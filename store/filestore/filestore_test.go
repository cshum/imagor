package filestore

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"
)

func TestFileStore_Path(t *testing.T) {
	tests := []struct {
		name       string
		baseDir    string
		baseURI    string
		image      string
		blacklist  *regexp.Regexp
		expected   string
		expectedOk bool
	}{
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
			).Path(tt.image)
			if res != tt.expected || ok != tt.expectedOk {
				t.Errorf(" = %s,%v want %s,%v", res, ok, tt.expected, tt.expectedOk)
			}
		})
	}
}

func TestFileStore_Load_Store(t *testing.T) {
	ctx := context.Background()
	dir, err := ioutil.TempDir("", "imagor-test")
	assert.NoError(t, err)

	t.Run("blacklisted path", func(t *testing.T) {
		s := New(dir)
		_, err = s.Load(&http.Request{}, "/abc/.git")
		assert.Equal(t, imagor.ErrNotFound, err)
		assert.Equal(t, imagor.ErrNotFound, s.Save(ctx, "/abc/.git", imagor.NewFileBytes([]byte("boo"))))
	})
	t.Run("insufficient permission", func(t *testing.T) {
		s := New(dir, WithMkdirPermission("0444"), WithWritePermission("0444"))
		_, err = s.Load(&http.Request{}, "/abc/.git")
		assert.Equal(t, imagor.ErrNotFound, err)
		assert.Error(t, s.Save(ctx, "/abc/fooo/asdf", imagor.NewFileBytes([]byte("boo"))))
	})
	t.Run("save and load", func(t *testing.T) {
		s := New(dir, WithMkdirPermission("0755"), WithWritePermission("0666"))
		_, err := s.Load(&http.Request{}, "/foo/fooo/asdf")
		assert.Equal(t, imagor.ErrNotFound, err)
		assert.NoError(t, s.Save(ctx, "/foo/fooo/asdf", imagor.NewFileBytes([]byte("bar"))))
		b, err := s.Load(&http.Request{}, "/foo/fooo/asdf")
		assert.NoError(t, err)
		buf, err := b.Bytes()
		assert.NoError(t, err)
		assert.Equal(t, "bar", string(buf))
	})

	t.Run("save err if exists", func(t *testing.T) {
		s := New(dir, WithSaveErrIfExists(true))
		assert.NoError(t, s.Save(ctx, "/foo/bar/asdf", imagor.NewFileBytes([]byte("bar"))))
		assert.Error(t, s.Save(ctx, "/foo/bar/asdf", imagor.NewFileBytes([]byte("boo"))))
		b, err := s.Load(&http.Request{}, "/foo/bar/asdf")
		assert.NoError(t, err)
		buf, err := b.Bytes()
		assert.NoError(t, err)
		assert.Equal(t, "bar", string(buf))
	})
}
