package imagorurl

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseGenerate(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		params Params
		secret string
	}{
		{
			name: "non url image",
			uri:  "meta/10x11:12x13/fit-in/-300x-200/left/top/smart/filters:some_filter()/img",
			params: Params{
				Path:       "meta/10x11:12x13/fit-in/-300x-200/left/top/smart/filters:some_filter()/img",
				Image:      "img",
				CropLeft:   10,
				CropTop:    11,
				CropRight:  12,
				CropBottom: 13,
				Width:      300,
				Height:     200,
				Meta:       true,
				HFlip:      true,
				VFlip:      true,
				HAlign:     "left",
				VAlign:     "top",
				Smart:      true,
				FitIn:      true,
				Filters:    []Filter{{Name: "some_filter"}},
			},
		},
		{
			name: "url image",
			uri:  "meta/10x11:12x13/fit-in/upscale/-300x-200/left/top/smart/filters:some_filter()/s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg",
			params: Params{
				Path:       "meta/10x11:12x13/fit-in/upscale/-300x-200/left/top/smart/filters:some_filter()/s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg",
				Image:      "s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg",
				CropLeft:   10,
				CropTop:    11,
				CropRight:  12,
				CropBottom: 13,
				Width:      300,
				Height:     200,
				Meta:       true,
				Upscale:    true,
				HFlip:      true,
				VFlip:      true,
				HAlign:     "left",
				VAlign:     "top",
				Smart:      true,
				FitIn:      true,
				Filters:    []Filter{{Name: "some_filter"}},
			},
		},
		{
			name: "url in filter",
			uri:  "filters:watermark(s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0)/img",
			params: Params{
				Path:    "filters:watermark(s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0)/img",
				Image:   "img",
				Filters: []Filter{{Name: "watermark", Args: "s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0"}},
			},
		},
		{
			name: "multiple filters",
			uri:  "filters:watermark(s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0):brightness(-50):grayscale()/img",
			params: Params{
				Path:  "filters:watermark(s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0):brightness(-50):grayscale()/img",
				Image: "img",
				Filters: []Filter{
					{
						Name: "watermark",
						Args: "s.glbimg.com/es/ge/f/original/2011/03/29/orlandosilva_60.jpg,0,0,0",
					},
					{
						Name: "brightness",
						Args: "-50",
					},
					{
						Name: "grayscale",
					},
				},
			},
		},
		{
			name: "no params",
			uri:  "unsafe/https://thumbor.readthedocs.io/en/latest/_images/man_before_sharpen.png",
			params: Params{
				Path:   "https://thumbor.readthedocs.io/en/latest/_images/man_before_sharpen.png",
				Image:  "https://thumbor.readthedocs.io/en/latest/_images/man_before_sharpen.png",
				Unsafe: true,
			},
		},
		{
			name: "url in filters",
			uri:  "unsafe/stretch/500x350/filters:watermark(http://thumborize.me/static/img/beach.jpg,100,100,50)/http://thumborize.me/static/img/beach.jpg",
			params: Params{
				Path:    "stretch/500x350/filters:watermark(http://thumborize.me/static/img/beach.jpg,100,100,50)/http://thumborize.me/static/img/beach.jpg",
				Image:   "http://thumborize.me/static/img/beach.jpg",
				Width:   500,
				Height:  350,
				Unsafe:  true,
				Stretch: true,
				Filters: []Filter{
					{
						Name: "watermark",
						Args: "http://thumborize.me/static/img/beach.jpg,100,100,50",
					},
				},
			},
		},
		{
			name:   "non url image with hash",
			uri:    "VTAq7YIRbEXgtwAcsTMhAjvBuT8=/meta/10x11:12x13/fit-in/-300x-200/5x6/left/top/smart/filters:some_filter()/img",
			secret: "1234",
			params: Params{
				Path:       "meta/10x11:12x13/fit-in/-300x-200/5x6/left/top/smart/filters:some_filter()/img",
				Hash:       "VTAq7YIRbEXgtwAcsTMhAjvBuT8=",
				Image:      "img",
				CropLeft:   10,
				CropTop:    11,
				CropRight:  12,
				CropBottom: 13,
				Width:      300,
				Height:     200,
				Meta:       true,
				HFlip:      true,
				VFlip:      true,
				HAlign:     "left",
				VAlign:     "top",
				Smart:      true,
				FitIn:      true,
				HPadding:   5,
				VPadding:   6,
				Filters:    []Filter{{Name: "some_filter"}},
			},
		},
	}
	for _, test := range tests {
		if test.name == "" {
			test.name = test.uri
		}
		t.Run(strings.TrimPrefix(test.name, "/"), func(t *testing.T) {
			resp := Parse(test.uri)
			respJSON, _ := json.MarshalIndent(resp, "", "  ")
			expectedJSON, _ := json.MarshalIndent(test.params, "", "  ")
			if !reflect.DeepEqual(resp, test.params) {
				t.Errorf(" = %s, want %s", string(respJSON), string(expectedJSON))
			}
			if test.secret != "" && Sign(resp.Path, test.secret) != resp.Hash {
				t.Errorf("signature mismatch = %s, want %s", resp.Hash, Sign(resp.Path, test.secret))
			}
			if test.params.Hash != "" {
				if uri := Generate(test.params, test.secret); uri != test.uri {
					t.Errorf(" = %s, want = %s", uri, test.uri)
				}
			} else if test.params.Unsafe {
				if uri := GenerateUnsafe(test.params); uri != test.uri {
					t.Errorf(" = %s, want = %s", uri, test.uri)
				}
			} else {
				if uri := generate(test.params); uri != test.uri {
					t.Errorf(" = %s, want = %s", uri, test.uri)
				}
			}
		})
	}
}
