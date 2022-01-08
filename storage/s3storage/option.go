package s3storage

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
)

type Option func(h *S3Storage)

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

func WithACL(acl string) Option {
	return func(h *S3Storage) {
		if aclValuesMap[acl] {
			h.ACL = acl
		}
	}
}

func WithSafeChars(chars string) Option {
	return func(h *S3Storage) {
		if chars != "" {
			h.SafeChars = chars
		}
	}
}
