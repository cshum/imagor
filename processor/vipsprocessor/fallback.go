package vipsprocessor

import (
	"image"
	"image/draw"
	"io"

	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
	"golang.org/x/image/bmp"
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
	goImg, err := bmp.Decode(r)
	if err != nil {
		return nil, err
	}
	rect := goImg.Bounds()
	size := rect.Size()
	if !v.Unlimited && (size.X > v.MaxWidth || size.Y > v.MaxHeight || size.X*size.Y > v.MaxResolution) {
		return nil, imagor.ErrMaxResolutionExceeded
	}
	rgba, ok := goImg.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(rect)
		draw.Draw(rgba, rect, goImg, rect.Min, draw.Src)
	}
	img, err := vips.NewImageFromMemory(rgba.Pix, size.X, size.Y, 4)
	if err != nil {
		return nil, err
	}
	// NewImageFromMemory assigns VIPS_INTERPRETATION_MULTIBAND by default.
	// BMP pixels are always RGBA/sRGB, so restore interpretation for color ops.
	if copied, copyErr := img.Copy(&vips.CopyOptions{
		Interpretation: vips.InterpretationSrgb,
	}); copyErr == nil {
		img.Close()
		return copied, nil
	}
	return img, nil
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
