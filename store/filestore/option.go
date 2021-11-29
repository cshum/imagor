package filestore

import "strings"

type Option func(h *fileStore)

func WithBaseURI(baseURI string) Option {
	return func(s *fileStore) {
		s.BaseURI = "/" + strings.Trim(baseURI, "/")
		if s.BaseURI != "/" {
			s.BaseURI += "/"
		}
	}
}
