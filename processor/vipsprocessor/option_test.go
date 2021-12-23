package vipsprocessor

import (
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestWithOption(t *testing.T) {
	t.Run("options", func(t *testing.T) {
		app := New(
			WithConcurrency(2),
			WithMaxFilterOps(167),
			WithMaxCacheSize(500),
			WithMaxCacheMem(501),
			WithMaxCacheFiles(10),
			WithMaxWidth(999),
			WithMaxHeight(998),
			WithDebug(true),
			WithDisableFilters("rgb", "fill, watermark"),
		)
		assert.Equal(t, 2, app.Concurrency)
		assert.Equal(t, 167, app.MaxFilterOps)
		assert.Equal(t, 500, app.MaxCacheSize)
		assert.Equal(t, 501, app.MaxCacheMem)
		assert.Equal(t, 10, app.MaxCacheFiles)
		assert.Equal(t, 999, app.MaxWidth)
		assert.Equal(t, 998, app.MaxHeight)
		assert.Equal(t, []string{"rgb", "fill", "watermark"}, app.DisableFilters)

	})
	t.Run("edge options", func(t *testing.T) {
		app := New(
			WithConcurrency(-1),
		)
		assert.Equal(t, runtime.NumCPU(), app.Concurrency)
	})
}
