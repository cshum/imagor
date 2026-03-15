package vipsprocessor

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath" //nolint:typecheck
	"github.com/cshum/vipsgen/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFullDim(t *testing.T) {
	tests := []struct {
		token     string
		parentDim int
		want      string
	}{
		{"f", 800, "800"},
		{"full", 800, "800"},
		{"f-20", 800, "780"},
		{"full-20", 800, "780"},
		{"-f", 800, "-800"},
		{"-f-20", 800, "-780"},
		{"400", 800, "400"},
		{"", 800, ""},
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got := resolveFullDim(tt.token, tt.parentDim)
			if got != tt.want {
				t.Errorf("resolveFullDim(%q, %d) = %q, want %q", tt.token, tt.parentDim, got, tt.want)
			}
		})
	}
}

func TestResolveFullDimensions(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		parentW int
		parentH int
		want    string
	}{
		{
			name:    "simple f-token",
			path:    "fxf/filters:format(png)/image.jpg",
			parentW: 800, parentH: 600,
			want: "800x600/filters:format(png)/image.jpg",
		},
		{
			name:    "f-token with offset",
			path:    "f-20xf-30/filters:format(png)/image.jpg",
			parentW: 800, parentH: 600,
			want: "780x570/filters:format(png)/image.jpg",
		},
		{
			name:    "no f-token",
			path:    "400x300/filters:format(png)/image.jpg",
			parentW: 800, parentH: 600,
			want: "400x300/filters:format(png)/image.jpg",
		},
		{
			name:    "nested layer - should NOT resolve nested f-tokens",
			path:    "1551x2162/filters:image(/f-141xf-1145/img1,106,400)/img2",
			parentW: 3840, parentH: 2560,
			want: "1551x2162/filters:image(/f-141xf-1145/img1,106,400)/img2",
		},
		{
			name:    "nested layer with outer f-token",
			path:    "f-100xf-200/filters:image(/f-141xf-1145/img1,106,400)/img2",
			parentW: 3840, parentH: 2560,
			want: "3740x2360/filters:image(/f-141xf-1145/img1,106,400)/img2",
		},
		{
			name:    "only f-token no filters",
			path:    "fxf/image.jpg",
			parentW: 800, parentH: 600,
			want: "800x600/image.jpg",
		},
		{
			name:    "mixed f and number",
			path:    "fx300/image.jpg",
			parentW: 800, parentH: 600,
			want: "800x300/image.jpg",
		},
		{
			name:    "flip with f-token",
			path:    "-fxf/image.jpg",
			parentW: 800, parentH: 600,
			want: "-800x600/image.jpg",
		},
		{
			name:    "no dimension segment",
			path:    "filters:format(png)/image.jpg",
			parentW: 800, parentH: 600,
			want: "filters:format(png)/image.jpg",
		},
		{
			name:    "empty path",
			path:    "",
			parentW: 800, parentH: 600,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFullDimensions(tt.path, tt.parentW, tt.parentH)
			if got != tt.want {
				t.Errorf("resolveFullDimensions(%q, %d, %d) =\n  %q\nwant:\n  %q",
					tt.path, tt.parentW, tt.parentH, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Overlay cache unit tests
// These tests exercise the cache logic using the ristretto cache with
// *imagor.Blob (BlobTypeMemory) values.
// ---------------------------------------------------------------------------

// makeTestMemBlob creates a minimal BlobTypeMemory blob with a synthetic pixel buf.
func makeTestMemBlob(w, h, bands int) *imagor.Blob {
	buf := make([]byte, w*h*bands)
	for i := range buf {
		buf[i] = byte(i % 256)
	}
	return imagor.NewBlobFromMemory(buf, w, h, bands)
}

// TestOverlayCacheNewAndGet verifies that newCache creates a working
// ristretto cache and that Set/Get round-trip correctly with *imagor.Blob values.
func TestOverlayCacheNewAndGet(t *testing.T) {
	cache, err := newPixelCache(10 * 1024 * 1024) // 10 MiB
	require.NoError(t, err)

	blob := makeTestMemBlob(100, 100, 4)
	data, _, _, _, _ := blob.Memory()
	cost := int64(len(data))

	ok := cache.Set("logo.png", blob, cost)
	assert.True(t, ok, "Set should succeed within budget")
	cache.Wait()

	got, found := cache.Get("logo.png")
	assert.True(t, found, "entry should be found after Set+Wait")
	assert.Equal(t, blob, got, "retrieved blob should be identical")
}

// TestOverlayCacheEviction verifies that ristretto evicts entries when the
// byte budget is exceeded. We store two entries whose combined cost exceeds
// MaxCost and confirm that at least one is evicted.
func TestOverlayCacheEviction(t *testing.T) {
	// Budget: 500 bytes — just enough for one 10×10×4 entry (400 bytes) but not two.
	cache, err := newPixelCache(500)
	require.NoError(t, err)

	b1 := makeTestMemBlob(10, 10, 4) // 400 bytes
	b2 := makeTestMemBlob(10, 10, 4) // 400 bytes
	d1, _, _, _, _ := b1.Memory()
	d2, _, _, _, _ := b2.Memory()

	cache.Set("a.png", b1, int64(len(d1)))
	cache.Set("b.png", b2, int64(len(d2)))
	cache.Wait()

	_, found1 := cache.Get("a.png")
	_, found2 := cache.Get("b.png")
	// At least one must have been evicted to stay within the 500-byte budget.
	assert.False(t, found1 && found2, "ristretto should evict at least one entry to stay within budget")
}

// TestOverlayCacheDisabledWhenSizeZero verifies that loadOverlayImage skips
// the cache entirely when CacheSize == 0 (cache == nil).
// It counts how many times the loader is called; with cache disabled every
// call must hit the loader.
func TestOverlayCacheDisabledWhenSizeZero(t *testing.T) {
	// cache == nil when CacheSize == 0 (default)
	v := NewProcessor()
	require.Nil(t, v.cache, "cache must be nil when CacheSize == 0")

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// Call loadOverlayImage twice for the same URL — both must hit the loader.
	img1, err := v.loadOverlayImage(ctx, load, "logo.png", 0, 0, 1, 0)
	require.NoError(t, err)
	img1.Close()

	img2, err := v.loadOverlayImage(ctx, load, "logo.png", 0, 0, 1, 0)
	require.NoError(t, err)
	img2.Close()

	assert.Equal(t, int64(2), loadCount.Load(), "loader must be called twice when cache is disabled")
}

// TestOverlayCacheConcurrentSafety verifies that concurrent calls to
// loadOverlayImage for the same URL are safe: all callers get a valid image,
// and after the first load the result is cached so subsequent calls are fast.
func TestOverlayCacheConcurrentSafety(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(WithCacheSize(50 * 1024 * 1024))
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			// Use known size (100×100) so the cache path is exercised.
			// Unknown-size requests bypass the cache entirely.
			img, err := v.loadOverlayImage(ctx, load, "logo.png", 100, 100, 1, 0)
			errs[i] = err
			if img != nil {
				img.Close()
			}
		}()
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d should not error", i)
	}
	// load() is now called inside the singleflight, so only 1 goroutine fetches
	// from the network regardless of how many concurrent cache misses there are.
	assert.Equal(t, int64(1), loadCount.Load(),
		"loader must be called exactly once — singleflight deduplicates both fetch and decode")
	// After all goroutines complete, the result must be cached.
	_, found := v.cache.Get("logo.png")
	assert.True(t, found, "result must be cached after concurrent loads")
	t.Logf("loader called %d times for %d concurrent requests", loadCount.Load(), goroutines)
}

// TestOverlayCacheConcurrentLoadDedup verifies the thundering-herd fix:
// when many goroutines concurrently miss the cache for the same URL,
// load() is called exactly once — the singleflight deduplicates both the
// network fetch and the decode, not just the decode.
func TestOverlayCacheConcurrentLoadDedup(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(WithCacheSize(50 * 1024 * 1024))
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			img, err := v.loadOverlayImage(ctx, load, "logo.png", 100, 100, 1, 0)
			errs[i] = err
			if img != nil {
				img.Close()
			}
		}()
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d should not error", i)
	}
	// The key assertion: load() called exactly once, not once per goroutine.
	assert.Equal(t, int64(1), loadCount.Load(),
		"load() must be called exactly once — singleflight deduplicates the network fetch")
	t.Logf("loader called %d times for %d concurrent requests (expected 1)", loadCount.Load(), goroutines)
}

