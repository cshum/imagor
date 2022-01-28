package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/davidbyttow/govips/v2/vips"
	"go.uber.org/zap"
	"golang.org/x/image/colornames"
	"image/color"
	"strings"
	"time"
)

func (v *VipsProcessor) process(
	ctx context.Context, img *vips.ImageRef, p imagorpath.Params, load imagor.LoadFunc, thumbnail, stretch, upscale bool,
) error {
	if p.Trim {
		if err := trim(img, p.TrimBy, p.TrimTolerance); err != nil {
			return err
		}
	}
	if p.CropBottom-p.CropTop > 0 || p.CropRight-p.CropLeft > 0 {
		cropRight := p.CropRight
		cropBottom := p.CropBottom
		if w := img.Width(); cropRight > w {
			cropRight = w
		}
		if h := img.PageHeight(); cropBottom > h {
			cropBottom = h
		}
		if err := img.ExtractArea(
			p.CropLeft, p.CropTop,
			cropRight-p.CropLeft, cropBottom-p.CropTop,
		); err != nil {
			return err
		}
	}
	var (
		w = p.Width
		h = p.Height
	)
	if w == 0 && h == 0 {
		w = img.Width() + p.HPadding*2
		h = img.PageHeight() + p.VPadding*2
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
			if upscale || w-p.HPadding*2 < img.Width() || h-p.VPadding*2 < img.PageHeight() {
				if err := img.Thumbnail(w-p.HPadding*2, h-p.VPadding*2, vips.InterestingNone); err != nil {
					return err
				}
			}
		} else if stretch {
			if upscale || (w-p.HPadding*2 < img.Width() && h-p.VPadding*2 < img.PageHeight()) {
				if err := img.ThumbnailWithSize(
					w-p.HPadding*2, h-p.VPadding*2, vips.InterestingNone, vips.SizeForce,
				); err != nil {
					return err
				}
			}
		} else if upscale || w-p.HPadding*2 < img.Width() || h-p.VPadding*2 < img.PageHeight() {
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
			if err := v.thumbnail(img, w-p.HPadding*2, h-p.VPadding*2, interest, vips.SizeBoth); err != nil {
				return err
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
		if i >= v.MaxFilterOps {
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
			if err := v.fill(ctx, img, w, h, p.HPadding, p.VPadding, upscale, filter.Args); err != nil {
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

func trim(img *vips.ImageRef, pos string, tolerance int) error {
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
