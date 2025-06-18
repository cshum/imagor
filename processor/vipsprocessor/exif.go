package vipsprocessor

import "C"
import (
	"strings"
)

func exifStringShort(s string) string {
	i := strings.Index(s, " (")
	if i > -1 {
		return s[:i]
	}
	return s
}

func extractExif(rawExif map[string]string) map[string]string {
	var exif = map[string]string{}
	for tag, value := range rawExif {
		if len(tag) < 10 {
			continue
		}
		name := tag[10:]
		value = strings.TrimSpace(exifStringShort(value))
		if value == "" {
			continue
		}
		exif[name] = value
	}
	return exif
}
