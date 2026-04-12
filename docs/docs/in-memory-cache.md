---
description: Cache decoded image pixels in memory with LRU eviction to avoid repeated I/O and decoding of the same source image across requests.
---

# In-Memory Cache

Imagor maintains an in-memory cache of decoded image pixels, keyed by image path. This avoids repeated I/O and decode for the same source image across different requests — base images, `watermark()` and `image()` filter overlays all share the same cache.

The cache stores pixel buffers keyed by image path. Each request gets its own independent image object reconstructed from the cached bytes, so concurrent requests are fully safe with no shared mutable state. Concurrent cache misses for the same path are deduplicated — only one fetch and decode occurs regardless of how many requests arrive simultaneously. The cache uses LRU eviction with a configurable byte budget.

```dotenv
VIPS_CACHE_SIZE=52428800      # Cache byte budget (e.g. 50 MiB). Default 0 = disabled
VIPS_CACHE_MAX_WIDTH=2400     # Max image width to cache (default 2400px)
VIPS_CACHE_MAX_HEIGHT=2000    # Max image height to cache (default 2000px)
VIPS_CACHE_TTL=1h             # Cache entry TTL. Default 0 = no expiry (LRU eviction only)
VIPS_CACHE_FORMAT=pixel       # Cache storage format: pixel (default), png, webp
```

## Cache Flow

```mermaid
flowchart TD
    A[Incoming request] --> B{Size > CacheMaxWidth×CacheMaxHeight,\nunknown size, or crop present?}
    B -- Yes --> C[Load at full resolution]
    C --> Z[Apply filters → encode → response]
    B -- No --> D{Cache hit?\nkey = image path}
    D -- Hit --> G
    D -- Miss --> E[Load from source]
    E --> F[Decode + SizeDown\nto ≤ CacheMaxWidth×CacheMaxHeight]
    F --> Anim{Animated\nimage?}
    Anim -- Yes --> Z
    Anim -- No --> Ser[Serialize to memory\npixel / png / webp]
    Ser --> Cache[(In-memory cache)]
    Cache --> G[Reconstruct from cache\n+ resize to requested size]
    G --> Z
```

**When to use:**
- Enable in preview contexts where the same source image is requested at multiple sizes (e.g. `800x600`, `400x300`, `200x150`). Add `filters:preview()` to opt base image requests into the in-memory cache — the first request decodes and caches; subsequent requests skip I/O entirely.
- Enable when the same `watermark()` or `image()` image path is reused across many requests (e.g. a logo watermark on every image).
- The cache key is the **image path only** — filter parameters such as position or size are not part of the key. `watermark(logo.png,100,100)` and `watermark(logo.png,200,200)` share a single cache entry; the cached downscaled version is resized to each requested size on the fly.
- Images larger than `VIPS_CACHE_MAX_WIDTH` × `VIPS_CACHE_MAX_HEIGHT` are still served normally, just not cached.
- Only known-size requests (explicit width × height) are served from cache. Unknown-size (0×0) and oversized requests always load from source to ensure correct native resolution.
- Requests with crop coordinates always bypass the cache, because the cache stores a downscaled copy and pixel coordinates from the original image space would be incorrect.
- Leave disabled (default) if source image paths are highly varied or user-supplied, as caching provides no benefit.
- Set `VIPS_CACHE_TTL` if source images may change at the same image path (e.g. mutable assets). Without a TTL, stale pixels are served until evicted by memory pressure or process restart. For stable assets (logos, static images), TTL is not needed.
- `VIPS_CACHE_FORMAT` controls how cached pixels are stored in memory. `pixel` (default) stores raw uncompressed pixels — fastest cache-hit and pixel-identical, but uses the most memory. `png` uses lossless compression — smaller memory footprint with pixel-identical quality. `webp` uses lossy compression — smallest memory footprint at the cost of slight quality difference.
