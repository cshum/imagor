package vipsprocessor

import "C"
import (
	"strconv"
	"strings"
)

// exifNames to extract, true to cast as int
var exifNames = map[string]bool{
	"Orientation":              true,
	"ResolutionUnit":           true,
	"FocalPlaneResolutionUnit": true,
	"YCbCrPositioning":         true,
	"Compression":              true,
	"exifExposureProgram":      true,
	"ISOSpeedRatings":          true,
	"MeteringMode":             true,
	"Flash":                    true,
	"ColorSpace":               true,
	"PixelXDimension":          true,
	"PixelYDimension":          true,
	"SensingMethod":            true,
	"ExposureMode":             true,
	"WhiteBalance":             true,
	"FocalLengthIn35mmFilm":    true,
	"SceneCaptureType":         true,
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
		if exifNames[name] {
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