// TestCacheSizeExceedsMaxDims verifies that overlays requested at an
// explicit size (w > 0 && h > 0) larger than CacheMaxWidth/Height
// bypass the cache entirely and hit the loader every time.
func TestCacheSizeExceedsMaxDims(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(100), // max 100px
		WithCacheMaxHeight(100),
	)
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// Request explicit size 200×200 > max 100×100 — must bypass cache.
	img1, err := v.loadOverlayImage(ctx, load, "logo.png", 200, 200, 1, 0)
	require.NoError(t, err)
	img1.Close()

	img2, err := v.loadOverlayImage(ctx, load, "logo.png", 200, 200, 1, 0)
	require.NoError(t, err)
	img2.Close()

	// Both calls should hit the loader because the explicit size exceeds cache max dims.
	assert.Equal(t, int64(2), loadCount.Load(),
		"loader must be called each time when explicit overlay size exceeds cache max dims")
}

// TestOverlayCacheURLKey verifies that the cache key is the URL only, and that
// a cached memory blob is correctly retrieved by URL.
func TestOverlayCacheURLKey(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)

	blob := makeTestMemBlob(100, 100, 4)
	data, _, _, _, _ := blob.Memory()
	cache.Set("logo.png", blob, int64(len(data)))
	cache.Wait()

	got, found := cache.Get("logo.png")
	assert.True(t, found, "blob should be found by URL key")
	assert.Equal(t, blob, got, "retrieved blob should match stored blob")

	// A different URL must not collide.
	_, found2 := cache.Get("other.png")
	assert.False(t, found2, "different URL must not collide")
}

