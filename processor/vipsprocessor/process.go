package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/davidbyttow/govips/v2/vips"
	"go.uber.org/zap"
	"golang.org/x/image/colornames"
	"image/color"
	"math"
	"strings"
	"time"
)

func (v *VipsProcessor) process(
	ctx context.Context, img *vips.ImageRef, p imagorpath.Params, load imagor.LoadFunc, thumbnail, stretch, upscale bool, focalRects []focal,
) error {
	var (
		origWidth  = float64(img.Width())
		origHeight = float64(img.PageHeight())
		cropLeft   = p.CropLeft
		cropRight  = p.CropRight
		cropTop    = p.CropTop
		cropBottom = p.CropBottom
	)
	if cropLeft < 0 {
		cropLeft = 0
	}
	if cropTop < 0 {
		cropTop = 0
	}
	if cropRight-cropLeft > 0 && cropBottom-cropTop > 0 {
		// percentage
		if cropLeft < 1.0 && cropTop < 1.0 && cropRight <= 1.0 && cropBottom <= 1.0 {
			cropLeft = math.Round(cropLeft * origWidth)
			cropTop = math.Round(cropTop * origHeight)
			cropRight = math.Round(cropRight * origWidth)
			cropBottom = math.Round(cropBottom * origHeight)
		}
		if cropRight > origWidth {
			cropRight = origWidth
		}
		if cropBottom > origHeight {
			cropBottom = origHeight
		}
		if err := img.ExtractArea(int(cropLeft), int(cropTop), int(cropRight-cropLeft), int(cropBottom-cropTop)); err != nil {
			return err
		}
	} else {
		cropTop = 0
		cropBottom = 0
		cropRight = 0
		cropLeft = 0
	}
	if p.Trim {
		if err := trim(ctx, img, p.TrimBy, p.TrimTolerance); err != nil {
			return err
		}
	}
	var (
		w = p.Width
		h = p.Height
	)
	if w == 0 && h == 0 {
		w = img.Width()
		h = img.PageHeight()
	} else if w == 0 {
		w = img.Width() * h / img.PageHeight()
		if !upscale && w > img.Width() {
			w = img.Width()
		}
	} else if h == 0 {
		h = img.PageHeight() * w / img.Width()
		if !upscale && h > img.PageHeight() {
			h = img.PageHeight()
		}
	}
	if !thumbnail {
		if p.FitIn {
			if upscale || w < img.Width() || h < img.PageHeight() {
				if err := img.Thumbnail(w, h, vips.InterestingNone); err != nil {
					return err
				}
			}
		} else if stretch {
			if upscale || (w < img.Width() && h < img.PageHeight()) {
				if err := img.ThumbnailWithSize(
					w, h, vips.InterestingNone, vips.SizeForce,
				); err != nil {
					return err
				}
			}
		} else if upscale || w < img.Width() || h < img.PageHeight() {
			interest := vips.InterestingCentre
			if p.Smart {
				interest = vips.InterestingAttention
			} else if float64(w)/float64(h) > float64(img.Width())/float64(img.PageHeight()) {
				if p.VAlign == imagorpath.VAlignTop {
					interest = vips.InterestingLow
				} else if p.VAlign == imagorpath.VAlignBottom {
					interest = vips.InterestingHigh
				}
			} else {
				if p.HAlign == imagorpath.HAlignLeft {
					interest = vips.InterestingLow
				} else if p.HAlign == imagorpath.HAlignRight {
					interest = vips.InterestingHigh
				}
			}
			if p.Smart && len(focalRects) > 0 {
				focalX, focalY := parseFocalPoint(focalRects...)
				if err := v.focalThumbnail(
					img, w, h,
					(focalX-cropLeft)/float64(img.Width()),
					(focalY-cropTop)/float64(img.PageHeight()),
				); err != nil {
					return err
				}
			} else {
				if err := v.thumbnail(img, w, h, interest, vips.SizeBoth); err != nil {
					return err
				}
			}
		}
	}
	if p.HFlip {
		if err := img.Flip(vips.DirectionHorizontal); err != nil {
			return err
		}
	}
	if p.VFlip {
		if err := img.Flip(vips.DirectionVertical); err != nil {
			return err
		}
	}
	for i, filter := range p.Filters {
		if err := ctx.Err(); err != nil {
			return err
		}
		if v.MaxFilterOps > 0 && i >= v.MaxFilterOps {
			if v.Debug {
				v.Logger.Debug("max-filter-ops-exceeded",
					zap.String("name", filter.Name), zap.String("args", filter.Args))
			}
			break
		}
		start := time.Now()
		args := strings.Split(filter.Args, ",")
		if fn := v.Filters[filter.Name]; fn != nil {
			if err := fn(ctx, img, load, args...); err != nil {
				return err
			}
		} else if filter.Name == "fill" {
			if err := v.fill(ctx, img, w, h,
				p.PaddingLeft, p.PaddingTop, p.PaddingRight, p.PaddingBottom,
				filter.Args); err != nil {
				return err
			}
		}
		if v.Debug {
			v.Logger.Debug("filter",
				zap.String("name", filter.Name), zap.String("args", filter.Args),
				zap.Duration("took", time.Since(start)))
		}
	}
	return nil
}

