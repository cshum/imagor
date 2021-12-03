package s3store

import (
	"strings"
)

type Option func(h *S3Store)

func WithBaseDir(baseDir string) Option {
	return func(s *S3Store) {
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
	return func(s *S3Store) {
		if prefix != "" {
			prefix = "/" + strings.Trim(prefix, "/")
			if prefix != "/" {
				prefix += "/"
			}
			s.PathPrefix = prefix
		}
	}
}
