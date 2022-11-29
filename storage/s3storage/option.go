package s3storage

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
	"time"
)

// Option S3Storage option
type Option func(h *S3Storage)

// WithBaseDir with base dir option
func WithBaseDir(baseDir string) Option {
	return func(s *S3Storage) {
		if baseDir != "" {
			baseDir = "/" + strings.Trim(baseDir, "/")
			if baseDir != "/" {
				baseDir += "/"
			}
			s.BaseDir = baseDir
		}
	}
}

// WithPathPrefix with path prefix option
func WithPathPrefix(prefix string) Option {
	return func(s *S3Storage) {
		if prefix != "" {
			prefix = "/" + strings.Trim(prefix, "/")
			if prefix != "/" {
				prefix += "/"
			}
			s.PathPrefix = prefix
		}
	}
}

var aclValuesMap = (func() map[string]bool {
	m := map[string]bool{}
	for _, acl := range s3.ObjectCannedACL_Values() {
		m[acl] = true
	}
	return m
})()

// WithACL with ACL option
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/acl-overview.html#canned-acl
func WithACL(acl string) Option {
	return func(h *S3Storage) {
		if aclValuesMap[acl] {
			h.ACL = acl
		}
	}
}

// WithSafeChars with safe chars option
func WithSafeChars(chars string) Option {
	return func(h *S3Storage) {
		if chars != "" {
			h.SafeChars = chars
		}
	}
}

// WithExpiration with modified time expiration option
func WithExpiration(exp time.Duration) Option {
	return func(h *S3Storage) {
		if exp > 0 {
			h.Expiration = exp
		}
	}
}
