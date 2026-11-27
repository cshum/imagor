package imagorpath

import (
	"path"
	"strings"
)

// ConfineJoin joins a storage key under baseDir and reports whether it stays there.
func ConfineJoin(baseDir, key string) (string, bool) {
	// Refused, not clamped: a clamped key names a different object.
	depth := 0
	for _, seg := range strings.Split(strings.Trim(key, "/"), "/") {
		switch seg {
		case "", ".":
		case "..":
			if depth == 0 {
				return "", false
			}
			depth--
		default:
			depth++
		}
	}
	base := path.Clean("/" + strings.Trim(baseDir, "/"))
	joined := path.Clean(path.Join(base, "/"+strings.TrimPrefix(key, "/")))
	if joined == "/" {
		return "", false
	}
	if base == "/" {
		return strings.TrimPrefix(joined, "/"), true
	}
	if joined != base && !strings.HasPrefix(joined, base+"/") {
		return "", false
	}
	return strings.TrimPrefix(joined, "/"), true
}
