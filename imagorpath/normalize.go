package imagorpath

import (
	"path"
	"strings"
)

const upperhex = "0123456789ABCDEF"

func defaultShouldEscape(c byte) bool {
	// alphanum
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}
	switch c {
	case '/': // should not escape path segment
		return false
	case '-', '_', '.', '~': // Unreserved characters (mark)
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
		if shouldEscape(c) {
			if c == ' ' {
				spaceCount++
			} else {
				hexCount++
			}
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

	if hexCount == 0 {
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
		switch c := s[i]; {
		case c == ' ':
			t[j] = '+'
			j++
		case shouldEscape(c):
			t[j] = '%'
			t[j+1] = upperhex[c>>4]
			t[j+2] = upperhex[c&15]
			j += 3
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

// Normalize imagor path to be file path friendly,
// optional shouldEscape func to
func Normalize(image string, shouldEscape ...func(c byte) bool) string {
	image = path.Clean(image)
	image = strings.Trim(image, "/")
	if len(shouldEscape) == 0 {
		return escape(image, defaultShouldEscape)
	} else {
		for _, fn := range shouldEscape {
			image = escape(image, fn)
		}
		return image
	}
}
