package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/davidbyttow/govips/v2/vips"
)

type Vips struct {
}

func New() *Vips {
	vips.Startup(nil)
	return &Vips{}
}

func (v *Vips) Process(
	_ context.Context, buf []byte, params imagor.Params,
) ([]byte, *imagor.Meta, error) {
	image, err := vips.NewImageFromBuffer(buf)
	if err != nil {
		return nil, nil, err
	}
	defer image.Close()

	// todo

	buf, meta, err := export(image, params)
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

func (v *Vips) Close() {
	vips.Shutdown()
}

func export(image *vips.ImageRef, params imagor.Params) ([]byte, *vips.ImageMetadata, error) {
	switch image.Format() {
	case vips.ImageTypeJPEG:
		return image.ExportJpeg(vips.NewJpegExportParams())
	case vips.ImageTypePNG:
		return image.ExportPng(vips.NewPngExportParams())
	case vips.ImageTypeWEBP:
		return image.ExportWebp(vips.NewWebpExportParams())
	case vips.ImageTypeHEIF:
		return image.ExportHeif(vips.NewHeifExportParams())
	case vips.ImageTypeTIFF:
		return image.ExportTiff(vips.NewTiffExportParams())
	case vips.ImageTypeAVIF:
		return image.ExportAvif(vips.NewAvifExportParams())
	default:
		return image.ExportJpeg(vips.NewJpegExportParams())
	}
}