type focal struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

func parseFocalPoint(focalRects ...focal) (focalX, focalY float64) {
	var sumWeight float64
	for _, f := range focalRects {
		sumWeight += (f.Right - f.Left) * (f.Bottom - f.Top)
	}
	for _, f := range focalRects {
		r := (f.Right - f.Left) * (f.Bottom - f.Top) / sumWeight
		focalX += (f.Left + f.Right) / 2 * r
		focalY += (f.Top + f.Bottom) / 2 * r
	}
	return
}

func trim(ctx context.Context, img *vips.ImageRef, pos string, tolerance int) error {
	if IsAnimated(ctx) {
		// skip animation support
		return nil
	}
	var x, y int
	if pos == imagorpath.TrimByBottomRight {
		x = img.Width() - 1
		y = img.PageHeight() - 1
	}
	if tolerance == 0 {
		tolerance = 1
	}
	p, err := img.GetPoint(x, y)
	if err != nil {
		return err
	}
	l, t, w, h, err := img.FindTrim(float64(tolerance), &vips.Color{
		R: uint8(p[0]), G: uint8(p[1]), B: uint8(p[2]),
	})
	if err != nil {
		return err
	}
	if err = img.ExtractArea(l, t, w, h); err != nil {
		return err
	}
	return nil
}

func crop(img *vips.ImageRef, left, top, right, bottom float64) (err error) {
	var (
		dw = float64(img.Width())
		dh = float64(img.PageHeight())
	)
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	if right-left <= 0 || bottom-top <= 0 {
		return // no-ops
	}
	// percentage
	if left < 1.0 && top < 1.0 && right <= 1.0 && bottom <= 1.0 {
		left = math.Round(left * dw)
		top = math.Round(top * dh)
		right = math.Round(right * dw)
		bottom = math.Round(bottom * dh)
	}
	if right > dw {
		right = dw
	}
	if bottom > dh {
		bottom = dh
	}
	return img.ExtractArea(int(left), int(top), int(right-left), int(bottom-top))
}

func isBlack(c *vips.Color) bool {
	return c.R == 0x00 && c.G == 0x00 && c.B == 0x00
}

func isWhite(c *vips.Color) bool {
	return c.R == 0xff && c.G == 0xff && c.B == 0xff
}

func getColor(img *vips.ImageRef, color string) *vips.Color {
	vc := &vips.Color{}
	args := strings.Split(strings.ToLower(color), ",")
	mode := ""
	name := strings.TrimPrefix(args[0], "#")
	if len(args) > 1 {
		mode = args[1]
	}
	if name == "auto" {
		if img != nil {
			x := 0
			y := 0
			if mode == "bottom-right" {
				x = img.Width() - 1
				y = img.PageHeight() - 1
			}
			p, _ := img.GetPoint(x, y)
			if len(p) >= 3 {
				vc.R = uint8(p[0])
				vc.G = uint8(p[1])
				vc.B = uint8(p[2])
			}
		}
	} else if c, ok := colornames.Map[name]; ok {
		vc.R = c.R
		vc.G = c.G
		vc.B = c.B
	} else if c, ok := parseHexColor(name); ok {
		vc.R = c.R
		vc.G = c.G
		vc.B = c.B
	}
	return vc
}

func parseHexColor(s string) (c color.RGBA, ok bool) {
	c.A = 0xff
	switch len(s) {
	case 6:
		c.R = hexToByte(s[0])<<4 + hexToByte(s[1])
		c.G = hexToByte(s[2])<<4 + hexToByte(s[3])
		c.B = hexToByte(s[4])<<4 + hexToByte(s[5])
		ok = true
	case 3:
		c.R = hexToByte(s[0]) * 17
		c.G = hexToByte(s[1]) * 17
		c.B = hexToByte(s[2]) * 17
		ok = true
	}
	return
}

func hexToByte(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	}
	return 0
}
