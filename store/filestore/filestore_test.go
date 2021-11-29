package filestore

import (
	"testing"
)

func TestFileStore_Path(t *testing.T) {
	tests := []struct {
		name       string
		root       string
		baseURI    string
		image      string
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, ok := New(tt.root, WithBaseURI(tt.baseURI)).Path(tt.image)
			if res != tt.expected {
				t.Errorf(" = %s want %s", res, tt.expected)
			}
			if ok != tt.expectedOk {
				t.Errorf(" = %v want %v", ok, tt.expectedOk)
			}
		})
	}
}
