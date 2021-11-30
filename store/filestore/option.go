package filestore

import (
	"regexp"
	"strings"
)

type Option func(h *FileStore)

func WithBaseURI(baseURI string) Option {
	return func(s *FileStore) {
		baseURI = "/" + strings.Trim(baseURI, "/")
		if baseURI != "/" {
			baseURI += "/"
		}
		s.BaseURI = baseURI
	}
}

func WithBlacklist(blacklist *regexp.Regexp) Option {
	return func(s *FileStore) {
		if blacklist != nil {
			s.Blacklists = append(s.Blacklists, blacklist)
		}
	}
}
