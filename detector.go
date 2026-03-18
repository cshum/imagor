package imagor

import "context"

// DetectorRegion is a normalised bounding box where all fields are ratios in [0.0, 1.0]
// relative to the image dimensions passed to Detector.Detect.
type DetectorRegion struct {
	Left, Top, Right, Bottom float64
	Score                    float64 // detection confidence; 0 means not provided
	Name                     string  // class name e.g. "face"; empty means not provided
}

// Detector detects regions of interest in a raw pixel buffer.
// Implementations are free to perform any kind of detection (faces, objects,
// text boxes, etc.) — the only contract is the coordinate space.
//
// imagePath identifies the source image (e.g. URL path) and is used as a cache
// key by implementations that cache results. Pass an empty string to opt out of
// caching for a particular call.
//
// blob must be a BlobTypeMemory blob created with NewBlobFromMemory, carrying
// row-major sRGB/sRGBA pixels. Retrieve dimensions via blob.Memory().
//
// Returned regions must use normalised coordinates in [0.0, 1.0] relative to
// the image width / height stored in blob.
type Detector interface {
	Startup(ctx context.Context) error
	Detect(ctx context.Context, imagePath string, blob *Blob) ([]DetectorRegion, error)
	Shutdown(ctx context.Context) error
}

// DetectorAdder is implemented by processors that accept one or more Detectors.
// imagorface and other detector plugins use this interface to wire a detector
// into a processor without importing the processor package directly.
type DetectorAdder interface {
	AddDetector(Detector)
}