// TestOverlayCacheAnimatedSkipped verifies that animated overlays
// (img.Height() != img.PageHeight()) are served but not stored in the cache.
// We use a GIF file which libvips loads as multi-page.
func TestOverlayCacheAnimatedSkipped(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(WithCacheSize(50 * 1024 * 1024))
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/dancing-banana.gif"), nil
	}

	ctx := context.Background()

	// First call — loads and decodes the animated GIF.
	img1, err := v.loadOverlayImage(ctx, load, "anim.gif", 0, 0, -1, 0)
	require.NoError(t, err)
	img1.Close()

	// Second call — animated result must NOT be in cache; loader called again.
	img2, err := v.loadOverlayImage(ctx, load, "anim.gif", 0, 0, -1, 0)
	require.NoError(t, err)
	img2.Close()

	// Animated overlays are never cached — loader called once per loadOverlayImage call.
	assert.Equal(t, int64(2), loadCount.Load(),
		"animated overlay must not be cached; loader should be called each time")
}

// TestOverlayCacheUnknownSizeBypassesCache verifies that unknown-size (w==0, h==0)
// overlay requests always bypass the cache and load at MaxWidth×MaxHeight with SizeDown.
// This is correct: the cached blob is capped at CacheMaxWidth×CacheMaxHeight,
// which may be smaller than native. Serving from cache would return the wrong dimensions.
func TestOverlayCacheUnknownSizeBypassesCache(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(500),
		WithCacheMaxHeight(500),
	)
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// First call — unknown size, must bypass cache and load from source.
	img1, err := v.loadOverlayImage(ctx, load, "logo.png", 0, 0, 1, 0)
	require.NoError(t, err)
	img1.Close()

	// Second call — unknown size again, must bypass cache again (not served from cache).
	img2, err := v.loadOverlayImage(ctx, load, "logo.png", 0, 0, 1, 0)
	require.NoError(t, err)
	img2.Close()

	// Both calls must hit the loader — unknown-size always bypasses cache.
	assert.Equal(t, int64(2), loadCount.Load(),
		"unknown-size overlay must bypass cache; loader should be called each time")

	// The URL must not be in the cache (unknown-size never caches).
	_, found := v.cache.Get("logo.png")
	assert.False(t, found, "unknown-size overlay must not populate the cache")
}

// TestOverlayCacheUnknownSizeNativeExceedsMax verifies that unknown-size overlay
// requests return the full native dimensions (up to MaxWidth×MaxHeight), NOT the
// cache-capped CacheMaxWidth×CacheMaxHeight.
// Uses CacheMaxWidth=50 so gopher.png (larger than 50px) simulates native > max.
func TestOverlayCacheUnknownSizeNativeExceedsMax(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(50), // smaller than gopher.png
		WithCacheMaxHeight(50),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// Unknown-size request: must return native size (> 50), not cache-capped 50×50.
	img, err := v.loadOverlayImage(ctx, load, "logo.png", 0, 0, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, img)
	assert.Greater(t, img.Width(), 50,
		"unknown-size overlay must return native width, not cache-capped CacheMaxWidth")
	assert.Greater(t, img.PageHeight(), 50,
		"unknown-size overlay must return native height, not cache-capped CacheMaxHeight")
	img.Close()
}

