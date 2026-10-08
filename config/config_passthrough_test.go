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

// TestPassthroughFlagWiring checks the operator surface reaches the parser: the
// flag is what an operator sets, so what it accepts has to be the policy the
// processor applies - a still format is served as-is, a refused one fails the
// process, and a separator typo is not a format.
func TestPassthroughFlagWiring(t *testing.T) {
	srv := CreateServer([]string{"-imagor-passthrough-formats", "svg,png"})
	assert.Equal(t, []imagor.BlobType{imagor.BlobTypeSVG, imagor.BlobTypePNG},
		srv.App.(*imagor.Imagor).PassthroughFormats)

	for _, value := range []string{"svg", "png", "jpeg", "svg,"} {
		t.Run("accepted "+value, func(t *testing.T) {
			assert.NotPanics(t, func() {
				CreateServer([]string{"-imagor-passthrough-formats", value})
			})
		})
	}

	for _, value := range []string{"pdf", "svvg", "exe", "svg,pdf"} {
		t.Run("refused "+value, func(t *testing.T) {
			assert.Panics(t, func() {
				CreateServer([]string{"-imagor-passthrough-formats", value})
			})
		})
	}
}
