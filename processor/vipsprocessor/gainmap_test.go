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
	gainmap, ok := img.Gainmap()
	require.True(t, ok)
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

func TestCropGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := gainmapSize(t, img)
	cropWidth, cropHeight := img.Width()/2, img.PageHeight()/2
	require.NoError(t, cropGainmap(img, img.Width()/4, img.PageHeight()/4, cropWidth, cropHeight))
	gotWidth, gotHeight := gainmapSize(t, img)
	assert.InDelta(t, width/2, gotWidth, 2)
	assert.InDelta(t, height/2, gotHeight, 2)
}

func TestFlipGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := gainmapSize(t, img)
	require.NoError(t, flipGainmap(img, vips.DirectionHorizontal))
	gotWidth, gotHeight := gainmapSize(t, img)
	assert.Equal(t, width, gotWidth)
	assert.Equal(t, height, gotHeight)
}

func TestAttachGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	require.False(t, img.HasField("gainmap"))
	require.NoError(t, attachGainmap(img))
	assert.True(t, img.HasField("gainmap"))
}

func TestEmbedGainmap(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := img.Width(), img.PageHeight()
	gainmapWidth, gainmapHeight := gainmapSize(t, img)
	require.NoError(t, embedGainmap(img, 10, 20, width+20, height+40,
		&vips.EmbedOptions{Extend: vips.ExtendBlack}))
	require.NoError(t, img.EmbedMultiPage(10, 20, width+20, height+40,
		&vips.EmbedMultiPageOptions{Extend: vips.ExtendBlack}))
	gotWidth, gotHeight := gainmapSize(t, img)
	assert.InDelta(t, img.Width()*gainmapWidth/width, gotWidth, 2)
	assert.InDelta(t, img.PageHeight()*gainmapHeight/height, gotHeight, 2)
}

func TestRotateGainmapArbitraryAngle(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	require.NoError(t, rotateGainmap(img, 45))
	width, height := gainmapSize(t, img)
	assert.InDelta(t, width, height, 2)
}

func TestFlipGainmapVertical(t *testing.T) {
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	width, height := gainmapSize(t, img)
	require.NoError(t, flipGainmap(img, vips.DirectionVertical))
	gotWidth, gotHeight := gainmapSize(t, img)
	assert.Equal(t, width, gotWidth)
	assert.Equal(t, height, gotHeight)
}

func TestRotateGainmapWithoutGainmap(t *testing.T) {
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "demo1.jpg"), nil)
	require.NoError(t, err)
	defer img.Close()

	assert.False(t, img.HasField("gainmap-data"))
	assert.NoError(t, rotateGainmap(img, 90))
	assert.False(t, img.HasField("gainmap-data"))
}

func TestAngleDegrees(t *testing.T) {
	tests := []struct {
		angle vips.Angle
		want  float64
	}{
		{vips.AngleD0, 0},
		{vips.AngleD90, 90},
		{vips.AngleD180, 180},
		{vips.AngleD270, 270},
		{vips.Angle(9), 0},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, angleDegrees(tt.angle))
	}
}

func TestEmbedGainmapUsesNeutralGain(t *testing.T) {
	img := loadUltraHDR(t)
	defer img.Close()

	require.NoError(t, embedGainmap(img, 10, 10, img.Width()+20, img.PageHeight()+20,
		&vips.EmbedOptions{Extend: vips.ExtendWhite}))
	require.NoError(t, img.EmbedMultiPage(10, 10, img.Width()+20, img.PageHeight()+20,
		&vips.EmbedMultiPageOptions{Extend: vips.ExtendWhite}))

	gainmap, ok := img.Gainmap()
	require.True(t, ok)
	defer gainmap.Close()
	corner, err := gainmap.Getpoint(0, 0, nil)
	require.NoError(t, err)
	assert.Less(t, corner[0], 10.0, "padding carries no boost, not the fill colour")
}

func TestCropGainmapTinyArea(t *testing.T) {
	img := loadUltraHDR(t)
	defer img.Close()

	width, height := img.Width(), img.PageHeight()
	require.NoError(t, cropGainmap(img, width-1, height-1, width, height))

	gainWidth, gainHeight := gainmapSize(t, img)
	assert.GreaterOrEqual(t, gainWidth, 1)
	assert.GreaterOrEqual(t, gainHeight, 1)
}
