package vipsprocessor

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strconv"
	"strings"

	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
)

// detectionPalette is a set of visually distinct colours used to auto-colour
// bounding boxes by detector class name. The index is chosen by hashing the
// name, so the same name always maps to the same colour.
var detectionPalette = []string{
	"ff3333", // red
	"33aaff", // blue
	"aa33ff", // purple
	"ffaa00", // orange
	"33ff66", // green
	"ff33aa", // pink
	"00cccc", // cyan
	"ffff33", // yellow
	"ff6600", // deep orange
	"0066ff", // royal blue
	"66ff00", // lime
	"cc0066", // magenta
}

// paletteColorForName returns a deterministic palette colour for the given
// detector class name using FNV-32a hashing, so the same name always maps
// to the same colour regardless of what other names are present.
func paletteColorForName(name string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return detectionPalette[h.Sum32()%uint32(len(detectionPalette))]
}

// detectionsFilter draws colour-coded bounding boxes for all detected regions
// onto the image for visual debugging. Each unique class name is automatically
// assigned a distinct colour via hash-based palette selection — the same name
// always maps to the same colour. No-op when no Detector is configured.
func (v *Processor) detectionsFilter(
	ctx context.Context, img *vips.Image, _ imagor.LoadFunc, _ ...string,
) (err error) {
	if v.Detector == nil {
		return
	}

	regions := v.detectRegions(ctx, img, "")
	if len(regions) == 0 {
		return
	}

	w := img.Width()
	h := img.PageHeight()
	bands := img.Bands()

	nameInks := make(map[string][]float64)

	inkForName := func(name string) []float64 {
		if ink, ok := nameInks[name]; ok {
			return ink
		}
		c := getColor(img, paletteColorForName(name))
		for len(c) < bands {
			c = append(c, 255)
		}
		nameInks[name] = c
		return c
	}

	for _, r := range regions {
		ink := inkForName(r.Name)
		left := int(math.Round(r.Left * float64(w)))
		top := int(math.Round(r.Top * float64(h)))
		rw := int(math.Round(r.Right*float64(w))) - left
		rh := int(math.Round(r.Bottom*float64(h))) - top
		if rw <= 0 || rh <= 0 {
			continue
		}
		if err = img.DrawRect(ink, left, top, rw, rh, &vips.DrawRectOptions{Fill: false}); err != nil {
			return
		}
	}
	return
}

// applyEllipseMask composites an ellipse-shaped alpha mask onto patch using
// BlendModeDestIn, so only the elliptical area remains visible.  The ellipse
// fills the full patch dimensions (rx = w/2, ry = h/2).
// patch must already have an alpha channel; call patch.Addalpha() first if needed.
func applyEllipseMask(patch *vips.Image) error {
	w := patch.Width()
	h := patch.Height()
	mask, err := vips.NewSvgloadBuffer([]byte(fmt.Sprintf(
		`<svg viewBox="0 0 %d %d"><ellipse cx="%d" cy="%d" rx="%d" ry="%d" fill="#fff"/></svg>`,
		w, h, w/2, h/2, w/2, h/2,
	)), nil)
	if err != nil {
		return err
	}
	defer mask.Close()
	return patch.Composite2(mask, vips.BlendModeDestIn, nil)
}

// parseRedactArgs extracts (mode, strength) from the filter args shared by
// redactFilter and redactOvalFilter.
func parseRedactArgs(args []string) (mode string, strength int) {
	mode = "blur"
	if len(args) > 0 && args[0] != "" {
		mode = strings.ToLower(args[0])
	}
	if len(args) > 1 {
		strength, _ = strconv.Atoi(args[1])
	}
	return
}

