package gcloudstorage

import (
	"strings"
	"time"
)

type Option func(h *GCloudStorage)

func WithBaseDir(baseDir string) Option {
	return func(s *GCloudStorage) {
		if baseDir != "" {
			baseDir = strings.Trim(baseDir, "/")
			s.BaseDir = baseDir
		}
	}
}

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

func WithACL(acl string) Option {
	// https://cloud.google.com/storage/docs/json_api/v1/objects/insert
	return func(h *GCloudStorage) {
		h.ACL = acl
	}
}

func WithSafeChars(chars string) Option {
	return func(h *GCloudStorage) {
		if chars != "" {
			h.SafeChars = chars
		}
	}
}

func WithExpiration(exp time.Duration) Option {
	return func(h *GCloudStorage) {
		if exp > 0 {
			h.Expiration = exp
		}
	}
}
