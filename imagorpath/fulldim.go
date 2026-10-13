package imagorpath

import (
	"regexp"
	"strconv"
	"strings"
)

// fullDimRegex matches a single dimension token: optionally a flip prefix -,
// then f or full, optionally followed by a negative integer offset
// e.g. f, f-20, full, full-20, -f, -full-20.
var fullDimRegex = regexp.MustCompile(`^(-?)(?:full|f)(-\d+)?$`)

// dimSegmentRegex matches a /‑separated WxH segment where either or both sides
// may be an f/full‑token or a plain integer.
var dimSegmentRegex = regexp.MustCompile(`^(-?(?:(?:full|f)(?:-\d+)?|\d*))x(-?(?:(?:full|f)(?:-\d+)?|\d*))$`)

// ResolveFullDim resolves a single dimension token against a parent pixel size.
// Tokens of the form f or f-NNN (with optional leading - for flip)
// resolve to parentDim - NNN. Any other token is returned unchanged.
func ResolveFullDim(token string, parentDim int) string {
	m := fullDimRegex.FindStringSubmatch(token)
	if m == nil {
		return token
	}
	flip := m[1]
	offset := 0
	if m[2] != "" {
		offset, _ = strconv.Atoi(m[2])
	}
	return flip + strconv.Itoa(parentDim+offset)
}

// ResolveFullDimensions rewrites f‑tokens in the WxH dimension segment of an
// imagor path, substituting the parent image's pixel dimensions before the path
// is parsed. Only the first WxH dimension segment is considered — it always
// appears before filters: in a valid imagor path. The function stops at the
// first dimSegmentRegex match or at a filters: prefix, ensuring nested layer
// paths inside filter arguments are not accidentally resolved at this level.
//
// This is how a nested path inherits the parent's dimensions: it lives here
// rather than in a processor so that every caller that resolves a nested path —
// the image() filter and a path: source alike — resolves f-tokens the same way.
func ResolveFullDimensions(imagorPath string, parentW, parentH int) string {
	start := 0
	for i := 0; i <= len(imagorPath); i++ {
		if i < len(imagorPath) && imagorPath[i] != '/' {
			continue
		}
		seg := imagorPath[start:i]
		// Stop before filters — nested layer paths inside filter arguments
		// must be resolved at their own processing level, not here.
		if strings.HasPrefix(seg, "filters:") {
			return imagorPath
		}
		if m := dimSegmentRegex.FindStringSubmatch(seg); m != nil {
			// Found the dimension segment. Resolve f-tokens if present.
			newLeft := ResolveFullDim(m[1], parentW)
			newRight := ResolveFullDim(m[2], parentH)
			if newLeft != m[1] || newRight != m[2] {
				return imagorPath[:start] + newLeft + "x" + newRight + imagorPath[i:]
			}
			return imagorPath
		}
		start = i + 1
	}
	return imagorPath
}
