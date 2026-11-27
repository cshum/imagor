package filestorage

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStore_Path(t *testing.T) {
	tests := []struct {
		name       string
		baseDir    string
		baseURI    string
		image      string
		blacklist  *regexp.Regexp
		safeChars  string
		expected   string
		expectedOk bool
	}{
		{
			name:       "escape unsafe chars",
			baseDir:    "/home/imagor",
			image:      "/foo/b{:}ar",
			expected:   "/home/imagor/foo/b%7B%3A%7Dar",
			expectedOk: true,
		},
		{
			name:       "escape safe chars",
			baseDir:    "/home/imagor",
			image:      "/foo/b{:}ar",
			expected:   "/home/imagor/foo/b{%3A}ar",
			safeChars:  "{}",
			expectedOk: true,
		},
		{
			name:       "no-op safe chars",
			baseDir:    "/home/imagor",
			image:      "/foo/b{:}ar",
			expected:   "/home/imagor/foo/b{:}ar",
			safeChars:  "--",
			expectedOk: true,
		},
		{
			name:       "path under with base uri",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar",
			expected:   "/home/imagor/bar",
			expectedOk: true,
		},
		{
			name:       "path under no base uri",
			baseDir:    "/home/imagor",
			image:      "/foo/bar",
			expected:   "/home/imagor/foo/bar",
			expectedOk: true,
		},
		{
			name:       "path not under",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/fooo/bar",
			expectedOk: false,
		},
		{
			name:       "path not under must not escalate",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/../../etc/passwd",
			expectedOk: false,
		},
		{
			name:       "path under must not escalate",
			baseDir:    "/home/imagor",
			baseURI:    "/",
			image:      "/../../etc/passwd",
			expected:   "/home/imagor/etc/passwd",
			expectedOk: true,
		},
		{
			name:       "path under must not expose sensitive",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar/.git",
			expectedOk: false,
		},
		{
			name:       "path under must not expose sensitive",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar/.git/logs/HEAD",
			expectedOk: false,
		},
		{
			name:       "path under",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar/abc/def/ghi.txt",
			expected:   "/home/imagor/bar/abc/def/ghi.txt",
			expectedOk: true,
		},
		{
			name:       "path under blacklist",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar/abc/def/ghi.txt",
			blacklist:  regexp.MustCompile("\\.txt"),
			expectedOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, ok := New(tt.baseDir,
				WithPathPrefix(tt.baseURI),
				WithBlacklist(tt.blacklist),
				WithSafeChars(tt.safeChars),
			).Path(tt.image)
			if res != tt.expected || ok != tt.expectedOk {
				t.Errorf(" = %s,%v want %s,%v", res, ok, tt.expected, tt.expectedOk)
			}
		})
	}
}

