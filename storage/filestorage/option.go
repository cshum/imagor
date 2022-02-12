package filestorage

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Option func(h *FileStorage)

func WithPathPrefix(prefix string) Option {
	return func(s *FileStorage) {
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
	return func(s *FileStorage) {
		if blacklist != nil {
			s.Blacklists = append(s.Blacklists, blacklist)
		}
	}
}

func WithMkdirPermission(perm string) Option {
	return func(h *FileStorage) {
		if perm != "" {
			if fm, err := strconv.ParseUint(perm, 0, 32); err == nil {
				h.MkdirPermission = os.FileMode(fm)
			}
		}
	}
}

func WithWritePermission(perm string) Option {
	return func(h *FileStorage) {
		if perm != "" {
			if fm, err := strconv.ParseUint(perm, 0, 32); err == nil {
				h.WritePermission = os.FileMode(fm)
			}
		}
	}
}

func WithSaveErrIfExists(saveErrIfExists bool) Option {
	return func(h *FileStorage) {
		h.SaveErrIfExists = saveErrIfExists
	}
}

func WithSafeChars(chars string) Option {
	return func(h *FileStorage) {
		if chars != "" {
			h.SafeChars = chars
		}
	}
}

func WithExpiration(exp time.Duration) Option {
	return func(h *FileStorage) {
		if exp > 0 {
			h.Expiration = exp
		}
	}
}
