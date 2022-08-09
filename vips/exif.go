package vips

// #include <vips/vips.h>
import "C"
import (
	"strconv"
	"strings"
)

var exifTags = []string{
	"exif-ifd0-Make",
	"exif-ifd0-Model",
	"exif-ifd0-Orientation",
	"exif-ifd0-XResolution",
	"exif-ifd0-YResolution",
	"exif-ifd0-ResolutionUnit",
	"exif-ifd0-Software",
	"exif-ifd0-DateTime",
	"exif-ifd0-YCbCrPositioning",
	"exif-ifd1-Compression",
	"exif-ifd2-ExposureTime",
	"exif-ifd2-FNumber",
	"exif-ifd2-ExposureProgram",
	"exif-ifd2-ISOSpeedRatings",
	"exif-ifd2-ExifVersion",
	"exif-ifd2-DateTimeOriginal",
	"exif-ifd2-DateTimeDigitized",
	"exif-ifd2-ComponentsConfiguration",
	"exif-ifd2-ShutterSpeedValue",
	"exif-ifd2-ApertureValue",
	"exif-ifd2-BrightnessValue",
	"exif-ifd2-ExposureBiasValue",
	"exif-ifd2-MeteringMode",
	"exif-ifd2-Flash",
	"exif-ifd2-FocalLength",
	"exif-ifd2-SubjectArea",
	"exif-ifd2-MakerNote",
	"exif-ifd2-SubSecTimeOriginal",
	"exif-ifd2-SubSecTimeDigitized",
	"exif-ifd2-ColorSpace",
	"exif-ifd2-PixelXDimension",
	"exif-ifd2-PixelYDimension",
	"exif-ifd2-SensingMethod",
	"exif-ifd2-SceneType",
	"exif-ifd2-ExposureMode",
	"exif-ifd2-WhiteBalance",
	"exif-ifd2-FocalLengthIn35mmFilm",
	"exif-ifd2-SceneCaptureType",
	"exif-ifd3-GPSLatitudeRef",
	"exif-ifd3-GPSLatitude",
	"exif-ifd3-GPSLongitudeRef",
	"exif-ifd3-GPSLongitude",
	"exif-ifd3-GPSAltitudeRef",
	"exif-ifd3-GPSAltitude",
	"exif-ifd3-GPSSpeedRef",
	"exif-ifd3-GPSSpeed",
	"exif-ifd3-GPSImgDirectionRef",
	"exif-ifd3-GPSImgDirection",
	"exif-ifd3-GPSDestBearingRef",
	"exif-ifd3-GPSDestBearing",
	"exif-ifd3-GPSDateStamp",
}

var exifInt = map[string]bool{
	"Orientation":           true,
	"ResolutionUnit":        true,
	"YCbCrPositioning":      true,
	"Compression":           true,
	"ExposureProgram":       true,
	"ISOSpeedRatings":       true,
	"MeteringMode":          true,
	"Flash":                 true,
	"ColorSpace":            true,
	"PixelXDimension":       true,
	"PixelYDimension":       true,
	"SensingMethod":         true,
	"ExposureMode":          true,
	"WhiteBalance":          true,
	"FocalLengthIn35mmFilm": true,
	"SceneCaptureType":      true,
}

func exifStringShort(s string) string {
	i := strings.Index(s, " (")
	if i > 0 {
		return s[:i]
	}
	return s
}

func vipsImageGetEXIF(img *C.VipsImage) map[string]any {
	var exif = map[string]any{}
	for _, tag := range exifTags {
		name := tag[10:]
		value := strings.TrimSpace(exifStringShort(vipsGetMetaString(img, tag)))
		if value == "" {
			continue
		}
		if exifInt[name] {
			exif[name], _ = strconv.Atoi(value)
		} else {
			exif[name] = value
		}
	}
	return exif
}