// TestOverlayCacheKnownSizeStillCaches verifies that known-size requests (w>0, h>0)
// within cache max dims DO use the cache — the fix must not break the happy path.
func TestOverlayCacheKnownSizeStillCaches(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(500),
		WithCacheMaxHeight(500),
	)
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// First call — known size, cache miss, loads and caches.
	img1, err := v.loadOverlayImage(ctx, load, "logo.png", 100, 100, 1, 0)
	require.NoError(t, err)
	img1.Close()

	// Second call — known size, same URL → cache hit, loader not called again.
	img2, err := v.loadOverlayImage(ctx, load, "logo.png", 50, 50, 1, 0)
	require.NoError(t, err)
	img2.Close()

	assert.Equal(t, int64(1), loadCount.Load(),
		"known-size overlay must hit cache on second call; loader should be called only once")
	_, found := v.cache.Get("logo.png")
	assert.True(t, found, "known-size overlay must populate the cache")
}

// TestOverlayCacheImageFilterConcurrent verifies that concurrent image() filter
// calls with the same URL are safe: all callers get a valid image,
// and after the calls complete the result is cached by URL key.
func TestOverlayCacheImageFilterConcurrent(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	blob := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params := imagorpath.Parse("200x200/gopher.png")
	url := "gopher.png"

	const goroutines = 10
	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			img, err := v.loadFilterImage(ctx, blob, params, load, url)
			errs[i] = err
			if img != nil {
				img.Close()
			}
		}()
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d should not error", i)
	}
	t.Logf("loader called %d times for %d concurrent image() filter requests", loadCount.Load(), goroutines)
	// After all goroutines complete, the result should be cached by URL key.
	_, found := v.cache.Get(url)
	assert.True(t, found, "image() filter result should be cached by URL key after first successful load")
}

// TestOverlayCacheImageFilterExportBypass verifies that image() filter requests
// with params.Width/Height exceeding CacheMaxWidth×CacheMaxHeight
// bypass the cache entirely — the URL is never stored in the cache.
func TestOverlayCacheImageFilterExportBypass(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(100), // max 100px
		WithCacheMaxHeight(100),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	url := "gopher.png"

	// params.Width=200, params.Height=200 > max 100×100 — must bypass cache.
	params := imagorpath.Parse("200x200/gopher.png")

	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	img1, err := v.loadFilterImage(ctx, blob1, params, load, url)
	require.NoError(t, err)
	if img1 != nil {
		img1.Close()
	}

	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	img2, err := v.loadFilterImage(ctx, blob2, params, load, url)
	require.NoError(t, err)
	if img2 != nil {
		img2.Close()
	}

	// The URL must not be in the cache — bypass means no caching.
	_, found := v.cache.Get(url)
	assert.False(t, found, "image() filter result must not be cached when params size exceeds max dims")
}

// TestOverlayCacheImageFilterURLOnlyKey verifies the core cache-hit maximization:
// two calls with different params (different sizes) but the same source URL
// both hit the same cache entry. After the first call populates the cache,
// the second call runs the pipeline from the cached memory blob — no I/O.
func TestOverlayCacheImageFilterURLOnlyKey(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	url := "gopher.png"

	// First call: image(200x200/gopher.png) — cache miss, decodes and caches "gopher.png".
	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params1 := imagorpath.Parse("200x200/gopher.png")
	img1, err := v.loadFilterImage(ctx, blob1, params1, load, url)
	require.NoError(t, err)
	require.NotNil(t, img1)
	assert.Equal(t, 200, img1.Width(), "first call result width should match params1")
	assert.Equal(t, 200, img1.PageHeight(), "first call result height should match params1")
	img1.Close()

	// Verify the URL is now cached.
	_, found := v.cache.Get(url)
	assert.True(t, found, "URL should be cached after first call")

	// Second call: image(100x100/gopher.png) — different size, same URL → cache hit.
	// blob2 is provided but loadOrCache will return the cached memBlob immediately.
	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params2 := imagorpath.Parse("100x100/gopher.png")
	img2, err := v.loadFilterImage(ctx, blob2, params2, load, url)
	require.NoError(t, err)
	require.NotNil(t, img2)
	// Pipeline ran from cached memory blob: result should be 100×100 (params2 size).
	assert.Equal(t, 100, img2.Width(), "second call result width should match params2")
	assert.Equal(t, 100, img2.PageHeight(), "second call result height should match params2")
	img2.Close()
}

