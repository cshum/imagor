package vipsprocessor

import "C"
import (
	"strconv"
	"strings"
)

// exifTags to extract, true to cast as int
var exifTags = map[string]bool{
	"exif-ifd0-Orientation":           true,
	"exif-ifd0-ResolutionUnit":        true,
	"exif-ifd0-YCbCrPositioning":      true,
	"exif-ifd1-Compression":           true,
	"exif-ifd2-ExposureProgram":       true,
	"exif-ifd2-ISOSpeedRatings":       true,
	"exif-ifd2-MeteringMode":          true,
	"exif-ifd2-Flash":                 true,
	"exif-ifd2-ColorSpace":            true,
	"exif-ifd2-PixelXDimension":       true,
	"exif-ifd2-PixelYDimension":       true,
	"exif-ifd2-SensingMethod":         true,
	"exif-ifd2-ExposureMode":          true,
	"exif-ifd2-WhiteBalance":          true,
	"exif-ifd2-FocalLengthIn35mmFilm": true,
	"exif-ifd2-SceneCaptureType":      true,
}

func exifStringShort(s string) string {
	i := strings.Index(s, " (")
	if i > -1 {
		return s[:i]
	}
	return s
}

func extractExif(rawExif map[string]string) map[string]any {
	var exif = map[string]any{}
	for tag, value := range rawExif {
		if len(tag) < 10 {
			continue
		}
		name := tag[10:]
		value = strings.TrimSpace(exifStringShort(value))
		if value == "" {
			continue
		}
		if exifTags[tag] {
			val, err := strconv.Atoi(value)
			if err == nil {
				exif[name] = val
			} else {
				exif[name] = value
			}
		} else {
			exif[name] = value
		}
	}
	return exif
}
