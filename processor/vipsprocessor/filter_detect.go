package vipsprocessor

import (
	"context"
	"math"

	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
)

// detectionsFilter draws bounding box outlines for all regions returned by the
// configured Detector onto the image.  It is intended for visual debugging.
//
// Usage:  filters:detections()
//
//	filters:detections(color)  e.g. detections(ff0000)
//
// color — any CSS colour name or hex string accepted by getColor (default: 00ff00, green)
//
// No-op when no Detector is configured.
func (v *Processor) detectionsFilter(
	ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string,
) (err error) {
	if v.Detector == nil {
		return
	}

	color := "00ff00"
	if len(args) >= 1 && args[0] != "" {
		color = args[0]
	}

	regions := v.detectRegions(ctx, img)
	if len(regions) == 0 {
		return
	}

	c := getColor(img, color)
	// DrawRect ink must have exactly as many elements as the image has bands.
	// getColor returns 3 (RGB); pad to 4 with full-opacity alpha when needed.
	for len(c) < img.Bands() {
		c = append(c, 255)
	}
	w := img.Width()
	h := img.PageHeight()

	for _, r := range regions {
		left := int(math.Round(r.Left * float64(w)))
		top := int(math.Round(r.Top * float64(h)))
		rw := int(math.Round(r.Right*float64(w))) - left
		rh := int(math.Round(r.Bottom*float64(h))) - top
		if rw <= 0 || rh <= 0 {
			continue
		}
		if err = img.DrawRect(c, left, top, rw, rh, &vips.DrawRectOptions{Fill: false}); err != nil {
			return
		}
	}
	return
}