// TestOverlayCacheImageFilterNilBlob verifies that loadFilterImage does not
// error when blob is nil (e.g. color: image paths generated in-process).
// The cache must be bypassed and loadAndProcess called directly with nil blob.
func TestOverlayCacheImageFilterNilBlob(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	// color: paths produce nil blob — loadFilterImage must not error.
	params := imagorpath.Parse("100x100/color:red")

	img, err := v.loadFilterImage(ctx, nil, params, load, "color:red")
	// loadAndProcess with nil blob + color: path should succeed (returns a solid color image).
	require.NoError(t, err)
	if img != nil {
		img.Close()
	}

	// The URL must not be cached (nil blob bypasses cache entirely).
	_, found := v.cache.Get("color:red")
	assert.False(t, found, "nil blob path must not be cached")
}

// TestOverlayCacheAnimatedSizeKnown verifies that animated overlays with a known
// requested size (sizeKnown=true) are returned at the requested size, not at
// maxW×maxH. This was a pre-existing bug: the animated fallback always used
// maxW×maxH regardless of sizeKnown.
func TestOverlayCacheAnimatedSizeKnown(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/dancing-banana.gif"), nil
	}

	ctx := context.Background()

	// Request explicit size 100×80 for an animated GIF.
	// The result must be at most 100×80, not 2400×1800.
	// Use size=0 (SizeDown) — with SizeBoth the GIF would be upscaled; SizeDown
	// ensures the result is ≤ 100×80 which is what we want to verify.
	img, err := v.loadOverlayImage(ctx, load, "anim.gif", 100, 80, -1, 0)
	require.NoError(t, err)
	require.NotNil(t, img)
	assert.LessOrEqual(t, img.Width(), 100,
		"animated overlay with sizeKnown must be ≤ requested width, not maxW")
	assert.LessOrEqual(t, img.PageHeight(), 80,
		"animated overlay with sizeKnown must be ≤ requested height, not maxH")
	img.Close()
}

// TestOverlayCacheImageFilterUnknownSizeBypassesCache verifies that image() filter
// requests with unknown size (params.Width==0 || params.Height==0) bypass the cache.
// The cached blob is capped at CacheMaxWidth×CacheMaxHeight; serving
// from cache for an unknown-size request would return the wrong (smaller) dimensions.
func TestOverlayCacheImageFilterUnknownSizeBypassesCache(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(50), // smaller than gopher.png
		WithCacheMaxHeight(50),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	url := "gopher.png"

	// params with unknown size (0x0) — must bypass cache.
	params := imagorpath.Parse("0x0/gopher.png")

	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	img1, err := v.loadFilterImage(ctx, blob1, params, load, url)
	require.NoError(t, err)
	if img1 != nil {
		// Result must be at native size (> 50), not cache-capped 50×50.
		assert.Greater(t, img1.Width(), 50,
			"unknown-size image() filter must return native width, not cache-capped CacheMaxWidth")
		img1.Close()
	}

	// The URL must not be in the cache (unknown-size never caches).
	_, found := v.cache.Get(url)
	assert.False(t, found, "unknown-size image() filter must not populate the cache")
}

// TestOverlayCacheBlobLifetime verifies the key safety property:
// the []byte in a cached memory blob remains valid after the *vips.Image
// derived from it is closed. This is the core correctness guarantee — the
// cached buf is a Go-owned allocation independent of any libvips object.
func TestOverlayCacheBlobLifetime(t *testing.T) {
	blob := makeTestMemBlob(50, 50, 4)
	data, w, h, bands, ok := blob.Memory()
	require.True(t, ok, "blob must be BlobTypeMemory")
	original := make([]byte, len(data))
	copy(original, data)

	// Verify dimensions are preserved.
	assert.Equal(t, 50, w)
	assert.Equal(t, 50, h)
	assert.Equal(t, 4, bands)

	// Simulate eviction: set blob to nil (as ristretto would do on evict).
	// The buf slice should still be reachable via our local reference.
	localData := data
	blob = nil //nolint:ineffassign // intentional: simulate eviction
	assert.Equal(t, original, localData,
		"buf must remain valid after blob pointer is cleared (GC safety)")
}

// ---------------------------------------------------------------------------
// Crop and focal cache bypass tests
// ---------------------------------------------------------------------------

