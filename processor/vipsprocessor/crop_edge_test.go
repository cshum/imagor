package vipsprocessor

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/vipsgen/v2/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// edgeCanvas returns a 1920x1280 grey PNG whose last column and last row are
// red, so a crop that reaches the right or bottom edge is visible in the result.
func edgeCanvas(t *testing.T) []byte {
	t.Helper()
	const w, h = 1920, 1280
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	for y := 0; y < h; y++ {
		img.Set(w-1, y, color.RGBA{255, 0, 0, 255})
	}
	for x := 0; x < w; x++ {
		img.Set(x, h-1, color.RGBA{255, 0, 0, 255})
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// TestCropEdge covers the manual crop in the URL path (AxB:CxD) where the
// bottom-right point reaches or passes the image edge. The right and bottom
// coordinates are exclusive bounds, so the full extent is the image size - a
// crop clamped to width-1 silently drops the last column and row.
func TestCropEdge(t *testing.T) {
	// vips is started by TestMain; a Processor here must not Shutdown, which
	// would stop libvips for the rest of the package run.
	v := NewProcessor(WithLogger(zap.NewNop()))
	canvas := edgeCanvas(t)

	for _, tt := range []struct {
		path      string
		width     int
		height    int
		rightRed  bool // crop includes the last column
		bottomRed bool // crop includes the last row
	}{
		{path: "0x0:100x100/filters:format(png)", width: 100, height: 100},
		{path: "0x100:1920x1180/filters:format(png)", width: 1920, height: 1080, rightRed: true},
		{path: "0x0:1920x1280/filters:format(png)", width: 1920, height: 1280, rightRed: true, bottomRed: true},
		{path: "0x0:1.0x1.0/filters:format(png)", width: 1920, height: 1280, rightRed: true, bottomRed: true},
		{path: "0x0:5000x5000/filters:format(png)", width: 1920, height: 1280, rightRed: true, bottomRed: true},
		{path: "1919x0:1920x1280/filters:format(png)", width: 1, height: 1280, rightRed: true, bottomRed: true},
		{path: "0x100:1920x1180/full-fit-in/1920x1080/filters:format(png)", width: 1920, height: 1080, rightRed: true},
	} {
		t.Run(tt.path, func(t *testing.T) {
			blob, err := v.Process(
				context.Background(), imagor.NewBlobFromBytes(canvas), imagorpath.Parse(tt.path), nil)
			require.NoError(t, err)
			buf, err := blob.ReadAll()
			require.NoError(t, err)
			img, err := vips.NewImageFromBuffer(buf, nil)
			require.NoError(t, err)
			defer img.Close()

			require.Equal(t, tt.width, img.Width(), "width")
			require.Equal(t, tt.height, img.Height(), "height")
			if tt.rightRed {
				require.NoError(t, img.ExtractArea(img.Width()-1, 0, 1, 1))
				px, err := img.Getpoint(0, 0, nil)
				require.NoError(t, err)
				require.Len(t, px, 3)
				assert.InDelta(t, 255, px[0], 1, "last column should be the red edge")
			}
			if tt.bottomRed {
				require.NoError(t, img.ExtractArea(0, img.Height()-1, 1, 1))
				px, err := img.Getpoint(0, 0, nil)
				require.NoError(t, err)
				require.Len(t, px, 3)
				assert.InDelta(t, 255, px[0], 1, "last row should be the red edge")
			}
		})
	}
}
