package vipsprocessor

import (
	"context"

	"github.com/cshum/imagor"
	"github.com/cshum/vipsgen/vips"
	"github.com/dgraph-io/ristretto/v2"
)

// imageCache is a ristretto cache storing blobs keyed by image path.
// Values are Go-owned buffers — no libvips lifecycle, no request context
// dependency, safe for concurrent reads and GC cleanup.
type imageCache = ristretto.Cache[string, *imagor.Blob]

// newImageCache creates a new ristretto pixel cache with the given byte budget.
func newImageCache(maxCost int64) (*imageCache, error) {
	return ristretto.NewCache[string, *imagor.Blob](&ristretto.Config[string, *imagor.Blob]{
		NumCounters: 10000,
		MaxCost:     maxCost,
		BufferItems: 64,
	})
}

// loadOrCacheResult is the singleflight result for loadOrCache.
// memBlob is non-nil for static images (cached pixels or compressed bytes).
// origBlob is non-nil for animated sources that cannot be cached.
type loadOrCacheResult struct {
	memBlob  *imagor.Blob
	origBlob *imagor.Blob
}

// LoadFromCache implements imagor.Cacher. Returns the cached blob for known-size requests
// (w > 0 && h > 0) within cache max dims, or (nil, false) on miss.
// Unknown-size and oversized requests always return (nil, false) — the cached blob is capped
// at CacheMaxWidth×CacheMaxHeight, which may be smaller than the original.
func (v *Processor) LoadFromCache(key string, w, h int) (*imagor.Blob, bool) {
	if v.cache == nil || w <= 0 || h <= 0 {
		return nil, false
	}
	if w > v.CacheMaxWidth || h > v.CacheMaxHeight {
		return nil, false
	}
	return v.cache.Get(key)
}

// loadOrCache returns a cached blob for the given image path, using the pixel cache.
// Cache key is image path only, so the same source serves all requested sizes.
// If load is non-nil and blob is nil, load is called inside the singleflight to fetch the blob,
// deduplicating network requests across concurrent cache misses.
// Returns (nil, nil, nil) if cache is disabled or the source is animated
// (multi-page structure cannot be preserved in the cache).
func (v *Processor) loadOrCache(
	blob *imagor.Blob, imagePath string, n int, load imagor.LoadFunc,
) (*imagor.Blob, *imagor.Blob, error) {
	if v.cache == nil {
		return nil, nil, nil
	}

	// Fast path: cache hit — return immediately without singleflight overhead.
	if memBlob, ok := v.cache.Get(imagePath); ok {
		return memBlob, nil, nil
	}

	// Deduplicate concurrent cache misses for the same image path.
	result, err, _ := v.cacheSF.Do(imagePath, func() (any, error) {
		// Re-check after acquiring the singleflight group.
		if memBlob, ok := v.cache.Get(imagePath); ok {
			return &loadOrCacheResult{memBlob: memBlob}, nil
		}

		// If blob not provided, fetch it inside the singleflight so concurrent
		// cache misses share a single network request.
		if blob == nil {
			if load == nil {
				return &loadOrCacheResult{}, nil
			}
			var err error
			blob, err = load(imagePath)
			if err != nil {
				return nil, err
			}
		}

		// Decode at maxW×maxH with SizeDown. Fresh context so the VipsSource
		// is released immediately after serialization.
		decodeCtx := withContext(context.Background())

		img, err := v.NewThumbnail(decodeCtx, blob, v.CacheMaxWidth, v.CacheMaxHeight,
			vips.InterestingNone, vips.SizeDown, n, 1, 0)
		if err != nil {
			contextDone(decodeCtx)
			return nil, err
		}

		// Animated source: multi-page structure cannot be preserved in the cache.
		// Return the original blob so the caller can serve it directly.
		if img.Height() != img.PageHeight() {
			img.Close()
			contextDone(decodeCtx)
			return &loadOrCacheResult{origBlob: blob}, nil
		}

		// Static image: serialize to Go-owned bytes, release libvips resources.
		// Storage format is controlled by CacheFormat:
		//   BlobTypeWEBP → WebpsaveBuffer (lossy, smaller memory, slight quality difference)
		//   BlobTypePNG  → PngsaveBuffer (lossless, smaller memory, pixel-identical)
		//   default      → WriteToMemory (raw pixels, fastest hit, most memory)
		imgW, imgH := img.Width(), img.PageHeight()

		var (
			buf     []byte
			memBlob *imagor.Blob
		)
		switch v.CacheFormat {
		case imagor.BlobTypeWEBP:
			buf, err = img.WebpsaveBuffer(nil)
			img.Close()
			contextDone(decodeCtx)
			if err != nil {
				return nil, err
			}
			memBlob = imagor.NewBlobFromBytes(buf)
		case imagor.BlobTypePNG:
			buf, err = img.PngsaveBuffer(nil)
			img.Close()
			contextDone(decodeCtx)
			if err != nil {
				return nil, err
			}
			memBlob = imagor.NewBlobFromBytes(buf)
		default:
			// Raw pixels (BlobTypeMemory) — fastest cache-hit path.
			bands := img.Bands()
			buf, err = img.WriteToMemory()
			img.Close()
			contextDone(decodeCtx)
			if err != nil {
				return nil, err
			}
			memBlob = imagor.NewBlobFromMemory(buf, imgW, imgH, bands)
		}

		// Cache if within max dims (result may be smaller than max due to SizeDown).
		if imgW <= v.CacheMaxWidth && imgH <= v.CacheMaxHeight {
			cost := int64(len(buf))
			if v.CacheTTL > 0 {
				v.cache.SetWithTTL(imagePath, memBlob, cost, v.CacheTTL)
			} else {
				v.cache.Set(imagePath, memBlob, cost)
			}
			v.cache.Wait()
		}
		return &loadOrCacheResult{memBlob: memBlob}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	r := result.(*loadOrCacheResult)
	return r.memBlob, r.origBlob, nil
}