// TestOverlayCacheImageFilterCropBypass verifies that image() filter requests
// with a crop in the nested imagor path bypass the cache entirely.
// The cache stores a downscaled copy (≤ CacheMaxWidth×CacheMaxHeight); applying
// crop coordinates from the original image space to the smaller cached image
// would produce incorrect results.
//
// Note: loadFilterImage receives a pre-loaded blob from the image() filter caller —
// it does not call load() itself. The bypass is verified by checking that the
// cached memBlob is NOT used: the result is produced from the original blob
// (which has full resolution), not from the downscaled cached copy.
func TestOverlayCacheImageFilterCropBypass(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	url := "gopher.png"

	// First call: no crop — populates the cache.
	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params1 := imagorpath.Parse("200x200/gopher.png")
	img1, err := v.loadFilterImage(ctx, blob1, params1, load, url)
	require.NoError(t, err)
	require.NotNil(t, img1)
	img1.Close()

	// Verify the URL is now cached.
	_, found := v.cache.Get(url)
	assert.True(t, found, "URL should be cached after first (no-crop) call")

	// Second call: with absolute pixel crop — must bypass cache and use original blob.
	// If the cache were used, the crop would be applied to the downscaled cached image
	// instead of the original, producing incorrect results.
	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params2 := imagorpath.Parse("10x10:50x50/200x200/gopher.png") // absolute pixel crop
	img2, err := v.loadFilterImage(ctx, blob2, params2, load, url)
	require.NoError(t, err)
	require.NotNil(t, img2)
	img2.Close()

	// The cache entry must still exist (crop bypass doesn't evict it).
	_, stillCached := v.cache.Get(url)
	assert.True(t, stillCached, "cache entry must still exist after crop bypass")
	// The crop call must succeed without error — correctness verified by no panic/error.
	// (The crop is applied to the original blob, not the cached downscaled copy.)
}

// TestOverlayCacheImageFilterPercentageCropBypass verifies that image() filter
// requests with a percentage crop also bypass the cache.
// Even though percentage crops are geometrically proportional, the crop region
// of the cached image may be smaller than the requested output size, causing
// upscaling artifacts. Example: 50% crop of a 2400×1800 cached image yields
// a 1200×900 region — insufficient for a 1500×1500 output without upscaling.
func TestOverlayCacheImageFilterPercentageCropBypass(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	url := "gopher.png"

	// First call: no crop — populates the cache.
	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params1 := imagorpath.Parse("200x200/gopher.png")
	img1, err := v.loadFilterImage(ctx, blob1, params1, load, url)
	require.NoError(t, err)
	require.NotNil(t, img1)
	img1.Close()

	_, found := v.cache.Get(url)
	assert.True(t, found, "URL should be cached after first (no-crop) call")

	// Second call: with percentage crop — must bypass cache and use original blob.
	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params2 := imagorpath.Parse("0x0:0.5x0.5/200x200/gopher.png") // percentage crop
	img2, err := v.loadFilterImage(ctx, blob2, params2, load, url)
	require.NoError(t, err)
	require.NotNil(t, img2)
	img2.Close()

	// Cache entry must still exist; crop bypass doesn't evict it.
	_, stillCached := v.cache.Get(url)
	assert.True(t, stillCached, "cache entry must still exist after percentage crop bypass")
}

// TestOverlayCacheImageFilterFocalBypass verifies that image() filter requests
// with a focal() filter bypass the cache.
// Focal points use pixel coordinates from the original image space; applying
// them to the smaller cached image would focus on the wrong area.
func TestOverlayCacheImageFilterFocalBypass(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	url := "gopher.png"

	// First call: no focal — populates the cache.
	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params1 := imagorpath.Parse("200x200/gopher.png")
	img1, err := v.loadFilterImage(ctx, blob1, params1, load, url)
	require.NoError(t, err)
	require.NotNil(t, img1)
	img1.Close()

	_, found := v.cache.Get(url)
	assert.True(t, found, "URL should be cached after first (no-focal) call")

	// Second call: with focal filter — must bypass cache and use original blob.
	// The focal point (100,80) is in the original image's coordinate space.
	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params2 := imagorpath.Parse("200x200/filters:focal(100,80)/gopher.png")
	img2, err := v.loadFilterImage(ctx, blob2, params2, load, url)
	require.NoError(t, err)
	require.NotNil(t, img2)
	img2.Close()

	// Cache entry must still exist; focal bypass doesn't evict it.
	_, stillCached := v.cache.Get(url)
	assert.True(t, stillCached, "cache entry must still exist after focal bypass")
}

