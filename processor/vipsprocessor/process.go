package vipsprocessor

import (
	"context"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"golang.org/x/image/colornames"
	"image/color"
	"math"
	"strings"
	"time"
)

func (v *VipsProcessor) process(
	ctx context.Context, img *vips.ImageRef, p imagorpath.Params, load imagor.LoadFunc,
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
		if h := img.Height(); cropBottom > h {
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
		stretch = p.Stretch
		upscale = p.Upscale
		w       = p.Width
		h       = p.Height
	)
	if w == 0 && h == 0 {
		w = img.Width()
		h = img.Height()
	} else if w == 0 {
		w = img.Width() * h / img.Height()
	} else if h == 0 {
		h = img.Height() * w / img.Width()
	}
	for _, p := range p.Filters {
		switch p.Name {
		case "stretch":
			stretch = true
			break
		case "upscale":
			upscale = true
			break
		case "no_upscale":
			upscale = false
			break
		}
	}
	if p.FitIn {
		if upscale || w-p.HPadding*2 < img.Width() || h-p.VPadding*2 < img.Height() {
			if err := img.Thumbnail(w-p.HPadding*2, h-p.VPadding*2, vips.InterestingNone); err != nil {
				return err
			}
		}
	} else if stretch {
		if err := img.ResizeWithVScale(
			float64(w)/float64(img.Width()),
			float64(h)/float64(img.Height()),
			vips.KernelAuto); err != nil {
			return err
		}
	} else if w < img.Width() || h < img.Height() {
		if err := img.Resize(math.Max(
			float64(w)/float64(img.Width()),
			float64(h)/float64(img.Height()),
		), vips.KernelAuto); err != nil {
			return err
		}
		interest := vips.InterestingCentre
		if p.Smart {
			interest = vips.InterestingAttention
		} else if (p.VAlign == "top" && img.Height() > h) || (p.HAlign == "left" && img.Width() > w) {
			interest = vips.InterestingLow
		} else if (p.VAlign == "bottom" && img.Height() > h) || (p.HAlign == "right" && img.Width() > w) {
			interest = vips.InterestingHigh
		}
		if err := img.SmartCrop(w, h, interest); err != nil {
			return err
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
	for i, p := range p.Filters {
		if err := ctx.Err(); err != nil {
			return err
		}
		if i >= v.MaxFilterOps {
			break
		}
		start := time.Now()
		args := strings.Split(p.Args, ",")
		if fn := v.Filters[p.Name]; fn != nil {
			if err := fn(img, load, args...); err != nil {
				return err
			}
		} else if p.Name == "fill" {
			if err := v.fill(img, w, h, upscale, args...); err != nil {
				return err
			}
		}
		if v.Debug {
			v.Logger.Debug("filter",
				zap.String("name", p.Name), zap.String("args", p.Args),
				zap.Duration("took", time.Since(start)))
		}
	}
	return nil
}

func (v *VipsProcessor) fill(img *vips.ImageRef, w, h int, upscale bool, args ...string) (err error) {
	var colour string
	var ln = len(args)
	if ln > 0 {
		colour = strings.ToLower(args[0])
	}
	c := getColor(img, colour)
	if img.HasAlpha() && colour != "blur" {
		if err = img.Flatten(getColor(img, colour)); err != nil {
			return
		}
	}
	if (colour != "blur" && isBlack(c)) || (colour == "blur" && v.DisableBlur) {
		if err = img.Embed(
			(w-img.Width())/2, (h-img.Height())/2,
			w, h, vips.ExtendBlack,
		); err != nil {
			return
		}
	} else if colour != "blur" && isWhite(c) {
		if err = img.Embed(
			(w-img.Width())/2, (h-img.Height())/2,
			w, h, vips.ExtendWhite,
		); err != nil {
			return
		}
	} else {
		var cp *vips.ImageRef
		if cp, err = img.Copy(); err != nil {
			return
		}
		defer cp.Close()
		if upscale || w < cp.Width() || h < cp.Height() {
			if err = cp.Thumbnail(w, h, vips.InterestingNone); err != nil {
				return
			}
		}
		if err = img.ResizeWithVScale(
			float64(w)/float64(img.Width()), float64(h)/float64(img.Height()),
			vips.KernelLinear,
		); err != nil {
			return
		}
		if colour == "blur" && !v.DisableBlur {
			if err = img.GaussianBlur(50); err != nil {
				return
			}
		} else {
			if err = img.DrawRect(vips.ColorRGBA{
				R: c.R, G: c.G, B: c.B, A: 255,
			}, 0, 0, w, h, true); err != nil {
				return
			}
		}
		if err = img.Composite(
			cp, vips.BlendModeOver, (w-cp.Width())/2, (h-cp.Height())/2); err != nil {
			return
		}
	}
	return
}

func trim(img *vips.ImageRef, pos string, tolerance int) error {
	var x, y int
	if pos == imagorpath.TrimByBottomRight {
		x = img.Width() - 1
		y = img.Height() - 1
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

func getColor(img *vips.ImageRef, name string) *vips.Color {
	vc := &vips.Color{}
	name = strings.TrimPrefix(strings.ToLower(name), "#")
	if name == "auto" || name == "" {
		p, _ := img.GetPoint(0, 0)
		if len(p) >= 3 {
			vc.R = uint8(p[0])
			vc.G = uint8(p[1])
			vc.B = uint8(p[2])
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
