package vipsprocessor

import (
	"context"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"math"
	"strconv"
	"strings"
)

func (v *Vips) process(
	_ context.Context, img *vips.ImageRef, p imagor.Params,
	load func(string) ([]byte, error),
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
		switch p.Type {
		case "fill", "background_color":
			if err := fill(img, w, h, p.Args, upscale); err != nil {
				return err
			}
			break
		case "watermark":
			if err := watermark(img, strings.Split(p.Args, ","), load); err != nil {
				return err
			}
			break
		case "round_corner":
			args := strings.Split(p.Args, ",")
			if len(args) > 0 {
				rx, _ := strconv.Atoi(args[0])
				ry := rx
				if len(args) > 1 {
					rx, _ = strconv.Atoi(args[1])
				}
				if err := roundCorner(img, rx, ry); err != nil {
					return err
				}
			}
		case "rotate":
			if angle, _ := strconv.Atoi(p.Args); angle > 0 {
				vAngle := vips.Angle0
				switch angle {
				case 90:
					vAngle = vips.Angle270
				case 180:
					vAngle = vips.Angle180
				case 270:
					vAngle = vips.Angle90
				}
				if err := img.Rotate(vAngle); err != nil {
					return err
				}
			}
			break
		case "grayscale":
			if err := img.Modulate(1, 0, 0); err != nil {
				return err
			}
			break
		case "brightness":
			b, _ := strconv.ParseFloat(p.Args, 64)
			b = b * 256 / 100
			if err := img.Linear([]float64{1, 1, 1}, []float64{b, b, b}); err != nil {
				return err
			}
			break
		case "contrast":
			a, _ := strconv.ParseFloat(p.Args, 64)
			a = a * 256 / 100
			b := 128 - a*128
			if err := img.Linear([]float64{a, a, a}, []float64{b, b, b}); err != nil {
				return err
			}
			break
		case "hue":
			h, _ := strconv.ParseFloat(p.Args, 64)
			if err := img.Modulate(1, 1, h); err != nil {
				return err
			}
			break
		case "saturation":
			s, _ := strconv.ParseFloat(p.Args, 64)
			s = 1 + s/100
			if err := img.Modulate(1, s, 0); err != nil {
				return err
			}
		case "rgb":
			if args := strings.Split(p.Args, ","); len(args) == 3 {
				r, _ := strconv.ParseFloat(args[0], 64)
				g, _ := strconv.ParseFloat(args[1], 64)
				b, _ := strconv.ParseFloat(args[2], 64)
				r = r * 256 / 100
				g = g * 256 / 100
				b = b * 256 / 100
				if err := img.Linear([]float64{1, 1, 1}, []float64{r, g, b}); err != nil {
					return err
				}
			}
			break
		case "blur":
			args := strings.Split(p.Args, ",")
			var sigma float64
			switch len(args) {
			case 2:
				sigma, _ = strconv.ParseFloat(args[1], 64)
				break
			case 1:
				sigma, _ = strconv.ParseFloat(args[0], 64)
				break
			}
			sigma /= 2
			if sigma > 0 {
				if err := img.GaussianBlur(sigma); err != nil {
					return err
				}
			}
			break
		case "sharpen":
			args := strings.Split(p.Args, ",")
			var sigma float64
			switch len(args) {
			case 1:
				sigma, _ = strconv.ParseFloat(args[0], 64)
				break
			case 2, 3:
				sigma, _ = strconv.ParseFloat(args[1], 64)
				break
			}
			sigma = 1 + sigma*2
			if err := img.Sharpen(sigma, 1, 2); err != nil {
				return err
			}
		case "strip_icc":
			if err := img.RemoveICCProfile(); err != nil {
				return err
			}
			break
		case "strip_exif":
			if err := img.RemoveMetadata(); err != nil {
				return err
			}
			break
		}
	}
	return nil
}
