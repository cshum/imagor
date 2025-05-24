package vipsprocessor

import (
	"github.com/cshum/imagor/vips"
	"golang.org/x/image/bmp"
	"image"
	"image/draw"
	"io"
)

func loadImageFromBMP(r io.Reader) (*vips.Image, error) {
	img, err := bmp.Decode(r)
	if err != nil {
		return nil, err
	}
	rect := img.Bounds()
	size := rect.Size()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(rect)
		draw.Draw(rgba, rect, img, rect.Min, draw.Src)
	}
	return vips.LoadImageFromMemory(rgba.Pix, size.X, size.Y, 4)
}
