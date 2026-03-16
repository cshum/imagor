// Package pigoprocessor implements the vipsprocessor.Detector interface using
// the pigo (https://github.com/esimov/pigo) pure-Go face detection library.
//
// A downscaled, grayscale probe image is passed to pigo's cascade classifier.
// The returned bounding boxes are converted to normalised [0.0, 1.0] ratios
// before being returned so that vipsprocessor can map them back to the
// original image dimensions independently of the probe size.
package pigoprocessor

import (
	"context"
	_ "embed"
	"math"

	pigo "github.com/esimov/pigo/core"

	"github.com/cshum/imagor/processor/vipsprocessor"
)

//go:embed cascade/facefinder
var defaultCascade []byte

// Detector detects faces using pigo's cascade classifier.
type Detector struct {
	classifier   *pigo.Pigo
	minSize      int
	maxSize      int
	minQuality   float32
	iouThreshold float64
}

// Option is a functional option for Detector.
type Option func(*Detector)

// WithMinSize sets the minimum face size in pixels (relative to probe image).
func WithMinSize(size int) Option {
	return func(d *Detector) { d.minSize = size }
}

// WithMaxSize sets the maximum face size in pixels (relative to probe image).
func WithMaxSize(size int) Option {
	return func(d *Detector) { d.maxSize = size }
}

// WithMinQuality sets the minimum detection quality threshold (default 5.0).
// Lower values produce more candidates; raise it to reduce false positives.
func WithMinQuality(q float32) Option {
	return func(d *Detector) { d.minQuality = q }
}

// WithIoUThreshold sets the intersection-over-union threshold for
// ClusterDetections non-maxima suppression (default 0.2).
func WithIoUThreshold(t float64) Option {
	return func(d *Detector) { d.iouThreshold = t }
}

// New creates a Detector using the embedded facefinder cascade.
func New(opts ...Option) (*Detector, error) {
	return NewWithCascade(defaultCascade, opts...)
}

// NewWithCascade creates a Detector using the provided cascade bytes.
// This allows callers to supply alternative or custom cascade files.
func NewWithCascade(cascade []byte, opts ...Option) (*Detector, error) {
	classifier, err := pigo.NewPigo().Unpack(cascade)
	if err != nil {
		return nil, err
	}
	d := &Detector{
		classifier:   classifier,
		minSize:      20,
		maxSize:      400,
		minQuality:   5.0,
		iouThreshold: 0.2,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d, nil
}

// Detect implements vipsprocessor.Detector.
//
// buf is a row-major sRGB or sRGBA pixel buffer from vips WriteToMemory with
// `bands` channels per pixel. width × height × bands must equal len(buf).
//
// Returned regions are normalised to [0.0, 1.0] relative to width / height.
func (d *Detector) Detect(_ context.Context, buf []uint8, width, height, bands int) ([]vipsprocessor.Region, error) {
	if bands < 3 || len(buf) != width*height*bands {
		return nil, nil
	}
	pixels := toGrayscale(buf, width, height, bands)
	maxSize := min(d.maxSize, min(width, height))
	cParams := pigo.CascadeParams{
		MinSize:     d.minSize,
		MaxSize:     maxSize,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   height,
			Cols:   width,
			Dim:    width,
		},
	}
	dets := d.classifier.RunCascade(cParams, 0.0)
	dets = d.classifier.ClusterDetections(dets, d.iouThreshold)

	var regions []vipsprocessor.Region
	for _, det := range dets {
		if det.Q < d.minQuality {
			continue
		}
		half := float64(det.Scale) / 2
		left := math.Max(0, float64(det.Col)-half) / float64(width)
		top := math.Max(0, float64(det.Row)-half) / float64(height)
		right := math.Min(float64(width), float64(det.Col)+half) / float64(width)
		bottom := math.Min(float64(height), float64(det.Row)+half) / float64(height)
		if right > left && bottom > top {
			regions = append(regions, vipsprocessor.Region{
				Left:   left,
				Top:    top,
				Right:  right,
				Bottom: bottom,
			})
		}
	}
	return regions, nil
}

// toGrayscale converts a row-major sRGB(A) pixel buffer to a flat grayscale
// []uint8 using ITU-R BT.601 luminance weights.
func toGrayscale(buf []uint8, width, height, bands int) []uint8 {
	pixels := make([]uint8, width*height)
	for i := range width * height {
		base := i * bands
		r := uint32(buf[base])
		g := uint32(buf[base+1])
		b := uint32(buf[base+2])
		// ITU-R BT.601 luminance: 0.299·R + 0.587·G + 0.114·B
		pixels[i] = uint8((299*r + 587*g + 114*b) / 1000)
	}
	return pixels
}
