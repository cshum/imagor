package filestore

import "regexp"

type Option func(h *fileStore)

func WithBasePath(basePath string) Option {
	return func(s *fileStore) {
		s.BasePath = basePath
	}
}

func WithBlacklist(blacklist *regexp.Regexp) Option {
	return func(s *fileStore) {
		if blacklist != nil {
			s.Blacklists = append(s.Blacklists, blacklist)
		}
	}
}
