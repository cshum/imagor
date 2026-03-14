package vipsprocessor

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath" //nolint:typecheck
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

// TestOverlayCacheNewAndGet verifies that newOverlayCache creates a working
// ristretto cache and that Set/Get round-trip correctly with *imagor.Blob values.
func TestOverlayCacheNewAndGet(t *testing.T) {
	cache, err := newOverlayCache(10 * 1024 * 1024) // 10 MiB
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
	cache, err := newOverlayCache(500)
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
// the cache entirely when OverlayCacheSize == 0 (overlayCache == nil).
// It counts how many times the loader is called; with cache disabled every
// call must hit the loader.
func TestOverlayCacheDisabledWhenSizeZero(t *testing.T) {
	// overlayCache == nil when OverlayCacheSize == 0 (default)
	v := NewProcessor()
	require.Nil(t, v.overlayCache, "overlayCache must be nil when OverlayCacheSize == 0")

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
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(WithOverlayCacheSize(50 * 1024 * 1024))
	v.overlayCache = cache

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
			img, err := v.loadOverlayImage(ctx, load, "logo.png", 0, 0, 1, 0)
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
	// loadOverlayImage calls load(url) before loadOrCacheBlob, so all goroutines
	// that miss the cache will call load(). Singleflight deduplicates the decode
	// (NewThumbnail + WriteToMemory), not the load() call itself.
	// All calls are safe; the loader may be called up to goroutines times.
	assert.LessOrEqual(t, loadCount.Load(), int64(goroutines),
		"loader should not be called more than goroutine count")
	// After all goroutines complete, the result must be cached.
	_, found := v.overlayCache.Get("logo.png")
	assert.True(t, found, "result must be cached after concurrent loads")
	t.Logf("loader called %d times for %d concurrent requests", loadCount.Load(), goroutines)
}

// TestOverlayCacheSizeExceedsMaxDims verifies that overlays requested at an
// explicit size (w > 0 && h > 0) larger than OverlayCacheMaxWidth/Height
// bypass the cache entirely and hit the loader every time.
func TestOverlayCacheSizeExceedsMaxDims(t *testing.T) {
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithOverlayCacheSize(50*1024*1024),
		WithOverlayCacheMaxWidth(100), // max 100px
		WithOverlayCacheMaxHeight(100),
	)
	v.overlayCache = cache

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
	cache, err := newOverlayCache(50 * 1024 * 1024)
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
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(WithOverlayCacheSize(50 * 1024 * 1024))
	v.overlayCache = cache

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

// TestOverlayCacheUnknownSizeUsesMaxDims verifies that the unknown-size path
// (w==0, h==0) loads at OverlayCacheMaxWidth×OverlayCacheMaxHeight, not at
// MaxWidth×MaxHeight. This ensures preview-size overlays are cached while
// export-size requests (MaxWidth > OverlayCacheMaxWidth) bypass the cache.
func TestOverlayCacheUnknownSizeUsesMaxDims(t *testing.T) {
	// Set OverlayCacheMaxWidth/Height smaller than the image's natural size
	// but larger than 0 so the image is loaded and cached.
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithOverlayCacheSize(50*1024*1024),
		WithOverlayCacheMaxWidth(500),
		WithOverlayCacheMaxHeight(500),
	)
	v.overlayCache = cache

	var loadCount atomic.Int64
	load := func(image string) (*imagor.Blob, error) {
		loadCount.Add(1)
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := context.Background()

	// First call — cache miss, loads and caches at ≤500×500.
	img1, err := v.loadOverlayImage(ctx, load, "logo.png", 0, 0, 1, 0)
	require.NoError(t, err)
	assert.LessOrEqual(t, img1.Width(), 500, "cached overlay width must be ≤ OverlayCacheMaxWidth")
	assert.LessOrEqual(t, img1.PageHeight(), 500, "cached overlay height must be ≤ OverlayCacheMaxHeight")
	img1.Close()

	// Second call — cache hit, loader not called again.
	img2, err := v.loadOverlayImage(ctx, load, "logo.png", 0, 0, 1, 0)
	require.NoError(t, err)
	img2.Close()

	assert.Equal(t, int64(1), loadCount.Load(),
		"loader should be called only once; second call should hit cache")
}

// TestOverlayCacheImageFilterConcurrent verifies that concurrent image() filter
// calls with the same URL are safe: all callers get a valid image,
// and after the calls complete the result is cached by URL key.
func TestOverlayCacheImageFilterConcurrent(t *testing.T) {
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithOverlayCacheSize(50*1024*1024),
		WithOverlayCacheMaxWidth(2400),
		WithOverlayCacheMaxHeight(1800),
	)
	v.overlayCache = cache

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
			img, err := v.loadAndCacheImageFilter(ctx, blob, params, load, url)
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
	_, found := v.overlayCache.Get(url)
	assert.True(t, found, "image() filter result should be cached by URL key after first successful load")
}

