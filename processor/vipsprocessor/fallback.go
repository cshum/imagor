package vipsprocessor

import (
	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
	"golang.org/x/image/bmp"
	"image"
	"image/draw"
	"io"
)

// FallbackFunc vips.Image fallback handler when vips.NewImageFromSource failed
type FallbackFunc func(blob *imagor.Blob, options *vips.LoadOptions) (*vips.Image, error)

// BufferFallbackFunc load image from buffer FallbackFunc
func BufferFallbackFunc(blob *imagor.Blob, options *vips.LoadOptions) (*vips.Image, error) {
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	return vips.NewImageFromBuffer(buf, options)
}

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
	return vips.NewImageFromMemory(rgba.Pix, size.X, size.Y, 4)
}
func BmpFallbackFunc(blob *imagor.Blob, _ *vips.LoadOptions) (*vips.Image, error) {
	if blob.BlobType() == imagor.BlobTypeBMP {
		// fallback with Go BMP decoder if vips error on BMP
		r, _, err := blob.NewReader()
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = r.Close()
		}()
		return loadImageFromBMP(r)
	} else {
		return nil, imagor.ErrUnsupportedFormat
	}
}
