package imagoruri

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strings"
)

// Params image resize and hash parameters
type Params struct {
	Params         bool     `json:"-"`
	Path           string   `json:"path,omitempty"`
	Image          string   `json:"image,omitempty"`
	Unsafe         bool     `json:"unsafe,omitempty"`
	Hash           string   `json:"hash,omitempty"`
	Meta           bool     `json:"meta,omitempty"`
	Trim           string   `json:"trim,omitempty"`
	TrimTolerance  int      `json:"trim_tolerance,omitempty"`
	CropLeft       int      `json:"crop_left,omitempty"`
	CropTop        int      `json:"crop_top,omitempty"`
	CropRight      int      `json:"crop_right,omitempty"`
	CropBottom     int      `json:"crop_bottom,omitempty"`
	FitIn          bool     `json:"fit_in,omitempty"`
	Stretch        bool     `json:"stretch,omitempty"`
	Upscale        bool     `json:"upscale,omitempty"`
	Width          int      `json:"width,omitempty"`
	Height         int      `json:"height,omitempty"`
	HPadding       int      `json:"hpadding,omitempty"`
	VPadding       int      `json:"vpadding,omitempty"`
	HorizontalFlip bool     `json:"horizontal_flip,omitempty"`
	VerticalFlip   bool     `json:"vertical_flip,omitempty"`
	HAlign         string   `json:"halign,omitempty"`
	VAlign         string   `json:"valign,omitempty"`
	Smart          bool     `json:"smart,omitempty"`
	Filters        []Filter `json:"filters,omitempty"`
}

type Filter struct {
	Name string `json:"type,omitempty"`
	Args string `json:"args,omitempty"`
}

// Verify if hash matches secret
func (p *Params) Verify(secret string) bool {
	return Sign(p.Path, secret) == p.Hash
}

// Sign an Imagor path with secret key
func Sign(path, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(strings.TrimPrefix(path, "/")))
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return hash
}
