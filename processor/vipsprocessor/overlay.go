package vipsprocessor

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/imagorpath"
	"github.com/cshum/vipsgen/vips"
	"github.com/dgraph-io/ristretto/v2"
)

// pixelCache is a ristretto cache storing BlobTypeMemory blobs keyed by image path.
// Values are Go-owned raw pixel buffers from WriteToMemory — no libvips lifecycle,
// no request context dependency, safe for concurrent reads and GC cleanup.
type pixelCache = ristretto.Cache[string, *imagor.Blob]

// newPixelCache creates a new ristretto pixel cache with the given byte budget.
func newPixelCache(maxCost int64) (*pixelCache, error) {
	return ristretto.NewCache[string, *imagor.Blob](&ristretto.Config[string, *imagor.Blob]{
		NumCounters: 10000,
		MaxCost:     maxCost,
		BufferItems: 64,
	})
}

// loadOrCacheResult is the singleflight result for loadOrCache.
// memBlob is non-nil for static images (cached raw pixels).
// origBlob is non-nil for animated sources that cannot be cached.
type loadOrCacheResult struct {
	memBlob  *imagor.Blob
	origBlob *imagor.Blob
}

// loadOrCache returns a BlobTypeMemory blob for the given image path, using the pixel cache.
// Cache key is image path only, so the same source serves all requested sizes.
// If load is non-nil and blob is nil, load is called inside the singleflight to fetch the blob,
// deduplicating network requests across concurrent cache misses.
// Returns (nil, nil, nil) if cache is disabled or the source is animated
// (WriteToMemory cannot preserve multi-page structure).
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
		// is released immediately after WriteToMemory.
		decodeCtx := withContext(context.Background())

		img, err := v.NewThumbnail(decodeCtx, blob, v.CacheMaxWidth, v.CacheMaxHeight,
			vips.InterestingNone, vips.SizeDown, n, 1, 0)
		if err != nil {
			contextDone(decodeCtx)
			return nil, err
		}

		// Animated source: WriteToMemory cannot preserve multi-page structure.
		// Return the original blob so the caller can serve it directly.
		if img.Height() != img.PageHeight() {
			img.Close()
			contextDone(decodeCtx)
			return &loadOrCacheResult{origBlob: blob}, nil
		}

		// Static image: serialize to Go-owned bytes, release libvips resources.
		// Storage format is controlled by CacheFormat:
		//   BlobTypeWEBP → WebpsaveBuffer (lossy, ~17× smaller, slight generation loss)
		//   BlobTypePNG  → PngsaveBuffer (lossless, ~6.6× smaller, pixel-identical)
		//   default      → WriteToMemory (raw pixels, fastest hit, most memory)
		// Capture dimensions before closing img.
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

// loadOverlayImage loads a watermark overlay image using the pixel cache.
// Cache key is image path only. Size must be known (w > 0 && h > 0) to use the cache;
// unknown size or size exceeding cache max dims bypasses the cache.
func (v *Processor) loadOverlayImage(
	ctx context.Context, load imagor.LoadFunc,
	url string, w, h, n int, size vips.Size,
) (*vips.Image, error) {
	sizeKnown := w > 0 && h > 0

	// Unknown size or cache disabled: load directly.
	if !sizeKnown || v.cache == nil {
		blob, err := load(url)
		if err != nil {
			return nil, err
		}
		if sizeKnown {
			return v.NewThumbnail(ctx, blob, w, h, vips.InterestingNone, size, n, 1, 0)
		}
		return v.NewThumbnail(ctx, blob, v.MaxWidth, v.MaxHeight, vips.InterestingNone, vips.SizeDown, n, 1, 0)
	}

	// Size exceeds cache max dims — bypass cache, load directly at requested size.
	if w > v.CacheMaxWidth || h > v.CacheMaxHeight {
		blob, err := load(url)
		if err != nil {
			return nil, err
		}
		return v.NewThumbnail(ctx, blob, w, h, vips.InterestingNone, size, n, 1, 0)
	}

	// Cache hit — serve from cached memory blob without loading.
	if memBlob, ok := v.cache.Get(url); ok {
		return v.NewThumbnail(ctx, memBlob, w, h, vips.InterestingNone, size, 1, 1, 0)
	}

	// Cache miss: fetch and decode inside the singleflight to deduplicate concurrent
	// network requests. load is passed so loadOrCache can call it inside the singleflight.
	memBlob, origBlob, err := v.loadOrCache(nil, url, n, load)
	if err != nil {
		return nil, err
	}

	// Animated source — origBlob is set; fall back to direct thumbnail at w×h.
	if origBlob != nil {
		return v.NewThumbnail(ctx, origBlob, w, h, vips.InterestingNone, size, n, 1, 0)
	}

	if memBlob == nil {
		// Cache disabled or blob could not be loaded — should not happen here.
		return nil, imagor.ErrNotFound
	}

	// Static: resize from cached memory blob to requested w×h.
	return v.NewThumbnail(ctx, memBlob, w, h, vips.InterestingNone, size, 1, 1, 0)
}

