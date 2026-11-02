package vipsprocessor

import (
	"path/filepath"
	"testing"

	"github.com/cshum/vipsgen/v2/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gainmapSize(t *testing.T, img *vips.Image) (int, int) {
	t.Helper()
	data, err := img.GetBlob("gainmap-data")
	require.NoError(t, err)
	gainmap, err := vips.NewImageFromBuffer(data, nil)
	require.NoError(t, err)
	defer gainmap.Close()
	return gainmap.Width(), gainmap.PageHeight()
}

func TestRotateGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := gainmapSize(t, img)
	require.NoError(t, rotateGainmap(img, 90))
	rotatedWidth, rotatedHeight := gainmapSize(t, img)
	assert.Equal(t, height, rotatedWidth)
	assert.Equal(t, width, rotatedHeight)
}

func TestRotateGainmapWithoutGainmap(t *testing.T) {
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "demo1.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	assert.False(t, img.HasField("gainmap-data"))
	assert.NoError(t, rotateGainmap(img, 90))
	assert.False(t, img.HasField("gainmap-data"))
}
