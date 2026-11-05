package svg

import (
	"regexp"
	"strings"
)

// These reference implementations preserve the pre-optimization behavior.
// Keep their logic independent of the optimized helpers under test.

func legacyDecodeEntities(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		c := s[i]
		if c != '&' {
			b.WriteByte(c)
			i++
			continue
		}
		semi := strings.IndexByte(s[i:], ';')
		if semi < 0 || semi > 12 {
			b.WriteByte(c)
			i++
			continue
		}
		entity := s[i+1 : i+semi]
		i += semi + 1
		switch entity {
		case "amp":
			b.WriteByte('&')
		case "lt":
			b.WriteByte('<')
		case "gt":
			b.WriteByte('>')
		case "quot":
			b.WriteByte('"')
		case "apos":
			b.WriteByte('\'')
		default:
			if r, ok := decodeNumericEntity(entity); ok {
				b.WriteRune(r)
			}
			// Unknown entities are dropped; encoding/xml would have failed
			// on them anyway unless they are predefined.
		}
	}
	return b.String()
}

func legacyRasterDataURI(value string) bool {
	lower := strings.ToLower(value)
	rest, ok := strings.CutPrefix(lower, "data:image/")
	if !ok {
		return false
	}
	for _, t := range imageRasterTypes {
		if after, ok := strings.CutPrefix(rest, t); ok && after != "" {
			switch after[0] {
			case ';', ',':
				return true
			}
		}
	}
	return false
}

func legacyHrefValue(value string) (string, bool) {
	v := strings.TrimSpace(legacyDecodeEntities(value))
	if !strings.HasPrefix(v, "#") || len(v) == 1 {
		return "", false
	}
	if strings.ContainsAny(v, " \t\r\n\"'<>\\") {
		return "", false
	}
	return v, true
}

func legacyImageHrefValue(value string) (string, bool) {
	v := strings.TrimSpace(legacyDecodeEntities(value))
	if legacyRasterDataURI(v) && !strings.ContainsAny(v, "<>\"'\\") {
		return v, true
	}
	return legacyHrefValue(v)
}

func legacyHasScheme(s string) (string, bool) {
	cleaned := strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f {
			return -1
		}
		return r
	}, strings.ToLower(s))
	m := legacySchemeRe.FindStringSubmatch(cleaned)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func legacyStyleValue(value string) (string, bool) {
	v := strings.TrimSpace(legacyDecodeEntities(value))
	if styleForbiddenRe.MatchString(v) {
		return "", false
	}
	for _, m := range styleURLRe.FindAllStringSubmatch(v, -1) {
		ref := strings.TrimSpace(m[1])
		if !strings.HasPrefix(ref, "#") || len(ref) == 1 {
			return "", false
		}
	}
	return v, true
}

func legacyURLAttrValue(value string) (string, bool) {
	v := strings.TrimSpace(legacyDecodeEntities(value))
	if !anyURLRe.MatchString(v) {
		if _, ok := legacyHasScheme(v); ok {
			return "", false
		}
		if strings.ContainsAny(v, "/\\") {
			return "", false
		}
		return v, true
	}
	for _, m := range styleURLRe.FindAllStringSubmatch(v, -1) {
		ref := strings.TrimSpace(m[1])
		if !strings.HasPrefix(ref, "#") || len(ref) == 1 {
			return "", false
		}
	}
	if strings.ContainsAny(v, "\\") || strings.Contains(strings.ToLower(v), "@import") {
		return "", false
	}
	return v, true
}

var legacySchemeRe = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.\-]*):`)