// applyRedactRegion applies the redact effect (blur/pixelate/solid) to a single
// bounding-box region on img.  When oval is true an ellipse mask is applied to
// the patch before compositing, producing a rounded redaction shape.
func applyRedactRegion(img *vips.Image, left, top, rw, rh int, mode string, strength int, oval bool) error {
	switch mode {
	case "pixelate":
		blockSize := 10
		if strength > 0 {
			blockSize = strength
		}
		patch, err := img.Copy(nil)
		if err != nil {
			return err
		}
		if err = patch.ExtractArea(left, top, rw, rh); err != nil {
			patch.Close()
			return err
		}
		if err = pixelateImage(patch, blockSize); err != nil {
			patch.Close()
			return err
		}
		if oval {
			if !patch.HasAlpha() {
				if err = patch.Addalpha(); err != nil {
					patch.Close()
					return err
				}
			}
			if err = applyEllipseMask(patch); err != nil {
				patch.Close()
				return err
			}
		}
		compErr := img.Composite2(patch, vips.BlendModeOver, &vips.Composite2Options{X: left, Y: top})
		patch.Close()
		return compErr

	case "blur":
		sigma := 15.0
		if strength > 0 {
			sigma = float64(strength)
		}
		patch, err := img.Copy(nil)
		if err != nil {
			return err
		}
		if err = patch.ExtractArea(left, top, rw, rh); err != nil {
			patch.Close()
			return err
		}
		if err = patch.Gaussblur(sigma, nil); err != nil {
			patch.Close()
			return err
		}
		if oval {
			if !patch.HasAlpha() {
				if err = patch.Addalpha(); err != nil {
					patch.Close()
					return err
				}
			}
			if err = applyEllipseMask(patch); err != nil {
				patch.Close()
				return err
			}
		}
		compErr := img.Composite2(patch, vips.BlendModeOver, &vips.Composite2Options{X: left, Y: top})
		patch.Close()
		return compErr

	default:
		// Treat mode as a color name or hex — solid fill redaction.
		// e.g. redact(black), redact(white), redact(ff0000)
		c := getColor(img, mode)
		for len(c) < img.Bands() {
			c = append(c, 255)
		}
		if !oval {
			return img.DrawRect(c, left, top, rw, rh, &vips.DrawRectOptions{Fill: true})
		}
		// Oval solid fill: render a filled ellipse patch and composite it.
		patch, err := vips.NewSvgloadBuffer([]byte(fmt.Sprintf(
			`<svg viewBox="0 0 %d %d"><ellipse cx="%d" cy="%d" rx="%d" ry="%d" fill="rgb(%d,%d,%d)"/></svg>`,
			rw, rh, rw/2, rh/2, rw/2, rh/2,
			int(c[0]), int(c[1]), int(c[2]),
		)), nil)
		if err != nil {
			return err
		}
		compErr := img.Composite2(patch, vips.BlendModeOver, &vips.Composite2Options{X: left, Y: top})
		patch.Close()
		return compErr
	}
}

// doRedact is the shared implementation for redactFilter and redactOvalFilter.
func (v *Processor) doRedact(ctx context.Context, img *vips.Image, oval bool, args ...string) error {
	if v.Detector == nil || isAnimated(img) {
		return nil
	}
	mode, strength := parseRedactArgs(args)
	regions := v.detectRegions(ctx, img, "")
	if len(regions) == 0 {
		return nil
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
		// Clamp to image bounds
		if left < 0 {
			rw += left
			left = 0
		}
		if top < 0 {
			rh += top
			top = 0
		}
		if left+rw > w {
			rw = w - left
		}
		if top+rh > h {
			rh = h - top
		}
		if rw <= 0 || rh <= 0 {
			continue
		}
		if err := applyRedactRegion(img, left, top, rw, rh, mode, strength, oval); err != nil {
			return err
		}
	}
	return nil
}

// redactFilter obscures all detected regions by applying blur, pixelate, or a
// solid color fill to each bounding box.
// No-op when no Detector is configured or no regions are detected.
// Skips animated images.
func (v *Processor) redactFilter(
	ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string,
) error {
	return v.doRedact(ctx, img, false, args...)
}

// redactOvalFilter is identical to redactFilter but applies an elliptical mask
func (v *Processor) redactOvalFilter(
	ctx context.Context, img *vips.Image, _ imagor.LoadFunc, args ...string,
) error {
	return v.doRedact(ctx, img, true, args...)
}
