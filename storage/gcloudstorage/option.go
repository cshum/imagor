package gcloudstorage

import (
	"strings"
	"time"
)

// Option GCloudStorage option
type Option func(h *GCloudStorage)

// WithBaseDir with base dir option
func WithBaseDir(baseDir string) Option {
	return func(s *GCloudStorage) {
		if baseDir != "" {
			baseDir = strings.Trim(baseDir, "/")
			s.BaseDir = baseDir
		}
	}
}

// WithPathPrefix with path prefix option
func WithPathPrefix(prefix string) Option {
	return func(s *GCloudStorage) {
		if prefix != "" {
			prefix = "/" + strings.Trim(prefix, "/")
			if prefix != "/" {
				prefix += "/"
			}
			s.PathPrefix = prefix
		}
	}
}

// WithACL with ACL option
// https://cloud.google.com/storage/docs/json_api/v1/objects/insert
func WithACL(acl string) Option {
	return func(h *GCloudStorage) {
		h.ACL = acl
	}
}

// WithSafeChars with safe chars option
func WithSafeChars(chars string) Option {
	return func(h *GCloudStorage) {
		if chars != "" {
			h.SafeChars = chars
		}
	}
}

// WithExpiration with modified time expiration option
func WithExpiration(exp time.Duration) Option {
	return func(h *GCloudStorage) {
		if exp > 0 {
			h.Expiration = exp
		}
	}
}
