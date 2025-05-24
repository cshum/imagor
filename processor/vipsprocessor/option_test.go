package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestWithOption(t *testing.T) {
	t.Run("options", func(t *testing.T) {
		v := NewProcessor(
			WithConcurrency(2),
			WithMaxFilterOps(167),
			WithMaxCacheSize(500),
			WithMaxCacheMem(501),
			WithMaxCacheFiles(10),
			WithMaxWidth(999),
			WithMaxHeight(998),
			WithMaxResolution(1666667),
			WithMozJPEG(true),
			WithAvifSpeed(9),
			WithStripMetadata(true),
			WithDebug(true),
			WithMaxAnimationFrames(3),
			WithDisableFilters("rgb", "fill, watermark"),
			WithFilter("noop", func(ctx context.Context, img *vips.Image, load imagor.LoadFunc, args ...string) (err error) {
				return nil
			}),
		)
		assert.Equal(t, 2, v.Concurrency)
		assert.Equal(t, 167, v.MaxFilterOps)
		assert.Equal(t, 500, v.MaxCacheSize)
		assert.Equal(t, 501, v.MaxCacheMem)
		assert.Equal(t, 10, v.MaxCacheFiles)
		assert.Equal(t, 999, v.MaxWidth)
		assert.Equal(t, 998, v.MaxHeight)
		assert.Equal(t, 1666667, v.MaxResolution)
		assert.Equal(t, 3, v.MaxAnimationFrames)
		assert.Equal(t, true, v.MozJPEG)
		assert.Equal(t, true, v.StripMetadata)
		assert.Equal(t, 9, v.AvifSpeed)
		assert.Equal(t, []string{"rgb", "fill", "watermark"}, v.DisableFilters)

	})
	t.Run("edge options", func(t *testing.T) {
		v := NewProcessor(
			WithConcurrency(-1),
		)
		assert.Equal(t, runtime.NumCPU(), v.Concurrency)
	})
}

func TestImportParamsOptionString(t *testing.T) {
	p := vips.NewImportParams()
	p.FailOnError.Set(true)
	p.AutoRotate.Set(false)
	p.Density.Set(13)
	p.Page.Set(167)
	p.HeifThumbnail.Set(true)
	p.SvgUnlimited.Set(false)
	p.JpegShrinkFactor.Set(12)
	p.HeifThumbnail.Set(true)
	assert.Equal(t, "page=167,dpi=13,fail=TRUE,shrink=12,autorotate=FALSE,unlimited=FALSE,thumbnail=TRUE", p.OptionString())
}
