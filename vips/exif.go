package vips

// #include <vips/vips.h>
import "C"
import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// exifTags to extract, true to cast as int
var exifTags = map[string]bool{
	"exif-ifd0-Make":                    false,
	"exif-ifd0-Model":                   false,
	"exif-ifd0-Orientation":             true,
	"exif-ifd0-XResolution":             false,
	"exif-ifd0-YResolution":             false,
	"exif-ifd0-ResolutionUnit":          true,
	"exif-ifd0-Software":                false,
	"exif-ifd0-DateTime":                false,
	"exif-ifd0-YCbCrPositioning":        true,
	"exif-ifd0-Copyright":               false,
	"exif-ifd0-Artist":                  false,
	"exif-ifd1-Compression":             true,
	"exif-ifd2-ExposureTime":            false,
	"exif-ifd2-FNumber":                 false,
	"exif-ifd2-ExposureProgram":         true,
	"exif-ifd2-ISOSpeedRatings":         true,
	"exif-ifd2-ExifVersion":             false,
	"exif-ifd2-DateTimeOriginal":        false,
	"exif-ifd2-DateTimeDigitized":       false,
	"exif-ifd2-ComponentsConfiguration": false,
	"exif-ifd2-ShutterSpeedValue":       false,
	"exif-ifd2-ApertureValue":           false,
	"exif-ifd2-BrightnessValue":         false,
	"exif-ifd2-ExposureBiasValue":       false,
	"exif-ifd2-MeteringMode":            true,
	"exif-ifd2-Flash":                   true,
	"exif-ifd2-FocalLength":             false,
	"exif-ifd2-SubjectArea":             false,
	"exif-ifd2-MakerNote":               false,
	"exif-ifd2-SubSecTimeOriginal":      false,
	"exif-ifd2-SubSecTimeDigitized":     false,
	"exif-ifd2-ColorSpace":              true,
	"exif-ifd2-PixelXDimension":         true,
	"exif-ifd2-PixelYDimension":         true,
	"exif-ifd2-SensingMethod":           true,
	"exif-ifd2-SceneType":               false,
	"exif-ifd2-ExposureMode":            true,
	"exif-ifd2-WhiteBalance":            true,
	"exif-ifd2-FocalLengthIn35mmFilm":   true,
	"exif-ifd2-SceneCaptureType":        true,
	"exif-ifd3-GPSLatitudeRef":          false,
	"exif-ifd3-GPSLatitude":             false,
	"exif-ifd3-GPSLongitudeRef":         false,
	"exif-ifd3-GPSLongitude":            false,
	"exif-ifd3-GPSAltitudeRef":          false,
	"exif-ifd3-GPSAltitude":             false,
	"exif-ifd3-GPSSpeedRef":             false,
	"exif-ifd3-GPSSpeed":                false,
	"exif-ifd3-GPSImgDirectionRef":      false,
	"exif-ifd3-GPSImgDirection":         false,
	"exif-ifd3-GPSDestBearingRef":       false,
	"exif-ifd3-GPSDestBearing":          false,
	"exif-ifd3-GPSDateStamp":            false,
}

func exifStringShort(s string) string {
	i := strings.Index(s, " (")
	if i > -1 {
		return s[:i]
	}
	return s
}

func vipsImageGetExif(img *C.VipsImage) map[string]any {
	var exif = map[string]any{}
	for tag, atoi := range exifTags {
		name := tag[10:]
		value := strings.TrimSpace(exifStringShort(vipsGetMetaString(img, tag)))
		if value == "" {
			continue
		}
		if atoi {
			exif[name], _ = strconv.Atoi(value)
		} else {
			exif[name] = value
		}
	}
	return exif
}

func convertDecimalToDMS(degree float64) string {
	degree = math.Abs(degree)
	seconds := degree * 3600

	degrees := math.Floor(degree)
	seconds -= degrees * 3600

	minutes := math.Floor(seconds / 60)
	seconds -= minutes * 60

	return fmt.Sprintf("%.2f° %.2f' %.4f\"", degrees, minutes, seconds)
}

func isValidDegree(degree float64) bool {
	return degree >= -180 && degree <= 180
}

func vipsSetGeo(img *C.VipsImage, latitude, longitude float64) error {
	if isValidDegree(longitude) == false || isValidDegree(latitude) == false {
		return errors.New("invalid longitude or latitude")
	}

	lonRef := "E"
	if longitude < 0 {
		lonRef = "W"
	}

	latRef := "N"
	if latitude < 0 {
		latRef = "S"
	}

	geoMeta := map[string]string{
		"exif-ifd3-GPSLatitude":     convertDecimalToDMS(latitude),
		"exif-ifd3-GPSLatitudeRef":  latRef,
		"exif-ifd3-GPSLongitude":    convertDecimalToDMS(longitude),
		"exif-ifd3-GPSLongitudeRef": lonRef,
	}

	vipsSetMetaString(img, geoMeta)

	return nil
}
