package filestore

import (
	"regexp"
	"strings"
)

type Option func(h *FileStore)

func WithPathPrefix(prefix string) Option {
	return func(s *FileStore) {
		if prefix != "" {
			prefix = "/" + strings.Trim(prefix, "/")
			if prefix != "/" {
				prefix += "/"
			}
			s.PathPrefix = prefix
		}
	}
}

func WithBlacklist(blacklist *regexp.Regexp) Option {
	return func(s *FileStore) {
		if blacklist != nil {
			s.Blacklists = append(s.Blacklists, blacklist)
		}
	}
}
