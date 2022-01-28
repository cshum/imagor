package vipsprocessor

import (
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestWithOption(t *testing.T) {
	t.Run("options", func(t *testing.T) {
		vips := New(
			WithConcurrency(2),
			WithMaxFilterOps(167),
			WithMaxCacheSize(500),
			WithMaxCacheMem(501),
			WithMaxCacheFiles(10),
			WithMaxWidth(999),
			WithMaxHeight(998),
			WithDebug(true),
			WithMaxAnimationFrames(3),
			WithDisableFilters("rgb", "fill, watermark"),
		)
		assert.Equal(t, 2, vips.Concurrency)
		assert.Equal(t, 167, vips.MaxFilterOps)
		assert.Equal(t, 500, vips.MaxCacheSize)
		assert.Equal(t, 501, vips.MaxCacheMem)
		assert.Equal(t, 10, vips.MaxCacheFiles)
		assert.Equal(t, 999, vips.MaxWidth)
		assert.Equal(t, 998, vips.MaxHeight)
		assert.Equal(t, 3, vips.MaxAnimationFrames)
		assert.Equal(t, []string{"rgb", "fill", "watermark"}, vips.DisableFilters)

	})
	t.Run("edge options", func(t *testing.T) {
		vips := New(
			WithConcurrency(-1),
		)
		assert.Equal(t, runtime.NumCPU(), vips.Concurrency)
	})
}
