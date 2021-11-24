package imagor

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Params struct {
	Path            string   `json:"path,omitempty"`
	Image           string   `json:"image,omitempty"`
	CropLeft        int      `json:"crop_left,omitempty"`
	CropTop         int      `json:"crop_top,omitempty"`
	CropRight       int      `json:"crop_right,omitempty"`
	CropBottom      int      `json:"crop_bottom,omitempty"`
	Width           int      `json:"width,omitempty"`
	Height          int      `json:"height,omitempty"`
	Meta            bool     `json:"meta,omitempty"`
	HorizontalFlip  bool     `json:"horizontal_flip,omitempty"`
	VerticalFlip    bool     `json:"vertical_flip,omitempty"`
	HAlign          string   `json:"h_align,omitempty"`
	VAlign          string   `json:"v_align,omitempty"`
	Smart           bool     `json:"smart,omitempty"`
	FitIn           bool     `json:"fit_in,omitempty"`
	TrimOrientation string   `json:"trim_orientation,omitempty"`
	TrimTolerance   int      `json:"trim_tolerance,omitempty"`
	Unsafe          bool     `json:"unsafe,omitempty"`
	Hash            string   `json:"hash,omitempty"`
	Filters         []Filter `json:"filters,omitempty"`
}

type Filter struct {
	Name string `json:"name,omitempty"`
	Args string `json:"args,omitempty"`
}

var pathRegex = regexp.MustCompile(
	"/?" +
		// hash
		"((unsafe/)|(([A-Za-z0-9-_]{26,28})[=]{0,2})/)?" +
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
		// dimensions
		"((\\-?)(\\d*)x(\\-?)(\\d*)/)?" +
		// halign
		"((left|right|center)/)?" +
		// valign
		"((top|bottom|middle)/)?" +
		// smart
		"(smart/)?" +
		// filters
		"(filters:(.+?\\))\\/)?" +
		// image
		"(.+)?",
)

var filterRegex = regexp.MustCompile("(.+)\\((.*)\\)")

func ParseParams(uri string) (params Params, err error) {
	match := pathRegex.FindStringSubmatch(uri)
	if len(match) < 6 {
		err = errors.New("invalid params")
		return
	}
	index := 1
	if match[index+1] == "unsafe/" {
		params.Unsafe = true
	} else if len(match[index+2]) <= 28 {
		params.Hash = match[index+3]
	}
	index += 4
	params.Path = match[index]

	match = paramsRegex.FindStringSubmatch(params.Path)
	if len(match) < 26 {
		err = errors.New("invalid params")
		return
	}
	index = 1
	if match[index] != "" {
		params.Meta = true
	}
	index += 1
	if match[index] != "" {
		params.TrimOrientation = "top-left"
		if s := match[index+2]; s != "" {
			params.TrimOrientation = s
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
		params.HorizontalFlip = match[index+1] != ""
		params.Width, _ = strconv.Atoi(match[index+2])
		params.VerticalFlip = match[index+3] != ""
		params.Height, _ = strconv.Atoi(match[index+4])
	}
	index += 5
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
	params.Image, err = url.QueryUnescape(match[index])
	return
}

func (p *Params) Verify(secret string) bool {
	return strings.TrimRight(Hash(p.Path, secret), "=") == p.Hash
}

func parseFilters(filters string) (results []Filter) {
	splits := strings.Split(filters, ":")
	for _, seg := range splits {
		if match := filterRegex.FindStringSubmatch(seg); len(match) >= 3 {
			results = append(results, Filter{
				Name: match[1],
				Args: match[2],
			})
		}
	}
	return
}

func Hash(path, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(strings.TrimPrefix(path, "/")))
	hash := base64.StdEncoding.EncodeToString(h.Sum(nil))
	hash = strings.Replace(hash, "/", "_", -1)
	hash = strings.Replace(hash, "+", "-", -1)
	return hash
}