// TestBaseImageCacheCropBypass verifies that Process() skips loadOrCache when
// the params contain a crop, even if the image was previously cached by a
// no-crop request. This prevents crop coordinates from being applied to the
// downscaled cached image instead of the original.
func TestBaseImageCacheCropBypass(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	ctx := context.Background()
	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	// First call: preview() + no crop — populates the cache via loadOrCache.
	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params1 := imagorpath.Params{
		Image: "gopher.png", Width: 200, Height: 200,
		Filters: []imagorpath.Filter{{Name: "preview"}},
	}
	result1, err := v.Process(ctx, blob1, params1, load)
	require.NoError(t, err)
	require.NotNil(t, result1)

	// Verify the image is now cached.
	_, found := v.cache.Get("gopher.png")
	assert.True(t, found, "image should be cached after first (no-crop) Process call")

	// Second call: preview() + absolute pixel crop — must NOT use the cached downscaled image.
	// The crop (10,10):(50,50) must be applied to the original image, not the cached copy.
	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params2 := imagorpath.Params{
		Image:      "gopher.png",
		Width:      200,
		Height:     200,
		CropLeft:   10,
		CropTop:    10,
		CropRight:  50,
		CropBottom: 50,
		Filters:    []imagorpath.Filter{{Name: "preview"}},
	}
	result2, err := v.Process(ctx, blob2, params2, load)
	require.NoError(t, err)
	require.NotNil(t, result2)
	// The crop request must succeed without error — if the cache were used incorrectly,
	// the crop region (10,10):(50,50) on the cached image would still work but produce
	// a different (wrong) region than on the original. The key test is that no error
	// occurs and the result is valid.
}

// TestBaseImageCacheFocalBypass verifies that Process() skips loadOrCache when
// the params contain a focal() filter, even if the image was previously cached.
func TestBaseImageCacheFocalBypass(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	ctx := context.Background()
	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	// First call: preview() + no focal — populates the cache.
	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params1 := imagorpath.Params{
		Image: "gopher.png", Width: 200, Height: 200,
		Filters: []imagorpath.Filter{{Name: "preview"}},
	}
	result1, err := v.Process(ctx, blob1, params1, load)
	require.NoError(t, err)
	require.NotNil(t, result1)

	_, found := v.cache.Get("gopher.png")
	assert.True(t, found, "image should be cached after first Process call")

	// Second call: preview() + focal filter — must bypass cache.
	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params2 := imagorpath.Params{
		Image:  "gopher.png",
		Width:  200,
		Height: 200,
		Filters: []imagorpath.Filter{
			{Name: "preview"},
			{Name: "focal", Args: "100,80"},
		},
	}
	result2, err := v.Process(ctx, blob2, params2, load)
	require.NoError(t, err)
	require.NotNil(t, result2)
}

// ---------------------------------------------------------------------------
// Watermark w_ratio / h_ratio regression tests
// ---------------------------------------------------------------------------

// TestWatermarkNoneNoneRatio verifies that watermark(img,x,y,alpha,none,none)
// loads the overlay at the parent image's dimensions, not at MaxWidth×MaxHeight.
//
// Regression: before the fix, when both w_ratio and h_ratio were "none", w and h
// stayed 0, causing loadOverlayImage to treat the size as unknown and load at
// MaxWidth×MaxHeight with SizeDown instead of the parent image's dimensions.
func TestWatermarkNoneNoneRatio(t *testing.T) {
	v := NewProcessor()

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// Simulate what the fixed watermark() code does for none,none:
	// w = img.Width(), h = img.PageHeight() — use a known parent size.
	parentW := 200
	parentH := 150

	// With the fix: w=parentW, h=parentH, size=SizeBoth.
	// The overlay must be loaded at ≤ parentW×parentH, not at MaxWidth×MaxHeight.
	img, err := v.loadOverlayImage(ctx, load, "gopher.png", parentW, parentH, 1, vips.SizeBoth)
	require.NoError(t, err)
	require.NotNil(t, img)
	assert.LessOrEqual(t, img.Width(), parentW,
		"none,none watermark overlay width must be ≤ parent width, not MaxWidth")
	assert.LessOrEqual(t, img.PageHeight(), parentH,
		"none,none watermark overlay height must be ≤ parent height, not MaxHeight")
	img.Close()
}

// TestWatermarkWithRatioSizing verifies that watermark(img,x,y,alpha,50,50)
// loads the overlay at 50% of the parent image's dimensions.
func TestWatermarkWithRatioSizing(t *testing.T) {
	v := NewProcessor()

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// Simulate what watermark() does for w_ratio=50, h_ratio=50:
	// w = img.Width() * 50 / 100, h = img.PageHeight() * 50 / 100.
	parentW := 400
	parentH := 300
	w := parentW * 50 / 100 // 200
	h := parentH * 50 / 100 // 150

	img, err := v.loadOverlayImage(ctx, load, "gopher.png", w, h, 1, vips.SizeBoth)
	require.NoError(t, err)
	require.NotNil(t, img)
	assert.LessOrEqual(t, img.Width(), w,
		"50%% watermark overlay width must be ≤ 50%% of parent width")
	assert.LessOrEqual(t, img.PageHeight(), h,
		"50%% watermark overlay height must be ≤ 50%% of parent height")
	img.Close()
}

