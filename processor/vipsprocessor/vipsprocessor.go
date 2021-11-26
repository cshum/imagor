package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/davidbyttow/govips/v2/vips"
	"math"
	"strconv"
	"strings"
)

type Vips struct {
}

func New() *Vips {
	vips.Startup(nil)
	return &Vips{}
}

func (v *Vips) Process(
	_ context.Context, buf []byte, p imagor.Params,
) ([]byte, *imagor.Meta, error) {
	img, err := vips.NewImageFromBuffer(buf)
	if err != nil {
		return nil, nil, err
	}
	defer img.Close()
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
			return nil, nil, err
		}
	}
	var (
		format  = img.Format()
		quality int
		fill    string
		stretch bool
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
		case "format":
			if typ, ok := imageTypeMap[p.Args]; ok {
				format = typ
			}
			break
		case "quality":
			quality, _ = strconv.Atoi(p.Args)
			break
		case "fill":
			fill = p.Args
			break
		case "stretch":
			stretch = true
			break
		}
	}
	if p.FitIn {
		if err := img.Thumbnail(w, h, vips.InterestingNone); err != nil {
			return nil, nil, err
		}
		if fill != "" {
			extend := vips.ExtendCopy
			switch fill {
			case "white":
				extend = vips.ExtendWhite
			case "mirror":
				extend = vips.ExtendMirror
			case "black":
				extend = vips.ExtendBlack
			case "repeat":
				extend = vips.ExtendRepeat
			}
			if err := img.Embed(
				(w-img.Width())/2,
				(h-img.Height())/2,
				w, h, extend,
			); err != nil {
				return nil, nil, err
			}
		}
	} else if stretch {
		if err := img.ResizeWithVScale(
			float64(w)/float64(img.Width()),
			float64(h)/float64(img.Height()),
			vips.KernelAuto); err != nil {
			return nil, nil, err
		}
	} else if w < img.Width() || h < img.Height() {
		if err := img.Resize(math.Max(
			float64(w)/float64(img.Width()),
			float64(h)/float64(img.Height()),
		), vips.KernelAuto); err != nil {
			return nil, nil, err
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
			return nil, nil, err
		}
	}
	if p.HorizontalFlip {
		if err := img.Flip(vips.DirectionHorizontal); err != nil {
			return nil, nil, err
		}
	}
	if p.VerticalFlip {
		if err := img.Flip(vips.DirectionVertical); err != nil {
			return nil, nil, err
		}
	}
	for _, p := range p.Filters {
		switch p.Type {
		case "blur":
			if sigma, _ := strconv.ParseFloat(strings.Split(p.Args, ",")[0], 64); sigma > 0 {
				if err := img.GaussianBlur(sigma); err != nil {
					return nil, nil, err
				}
			}
			break
		case "rotate":
			if angle, _ := strconv.Atoi(p.Args); angle > 0 {
				vAngle := vips.Angle0
				if angle == 90 {
					vAngle = vips.Angle270
				} else if angle == 180 {
					vAngle = vips.Angle180
				} else if angle == 270 {
					vAngle = vips.Angle90
				}
				if err := img.Rotate(vAngle); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	buf, meta, err := export(img, format, quality)
	if err != nil {
		return nil, nil, err
	}
	return buf, &imagor.Meta{
		Format:      vips.ImageTypes[meta.Format],
		Width:       meta.Width,
		Height:      meta.Height,
		Orientation: meta.Orientation,
	}, nil
}

func (v *Vips) Close() error {
	vips.Shutdown()
	return nil
}

var imageTypeMap = map[string]vips.ImageType{
	"gif":    vips.ImageTypeGIF,
	"jpeg":   vips.ImageTypeJPEG,
	"jpg":    vips.ImageTypeJPEG,
	"magick": vips.ImageTypeMagick,
	"pdf":    vips.ImageTypePDF,
	"png":    vips.ImageTypePNG,
	"svg":    vips.ImageTypeSVG,
	"tiff":   vips.ImageTypeTIFF,
	"webp":   vips.ImageTypeWEBP,
	"heif":   vips.ImageTypeHEIF,
	"bmp":    vips.ImageTypeBMP,
	"avif":   vips.ImageTypeAVIF,
}

func export(image *vips.ImageRef, format vips.ImageType, quality int) ([]byte, *vips.ImageMetadata, error) {
	switch format {
	case vips.ImageTypePNG:
		opts := vips.NewPngExportParams()
		return image.ExportPng(opts)
	case vips.ImageTypeWEBP:
		opts := vips.NewWebpExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportWebp(opts)
	case vips.ImageTypeHEIF:
		opts := vips.NewHeifExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportHeif(opts)
	case vips.ImageTypeTIFF:
		opts := vips.NewTiffExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportTiff(opts)
	case vips.ImageTypeGIF:
		opts := vips.NewGifExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportGIF(opts)
	case vips.ImageTypeAVIF:
		opts := vips.NewAvifExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportAvif(opts)
	default:
		opts := vips.NewJpegExportParams()
		if quality > 0 {
			opts.Quality = quality
		}
		return image.ExportJpeg(opts)
	}
}
