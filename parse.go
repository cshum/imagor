package imagor

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Params struct {
	Image           string
	CropLeft        int
	CropTop         int
	CropRight       int
	CropBottom      int
	Width           int
	Height          int
	Meta            bool
	HorizontalFlip  bool
	VerticalFlip    bool
	HAlign          string
	VAlign          string
	Smart           bool
	FitIn           bool
	TrimOrientation string
	TrimTolerance   int
	Unsafe          bool
	Hash            string
	Filters         []Filter
}

type Filter struct {
	Name string
	Args string
}

var paramsRegex = regexp.MustCompile(
	"/?" +
		// unsafe
		"((unsafe/)|(([A-Za-z0-9-_]{26,28})[=]{0,2})/)?" +
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

func Parse(u *url.URL) (params *Params, err error) {
	params = &Params{}
	match := paramsRegex.FindStringSubmatch(u.Path)
	if len(match) < 30 {
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
