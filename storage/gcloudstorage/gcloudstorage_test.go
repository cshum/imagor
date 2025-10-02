package gcloudstorage

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/cshum/imagor"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCloudStorage_Path(t *testing.T) {

	tests := []struct {
		name           string
		bucket         string
		baseDir        string
		baseURI        string
		image          string
		safeChars      string
		expectedPath   string
		expectedBucket string
		expectedOk     bool
	}{
		{
			name:         "defaults ok",
			image:        "/foo/bar",
			expectedPath: "foo/bar",
			expectedOk:   true,
		},
		{
			name:         "escape unsafe chars",
			image:        "/foo/b{:}ar",
			expectedPath: "foo/b%7B%3A%7Dar",
			expectedOk:   true,
		},
		{
			name:         "escape safe chars",
			image:        "/foo/b{:}ar",
			expectedPath: "foo/b{%3A}ar",
			safeChars:    "{}",
			expectedOk:   true,
		},
		{
			name:         "no-op safe chars",
			image:        "/foo/b{:}ar",
			expectedPath: "foo/b{:}ar",
			safeChars:    "--",
			expectedOk:   true,
		},
		{
			name:         "path under with base uri",
			baseDir:      "home/imagor",
			baseURI:      "/foo",
			image:        "/foo/bar",
			expectedPath: "home/imagor/bar",
			expectedOk:   true,
		},
		{
			name:         "path under no base uri",
			baseDir:      "/home/imagor",
			image:        "/foo/bar",
			expectedPath: "home/imagor/foo/bar",
			expectedOk:   true,
		},
		{
			name:       "path not under",
			baseDir:    "/home/imagor",
			baseURI:    "/foo",
			image:      "/fooo/bar",
			expectedOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.baseURI != "" {
				opts = append(opts, WithPathPrefix(tt.baseURI))
			}
			if tt.baseDir != "" {
				opts = append(opts, WithBaseDir(tt.baseDir))
			}
			opts = append(opts, WithSafeChars(tt.safeChars))

			s := New(nil, tt.bucket, opts...)
			res, ok := s.Path(tt.image)
			if res != tt.expectedPath || ok != tt.expectedOk {
				t.Errorf("= %s,%v want %s,%v", res, ok, tt.expectedPath, tt.expectedOk)
			}
		})
	}
}

func TestCRUD(t *testing.T) {
	srv := fakestorage.NewServer([]fakestorage.Object{{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "test",
			Name:       "placeholder",
		},
		Content: []byte(""),
	}})
	ctx := context.Background()
	r := (&http.Request{}).WithContext(ctx)
	s := New(srv.Client(), "test", WithPathPrefix("/foo"), WithACL("publicRead"))
	var err error

	_, err = s.Get(r, "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrInvalid, err)

	_, err = s.Stat(ctx, "/bar/fooo/asdf")
	assert.Equal(t, imagor.ErrInvalid, err)

	_, err = s.Stat(ctx, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	b, err := s.Get(r, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	_, err = s.Stat(ctx, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	assert.ErrorIs(t, s.Put(ctx, "/bar/fooo/asdf", imagor.NewBlobFromBytes([]byte("bar"))), imagor.ErrInvalid)

	blob := imagor.NewBlobFromBytes([]byte("bar"))

	require.NoError(t, s.Put(ctx, "/foo/fooo/asdf", blob))

	stat, err := s.Stat(ctx, "/foo/fooo/asdf")
	require.NoError(t, err)
	assert.True(t, stat.ModifiedTime.Before(time.Now()))
	assert.NotEmpty(t, stat.ETag)

	b, err = s.Get(r, "/foo/fooo/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))
	assert.NotEmpty(t, b.Stat)
	assert.Equal(t, stat.ModifiedTime, b.Stat.ModifiedTime)
	assert.Equal(t, stat.ETag, b.Stat.ETag)

	err = s.Delete(ctx, "/foo/fooo/asdf")
	require.NoError(t, err)

	b, err = s.Get(r, "/foo/fooo/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)

	assert.Equal(t, imagor.ErrInvalid, s.Delete(ctx, "/bar/fooo/asdf"))

	require.NoError(t, s.Put(ctx, "/foo/boo/asdf", imagor.NewBlobFromBytes([]byte("bar"))))
}

func TestExpiration(t *testing.T) {
	srv := fakestorage.NewServer([]fakestorage.Object{{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "test",
			Name:       "placeholder",
		},
		Content: []byte(""),
	}})
	s := New(srv.Client(), "test", WithExpiration(time.Millisecond*10))
	ctx := context.Background()
	var err error

	_, err = s.Get(&http.Request{}, "/foo/bar/asdf")
	assert.Equal(t, imagor.ErrNotFound, err)
	blob := imagor.NewBlobFromBytes([]byte("bar"))
	require.NoError(t, s.Put(ctx, "/foo/bar/asdf", blob))
	b, err := s.Get(&http.Request{}, "/foo/bar/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))

	time.Sleep(time.Second)
	_, err = s.Get(&http.Request{}, "/foo/bar/asdf")
	require.ErrorIs(t, err, imagor.ErrExpired)
}

func TestContextCancel(t *testing.T) {
	srv := fakestorage.NewServer([]fakestorage.Object{{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "test",
			Name:       "placeholder",
		},
		Content: []byte(""),
	}})
	s := New(srv.Client(), "test")
	ctx, cancel := context.WithCancel(context.Background())
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	require.NoError(t, err)
	blob := imagor.NewBlobFromBytes([]byte("bar"))
	require.NoError(t, s.Put(ctx, "/foo/bar/asdf", blob))
	b, err := s.Get(r, "/foo/bar/asdf")
	require.NoError(t, err)
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, "bar", string(buf))
	cancel()
	b, err = s.Get(r, "/foo/bar/asdf")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGCloudStorage_GzipContentEncoding(t *testing.T) {
	// Test that gzip-compressed objects don't cause fanout buffer size issues
	
	// Create properly gzipped content
	originalContent := "this content is longer than 20 bytes when decompressed and should not be truncated"
	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	_, err := gzipWriter.Write([]byte(originalContent))
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())
	
	gzippedContent := gzipBuf.Bytes()
	
	srv := fakestorage.NewServer([]fakestorage.Object{{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName:      "test",
			Name:           "test-gzip",
			ContentEncoding: "gzip",
			Size:           int64(len(gzippedContent)), // Actual compressed size
		},
		Content: gzippedContent,
	}})
	
	ctx := context.Background()
	r := (&http.Request{}).WithContext(ctx)
	s := New(srv.Client(), "test")
	
	// Test the fix - should not truncate despite size mismatch
	b, err := s.Get(r, "/test-gzip")
	require.NoError(t, err)
	
	buf, err := b.ReadAll()
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(buf))
	
	// Verify blob stats are still set correctly
	assert.NotNil(t, b.Stat)
	// The stat size will still show compressed size, which is correct behavior
	assert.Equal(t, int64(len(gzippedContent)), b.Stat.Size)
	
	// The key fix is that content is fully readable despite size mismatch
	// (the fanout optimization is disabled internally for gzip content)
}