func TestFileStorage_Load_Save(t *testing.T) {
	ctx := context.Background()
	r := (&http.Request{}).WithContext(ctx)
	dir, err := os.MkdirTemp("", "imagor-test")
	require.NoError(t, err)

	t.Run("blacklisted path", func(t *testing.T) {
		s := New(dir)
		_, err = s.Get(r, "/abc/.git")
		assert.Equal(t, imagor.ErrInvalid, err)
		assert.Equal(t, imagor.ErrInvalid, s.Put(ctx, "/abc/.git", imagor.NewBlobFromBytes([]byte("boo"))))
	})
	t.Run("CRUD", func(t *testing.T) {
		s := New(dir, WithPathPrefix("/foo"), WithMkdirPermission("0755"), WithWritePermission("0666"))

		_, err := checkBlob(s.Get(r, "/bar/fooo/asdf"))
		assert.Equal(t, imagor.ErrInvalid, err)

		_, err = s.Stat(ctx, "/bar/fooo/asdf")
		assert.Equal(t, imagor.ErrInvalid, err)

		_, err = checkBlob(s.Get(r, "/foo/fooo/asdf"))
		assert.Equal(t, imagor.ErrNotFound, err)

		_, err = s.Stat(ctx, "/foo/fooo/asdf")
		assert.Equal(t, imagor.ErrNotFound, err)

		assert.ErrorIs(t, s.Put(ctx, "/bar/fooo/asdf", imagor.NewBlobFromBytes([]byte("bar"))), imagor.ErrInvalid)

		assert.Equal(t, imagor.ErrInvalid, s.Delete(ctx, "/bar/fooo/asdf"))

		blob := imagor.NewBlobFromBytes([]byte("bar"))

		require.NoError(t, s.Put(ctx, "/foo/fooo/asdf", blob))

		stat, err := s.Stat(ctx, "/foo/fooo/asdf")
		require.NoError(t, err)
		assert.True(t, stat.ModifiedTime.Before(time.Now()))

		b, err := checkBlob(s.Get(r, "/foo/fooo/asdf"))
		require.NoError(t, err)
		buf, err := b.ReadAll()
		require.NoError(t, err)
		assert.Equal(t, "bar", string(buf))

		stat, err = s.Stat(ctx, "/foo/fooo/asdf")
		require.NoError(t, err)
		assert.True(t, stat.ModifiedTime.Before(time.Now()))

		err = s.Delete(ctx, "/foo/fooo/asdf")
		require.NoError(t, err)

		b, err = checkBlob(s.Get(r, "/foo/fooo/asdf"))
		assert.Equal(t, imagor.ErrNotFound, err)

	})

	t.Run("save err if exists", func(t *testing.T) {
		s := New(dir, WithSaveErrIfExists(true))
		require.NoError(t, s.Put(ctx, "/foo/tar/asdf", imagor.NewBlobFromBytes([]byte("bar"))))
		assert.Error(t, s.Put(ctx, "/foo/tar/asdf", imagor.NewBlobFromBytes([]byte("boo"))))
		b, err := checkBlob(s.Get(r, "/foo/tar/asdf"))
		require.NoError(t, err)
		buf, err := b.ReadAll()
		require.NoError(t, err)
		assert.Equal(t, "bar", string(buf))
	})

	t.Run("expiration", func(t *testing.T) {
		s := New(dir, WithExpiration(time.Millisecond*100))
		var err error

		_, err = checkBlob(s.Get(r, "/foo/bar/asdf"))
		assert.Equal(t, imagor.ErrNotFound, err)
		blob := imagor.NewBlobFromBytes([]byte("bar"))
		require.NoError(t, s.Put(ctx, "/foo/bar/asdf", blob))
		b, err := checkBlob(s.Get(r, "/foo/bar/asdf"))
		require.NoError(t, err)
		buf, err := b.ReadAll()
		require.NoError(t, err)
		assert.Equal(t, "bar", string(buf))

		time.Sleep(time.Second)
		_, err = checkBlob(s.Get(r, "/foo/bar/asdf"))
		require.ErrorIs(t, err, imagor.ErrExpired)
	})
}

func checkBlob(blob *imagor.Blob, err error) (*imagor.Blob, error) {
	if blob != nil && err == nil {
		err = blob.Err()
	}
	return blob, err
}

// gatedReader signals on first read and then blocks until released, so a test
// can look at the target path while a Put is in flight.
type gatedReader struct {
	started chan struct{}
	release chan struct{}
	data    []byte
	err     error
	fired   bool
}

func (g *gatedReader) Read(p []byte) (int, error) {
	if !g.fired {
		g.fired = true
		close(g.started)
		<-g.release
	}
	if len(g.data) == 0 {
		if g.err != nil {
			return 0, g.err
		}
		return 0, io.EOF
	}
	n := copy(p, g.data)
	g.data = g.data[n:]
	return n, nil
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(info.Name(), ".imagor-tmp-") {
			t.Errorf("temporary file left behind: %s", p)
		}
		return nil
	})
	require.NoError(t, err)
}

