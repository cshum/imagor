package imagorpath

import (
	"path"
	"strings"
)

// ConfineJoin joins a storage key under baseDir and reports whether it stays there.
func ConfineJoin(baseDir, key string) (string, bool) {
	// Refused even with no base dir: clamping it would serve the wrong object.
	if key == ".." || strings.HasPrefix(key, "../") || strings.HasPrefix(key, "/../") {
		return "", false
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
