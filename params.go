package imagor

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"regexp"
	"strconv"
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
	HPadding       int      `json:"hpadding,omitempty"`
	VPadding       int      `json:"vpadding,omitempty"`
	Stretch        bool     `json:"stretch,omitempty"`
	Upscale        bool     `json:"upscale,omitempty"`
	Width          int      `json:"width,omitempty"`
	Height         int      `json:"height,omitempty"`
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

type Meta struct {
	Format      string `json:"format"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Orientation int    `json:"orientation"`
}

var pathRegex = regexp.MustCompile(
	"/?" +
		// params
		"(params/)?" +
		// hash
		"((unsafe/)|([A-Za-z0-9-_=]{26,30})/)?" +
		// path
		"(.+)?",
)

var paramsRegex = regexp.MustCompile(
	"/?" +
		// meta
		"(meta/)?" +
		// trim
		"(trim(:(top-left|bottom-right))?(:(\\d+))?/)?" +
		// crop
		"((\\d+)x(\\d+):(\\d+)x(\\d+)/)?" +
		// fit-in
		"(fit-in/)?" +
		// stretch
		"(stretch/)?" +
		// upscale
		"(upscale/)?" +
		// dimensions
		"((\\-?)(\\d*)x(\\-?)(\\d*)/)?" +
		// paddings
		"((\\d+)x(\\d+)/)?" +
		// halign
		"((left|right|center)/)?" +
		// valign
		"((top|bottom|middle)/)?" +
		// smart
		"(smart/)?" +
		// filters
		"(filters:(.+?\\))/)?" +
		// image
		"(.+)?",
)

var filterRegex = regexp.MustCompile("(.+)\\((.*)\\)")

// ParseParams parse params object from uri string
func ParseParams(uri string) (params Params) {
	match := pathRegex.FindStringSubmatch(uri)
	if len(match) < 6 {
		return
	}
	index := 1
	if match[index] != "" {
		params.Params = true
	}
	index += 1
	if match[index+1] == "unsafe/" {
		params.Unsafe = true
	} else if len(match[index+2]) <= 28 {
		params.Hash = match[index+2]
	}
	index += 3
	params.Path = match[index]

	match = paramsRegex.FindStringSubmatch(params.Path)
	if len(match) == 0 {
		return
	}
	index = 1
	if match[index] != "" {
		params.Meta = true
	}
	index += 1
	if match[index] != "" {
		params.Trim = "top-left"
		if s := match[index+2]; s != "" {
			params.Trim = s
		}
		params.TrimTolerance, _ = strconv.Atoi(match[index+4])
	}
	index += 5
	if match[index] != "" {
		params.CropLeft, _ = strconv.Atoi(match[index+1])
		params.CropTop, _ = strconv.Atoi(match[index+2])
		params.CropRight, _ = strconv.Atoi(match[index+3])
		params.CropBottom, _ = strconv.Atoi(match[index+4])
	}
	index += 5
	if match[index] != "" {
		params.FitIn = true
	}
	index += 1
	if match[index] != "" {
		params.Stretch = true
	}
	index += 1
	if match[index] != "" {
		params.Upscale = true
	}
	index += 1
	if match[index] != "" {
		params.HorizontalFlip = match[index+1] != ""
		params.Width, _ = strconv.Atoi(match[index+2])
		params.VerticalFlip = match[index+3] != ""
		params.Height, _ = strconv.Atoi(match[index+4])
	}
	index += 5
	if match[index] != "" {
		params.HPadding, _ = strconv.Atoi(match[index+1])
		params.VPadding, _ = strconv.Atoi(match[index+2])
	}
	index += 3
	if match[index] != "" {
		params.HAlign = match[index+1]
	}
	index += 2
	if match[index] != "" {
		params.VAlign = match[index+1]
	}
	index += 2
	if match[index] != "" {
		params.Smart = true
	}
	index += 1
	if match[index] != "" {
		params.Filters = parseFilters(match[index+1])
	}
	index += 2
	params.Image = match[index]
	if u, err := url.QueryUnescape(match[index]); err == nil {
		params.Image = u
	}
	return
}

// Verify if hash matches secret
func (p *Params) Verify(secret string) bool {
	return Sign(p.Path, secret) == p.Hash
}

func parseFilters(filters string) (results []Filter) {
	splits := strings.Split(filters, "):")
	for _, seg := range splits {
		seg = strings.TrimSuffix(seg, ")") + ")"
		if match := filterRegex.FindStringSubmatch(seg); len(match) >= 3 {
			results = append(results, Filter{
				Name: strings.ToLower(match[1]),
				Args: match[2],
			})
		}
	}
	return
}

// Sign an Imagor path with secret key
func Sign(path, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(strings.TrimPrefix(path, "/")))
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return hash
}
