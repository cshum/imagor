package imagor

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
// buf is a row-major sRGB or sRGBA byte slice produced by vips WriteToMemory,
// with `bands` channels per pixel (typically 3 or 4).
// len(buf) == width * height * bands.
//
// Returned regions must use normalised coordinates in [0.0, 1.0] relative to
// width / height.
type Detector interface {
	Startup(ctx context.Context) error
	Detect(ctx context.Context, buf []uint8, width, height, bands int) ([]Region, error)
	Shutdown(ctx context.Context) error
}

// DetectorSetter is implemented by processors that accept a Detector.
// imagorface uses this interface to wire a detector into a processor without
// importing the processor package directly.
type DetectorSetter interface {
	SetDetector(Detector)
}