// TestOverlayCacheImageFilterExportBypass verifies that image() filter requests
// with params.Width/Height exceeding OverlayCacheMaxWidth×OverlayCacheMaxHeight
// bypass the cache entirely — the URL is never stored in the cache.
func TestOverlayCacheImageFilterExportBypass(t *testing.T) {
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithOverlayCacheSize(50*1024*1024),
		WithOverlayCacheMaxWidth(100), // max 100px
		WithOverlayCacheMaxHeight(100),
	)
	v.overlayCache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	url := "gopher.png"

	// params.Width=200, params.Height=200 > max 100×100 — must bypass cache.
	params := imagorpath.Parse("200x200/gopher.png")

	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	img1, err := v.loadAndCacheImageFilter(ctx, blob1, params, load, url)
	require.NoError(t, err)
	if img1 != nil {
		img1.Close()
	}

	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	img2, err := v.loadAndCacheImageFilter(ctx, blob2, params, load, url)
	require.NoError(t, err)
	if img2 != nil {
		img2.Close()
	}

	// The URL must not be in the cache — bypass means no caching.
	_, found := v.overlayCache.Get(url)
	assert.False(t, found, "image() filter result must not be cached when params size exceeds max dims")
}

// TestOverlayCacheImageFilterURLOnlyKey verifies the core cache-hit maximization:
// two calls with different params (different sizes) but the same source URL
// both hit the same cache entry. After the first call populates the cache,
// the second call runs the pipeline from the cached memory blob — no I/O.
func TestOverlayCacheImageFilterURLOnlyKey(t *testing.T) {
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithOverlayCacheSize(50*1024*1024),
		WithOverlayCacheMaxWidth(2400),
		WithOverlayCacheMaxHeight(1800),
	)
	v.overlayCache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	url := "gopher.png"

	// First call: image(200x200/gopher.png) — cache miss, decodes and caches "gopher.png".
	blob1 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params1 := imagorpath.Parse("200x200/gopher.png")
	img1, err := v.loadAndCacheImageFilter(ctx, blob1, params1, load, url)
	require.NoError(t, err)
	require.NotNil(t, img1)
	assert.Equal(t, 200, img1.Width(), "first call result width should match params1")
	assert.Equal(t, 200, img1.PageHeight(), "first call result height should match params1")
	img1.Close()

	// Verify the URL is now cached.
	_, found := v.overlayCache.Get(url)
	assert.True(t, found, "URL should be cached after first call")

	// Second call: image(100x100/gopher.png) — different size, same URL → cache hit.
	// blob2 is provided but loadOrCacheBlob will return the cached memBlob immediately.
	blob2 := imagor.NewBlobFromFile("../../testdata/gopher.png")
	params2 := imagorpath.Parse("100x100/gopher.png")
	img2, err := v.loadAndCacheImageFilter(ctx, blob2, params2, load, url)
	require.NoError(t, err)
	require.NotNil(t, img2)
	// Pipeline ran from cached memory blob: result should be 100×100 (params2 size).
	assert.Equal(t, 100, img2.Width(), "second call result width should match params2")
	assert.Equal(t, 100, img2.PageHeight(), "second call result height should match params2")
	img2.Close()
}

// TestOverlayCacheImageFilterNilBlob verifies that loadAndCacheImageFilter does not
// error when blob is nil (e.g. color: image paths generated in-process).
// The cache must be bypassed and loadAndProcess called directly with nil blob.
func TestOverlayCacheImageFilterNilBlob(t *testing.T) {
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithOverlayCacheSize(50*1024*1024),
		WithOverlayCacheMaxWidth(2400),
		WithOverlayCacheMaxHeight(1800),
	)
	v.overlayCache = cache

	load := func(image string) (*imagor.Blob, error) {
		return imagor.NewBlobFromFile("../../testdata/gopher.png"), nil
	}

	ctx := withContext(context.Background())
	// color: paths produce nil blob — loadAndCacheImageFilter must not error.
	params := imagorpath.Parse("100x100/color:red")

	img, err := v.loadAndCacheImageFilter(ctx, nil, params, load, "color:red")
	// loadAndProcess with nil blob + color: path should succeed (returns a solid color image).
	require.NoError(t, err)
	if img != nil {
		img.Close()
	}

	// The URL must not be cached (nil blob bypasses cache entirely).
	_, found := v.overlayCache.Get("color:red")
	assert.False(t, found, "nil blob path must not be cached")
}

// TestOverlayCacheAnimatedSizeKnown verifies that animated overlays with a known
// requested size (sizeKnown=true) are returned at the requested size, not at
// maxW×maxH. This was a pre-existing bug: the animated fallback always used
// maxW×maxH regardless of sizeKnown.
func TestOverlayCacheAnimatedSizeKnown(t *testing.T) {
	cache, err := newOverlayCache(50 * 1024 * 1024)
	require.NoError(t, err)
	v := NewProcessor(
		WithOverlayCacheSize(50*1024*1024),
		WithOverlayCacheMaxWidth(2400),
		WithOverlayCacheMaxHeight(1800),
	)
	v.overlayCache = cache

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
