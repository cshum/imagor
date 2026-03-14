package vipsprocessor

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
	"github.com/stretchr/testify/assert"
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
			WithUnlimited(true),
			WithForceBmpFallback(),
			WithFilter("noop", func(ctx context.Context, img *vips.Image, load imagor.LoadFunc, args ...string) (err error) {
				return nil
			}),
			WithCacheSize(52428800),
			WithCacheMaxWidth(3000),
			WithCacheMaxHeight(2000),
			WithCacheTTL(time.Hour),
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
		assert.Equal(t, true, v.Unlimited)
		assert.Equal(t, 9, v.AvifSpeed)
		assert.Equal(t, []string{"rgb", "fill", "watermark"}, v.DisableFilters)
		assert.NotNil(t, v.FallbackFunc)
		assert.Equal(t, int64(52428800), v.CacheSize)
		assert.Equal(t, 3000, v.CacheMaxWidth)
		assert.Equal(t, 2000, v.CacheMaxHeight)
		assert.Equal(t, time.Hour, v.CacheTTL)
	})
	t.Run("edge options", func(t *testing.T) {
		v := NewProcessor(
			WithConcurrency(-1),
		)
		assert.Equal(t, runtime.NumCPU(), v.Concurrency)
	})
	t.Run("cache option defaults and guards", func(t *testing.T) {
		// Defaults
		v := NewProcessor()
		assert.Equal(t, int64(0), v.CacheSize)
		assert.Equal(t, 2400, v.CacheMaxWidth)
		assert.Equal(t, 1800, v.CacheMaxHeight)
		assert.Equal(t, time.Duration(0), v.CacheTTL)

		// Zero/negative values are ignored (guards)
		v2 := NewProcessor(
			WithCacheMaxWidth(0),
			WithCacheMaxHeight(0),
			WithCacheTTL(0),
		)
		assert.Equal(t, 2400, v2.CacheMaxWidth)  // unchanged default
		assert.Equal(t, 1800, v2.CacheMaxHeight) // unchanged default
		assert.Equal(t, time.Duration(0), v2.CacheTTL)
	})
}
