package vipsprocessor

import (
	"context"
	"github.com/cshum/govips/v2/vips"
	"github.com/cshum/imagor"
	"strconv"
)

type FilterFunc func(img *vips.ImageRef, load imagor.LoadFunc, args ...string) (err error)

type vipsProcessor struct {
	Filters map[string]FilterFunc
}

func (v *vipsProcessor) Process(
	ctx context.Context, buf []byte, p imagor.Params, load imagor.LoadFunc,
) ([]byte, *imagor.Meta, error) {
	img, err := vips.NewImageFromBuffer(buf)
	if err != nil {
		return nil, nil, err
	}
	defer img.Close()
	var (
		format  = img.Format()
		quality int
	)
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
		case "autojpg":
			format = vips.ImageTypeJPEG
			break
		}
	}
	if err := v.process(ctx, img, p, load); err != nil {
		return nil, nil, err
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
