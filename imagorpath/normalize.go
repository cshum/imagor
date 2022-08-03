package imagorpath

import (
	"path"
	"strings"
)

const upperHex = "0123456789ABCDEF"

type SafeChars interface {
	ShouldEscape(c byte) bool
}

var defaultSafeChars = NewSafeChars("")

func NewSafeChars(safechars string) SafeChars {
	s := &safeChars{safeChars: map[byte]bool{}}
	for _, c := range safechars {
		s.safeChars[byte(c)] = true
		s.hasCustom = true
	}
	return s
}

type safeChars struct {
	hasCustom bool
	safeChars map[byte]bool
}

func (s *safeChars) ShouldEscape(c byte) bool {
	// alphanum
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}
	switch c {
	case '/': // should not escape path segment
		return false
	case '-', '_', '.', '~': // Unreserved characters
		return false
	}
	if s.hasCustom && s.safeChars[c] {
		// safe chars from config
		return false
	}
	// Everything else must be escaped.
	return true
}

// extracted from url.escape plus allowing custom shouldEscape func
func escape(s string, shouldEscape func(c byte) bool) string {
	spaceCount, hexCount := 0, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' {
			spaceCount++
		} else if shouldEscape(c) {
			hexCount++
		}
	}

	if spaceCount == 0 && hexCount == 0 {
		return s
	}

	var buf [64]byte
	var t []byte

	required := len(s) + 2*hexCount
	if required <= len(buf) {
		t = buf[:required]
	} else {
		t = make([]byte, required)
	}

	if hexCount == 0 && shouldEscape(' ') {
		copy(t, s)
		for i := 0; i < len(s); i++ {
			if s[i] == ' ' {
				t[i] = '+'
			}
		}
		return string(t)
	}

	j := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			if c != ' ' {
				t[j] = '%'
				t[j+1] = upperHex[c>>4]
				t[j+2] = upperHex[c&15]
				j += 3
			} else {
				t[j] = '+'
				j++
			}
		} else {
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

var breaksCleaner = strings.NewReplacer(
	"\r\n", "",
	"\r", "",
	"\n", "",
	"\v", "",
	"\f", "",
	"\u0085", "",
	"\u2028", "",
	"\u2029", "",
)

// Normalize imagor path to be file path friendly,
// optional escapeByte func for custom SafeChars
func Normalize(image string, safeChars SafeChars) string {
	image = path.Clean(image)
	image = breaksCleaner.Replace(image)
	image = strings.Trim(image, "/")
	if safeChars == nil {
		return escape(image, defaultSafeChars.ShouldEscape)
	} else {
		return escape(image, safeChars.ShouldEscape)
	}
}