// loadFilterImage runs the imagor pipeline for an image() filter using the pixel cache.
// Bypasses cache for unknown size, exceeds max dims, or crop/focal filters
// (which depend on original image coordinates, not the downscaled cached copy).
func (v *Processor) loadFilterImage(
	ctx context.Context, blob *imagor.Blob, params imagorpath.Params, load imagor.LoadFunc,
	url string,
) (*vips.Image, error) {
	sizeKnown := params.Width > 0 && params.Height > 0

	if v.cache == nil || blob == nil || !sizeKnown || imagorpath.HasCrop(params) || imagorpath.HasFilter(params, "focal") ||
		params.Width > v.CacheMaxWidth || params.Height > v.CacheMaxHeight {
		return v.loadAndProcess(ctx, blob, params, load)
	}

	memBlob, origBlob, err := v.loadOrCache(blob, url, 1, nil)
	if err != nil {
		return nil, err
	}

	if origBlob != nil || memBlob == nil {
		// Animated source or cache miss — run pipeline on original blob.
		return v.loadAndProcess(ctx, blob, params, load)
	}
	return v.loadAndProcess(ctx, memBlob, params, load)
}

// fullDimRegex matches a single dimension token: optionally a flip prefix -,
// then f or full, optionally followed by a negative integer offset
// e.g. f, f-20, full, full-20, -f, -full-20.
var fullDimRegex = regexp.MustCompile(`^(-?)(?:full|f)(-\d+)?$`)

// dimSegmentRegex matches a /‑separated WxH segment where either or both sides
// may be an f/full‑token or a plain integer.
var dimSegmentRegex = regexp.MustCompile(`^(-?(?:(?:full|f)(?:-\d+)?|\d*))x(-?(?:(?:full|f)(?:-\d+)?|\d*))$`)

// resolveFullDim resolves a single dimension token against a parent pixel size.
// Tokens of the form f or f-NNN (with optional leading - for flip)
// resolve to parentDim - NNN. Any other token is returned unchanged.
func resolveFullDim(token string, parentDim int) string {
	m := fullDimRegex.FindStringSubmatch(token)
	if m == nil {
		return token
	}
	flip := m[1]
	offset := 0
	if m[2] != "" {
		offset, _ = strconv.Atoi(m[2])
	}
	return flip + strconv.Itoa(parentDim+offset)
}

// resolveFullDimensions rewrites f‑tokens in the WxH dimension segment of an
// imagor path, substituting the parent image's pixel dimensions before the path
// is parsed. Only the first WxH dimension segment is considered — it always
// appears before filters: in a valid imagor path. The function stops at the
// first dimSegmentRegex match or at a filters: prefix, ensuring nested layer
// paths inside filter arguments are not accidentally resolved at this level.
func resolveFullDimensions(imagorPath string, parentW, parentH int) string {
	start := 0
	for i := 0; i <= len(imagorPath); i++ {
		if i < len(imagorPath) && imagorPath[i] != '/' {
			continue
		}
		seg := imagorPath[start:i]
		// Stop before filters — nested layer paths inside filter arguments
		// must be resolved at their own processing level, not here.
		if strings.HasPrefix(seg, "filters:") {
			return imagorPath
		}
		if m := dimSegmentRegex.FindStringSubmatch(seg); m != nil {
			// Found the dimension segment. Resolve f-tokens if present.
			newLeft := resolveFullDim(m[1], parentW)
			newRight := resolveFullDim(m[2], parentH)
			if newLeft != m[1] || newRight != m[2] {
				return imagorPath[:start] + newLeft + "x" + newRight + imagorPath[i:]
			}
			return imagorPath
		}
		start = i + 1
	}
	return imagorPath
}

