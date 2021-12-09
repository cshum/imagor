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
	Params        bool    `json:"-"`
	Path          string  `json:"path,omitempty"`
	Image         string  `json:"image,omitempty"`
	Unsafe        bool    `json:"unsafe,omitempty"`
	Hash          string  `json:"hash,omitempty"`
	Meta          bool    `json:"meta,omitempty"`
	Trim          bool    `json:"trim,omitempty"`
	TrimBy        string  `json:"trim_by,omitempty"`
	TrimTolerance int     `json:"trim_tolerance,omitempty"`
	CropLeft      int     `json:"crop_left,omitempty"`
	CropTop       int     `json:"crop_top,omitempty"`
	CropRight     int     `json:"crop_right,omitempty"`
	CropBottom    int     `json:"crop_bottom,omitempty"`
	FitIn         bool    `json:"fit_in,omitempty"`
	Stretch       bool    `json:"stretch,omitempty"`
	Upscale       bool    `json:"upscale,omitempty"`
	Width         int     `json:"width,omitempty"`
	Height        int     `json:"height,omitempty"`
	HPadding      int     `json:"h_padding,omitempty"`
	VPadding      int     `json:"v_padding,omitempty"`
	HFlip         bool    `json:"h_flip,omitempty"`
	VFlip         bool    `json:"v_flip,omitempty"`
	HAlign        string  `json:"h_align,omitempty"`
	VAlign        string  `json:"v_align,omitempty"`
	Smart         bool    `json:"smart,omitempty"`
	Filters       Filters `json:"filters,omitempty"`
}

type Filter struct {
	Name string `json:"type,omitempty"`
	Args string `json:"args,omitempty"`
}

// Sign an Imagor endpoint with secret key
func Sign(path, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(strings.TrimPrefix(path, "/")))
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return hash
}
