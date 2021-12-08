package params

import (
	"reflect"
	"testing"
)

func TestURL_ParseGenerate(t *testing.T) {
	tests := []struct {
		u      URL
		name   string
		url    string
		params Params
	}{
		{
			name: "unsafe",
			u:    NewURLUnsafe("http://localhost:8000"),
			url:  "http://localhost:8000/unsafe/-0x0/https://foo.com/bar.png",
			params: Params{
				HFlip: true,
				Image: "https://foo.com/bar.png",
			},
		},
		{
			name: "unsafe with query",
			u:    NewURLUnsafe("http://localhost:8000"),
			url:  "http://localhost:8000/unsafe/fit-in/https%3A%2F%2Ffoo.com%2Fbar.png%3Fv%3D1234",
			params: Params{
				FitIn: true,
				Image: "https://foo.com/bar.png?v=1234",
			},
		},
		{
			u:    NewURL("https://localhost:8000", "1234"),
			name: "unsafe",
			url:  "https://localhost:8000/3De8zlPL3Dty08swVLGEcdpI1tc=/-0x0/https://foo.com/bar.png",
			params: Params{
				HFlip: true,
				Image: "https://foo.com/bar.png",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.u.Generate(tt.params); tt.url != result {
				t.Errorf(" = %s, want = %s", result, tt.url)
			}
			if result, ok := tt.u.Parse(tt.url); !ok || reflect.DeepEqual(result, tt.params) {
				t.Errorf(" = %v, want = %v", result, tt.params)
			}
		})
	}
}
