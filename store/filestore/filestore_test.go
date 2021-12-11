package filestore

import (
	"context"
	"github.com/cshum/imagor"
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
	dir, err := ioutil.TempDir("", "imagor-test")
	if err != nil {
		t.Fatal(err)
	}
	s := New(dir, WithMkdirPermission("0755"), WithWritePermission("0444"))
	b, err := s.Load(&http.Request{}, "/foo/fooo/asdf")
	if err != imagor.ErrNotFound {
		t.Errorf("= %v, want ErrNotFound", err)
	}
	ctx := context.Background()
	if err := s.Save(ctx, "/foo/fooo/asdf", []byte("bar")); err != nil {
		t.Error(err)
	}
	b, err = s.Load(&http.Request{}, "/foo/fooo/asdf")
	if err != nil {
		t.Error(err)
	}
	if string(b) != "bar" {
		t.Errorf(" = %s want %s", string(b), "bar")
	}
}
