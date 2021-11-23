package imagor

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type Params struct {
	Image           string
	CropLeft        float64
	CropTop         float64
	CropRight       float64
	CropBottom      float64
	Width           float64
	Height          float64
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

func Parse(u *url.URL) (params *Params, err error) {
	var from int
	params = &Params{}
	segs := strings.Split(u.Path, "/")
	fmt.Println(strings.Join(paramsRegex.FindStringSubmatch(u.Path), " | "))
	for to, seg := range segs {
		if seg == "" {
			from++
		}
		if strings.HasPrefix(seg, "http") {
			params.Image, _ = url.QueryUnescape(strings.Join(segs[to:], "/"))
			segs = segs[from:to]
			return
		}
	}
	return
}

//func testing() {
//	var from int
//	results = &Params{}
//	match := paramsRegex.FindStringSubmatch(u.Path)
//	index := 1
//	ln := len(match)
//	if ln > index+1 && match[index+1] == "unsafe/" {
//		results.Unsafe = true
//	} else if ln > index+3 {
//		if h := match[index+2]; len(h) <= 28 {
//			results.Hash = match[index+3]
//		}
//	}
//	index += 4
//	fmt.Println(match)
//}
