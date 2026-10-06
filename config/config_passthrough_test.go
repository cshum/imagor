package config

import (
	"testing"

	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
)

// TestParsePassthroughFormats covers the csv the operator writes: separators,
// spacing, case and repeats are all tolerated, and the order is kept.
func TestParsePassthroughFormats(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want []imagor.BlobType
	}{
		{"single", "svg", []imagor.BlobType{imagor.BlobTypeSVG}},
		{"uppercase", "SVG", []imagor.BlobType{imagor.BlobTypeSVG}},
		{"csv with spaces", " svg , png ", []imagor.BlobType{imagor.BlobTypeSVG, imagor.BlobTypePNG}},
		{"trailing comma", "svg,", []imagor.BlobType{imagor.BlobTypeSVG}},
		{"repeats collapse", "svg,svg,PNG,png", []imagor.BlobType{imagor.BlobTypeSVG, imagor.BlobTypePNG}},
		{"order kept", "webp,png,avif", []imagor.BlobType{imagor.BlobTypeWEBP, imagor.BlobTypePNG, imagor.BlobTypeAVIF}},
		{"alias for jpeg", "jpg", []imagor.BlobType{imagor.BlobTypeJPEG}},
		{"unset", "", nil},
		{"separators only", ",,", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parsePassthroughFormats(tc.in))
		})
	}
}

// TestParsePassthroughFormatsPanics pins the startup behaviour. A name the
// server would never serve is a misconfiguration, and a container that refuses
// to start says so in one line, where ignoring the value would leave the
// feature quietly off - the silent substitution issue #835 was about. The test
// also pins the message an operator sees, which names what was expected.
func TestParsePassthroughFormatsPanics(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"typo", "svvg"},
		{"unknown format", "exe"},
		{"refused format", "pdf"},
		{"refused among valid", "svg,pdf"},
		{"unknown among valid", "svg,exe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() { parsePassthroughFormats(tc.in) })
		})
	}

	assert.PanicsWithError(t,
		"imagor: unknown passthrough format \"svvg\", expected one of avif, bmp, gif, heif, jp2, jpeg, jpg, jxl, png, svg, tiff, webp",
		func() { parsePassthroughFormats("svvg") })
	assert.PanicsWithError(t,
		"imagor: passthrough format \"pdf\" is refused: its bytes are active content in a browser",
		func() { parsePassthroughFormats("pdf") })
}
