package s3store

import (
	"strings"
)

type Option func(h *S3Store)

func WithBaseURI(baseURI string) Option {
	return func(s *S3Store) {
		baseURI = "/" + strings.Trim(baseURI, "/")
		if baseURI != "/" {
			baseURI += "/"
		}
		s.BaseURI = baseURI
	}
}

func WithBaseDir(baseDir string) Option {
	return func(s *S3Store) {
		baseDir = "/" + strings.Trim(baseDir, "/")
		if baseDir != "/" {
			baseDir += "/"
		}
		s.BaseDir = baseDir
	}
}
