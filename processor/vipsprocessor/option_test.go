package vipsprocessor

import (
	"context"
	"github.com/cshum/imagor"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestWithOption(t *testing.T) {
	t.Run("options", func(t *testing.T) {
		v := New(
			WithConcurrency(2),
			WithMaxFilterOps(167),
			WithMaxCacheSize(500),
			WithMaxCacheMem(501),
			WithMaxCacheFiles(10),
			WithMaxWidth(999),
			WithMaxHeight(998),
			WithMaxResolution(1666667),
			WithMozJPEG(true),
			WithDebug(true),
			WithMaxAnimationFrames(3),
			WithDisableFilters("rgb", "fill, watermark"),
			WithFilter("noop", func(ctx context.Context, img *ImageRef, load imagor.LoadFunc, args ...string) (err error) {
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
		assert.Equal(t, []string{"rgb", "fill", "watermark"}, v.DisableFilters)

	})
	t.Run("edge options", func(t *testing.T) {
		v := New(
			WithConcurrency(-1),
		)
		assert.Equal(t, runtime.NumCPU(), v.Concurrency)
	})
}
