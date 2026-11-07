package imagor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/processor/vipsprocessor"
	"github.com/cshum/vipsgen/v2/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ultraHDRLoader func(*http.Request, string) (*imagor.Blob, error)

func (f ultraHDRLoader) Get(r *http.Request, image string) (*imagor.Blob, error) {
	return f(r, image)
}

// TestUltraHDRIPhoneEncode runs an iPhone photo, whose gain map has no colour
// profile, through the whole request path and checks the encoded result still
// carries a gain map. Without gainmap-use-base-cg the encode drops it.
func TestUltraHDRIPhoneEncode(t *testing.T) {
	app := imagor.New(
		imagor.WithUnsafe(true),
		imagor.WithLoaders(ultraHDRLoader(func(_ *http.Request, image string) (*imagor.Blob, error) {
			return imagor.NewBlobFromFile(filepath.Join("testdata", image)), nil
		})),
		imagor.WithProcessors(vipsprocessor.NewProcessor()),
	)
	require.NoError(t, app.Startup(context.Background()))
	defer func() { require.NoError(t, app.Shutdown(context.Background())) }()

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(
		http.MethodGet, "https://example.com/unsafe/fit-in/400x400/iphone-ultrahdr.jpg", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	img, err := vips.NewImageFromBuffer(w.Body.Bytes(), nil)
	require.NoError(t, err)
	defer img.Close()

	gainmap, ok := img.Gainmap()
	require.True(t, ok, "the encoded image lost its gain map")
	defer gainmap.Close()
	assert.InDelta(t, img.Width()/2, gainmap.Width(), 2)
	mean, err := gainmap.Avg()
	require.NoError(t, err)
	assert.Greater(t, mean, 1.0)
	assert.Less(t, mean, 254.0)
}
