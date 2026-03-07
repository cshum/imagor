package vipsprocessor

import "testing"

func TestResolveFullDim(t *testing.T) {
	tests := []struct {
		token     string
		parentDim int
		want      string
	}{
		{"f", 800, "800"},
		{"full", 800, "800"},
		{"f-20", 800, "780"},
		{"full-20", 800, "780"},
		{"-f", 800, "-800"},
		{"-f-20", 800, "-780"},
		{"400", 800, "400"},
		{"", 800, ""},
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got := resolveFullDim(tt.token, tt.parentDim)
			if got != tt.want {
				t.Errorf("resolveFullDim(%q, %d) = %q, want %q", tt.token, tt.parentDim, got, tt.want)
			}
		})
	}
}

func TestResolveFullDimensions(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		parentW int
		parentH int
		want    string
	}{
		{
			name:    "simple f-token",
			path:    "fxf/filters:format(png)/image.jpg",
			parentW: 800, parentH: 600,
			want: "800x600/filters:format(png)/image.jpg",
		},
		{
			name:    "f-token with offset",
			path:    "f-20xf-30/filters:format(png)/image.jpg",
			parentW: 800, parentH: 600,
			want: "780x570/filters:format(png)/image.jpg",
		},
		{
			name:    "no f-token",
			path:    "400x300/filters:format(png)/image.jpg",
			parentW: 800, parentH: 600,
			want: "400x300/filters:format(png)/image.jpg",
		},
		{
			name:    "nested layer - should NOT resolve nested f-tokens",
			path:    "1551x2162/filters:image(/f-141xf-1145/img1,106,400)/img2",
			parentW: 3840, parentH: 2560,
			want: "1551x2162/filters:image(/f-141xf-1145/img1,106,400)/img2",
		},
		{
			name:    "nested layer with outer f-token",
			path:    "f-100xf-200/filters:image(/f-141xf-1145/img1,106,400)/img2",
			parentW: 3840, parentH: 2560,
			want: "3740x2360/filters:image(/f-141xf-1145/img1,106,400)/img2",
		},
		{
			name:    "only f-token no filters",
			path:    "fxf/image.jpg",
			parentW: 800, parentH: 600,
			want: "800x600/image.jpg",
		},
		{
			name:    "mixed f and number",
			path:    "fx300/image.jpg",
			parentW: 800, parentH: 600,
			want: "800x300/image.jpg",
		},
		{
			name:    "flip with f-token",
			path:    "-fxf/image.jpg",
			parentW: 800, parentH: 600,
			want: "-800x600/image.jpg",
		},
		{
			name:    "no dimension segment",
			path:    "filters:format(png)/image.jpg",
			parentW: 800, parentH: 600,
			want: "filters:format(png)/image.jpg",
		},
		{
			name:    "empty path",
			path:    "",
			parentW: 800, parentH: 600,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFullDimensions(tt.path, tt.parentW, tt.parentH)
			if got != tt.want {
				t.Errorf("resolveFullDimensions(%q, %d, %d) =\n  %q\nwant:\n  %q",
					tt.path, tt.parentW, tt.parentH, got, tt.want)
			}
		})
	}
}
