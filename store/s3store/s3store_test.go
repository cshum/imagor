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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, err := session.NewSession()
			if err != nil {
				t.Error(err)
			}
			var s *S3Store
			var opts []Option
			if tt.baseURI != "" {
				opts = append(opts, WithBaseURI(tt.baseURI))
			}
			if tt.baseDir != "" {
				opts = append(opts, WithBaseDir(tt.baseDir))
			}
			res, ok := New(sess, tt.bucket, opts...).Path(tt.image)
			if res != tt.expectedPath || ok != tt.expectedOk || s.Bucket != tt.expectedBucket {
				t.Errorf("= %s,%s,%v want %s,%s,%v", tt.bucket, res, ok, tt.expectedBucket, tt.expectedPath, tt.expectedOk)
			}
		})
	}
}
