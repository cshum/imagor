package imagorpath

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strings"
)

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
	Params            bool    `json:"-"`
	Path              string  `json:"path,omitempty"`
	Image             string  `json:"image,omitempty"`
	Unsafe            bool    `json:"unsafe,omitempty"`
	Hash              string  `json:"hash,omitempty"`
	Meta              bool    `json:"meta,omitempty"`
	Trim              bool    `json:"trim,omitempty"`
	TrimBy            string  `json:"trim_by,omitempty"`
	TrimTolerance     int     `json:"trim_tolerance,omitempty"`
	CropLeft          int     `json:"crop_left,omitempty"`
	CropLeftPercent   float64 `json:"crop_left_percent,omitempty"`
	CropTop           int     `json:"crop_top,omitempty"`
	CropTopPercent    float64 `json:"crop_top_percent,omitempty"`
	CropRight         int     `json:"crop_right,omitempty"`
	CropRightPercent  float64 `json:"crop_right_percent,omitempty"`
	CropBottom        int     `json:"crop_bottom,omitempty"`
	CropBottomPercent float64 `json:"crop_bottom_percent,omitempty"`
	FitIn             bool    `json:"fit_in,omitempty"`
	Stretch           bool    `json:"stretch,omitempty"`
	Width             int     `json:"width,omitempty"`
	Height            int     `json:"height,omitempty"`
	PaddingLeft       int     `json:"padding_left,omitempty"`
	PaddingTop        int     `json:"padding_top,omitempty"`
	PaddingRight      int     `json:"padding_right,omitempty"`
	PaddingBottom     int     `json:"padding_bottom,omitempty"`
	HFlip             bool    `json:"h_flip,omitempty"`
	VFlip             bool    `json:"v_flip,omitempty"`
	HAlign            string  `json:"h_align,omitempty"`
	VAlign            string  `json:"v_align,omitempty"`
	Smart             bool    `json:"smart,omitempty"`
	Filters           Filters `json:"filters,omitempty"`
}

type Filter struct {
	Name string `json:"name,omitempty"`
	Args string `json:"args,omitempty"`
}

// Sign an Imagor endpoint with secret key
func Sign(path, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(strings.TrimPrefix(path, "/")))
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return hash
}
