package vipsprocessor

import (
	"context"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
	"math"
	"strings"
	"time"
)

func (v *VipsProcessor) process(
	ctx context.Context, img *vips.ImageRef, p imagor.Params, load imagor.LoadFunc,
) error {
	if p.TrimPosition != "" {
		if err := v.trim(img, p.TrimPosition, p.TrimTolerance); err != nil {
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
		if upscale || w < img.Width() || h < img.Height() {
			if err := img.Thumbnail(w, h, vips.InterestingNone); err != nil {
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
			interest = vips.InterestingEntropy
		} else if (p.VAlign == "top" && img.Height() > h) || (p.HAlign == "left" && img.Width() > w) {
			interest = vips.InterestingLow
		} else if (p.VAlign == "bottom" && img.Height() > h) || (p.HAlign == "right" && img.Width() > w) {
			interest = vips.InterestingHigh
		}
		if err := img.SmartCrop(w, h, interest); err != nil {
			return err
		}
	}
	if p.HorizontalFlip {
		if err := img.Flip(vips.DirectionHorizontal); err != nil {
			return err
		}
	}
	if p.VerticalFlip {
		if err := img.Flip(vips.DirectionVertical); err != nil {
			return err
		}
	}
	for _, p := range p.Filters {
		if err := ctx.Err(); err != nil {
			return err
		}
		start := time.Now()
		if fn := v.Filters[p.Name]; fn != nil {
			if err := fn(img, load, strings.Split(p.Args, ",")...); err != nil {
				return err
			}
		}
		switch p.Name {
		case "fill":
			if err := v.fill(img, w, h, p.Args, upscale); err != nil {
				return err
			}
			break
		}
		if v.Debug {
			v.Logger.Debug("filter",
				zap.String("name", p.Name), zap.String("args", p.Args),
				zap.Duration("took", time.Since(start)))
		}
	}
	return nil
}

func (v *VipsProcessor) trim(img *vips.ImageRef, pos string, tolerance int) error {
	var x, y int
	if pos == "bottom-right" {
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

func (v *VipsProcessor) fill(img *vips.ImageRef, w, h int, color string, upscale bool) (err error) {
	color = strings.ToLower(color)
	if img.HasAlpha() && color != "blur" {
		if err = img.Flatten(getColor(color)); err != nil {
			return
		}
	}
	if color == "black" || (color == "blur" && v.DisableBlur) {
		if err = img.Embed(
			(w-img.Width())/2, (h-img.Height())/2,
			w, h, vips.ExtendBlack,
		); err != nil {
			return
		}
	} else if color == "white" {
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
		if color == "blur" && !v.DisableBlur {
			if err = img.GaussianBlur(50); err != nil {
				return
			}
		} else {
			c := getColor(color)
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