// blendModeMap maps blend mode names to vips.BlendMode constants
var blendModeMap = map[string]vips.BlendMode{
	"normal":      vips.BlendModeOver,
	"multiply":    vips.BlendModeMultiply,
	"color-burn":  vips.BlendModeColourBurn,
	"darken":      vips.BlendModeDarken,
	"screen":      vips.BlendModeScreen,
	"color-dodge": vips.BlendModeColourDodge,
	"lighten":     vips.BlendModeLighten,
	"add":         vips.BlendModeAdd,
	"overlay":     vips.BlendModeOverlay,
	"soft-light":  vips.BlendModeSoftLight,
	"hard-light":  vips.BlendModeHardLight,
	"difference":  vips.BlendModeDifference,
	"exclusion":   vips.BlendModeExclusion,
	"mask":        vips.BlendModeDestIn,
	"mask-out":    vips.BlendModeDestOut,
}

// parseOverlayPosition parses position argument and returns position value and repeat count
func parseOverlayPosition(arg string, canvasSize, overlaySize int, hAlign, vAlign string) (pos int, repeat int) {
	repeat = 1
	if arg == "" {
		return 0, 1
	}

	// Check for alignment keyword with negative offset (e.g., left-20, l-20, right-30, r-30, top-20, t-20, bottom-20, b-20)
	if strings.HasPrefix(arg, "left-") || strings.HasPrefix(arg, "l-") {
		offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(arg, "left-"), "l-"))
		return -offset, 1
	} else if strings.HasPrefix(arg, "right-") || strings.HasPrefix(arg, "r-") {
		offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(arg, "right-"), "r-"))
		return canvasSize - overlaySize + offset, 1
	} else if strings.HasPrefix(arg, "top-") || strings.HasPrefix(arg, "t-") {
		offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(arg, "top-"), "t-"))
		return -offset, 1
	} else if strings.HasPrefix(arg, "bottom-") || strings.HasPrefix(arg, "b-") {
		offset, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(arg, "bottom-"), "b-"))
		return canvasSize - overlaySize + offset, 1
	}

	if arg == "center" {
		return (canvasSize - overlaySize) / 2, 1
	} else if arg == hAlign || arg == vAlign {
		if arg == imagorpath.HAlignRight || arg == imagorpath.VAlignBottom {
			return canvasSize - overlaySize, 1
		}
		return 0, 1
	} else if arg == "repeat" {
		return 0, canvasSize/overlaySize + 1
	} else if strings.HasPrefix(strings.TrimPrefix(arg, "-"), "0.") {
		pec, _ := strconv.ParseFloat(arg, 64)
		return int(pec * float64(canvasSize)), 1
	} else if strings.HasSuffix(arg, "p") {
		val, _ := strconv.Atoi(strings.TrimSuffix(arg, "p"))
		return val * canvasSize / 100, 1
	}

	pos, _ = strconv.Atoi(arg)
	return pos, 1
}