func TestFileStorage_PutIsAtomic(t *testing.T) {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "imagor-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(dir)

	t.Run("the target appears only when it is complete", func(t *testing.T) {
		started, release := make(chan struct{}), make(chan struct{})
		blob := imagor.NewBlob(func() (io.ReadCloser, int64, error) {
			return io.NopCloser(&gatedReader{started: started, release: release, data: []byte("complete")}), 8, nil
		})
		done := make(chan error, 1)
		go func() { done <- s.Put(ctx, "/a/file.jpg", blob) }()
		<-started
		_, err := os.Stat(filepath.Join(dir, "a", "file.jpg"))
		assert.True(t, errors.Is(err, fs.ErrNotExist), "target must not exist while the body is still being written")
		close(release)
		require.NoError(t, <-done)
		buf, err := os.ReadFile(filepath.Join(dir, "a", "file.jpg"))
		require.NoError(t, err)
		assert.Equal(t, "complete", string(buf))
		assertNoTempFiles(t, dir)
	})

	t.Run("an existing target is replaced, never truncated in place", func(t *testing.T) {
		require.NoError(t, s.Put(ctx, "/b/file.jpg", imagor.NewBlobFromBytes([]byte("old"))))
		started, release := make(chan struct{}), make(chan struct{})
		blob := imagor.NewBlob(func() (io.ReadCloser, int64, error) {
			return io.NopCloser(&gatedReader{started: started, release: release, data: []byte("new")}), 3, nil
		})
		done := make(chan error, 1)
		go func() { done <- s.Put(ctx, "/b/file.jpg", blob) }()
		<-started
		buf, err := os.ReadFile(filepath.Join(dir, "b", "file.jpg"))
		require.NoError(t, err)
		assert.Equal(t, "old", string(buf), "a reader must see the old file until the new one is complete")
		close(release)
		require.NoError(t, <-done)
		buf, err = os.ReadFile(filepath.Join(dir, "b", "file.jpg"))
		require.NoError(t, err)
		assert.Equal(t, "new", string(buf))
		assertNoTempFiles(t, dir)
	})

	t.Run("a failed copy leaves nothing behind", func(t *testing.T) {
		started, release := make(chan struct{}), make(chan struct{})
		blob := imagor.NewBlob(func() (io.ReadCloser, int64, error) {
			return io.NopCloser(&gatedReader{started: started, release: release, data: []byte("half"), err: io.ErrUnexpectedEOF}), 9, nil
		})
		done := make(chan error, 1)
		go func() { done <- s.Put(ctx, "/c/file.jpg", blob) }()
		<-started
		close(release)
		assert.Error(t, <-done)
		_, err := os.Stat(filepath.Join(dir, "c", "file.jpg"))
		assert.True(t, errors.Is(err, fs.ErrNotExist), "a failed write must not leave a target")
		assertNoTempFiles(t, dir)
	})

	t.Run("save err if exists leaves the target untouched", func(t *testing.T) {
		exists := New(dir, WithSaveErrIfExists(true))
		require.NoError(t, exists.Put(ctx, "/d/file.jpg", imagor.NewBlobFromBytes([]byte("keep"))))
		assert.Error(t, exists.Put(ctx, "/d/file.jpg", imagor.NewBlobFromBytes([]byte("boo"))))
		buf, err := os.ReadFile(filepath.Join(dir, "d", "file.jpg"))
		require.NoError(t, err)
		assert.Equal(t, "keep", string(buf))
		assertNoTempFiles(t, dir)
	})
}

func TestFileStorage_PutKeepsModeAndReplacesLinks(t *testing.T) {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "imagor-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(dir)

	t.Run("an overwrite keeps the file's permissions", func(t *testing.T) {
		target := filepath.Join(dir, "mode", "file.jpg")
		require.NoError(t, s.Put(ctx, "/mode/file.jpg", imagor.NewBlobFromBytes([]byte("one"))))
		require.NoError(t, os.Chmod(target, 0o600))
		require.NoError(t, s.Put(ctx, "/mode/file.jpg", imagor.NewBlobFromBytes([]byte("two"))))
		st, err := os.Stat(target)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), st.Mode().Perm())
	})

	t.Run("a group writable mode survives an overwrite despite the umask", func(t *testing.T) {
		target := filepath.Join(dir, "mode2", "file.jpg")
		require.NoError(t, s.Put(ctx, "/mode2/file.jpg", imagor.NewBlobFromBytes([]byte("one"))))
		require.NoError(t, os.Chmod(target, 0o666))
		require.NoError(t, s.Put(ctx, "/mode2/file.jpg", imagor.NewBlobFromBytes([]byte("two"))))
		st, err := os.Stat(target)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o666), st.Mode().Perm())
	})

	t.Run("a symlink in the storage is replaced, not followed", func(t *testing.T) {
		outside := filepath.Join(dir, "outside.jpg")
		require.NoError(t, os.WriteFile(outside, []byte("untouched"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "link"), 0o755))
		target := filepath.Join(dir, "link", "file.jpg")
		require.NoError(t, os.Symlink(outside, target))

		require.NoError(t, s.Put(ctx, "/link/file.jpg", imagor.NewBlobFromBytes([]byte("new"))))

		buf, err := os.ReadFile(outside)
		require.NoError(t, err)
		assert.Equal(t, "untouched", string(buf), "a write must not follow a link out of the base dir")
		st, err := os.Lstat(target)
		require.NoError(t, err)
		assert.Zero(t, st.Mode()&os.ModeSymlink)
		buf, err = os.ReadFile(target)
		require.NoError(t, err)
		assert.Equal(t, "new", string(buf))
	})
}

func TestFileStorage_PutRejectsShortBody(t *testing.T) {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "imagor-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	s := New(dir)

	blob := imagor.NewBlob(func() (io.ReadCloser, int64, error) {
		return io.NopCloser(strings.NewReader("short")), 1024, nil
	})
	err = s.Put(ctx, "/short/file.jpg", blob)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "truncated write")

	_, statErr := os.Stat(filepath.Join(dir, "short", "file.jpg"))
	assert.True(t, errors.Is(statErr, fs.ErrNotExist), "a short body must not reach the target")
	assertNoTempFiles(t, dir)
}
