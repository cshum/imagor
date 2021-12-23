package filestore

import (
	"os"
	"regexp"
	"strconv"
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

func WithMkdirPermission(perm string) Option {
	return func(h *FileStore) {
		if perm != "" {
			if fm, err := strconv.ParseUint(perm, 0, 32); err == nil {
				h.MkdirPermission = os.FileMode(fm)
			}
		}
	}
}

func WithWritePermission(perm string) Option {
	return func(h *FileStore) {
		if perm != "" {
			if fm, err := strconv.ParseUint(perm, 0, 32); err == nil {
				h.WritePermission = os.FileMode(fm)
			}
		}
	}
}

func WithSaveErrIfExists(saveErrIfExists bool) Option {
	return func(h *FileStore) {
		h.SaveErrIfExists = saveErrIfExists
	}
}