// compositeOverlay transforms and composites overlay image onto the base image
// Handles color space, alpha channel, positioning, repeat patterns, cropping, and animation frames
// Returns early without compositing if overlay is completely outside canvas bounds
func compositeOverlay(img *vips.Image, overlay *vips.Image, xArg, yArg string, alpha float64, blendMode vips.BlendMode) error {
	// Ensure overlay has proper color space and alpha
	if overlay.Bands() < 3 {
		if err := overlay.Colourspace(vips.InterpretationSrgb, nil); err != nil {
			return err
		}
	}
	if !overlay.HasAlpha() {
		if err := overlay.Addalpha(); err != nil {
			return err
		}
	}

	// Apply alpha if provided
	if alpha > 0 {
		alphaMultiplier := 1 - alpha/100
		if alphaMultiplier != 1 {
			if err := overlay.Linear([]float64{1, 1, 1, alphaMultiplier}, []float64{0, 0, 0, 0}, nil); err != nil {
				return err
			}
		}
	}

	// Parse position
	overlayWidth := overlay.Width()
	overlayHeight := overlay.PageHeight()

	x, across := parseOverlayPosition(xArg, img.Width(), overlayWidth, imagorpath.HAlignLeft, imagorpath.HAlignRight)
	y, down := parseOverlayPosition(yArg, img.PageHeight(), overlayHeight, imagorpath.VAlignTop, imagorpath.VAlignBottom)

	// Apply negative adjustment for plain numeric values only (not prefixed keywords)
	if x < 0 && xArg != "center" &&
		!strings.HasPrefix(xArg, "left-") && !strings.HasPrefix(xArg, "l-") &&
		!strings.HasPrefix(xArg, "right-") && !strings.HasPrefix(xArg, "r-") {
		x += img.Width() - overlayWidth
	}
	if y < 0 && yArg != "center" &&
		!strings.HasPrefix(yArg, "top-") && !strings.HasPrefix(yArg, "t-") &&
		!strings.HasPrefix(yArg, "bottom-") && !strings.HasPrefix(yArg, "b-") {
		y += img.PageHeight() - overlayHeight
	}

	// Handle repeat pattern
	if across*down > 1 {
		if err := overlay.EmbedMultiPage(0, 0, across*overlayWidth, down*overlayHeight,
			&vips.EmbedMultiPageOptions{Extend: vips.ExtendRepeat}); err != nil {
			return err
		}
		// Update dimensions after repeat
		overlayWidth = overlay.Width()
		overlayHeight = overlay.PageHeight()
	}

	// Check if overlay is completely outside canvas bounds
	// Skip compositing if there's no intersection with the canvas
	if x >= img.Width() || y >= img.PageHeight() ||
		x+overlayWidth <= 0 || y+overlayHeight <= 0 {
		// Overlay is completely outside canvas bounds, skip it
		return nil
	}

	// Position overlay on canvas
	// Crop overlay to only the visible portion within canvas bounds
	visibleLeft := 0
	visibleTop := 0
	visibleWidth := overlayWidth
	visibleHeight := overlayHeight
	embedX := x
	embedY := y

	// Handle overlay extending beyond right/bottom edges
	if x+overlayWidth > img.Width() {
		visibleWidth = img.Width() - x
	}
	if y+overlayHeight > img.PageHeight() {
		visibleHeight = img.PageHeight() - y
	}

	// Handle overlay starting before left/top edges (negative positions)
	if x < 0 {
		visibleLeft = -x
		visibleWidth = overlayWidth + x // reduce width
		embedX = 0
	}
	if y < 0 {
		visibleTop = -y
		visibleHeight = overlayHeight + y // reduce height
		embedY = 0
	}

	// Crop overlay to visible portion if needed
	if visibleLeft > 0 || visibleTop > 0 ||
		visibleWidth < overlayWidth || visibleHeight < overlayHeight {
		if visibleWidth > 0 && visibleHeight > 0 {
			if err := overlay.ExtractAreaMultiPage(
				visibleLeft, visibleTop, visibleWidth, visibleHeight,
			); err != nil {
				return err
			}
		} else {
			// Overlay is completely outside canvas bounds, skip it
			return nil
		}
	}

	// Embed the cropped overlay at adjusted position
	if err := overlay.EmbedMultiPage(
		embedX, embedY, img.Width(), img.PageHeight(), nil,
	); err != nil {
		return err
	}

	// Handle animation frames
	overlayN := overlay.Height() / overlay.PageHeight()
	if n := img.Height() / img.PageHeight(); n > overlayN {
		cnt := n / overlayN
		if n%overlayN > 0 {
			cnt++
		}
		if err := overlay.Replicate(1, cnt); err != nil {
			return err
		}
	}

	// Composite overlay onto image with specified blend mode
	return img.Composite2(overlay, blendMode, nil)
}

// getBlendMode returns the vips.BlendMode for a given mode string
// Defaults to BlendModeOver (normal) if mode is empty or invalid
func getBlendMode(mode string) vips.BlendMode {
	if mode == "" {
		return vips.BlendModeOver
	}
	if blendMode, ok := blendModeMap[strings.ToLower(mode)]; ok {
		return blendMode
	}
	// Default to normal if invalid mode
	return vips.BlendModeOver
}
