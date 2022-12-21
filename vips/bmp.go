package vips

import (
	"golang.org/x/image/bmp"
	"image"
	"image/draw"
	"io"
)

func loadImageFromBMP(r io.Reader) (*Image, error) {
	img, err := bmp.Decode(r)
	if err != nil {
		return nil, err
	}
	rect := img.Bounds()
	size := rect.Size()
	rgba := image.NewRGBA(rect)
	draw.Draw(rgba, rect, img, rect.Min, draw.Src)
	return LoadImageFromMemory(rgba.Pix, size.X, size.Y, 4)
}
