package filestorage

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Option FileStorage option
type Option func(h *FileStorage)

// WithPathPrefix with path prefix option
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

// WithBlacklist with regexp path blacklist option
func WithBlacklist(blacklist *regexp.Regexp) Option {
	return func(s *FileStorage) {
		if blacklist != nil {
			s.Blacklists = append(s.Blacklists, blacklist)
		}
	}
}

// WithMkdirPermission with mkdir permission option
func WithMkdirPermission(perm string) Option {
	return func(h *FileStorage) {
		if perm != "" {
			if fm, err := strconv.ParseUint(perm, 0, 32); err == nil {
				h.MkdirPermission = os.FileMode(fm)
			}
		}
	}
}

// WithWritePermission with write permission option
func WithWritePermission(perm string) Option {
	return func(h *FileStorage) {
		if perm != "" {
			if fm, err := strconv.ParseUint(perm, 0, 32); err == nil {
				h.WritePermission = os.FileMode(fm)
			}
		}
	}
}

// WithSaveErrIfExists with save error if exists option
func WithSaveErrIfExists(saveErrIfExists bool) Option {
	return func(h *FileStorage) {
		h.SaveErrIfExists = saveErrIfExists
	}
}

// WithSafeChars with safe chars option
func WithSafeChars(chars string) Option {
	return func(h *FileStorage) {
		if chars != "" {
			h.SafeChars = chars
		}
	}
}

// WithExpiration with last modified expiration option
func WithExpiration(exp time.Duration) Option {
	return func(h *FileStorage) {
		if exp > 0 {
			h.Expiration = exp
		}
	}
}
