package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/davidbyttow/govips/v2/vips"
	"strconv"
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
	image, err := vips.NewImageFromBuffer(buf)
	if err != nil {
		return nil, nil, err
	}
	defer image.Close()
	var (
		format  = image.Format()
		w       = p.Width
		h       = p.Height
		quality int
		fill    string
	)
	if w == 0 && h == 0 {
		w = image.Width()
		h = image.Height()
	} else if w == 0 {
		w = image.Width() * h / image.Height()
	} else if h == 0 {
		h = image.Height() * w / image.Width()
	}
	for _, p := range p.Filters {
		switch p.Name {
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
		}
	}
	if p.FitIn {
		if p.Smart {
			if err := image.Thumbnail(w, h, vips.InterestingAttention); err != nil {
				return nil, nil, err
			}
		} else {
			if err := image.Thumbnail(w, h, vips.InterestingNone); err != nil {
				return nil, nil, err
			}
		}
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
		if err := image.Embed(
			(w-image.Width())/2,
			(h-image.Height())/2,
			w, h, extend,
		); err != nil {
			return nil, nil, err
		}
	}
	buf, meta, err := export(image, format, quality)
	if err != nil {
		return nil, nil, err
	}
	return buf, &imagor.Meta{
		ImageType:   vips.ImageTypes[meta.Format],
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
