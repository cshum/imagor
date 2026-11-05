package vipsprocessor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/vipsgen/v2/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadUltraHDR(t *testing.T) *vips.Image {
	t.Helper()
	if !vips.HasOperation("uhdrload") {
		t.Skip("libvips is built without libultrahdr")
	}
	img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"), nil)
	require.NoError(t, err)
	return img
}

func TestGainmapFilters(t *testing.T) {
	src := loadUltraHDR(t)
	srcWidth, srcHeight := src.Width(), src.PageHeight()
	src.Close()

	t.Run("crop", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		processed, err := crop(context.Background(), img, nil, "0.25", "0.25", "0.5", "0.5")
		require.NoError(t, err)
		assert.True(t, processed)
		assert.InDelta(t, srcWidth/2, img.Width(), 2)
		assert.InDelta(t, srcHeight/2, img.PageHeight(), 2)
		width, height := gainmapSize(t, img)
		assert.InDelta(t, img.Width()/2, width, 2)
		assert.InDelta(t, img.PageHeight()/2, height, 2)
	})

	t.Run("proportion", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		processed, err := proportion(context.Background(), img, nil, "50")
		require.NoError(t, err)
		assert.True(t, processed)
		assert.InDelta(t, srcWidth/2, img.Width(), 2)
		assert.InDelta(t, srcHeight/2, img.PageHeight(), 2)
		width, height := gainmapSize(t, img)
		assert.InDelta(t, img.Width()/2, width, 2)
		assert.InDelta(t, img.PageHeight()/2, height, 2)
	})

	t.Run("trim", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		opts := &vips.EmbedMultiPageOptions{Extend: vips.ExtendWhite}
		require.NoError(t, embedGainmap(img, 60, 60, srcWidth+120, srcHeight+120, opts))
		require.NoError(t, img.EmbedMultiPage(60, 60, srcWidth+120, srcHeight+120, opts))

		processed, err := trim(context.Background(), img, nil, "10")
		require.NoError(t, err)
		assert.True(t, processed)
		assert.InDelta(t, srcWidth, img.Width(), 4)
		assert.InDelta(t, srcHeight, img.PageHeight(), 4)
		gainmapWidth, gainmapHeight := gainmapSize(t, img)
		assert.InDelta(t, img.Width()/2, gainmapWidth, 4)
		assert.InDelta(t, img.PageHeight()/2, gainmapHeight, 4)
	})
}

func TestGainmapHelperEdges(t *testing.T) {
	t.Run("crop clamps to the gain map", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		width, height := gainmapSize(t, img)
		require.NoError(t, cropGainmap(img, -10, -10, 100000, 100000))
		gotWidth, gotHeight := gainmapSize(t, img)
		assert.Equal(t, width, gotWidth)
		assert.Equal(t, height, gotHeight)
	})

	t.Run("crop with no area is a no-op", func(t *testing.T) {
		img := loadUltraHDR(t)
		defer img.Close()
		require.NoError(t, cropGainmap(img, 10, 10, 0, 0))
	})

	t.Run("images without a gain map are skipped", func(t *testing.T) {
		img, err := vips.NewImageFromFile(filepath.Join(testDataDir, "demo1.jpg"), nil)
		require.NoError(t, err)
		defer img.Close()
		require.NoError(t, attachGainmap(img))
		require.NoError(t, cropGainmap(img, 10, 10, 100, 100))
		require.NoError(t, embedGainmap(img, 10, 10, img.Width()+20, img.PageHeight()+20,
			&vips.EmbedMultiPageOptions{Extend: vips.ExtendBlack}))
		require.NoError(t, rotateGainmap(img, 90))
		require.NoError(t, flipGainmap(img, vips.DirectionHorizontal))
	})
}

func TestUltraHDRPipeline(t *testing.T) {
	src := loadUltraHDR(t)
	srcWidth, srcHeight := src.Width(), src.PageHeight()
	gainWidth, gainHeight := gainmapSize(t, src)
	scaleX := float64(gainWidth) / float64(srcWidth)
	scaleY := float64(gainHeight) / float64(srcHeight)
	src.Close()

	v := NewProcessor()
	load := func(string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg")), nil
	}

	tests := []struct {
		path          string
		width, height int
	}{
		{"100x100/ultra-hdr.jpg", 100, 100},
		{"-100x100/ultra-hdr.jpg", 100, 100},
		{"100x100/10x10/filters:fill(white)/ultra-hdr.jpg", 120, 120},
		{"filters:proportion(50)/ultra-hdr.jpg", int(float64(srcWidth) * 0.5), int(float64(srcHeight) * 0.5)},
		{"filters:crop(0.1,0.1,0.8,0.8)/ultra-hdr.jpg", 0, 0},
		{"filters:rotate(90)/ultra-hdr.jpg", srcHeight, srcWidth},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			blob := imagor.NewBlobFromFile(filepath.Join(testDataDir, "ultra-hdr.jpg"))
			out, err := v.Process(context.Background(), blob, imagorpath.Parse(tt.path), load)
			require.NoError(t, err)
			data, err := out.ReadAll()
			require.NoError(t, err)

			img, err := vips.NewImageFromBuffer(data, nil)
			require.NoError(t, err)
			defer img.Close()
			if tt.width > 0 {
				assert.InDelta(t, tt.width, img.Width(), 2)
				assert.InDelta(t, tt.height, img.PageHeight(), 2)
			}
			gotWidth, gotHeight := gainmapSize(t, img)
			assert.InDelta(t, float64(img.Width())*scaleX, gotWidth, 3)
			assert.InDelta(t, float64(img.PageHeight())*scaleY, gotHeight, 3)
		})
	}
}
