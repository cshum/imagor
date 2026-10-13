package imagorpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveFullDim(t *testing.T) {
	for _, tc := range []struct {
		name      string
		token     string
		parentDim int
		want      string
	}{
		{"f", "f", 400, "400"},
		{"full", "full", 400, "400"},
		{"f with offset", "f-20", 400, "380"},
		{"full with offset", "full-20", 400, "380"},
		{"offset larger than parent", "f-500", 400, "-100"},
		{"flip f", "-f", 400, "-400"},
		{"flip full with offset", "-full-20", 400, "-380"},
		{"zero parent", "f", 0, "0"},
		{"negative parent", "f-10", -200, "-210"},

		// not a full-dim token: returned unchanged
		{"plain int", "300", 400, "300"},
		{"empty", "", 400, ""},
		{"bare f prefix only, no offset", "f-", 400, "f-"},
		{"unknown word", "foo", 400, "foo"},
		{"x", "x", 400, "x"},
		{"leading uppercase", "F", 400, "F"},
		{"offset not integer", "f-2a", 400, "f-2a"},
		{"percent", "50p", 400, "50p"},
		{"fraction", "0.5", 400, "0.5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveFullDim(tc.token, tc.parentDim))
		})
	}
}

func TestResolveFullDimensions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		path   string
		w, h   int
		want   string
		wantEq bool // want identical to input
	}{
		{
			name: "fxf resolves to the parent",
			path: "fxf/color:red", w: 400, h: 300,
			want: "400x300/color:red",
		},
		{
			name: "f with offsets",
			path: "f-20xf-20/color:red", w: 400, h: 300,
			want: "380x280/color:red",
		},
		{
			name: "full tokens",
			path: "full-20xfull-20/color:red", w: 400, h: 300,
			want: "380x280/color:red",
		},
		{
			name: "one side only",
			path: "300xf/color:red", w: 400, h: 300,
			want: "300x300/color:red",
		},
		{
			name: "other side only",
			path: "fx100/color:red", w: 400, h: 300,
			want: "400x100/color:red",
		},
		{
			name: "crop parameter before the dimension segment",
			path: "0x0:100x100/fxf/img.jpg", w: 400, h: 300,
			want: "0x0:100x100/400x300/img.jpg",
		},
		{
			name: "fit-in before the dimension segment",
			path: "fit-in/fxf/img.jpg", w: 400, h: 300,
			want: "fit-in/400x300/img.jpg",
		},
		{
			name: "filters preserved after the dimension segment",
			path: "fxf/filters:blur(5):grayscale()/img.jpg", w: 400, h: 300,
			want: "400x300/filters:blur(5):grayscale()/img.jpg",
		},
		{
			name: "meta prefix",
			path: "meta/fxf/color:red", w: 400, h: 300,
			want: "meta/400x300/color:red",
		},
		{
			name: "flip token",
			path: "-fxf/color:red", w: 400, h: 300,
			want: "-400x300/color:red",
		},

		// stops at filters: — nested layer paths resolve at their own level
		{
			name: "does not resolve inside filters:",
			path: "filters:image(fxf/color:red,0,0)/color:blue", w: 400, h: 300,
			wantEq: true,
		},
		{
			name: "filters before any dimension segment",
			path: "filters:blur(5)/fxf/img.jpg", w: 400, h: 300,
			wantEq: true,
		},

		// nothing to resolve
		{
			name: "plain dimensions",
			path: "300x200/color:red", w: 400, h: 300,
			wantEq: true,
		},
		{
			name: "no dimension segment",
			path: "color:red", w: 400, h: 300,
			wantEq: true,
		},
		{
			name: "empty path",
			path: "", w: 400, h: 300,
			wantEq: true,
		},
		{
			name: "bare f as the image",
			path: "f", w: 400, h: 300,
			wantEq: true,
		},
		{
			name: "no trailing segment after dimensions",
			path: "fxf", w: 400, h: 300,
			want: "400x300",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveFullDimensions(tc.path, tc.w, tc.h)
			if tc.wantEq {
				assert.Equal(t, tc.path, got, "path with nothing to resolve must be returned untouched")
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

// The point of resolving f-tokens before parsing is that the parser would
// otherwise read f-20xf-20 as a source, not a dimension: it ends up at the
// loader as a hostname. Assert the resolved path actually parses to the
// expected dimensions rather than merely looking right as a string.
func TestResolveFullDimensionsParsesToTheParentDimensions(t *testing.T) {
	for _, tc := range []struct {
		path  string
		w, h  int
		wantW int
		wantH int
	}{
		{"fxf/color:red", 400, 300, 400, 300},
		{"f-20xf-20/color:red", 400, 300, 380, 280},
		{"fit-in/fxf/img.jpg", 400, 300, 400, 300},
		{"0x0:100x100/fxf/img.jpg", 400, 300, 400, 300},
	} {
		t.Run(tc.path, func(t *testing.T) {
			p := Parse(ResolveFullDimensions(tc.path, tc.w, tc.h))
			assert.Equal(t, tc.wantW, p.Width)
			assert.Equal(t, tc.wantH, p.Height)
		})
	}
}
