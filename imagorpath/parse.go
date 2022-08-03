package imagorpath

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var pathRegex = regexp.MustCompile(
	"/*" +
		// params
		"(params/)?" +
		// hash
		"((unsafe/)|([A-Za-z0-9-_=]{8,})/)?" +
		// path
		"(.+)?",
)

var paramsRegex = regexp.MustCompile(
	"/*" +
		// meta
		"(meta/)?" +
		// trim
		"(trim(:(top-left|bottom-right))?(:(\\d+))?/)?" +
		// crop
		"(((0?\\.)?\\d+)x((0?\\.)?\\d+):(([0-1]?\\.)?\\d+)x(([0-1]?\\.)?\\d+)/)?" +
		// fit-in
		"(fit-in/)?" +
		// stretch
		"(stretch/)?" +
		// dimensions
		"((\\-?)(\\d*)x(\\-?)(\\d*)/)?" +
		// paddings
		"((\\d+)x(\\d+)(:(\\d+)x(\\d+))?/)?" +
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

// Parse Params struct from Imagor endpoint URI
func Parse(path string) Params {
	var p Params
	return Apply(p, path)
}

// Apply Params struct from Imagor endpoint URI on top of existing Params
func Apply(p Params, path string) Params {
	match := pathRegex.FindStringSubmatch(path)
	if len(match) < 6 {
		return p
	}
	index := 1
	if match[index] != "" {
		p.Params = true
	}
	index += 1
	if match[index+1] == "unsafe/" {
		p.Unsafe = true
	} else if len(match[index+2]) > 8 {
		p.Hash = match[index+2]
	}
	index += 3
	p.Path = match[index]

	match = paramsRegex.FindStringSubmatch(p.Path)
	if len(match) == 0 {
		return p
	}
	index = 1
	if match[index] != "" {
		p.Meta = true
	}
	index += 1
	if match[index] != "" {
		p.Trim = true
		p.TrimBy = TrimByTopLeft
		if s := match[index+2]; s != "" {
			p.TrimBy = s
		}
		p.TrimTolerance, _ = strconv.Atoi(match[index+4])
	}
	index += 5
	if match[index] != "" {
		p.CropLeft, _ = strconv.ParseFloat(match[index+1], 64)
		p.CropTop, _ = strconv.ParseFloat(match[index+3], 64)
		p.CropRight, _ = strconv.ParseFloat(match[index+5], 64)
		p.CropBottom, _ = strconv.ParseFloat(match[index+7], 64)
	}
	index += 9
	if match[index] != "" {
		p.FitIn = true
	}
	index += 1
	if match[index] != "" {
		p.Stretch = true
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
		p.PaddingLeft, _ = strconv.Atoi(match[index+1])
		p.PaddingTop, _ = strconv.Atoi(match[index+2])
		if match[index+3] != "" {
			p.PaddingRight, _ = strconv.Atoi(match[index+4])
			p.PaddingBottom, _ = strconv.Atoi(match[index+5])
		} else {
			p.PaddingRight = p.PaddingLeft
			p.PaddingBottom = p.PaddingTop
		}
	}
	index += 6
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
		p.Filters = append(p.Filters, parseFilters(match[index+1])...)
	}
	index += 2
	if str := match[index]; str != "" {
		p.Image = str
		if u, err := url.QueryUnescape(str); err == nil {
			p.Image = u
		}
	}
	return p
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
