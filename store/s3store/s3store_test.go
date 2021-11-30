package s3store

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"testing"
)

func TestS3Store_Path(t *testing.T) {
	tests := []struct {
		name           string
		bucket         string
		baseDir        string
		baseURI        string
		image          string
		expectedPath   string
		expectedBucket string
		expectedOk     bool
	}{
		{
			name:           "path under with base uri",
			bucket:         "mybucket",
			baseDir:        "/home/imagor",
			baseURI:        "/foo",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "/home/imagor/bar",
			expectedOk:     true,
		},
		{
			name:           "path under no base uri",
			bucket:         "mybucket",
			baseDir:        "/home/imagor",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "/home/imagor/foo/bar",
			expectedOk:     true,
		},
		{
			name:           "path not under",
			bucket:         "mybucket",
			baseDir:        "/home/imagor",
			baseURI:        "/foo",
			image:          "/fooo/bar",
			expectedBucket: "mybucket",
			expectedOk:     false,
		},
		{
			name:           "extract bucket path under",
			bucket:         "mybucket/home/imagor",
			baseURI:        "/foo",
			image:          "/foo/bar",
			expectedBucket: "mybucket",
			expectedPath:   "/home/imagor/bar",
			expectedOk:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, err := session.NewSession()
			if err != nil {
				t.Error(err)
			}
			var opts []Option
			if tt.baseURI != "" {
				opts = append(opts, WithBaseURI(tt.baseURI))
			}
			if tt.baseDir != "" {
				opts = append(opts, WithBaseDir(tt.baseDir))
			}
			s := New(sess, tt.bucket, opts...)
			res, ok := s.Path(tt.image)
			if res != tt.expectedPath || ok != tt.expectedOk || s.Bucket != tt.expectedBucket {
				t.Errorf("= %s,%s,%v want %s,%s,%v", tt.bucket, res, ok, tt.expectedBucket, tt.expectedPath, tt.expectedOk)
			}
		})
	}
}
