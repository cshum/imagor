package vipsprocessor

import (
	"github.com/cshum/vipsgen/vips"
)

// linearRGB applies linear RGB transformation to an image
// Automatically handles alpha channel if present
func linearRGB(img *vips.Image, a, b []float64) error {
	if img.HasAlpha() {
		a = append(a, 1)
		b = append(b, 0)
	}
	return img.Linear(a, b, nil)
}

// isAnimated checks if an image is animated (multi-page)
func isAnimated(img *vips.Image) bool {
	return img.Height() > img.PageHeight()
}

// getAngle converts an integer angle to vips.Angle enum
func getAngle(angle int) vips.Angle {
	switch angle {
	case 90:
		return vips.AngleD270
	case 180:
		return vips.AngleD180
	case 270:
		return vips.AngleD90
	default:
		return vips.AngleD0
	}
}
