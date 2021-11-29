package vipsprocessor

import (
	"context"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"math"
	"strings"
)

func (v *VipsProcessor) process(
	ctx context.Context, img *vips.ImageRef, p imagor.Params, load imagor.LoadFunc,
) error {
	if p.TrimPosition != "" {
		if err := trim(img, p.TrimPosition, p.TrimTolerance); err != nil {
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
		switch p.Type {
		case "stretch":
			stretch = true
			break
		case "upscale":
			upscale = true
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
		if v.Filters != nil {
			if fn := v.Filters[p.Type]; fn != nil {
				if err := fn(img, load, strings.Split(p.Args, ",")...); err != nil {
					return err
				}
			}
		}
		switch p.Type {
		case "fill", "background_color":
			if err := fill(img, w, h, p.Args, upscale); err != nil {
				return err
			}
			break
		}
	}
	return nil
}
