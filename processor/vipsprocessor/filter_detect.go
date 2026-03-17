package vipsprocessor

import (
	"context"
	"math"
	"strconv"

	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
)

// detectionsFilter draws bounding boxes for all regions returned by the
// configured Detector onto the image.  It is intended for visual debugging.
//
// Usage:  filters:detections()
//
//	filters:detections(color)          e.g. detections(ff0000)
//	filters:detections(color,opacity)  e.g. detections(00ff00,60)
//
// color    — any CSS colour name or hex string accepted by getColor (default: 00ff00, green)
// opacity  — fill opacity 0-100 (default: 40); outline is always fully opaque
//
// No-op when no Detector is configured.
func (v *Processor) detectionsFilter(
	ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string,
) (err error) {
	if v.Detector == nil {
		return
	}

	color := "00ff00"
	opacity := 40
	if len(args) >= 1 && args[0] != "" {
		color = args[0]
	}
	if len(args) >= 2 {
		if n, e := strconv.Atoi(args[1]); e == nil && n >= 0 && n <= 100 {
			opacity = n
		}
	}

	regions := v.detectRegions(ctx, img)
	if len(regions) == 0 {
		return
	}

	c := getColor(img, color)
	fillAlpha := math.Round(float64(opacity) * 255 / 100)

	w := img.Width()
	h := img.PageHeight()

	for _, r := range regions {
		left := int(math.Round(r.Left * float64(w)))
		top := int(math.Round(r.Top * float64(h)))
		right := int(math.Round(r.Right * float64(w)))
		bottom := int(math.Round(r.Bottom * float64(h)))

		rw := right - left
		rh := bottom - top
		if rw <= 0 || rh <= 0 {
			continue
		}

		// Draw semi-transparent fill
		if opacity > 0 {
			fill, e := newColorImage(rw, rh, append(c, fillAlpha))
			if e != nil {
				continue
			}
			e = img.Composite2(fill, vips.BlendModeOver, &vips.Composite2Options{X: left, Y: top})
			fill.Close()
			if e != nil {
				return e
			}
		}

		// Draw 2px solid outline — top edge
		if err = drawRect(img, c, left, top, rw, 2); err != nil {
			return
		}
		// bottom edge
		if err = drawRect(img, c, left, bottom-2, rw, 2); err != nil {
			return
		}
		// left edge
		if err = drawRect(img, c, left, top, 2, rh); err != nil {
			return
		}
		// right edge
		if err = drawRect(img, c, right-2, top, 2, rh); err != nil {
			return
		}
	}
	return
}

// drawRect composites a solid opaque coloured rectangle onto img.
func drawRect(img *vips.Image, c []float64, x, y, w, h int) error {
	if w <= 0 || h <= 0 {
		return nil
	}
	rect, err := newColorImage(w, h, append(c, float64(255)))
	if err != nil {
		return err
	}
	err = img.Composite2(rect, vips.BlendModeOver, &vips.Composite2Options{X: x, Y: y})
	rect.Close()
	return err
}
