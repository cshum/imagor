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

// bufferFallbackFunc load image from buffer FallbackFunc
func bufferFallbackFunc(blob *imagor.Blob, options *vips.LoadOptions) (*vips.Image, error) {
	buf, err := blob.ReadAll()
	if err != nil {
		return nil, err
	}
	return vips.NewImageFromBuffer(buf, options)
}

func estimateMaxBMPFileSize(maxResolution int64) int64 {
	const (
		bmpHeaderSize = 54
		bytesPerPixel = 4   // 32-bit RGBA (worst case)
		safetyMargin  = 1.2 // 20% buffer
	)
	return int64(float64(bmpHeaderSize+maxResolution*bytesPerPixel) * safetyMargin)
}

func (v *Processor) loadImageFromBMP(r io.Reader) (*vips.Image, error) {
	img, err := bmp.Decode(r)
	if err != nil {
		return nil, err
	}
	rect := img.Bounds()
	size := rect.Size()
	if !v.Unlimited && (size.X > v.MaxWidth || size.Y > v.MaxHeight || size.X*size.Y > v.MaxResolution) {
		return nil, imagor.ErrMaxResolutionExceeded
	}
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(rect)
		draw.Draw(rgba, rect, img, rect.Min, draw.Src)
	}
	return vips.NewImageFromMemory(rgba.Pix, size.X, size.Y, 4)
}

func (v *Processor) bmpFallbackFunc(blob *imagor.Blob, _ *vips.LoadOptions) (*vips.Image, error) {
	if blob.BlobType() == imagor.BlobTypeBMP {
		if blob.Size() > estimateMaxBMPFileSize(int64(v.MaxResolution)) {
			return nil, imagor.ErrMaxResolutionExceeded
		}
		r, _, err := blob.NewReader()
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = r.Close()
		}()
		return v.loadImageFromBMP(r)
	} else {
		return nil, imagor.ErrUnsupportedFormat
	}
}
