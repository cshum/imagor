package imagorpath

import "strconv"

const (
	// TrimByTopLeft trim by top-left keyword
	TrimByTopLeft = "top-left"
	// TrimByBottomRight trim by bottom-right keyword
	TrimByBottomRight = "bottom-right"
	// HAlignLeft horizontal align left keyword
	HAlignLeft = "left"
	// HAlignRight horizontal align right keyword
	HAlignRight = "right"
	// VAlignTop vertical align top keyword
	VAlignTop = "top"
	// VAlignBottom vertical align bottom keyword
	VAlignBottom = "bottom"
)

// Filters a slice of Filter
type Filters []Filter

// Params image endpoint parameters
type Params struct {
	Params        bool    `json:"-"`
	Path          string  `json:"path,omitempty"`
	Image         string  `json:"image,omitempty"`
	Base64Image   bool    `json:"base64_image,omitempty"`
	Unsafe        bool    `json:"unsafe,omitempty"`
	Hash          string  `json:"hash,omitempty"`
	Meta          bool    `json:"meta,omitempty"`
	Trim          bool    `json:"trim,omitempty"`
	TrimBy        string  `json:"trim_by,omitempty"`
	TrimTolerance int     `json:"trim_tolerance,omitempty"`
	CropLeft      float64 `json:"crop_left,omitempty"`
	CropTop       float64 `json:"crop_top,omitempty"`
	CropRight     float64 `json:"crop_right,omitempty"`
	CropBottom    float64 `json:"crop_bottom,omitempty"`
	FitIn         bool    `json:"fit_in,omitempty"`
	AdaptiveFitIn bool    `json:"adaptive_fit_in,omitempty"`
	FullFitIn     bool    `json:"full_fit_in,omitempty"`
	Stretch       bool    `json:"stretch,omitempty"`
	Width         int     `json:"width,omitempty"`
	Height        int     `json:"height,omitempty"`
	PaddingLeft   int     `json:"padding_left,omitempty"`
	PaddingTop    int     `json:"padding_top,omitempty"`
	PaddingRight  int     `json:"padding_right,omitempty"`
	PaddingBottom int     `json:"padding_bottom,omitempty"`
	HFlip         bool    `json:"h_flip,omitempty"`
	VFlip         bool    `json:"v_flip,omitempty"`
	HAlign        string  `json:"h_align,omitempty"`
	VAlign        string  `json:"v_align,omitempty"`
	Smart         bool    `json:"smart,omitempty"`
	Filters       Filters `json:"filters,omitempty"`
}

// Filter imagor endpoint filter
type Filter struct {
	Name string `json:"name,omitempty"`
	Args string `json:"args,omitempty"`
}

// HasCrop reports whether the params specify a crop region.
// Any non-zero crop coordinate (left, top, right, or bottom) counts as a crop.
func HasCrop(p Params) bool {
	return p.CropLeft > 0 || p.CropTop > 0 || p.CropRight > 0 || p.CropBottom > 0
}

// HasFilter reports whether the params include at least one filter with the given name.
func HasFilter(p Params, name string) bool {
	for _, f := range p.Filters {
		if f.Name == name {
			return true
		}
	}
	return false
}

// HasCacheBypass reports whether the params require bypassing the image cache.
// The cache stores a downscaled copy keyed by image path only. Requests that
// depend on original-space coordinates or per-request decode parameters must
// bypass the cache to avoid incorrect results:
//   - crop coordinates (absolute or percentage) are in original image space
//   - focal() filter uses original image coordinates for smart crop
//   - page(n>1) selects a specific page/frame; cache stores page 1 only
//   - dpi(n) affects SVG/PDF render resolution; cache stores at default DPI
func HasCacheBypass(p Params) bool {
	if HasCrop(p) {
		return true
	}
	for _, f := range p.Filters {
		switch f.Name {
		case "focal":
			return true
		case "page":
			if n, _ := strconv.Atoi(f.Args); n > 1 {
				return true
			}
		case "dpi":
			if n, _ := strconv.Atoi(f.Args); n > 0 {
				return true
			}
		}
	}
	return false
}
