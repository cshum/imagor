package vipsprocessor

import (
	"bytes"
	"context"
	"image"
	_ "image/png"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The branches differ by colour, so the rendered pixel says which language was
// selected without anyone having to read glyphs. There is no unconditional
// branch: a document whose condition is left to the renderer draws nothing here,
// which is what makes this pin the selection rather than the fixture.
const langFixture = `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32">
<switch>
<rect systemLanguage="fr" x="0" y="0" width="32" height="32" fill="#ff0000"/>
<rect systemLanguage="en" x="0" y="0" width="32" height="32" fill="#0000ff"/>
</switch>
</svg>`

func TestProcessorLangFilterSelectsTheRenderedLanguage(t *testing.T) {
	v := NewProcessor()
	for _, tc := range []struct {
		lang string
		want [3]uint32
	}{
		{"fr", [3]uint32{255, 0, 0}},
		{"en", [3]uint32{0, 0, 255}},
	} {
		t.Run(tc.lang, func(t *testing.T) {
			out, err := v.Process(context.Background(),
				imagor.NewBlobFromBytes([]byte(langFixture)),
				imagorpath.Params{
					Image: "photo.svg",
					Filters: []imagorpath.Filter{
						{Name: "lang", Args: tc.lang},
						{Name: "format", Args: "png"},
					},
				}, nil)
			require.NoError(t, err)
			data, err := out.ReadAll()
			require.NoError(t, err)
			img, _, err := image.Decode(bytes.NewReader(data))
			require.NoError(t, err)
			r, g, b, _ := img.At(16, 16).RGBA()
			// 16-bit channels; a colour profile round trip may shift a step.
			assert.InDelta(t, float64(tc.want[0])*257, float64(r), 257*4, "red")
			assert.InDelta(t, float64(tc.want[1])*257, float64(g), 257*4, "green")
			assert.InDelta(t, float64(tc.want[2])*257, float64(b), 257*4, "blue")
		})
	}
}

// TestProcessorLangFilterRejectsAMalformedTag: the filter value is caller input,
// so a value that is not a language tag is a request error rather than something
// to pass to the renderer.
func TestProcessorLangFilterRejectsAMalformedTag(t *testing.T) {
	v := NewProcessor()
	_, err := v.Process(context.Background(),
		imagor.NewBlobFromBytes([]byte(langFixture)),
		imagorpath.Params{
			Image:   "photo.svg",
			Filters: []imagorpath.Filter{{Name: "lang", Args: "!!"}},
		}, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, imagor.ErrInvalid)
}
