package imagorpath

const (
	TrimByTopLeft     = "top-left"
	TrimByBottomRight = "bottom-right"
	HAlignLeft        = "left"
	HAlignRight       = "right"
	VAlignTop         = "top"
	VAlignBottom      = "bottom"
)

type Filters []Filter

// Params image endpoint parameters
type Params struct {
	Params        bool    `json:"-"`
	Path          string  `json:"path,omitempty"`
	Image         string  `json:"image,omitempty"`
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

type Filter struct {
	Name string `json:"name,omitempty"`
	Args string `json:"args,omitempty"`
}
