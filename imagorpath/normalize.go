package imagorpath

import (
	"path"
	"strings"
)

const upperhex = "0123456789ABCDEF"

type EscapeByte func(byte) bool

func DefaultEscapeByte(c byte) bool {
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
				t[j+1] = upperhex[c>>4]
				t[j+2] = upperhex[c&15]
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

// Normalize imagor path to be file path friendly,
// optional escapeByte func for custom safe characters
func Normalize(image string, escapeByte EscapeByte) string {
	image = path.Clean(image)
	image = strings.Trim(image, "/")
	if escapeByte == nil {
		return escape(image, DefaultEscapeByte)
	} else {
		return escape(image, escapeByte)
	}
}
