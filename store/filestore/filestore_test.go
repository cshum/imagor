package filestore

import (
	"regexp"
	"testing"
)

func TestFileStore_Path(t *testing.T) {
	tests := []struct {
		name       string
		root       string
		baseURI    string
		image      string
		blacklist  *regexp.Regexp
		expected   string
		expectedOk bool
	}{
		{
			name:       "path under",
			root:       "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar",
			expected:   "/home/imagor/bar",
			expectedOk: true,
		},
		{
			name:       "path under",
			root:       "/home/imagor",
			image:      "/foo/bar",
			expected:   "/home/imagor/foo/bar",
			expectedOk: true,
		},
		{
			name:       "path not under",
			root:       "/home/imagor",
			baseURI:    "/foo",
			image:      "/fooo/bar",
			expectedOk: false,
		},
		{
			name:       "path not under must not escalate",
			root:       "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/../../etc/passwd",
			expectedOk: false,
		},
		{
			name:       "path under must not escalate",
			root:       "/home/imagor",
			baseURI:    "/",
			image:      "/../../etc/passwd",
			expected:   "/home/imagor/etc/passwd",
			expectedOk: true,
		},
		{
			name:       "path under must not expose sensitive",
			root:       "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar/.git",
			expectedOk: false,
		},
		{
			name:       "path under must not expose sensitive",
			root:       "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar/.git/logs/HEAD",
			expectedOk: false,
		},
		{
			name:       "path under",
			root:       "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar/abc/def/ghi.txt",
			expected:   "/home/imagor/bar/abc/def/ghi.txt",
			expectedOk: true,
		},
		{
			name:       "path under blacklist",
			root:       "/home/imagor",
			baseURI:    "/foo",
			image:      "/foo/bar/abc/def/ghi.txt",
			blacklist:  regexp.MustCompile("\\.txt"),
			expectedOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, ok := New(tt.root,
				WithBaseURI(tt.baseURI),
				WithBlacklist(tt.blacklist),
			).Path(tt.image)
			if res != tt.expected || ok != tt.expectedOk {
				t.Errorf(" = %s,%v want %s,%v", res, ok, tt.expected, tt.expectedOk)
			}
		})
	}
}
