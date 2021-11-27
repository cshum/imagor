package imagor

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected Params
		secret   string
	}{
		{
			name: "non url image",
			uri:  "/meta/10x11:12x13/fit-in/-300x-200/left/top/smart/filters:some_filter()/img",
			expected: Params{
				Path:           "meta/10x11:12x13/fit-in/-300x-200/left/top/smart/filters:some_filter()/img",
				Image:          "img",
				CropLeft:       10,
				CropTop:        11,
				CropRight:      12,
				CropBottom:     13,
				Width:          300,
				Height:         200,
				Meta:           true,
				HorizontalFlip: true,
				VerticalFlip:   true,
				HAlign:         "left",
				VAlign:         "top",
				Smart:          true,
				FitIn:          true,
				Filters:        []Filter{{Type: "some_filter"}},
			},
		},
		{
			name: "url image",
			uri:  "/meta/10x11:12x13/fit-in/-300x-200/left/top/smart/filters:some_filter()/s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg",
			expected: Params{
				Path:           "meta/10x11:12x13/fit-in/-300x-200/left/top/smart/filters:some_filter()/s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg",
				Image:          "s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg",
				CropLeft:       10,
				CropTop:        11,
				CropRight:      12,
				CropBottom:     13,
				Width:          300,
				Height:         200,
				Meta:           true,
				HorizontalFlip: true,
				VerticalFlip:   true,
				HAlign:         "left",
				VAlign:         "top",
				Smart:          true,
				FitIn:          true,
				Filters:        []Filter{{Type: "some_filter"}},
			},
		},
		{
			name: "url in filter",
			uri:  "/filters:watermark(s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0)/img",
			expected: Params{
				Path:    "filters:watermark(s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0)/img",
				Image:   "img",
				Filters: []Filter{{Type: "watermark", Args: "s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0"}},
			},
		},
		{
			name: "multiple filters",
			uri:  "/filters:watermark(s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0):brightness(-50):grayscale()/img",
			expected: Params{
				Path:  "filters:watermark(s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0):brightness(-50):grayscale()/img",
				Image: "img",
				Filters: []Filter{
					{
						Type: "watermark",
						Args: "s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0",
					},
					{
						Type: "brightness",
						Args: "-50",
					},
					{
						Type: "grayscale",
					},
				},
			},
		},
		{
			name: "no params",
			uri:  "/unsafe/https://thumbor.readthedocs.io/en/latest/_images/man_before_sharpen.png",
			expected: Params{
				Path:   "https://thumbor.readthedocs.io/en/latest/_images/man_before_sharpen.png",
				Image:  "https://thumbor.readthedocs.io/en/latest/_images/man_before_sharpen.png",
				Unsafe: true,
			},
		},
		{
			name: "url in filters",
			uri:  "/unsafe/stretch/500x350/filters:watermark(http://thumborize.me/static/img/beach.jpg,100,100,50)/http://thumborize.me/static/img/beach.jpg?v=93ce8775572809c2fa498f3ba53c9ef6",
			expected: Params{
				Path:    "stretch/500x350/filters:watermark(http://thumborize.me/static/img/beach.jpg,100,100,50)/http://thumborize.me/static/img/beach.jpg?v=93ce8775572809c2fa498f3ba53c9ef6",
				Image:   "http://thumborize.me/static/img/beach.jpg?v=93ce8775572809c2fa498f3ba53c9ef6",
				Width:   500,
				Height:  350,
				Unsafe:  true,
				Stretch: true,
				Filters: []Filter{
					{
						Type: "watermark",
						Args: "http://thumborize.me/static/img/beach.jpg,100,100,50",
					},
				},
			},
		},
	}
	for _, test := range tests {
		if test.name == "" {
			test.name = test.uri
		}
		t.Run(strings.TrimPrefix(test.name, "/"), func(t *testing.T) {
			test.expected.URI = test.uri
			resp := ParseParams(test.uri)
			respJSON, _ := json.MarshalIndent(resp, "", "  ")
			expectedJSON, _ := json.MarshalIndent(test.expected, "", "  ")
			if !reflect.DeepEqual(resp, test.expected) {
				t.Errorf(" = %s, want %s", string(respJSON), string(expectedJSON))
			}
			if test.secret != "" && !resp.Verify(test.secret) {
				t.Errorf("hash %s mismatch", resp.Hash)
			}
		})
	}
}
