package imagorurl

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

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
		// h_align
		"((left|right|center)/)?" +
		// v_align
		"((top|bottom|middle)/)?" +
		// smart
		"(smart/)?" +
		// filters
		"(filters:(.+?\\))/)?" +
		// image
		"(.+)?",
)

var filterRegex = regexp.MustCompile("(.+)\\((.*)\\)")

// Parse params object from uri string
func Parse(uri string) (p Params) {
	match := pathRegex.FindStringSubmatch(uri)
	if len(match) < 6 {
		return
	}
	index := 1
	if match[index] != "" {
		p.Params = true
	}
	index += 1
	if match[index+1] == "unsafe/" {
		p.Unsafe = true
	} else if len(match[index+2]) <= 28 {
		p.Hash = match[index+2]
	}
	index += 3
	p.Path = match[index]

	match = paramsRegex.FindStringSubmatch(p.Path)
	if len(match) == 0 {
		return
	}
	index = 1
	if match[index] != "" {
		p.Meta = true
	}
	index += 1
	if match[index] != "" {
		p.Trim = "top-left"
		if s := match[index+2]; s != "" {
			p.Trim = s
		}
		p.TrimTolerance, _ = strconv.Atoi(match[index+4])
	}
	index += 5
	if match[index] != "" {
		p.CropLeft, _ = strconv.Atoi(match[index+1])
		p.CropTop, _ = strconv.Atoi(match[index+2])
		p.CropRight, _ = strconv.Atoi(match[index+3])
		p.CropBottom, _ = strconv.Atoi(match[index+4])
	}
	index += 5
	if match[index] != "" {
		p.FitIn = true
	}
	index += 1
	if match[index] != "" {
		p.Stretch = true
	}
	index += 1
	if match[index] != "" {
		p.Upscale = true
	}
	index += 1
	if match[index] != "" {
		p.HFlip = match[index+1] != ""
		p.Width, _ = strconv.Atoi(match[index+2])
		p.VFlip = match[index+3] != ""
		p.Height, _ = strconv.Atoi(match[index+4])
	}
	index += 5
	if match[index] != "" {
		p.HPadding, _ = strconv.Atoi(match[index+1])
		p.VPadding, _ = strconv.Atoi(match[index+2])
	}
	index += 3
	if match[index] != "" {
		p.HAlign = match[index+1]
	}
	index += 2
	if match[index] != "" {
		p.VAlign = match[index+1]
	}
	index += 2
	if match[index] != "" {
		p.Smart = true
	}
	index += 1
	if match[index] != "" {
		p.Filters = parseFilters(match[index+1])
	}
	index += 2
	p.Image = match[index]
	if u, err := url.QueryUnescape(match[index]); err == nil {
		p.Image = u
	}
	return
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
