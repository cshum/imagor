package imagorpath

import "testing"

func TestJoin(t *testing.T) {
	for _, tt := range []struct {
		base, key string
		want      string
		ok        bool
	}{
		{"", "photo.jpg", "photo.jpg", true},
		{"/", "photo.jpg", "photo.jpg", true},
		{"home/imagor", "photo.jpg", "home/imagor/photo.jpg", true},
		{"home/imagor", "/photo.jpg", "home/imagor/photo.jpg", true},
		{"home/imagor", "sub/photo.jpg", "home/imagor/sub/photo.jpg", true},
		{"home/imagor", "sub/../photo.jpg", "home/imagor/photo.jpg", true},
		{"home/imagor", "..", "", false},
		{"home/imagor", "../other/photo.jpg", "", false},
		{"home/imagor", "../../etc/passwd", "", false},
		{"home/imagor", "sub/../../other/photo.jpg", "", false},
		{"home/imagor/", "photo.jpg", "home/imagor/photo.jpg", true},
		{"home/imagor", "a..b/photo.jpg", "home/imagor/a..b/photo.jpg", true},
		{"home/imagor", "../imagorX/photo.jpg", "", false},
		{"", "../other/photo.jpg", "", false},
		{"/", "../other/photo.jpg", "", false},
		{"", "..", "", false},
		{"", "..photo.jpg", "..photo.jpg", true},
		{"", "a/../../x", "", false},
		{"", "a/b/../../../y.jpg", "", false},
		{"home/imagor", "a/../../x", "", false},
	} {
		got, ok := ConfineJoin(tt.base, tt.key)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ConfineJoin(%q, %q) = %q,%v want %q,%v", tt.base, tt.key, got, ok, tt.want, tt.ok)
		}
	}
}
