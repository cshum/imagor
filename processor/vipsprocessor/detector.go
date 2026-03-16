package vipsprocessor

import "context"

// Region is a normalised bounding box where all fields are ratios in [0.0, 1.0]
// relative to the image dimensions passed to Detector.Detect.
type Region struct {
	Left, Top, Right, Bottom float64
}

// Detector detects regions of interest in a raw pixel buffer.
// Implementations are free to perform any kind of detection (faces, objects,
// text boxes, etc.) — the only contract is the coordinate space.
//
// buf is a row-major sRGB or sRGBA byte slice produced by WriteToMemory, with
// `bands` channels per pixel (typically 3 or 4). width and height describe the
// probe image dimensions, so len(buf) == width * height * bands.
//
// Returned regions must use normalised coordinates in [0.0, 1.0] relative to
// width / height.  vipsprocessor multiplies them by the original image
// dimensions to produce absolute focal rects before cropping.
type Detector interface {
	Detect(ctx context.Context, buf []uint8, width, height, bands int) ([]Region, error)
}
