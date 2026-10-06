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
		{"spaces around", " svg ", []imagor.BlobType{imagor.BlobTypeSVG}},
		{"trailing comma", "svg,", []imagor.BlobType{imagor.BlobTypeSVG}},
		{"repeats collapse", "svg,svg,SVG", []imagor.BlobType{imagor.BlobTypeSVG}},
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
// also pins the messages an operator sees, which name what was expected.
func TestParsePassthroughFormatsPanics(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"typo", "svvg"},
		{"unknown format", "exe"},
		{"refused format", "pdf"},
		{"refused among valid", "svg,pdf"},
		{"still format", "png"},
		{"still format among valid", "svg,webp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() { parsePassthroughFormats(tc.in) })
		})
	}

	assert.PanicsWithError(t,
		"imagor: unknown passthrough format \"svvg\", expected one of svg",
		func() { parsePassthroughFormats("svvg") })
	assert.PanicsWithError(t,
		"imagor: passthrough format \"pdf\" is refused: its bytes are active content in a browser",
		func() { parsePassthroughFormats("pdf") })
	assert.PanicsWithError(t,
		"imagor: passthrough format \"png\" is not configurable: only a source that is sanitized before it is served, such as svg, can be set here",
		func() { parsePassthroughFormats("png") })
}

// TestPassthroughFlagWiring checks the operator surface reaches the parser: the
// flag is what an operator actually sets, and a value this release does not
// serve has to fail the process, not the request.
func TestPassthroughFlagWiring(t *testing.T) {
	srv := CreateServer([]string{"-imagor-passthrough-formats", "svg"})
	assert.Equal(t, []imagor.BlobType{imagor.BlobTypeSVG},
		srv.App.(*imagor.Imagor).PassthroughFormats)

	for _, value := range []string{"png", "pdf", "svvg", "svg,png", "svg,"} {
		t.Run(value, func(t *testing.T) {
			if value == "svg," {
				// A trailing comma is a typo in the separators, not a format.
				assert.NotPanics(t, func() {
					CreateServer([]string{"-imagor-passthrough-formats", value})
				})
				return
			}
			assert.Panics(t, func() {
				CreateServer([]string{"-imagor-passthrough-formats", value})
			})
		})
	}
}
