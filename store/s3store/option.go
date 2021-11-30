package s3store

import (
	"strings"
)

type Option func(h *s3Store)

func WithBaseURI(baseURI string) Option {
	return func(s *s3Store) {
		baseURI = "/" + strings.Trim(baseURI, "/")
		if baseURI != "/" {
			baseURI += "/"
		}
		s.BaseURI = baseURI
	}
}

func WithBaseDir(baseDir string) Option {
	return func(s *s3Store) {
		baseDir = "/" + strings.Trim(baseDir, "/")
		if baseDir != "/" {
			baseDir += "/"
		}
		s.BaseURI = baseDir
	}
}