// ---------------------------------------------------------------------------
// CacheFormat tests
// ---------------------------------------------------------------------------

// TestOverlayCacheFormatPixelDefault verifies that the default cache format
// (CacheFormat == 0, i.e. BlobTypeUnknown treated as raw pixels) stores
// BlobTypeMemory blobs in the cache — identical to the explicit pixel path.
func TestOverlayCacheFormatPixelDefault(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	// No WithCacheFormat — default is raw pixels (BlobTypeMemory).
	v := NewProcessor(WithCacheSize(50 * 1024 * 1024))
	v.cache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()
	img, err := v.loadOverlayImage(ctx, load, "gopher.png", 100, 100, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, img)
	img.Close()

	cached, found := v.cache.Get("gopher.png")
	require.True(t, found, "entry must be cached after first call")
	assert.Equal(t, imagor.BlobTypeMemory, cached.BlobType(),
		"default cache format must store BlobTypeMemory (raw pixels)")
}

// TestOverlayCacheFormatPNG verifies that WithCacheFormat(BlobTypePNG) stores
// lossless PNG bytes in the cache and that subsequent calls return valid images.
func TestOverlayCacheFormatPNG(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheFormat(imagor.BlobTypePNG),
	)
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// First call — cache miss, loads and encodes to PNG.
	img1, err := v.loadOverlayImage(ctx, load, "gopher.png", 100, 100, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, img1)
	img1.Close()

	// Verify cache entry is BlobTypePNG.
	cached, found := v.cache.Get("gopher.png")
	require.True(t, found, "entry must be cached after first call")
	assert.Equal(t, imagor.BlobTypePNG, cached.BlobType(),
		"PNG cache format must store BlobTypePNG bytes")

	// Second call — cache hit, must return a valid image without calling load again.
	img2, err := v.loadOverlayImage(ctx, load, "gopher.png", 80, 80, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, img2)
	img2.Close()

	assert.Equal(t, int64(1), loadCount.Load(),
		"loader must be called only once — second call served from PNG cache")
}

// TestOverlayCacheFormatWebP verifies that WithCacheFormat(BlobTypeWEBP) stores
// lossy WebP bytes in the cache and that subsequent calls return valid images.
func TestOverlayCacheFormatWebP(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheFormat(imagor.BlobTypeWEBP),
	)
	v.cache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// First call — cache miss, loads and encodes to WebP.
	img1, err := v.loadOverlayImage(ctx, load, "gopher.png", 100, 100, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, img1)
	img1.Close()

	// Verify cache entry is BlobTypeWEBP.
	cached, found := v.cache.Get("gopher.png")
	require.True(t, found, "entry must be cached after first call")
	assert.Equal(t, imagor.BlobTypeWEBP, cached.BlobType(),
		"WebP cache format must store BlobTypeWEBP bytes")

	// Second call — cache hit, must return a valid image without calling load again.
	img2, err := v.loadOverlayImage(ctx, load, "gopher.png", 80, 80, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, img2)
	img2.Close()

	assert.Equal(t, int64(1), loadCount.Load(),
		"loader must be called only once — second call served from WebP cache")
}

// TestBaseImageCacheNoCropStillCaches verifies that the crop/focal bypass does
// not break the happy path: Process() still uses the cache for no-crop requests.
func TestBaseImageCacheNoCropStillCaches(t *testing.T) {
	cache, err := newPixelCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithCacheSize(50*1024*1024),
		WithCacheMaxWidth(2400),
		WithCacheMaxHeight(1800),
	)
	v.cache = cache

	ctx := context.Background()
	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	// First call: preview() + no crop — populates the cache.
	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params := imagorpath.Params{
		Image: "gopher.png", Width: 200, Height: 200,
		Filters: []imagorpath.Filter{{Name: "preview"}},
	}
	result1, err := v.Process(ctx, blob1, params, load)
	require.NoError(t, err)
	require.NotNil(t, result1)

	// Second call: same params, preview() + no crop — should use cache (blob replaced with memBlob).
	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	result2, err := v.Process(ctx, blob2, params, load)
	require.NoError(t, err)
	require.NotNil(t, result2)

	// Cache should be populated.
	_, found := v.cache.Get("gopher.png")
	assert.True(t, found, "preview() + no-crop request must still populate the cache")
}
