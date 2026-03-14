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
	"golang.org/x/sync/singleflight"
)

// singleflightGroup is an alias for singleflight.Group used for overlay cache coalescing.
type singleflightGroup = singleflight.Group

// overlayRistrettoCache is a type alias for the ristretto cache used for overlay images.
type overlayRistrettoCache = ristretto.Cache[string, *overlayCacheEntry]

// overlayCacheEntry holds decoded raw pixel data for a cached overlay image.
// buf is a Go-owned []byte produced by WriteToMemory — independent of any
// libvips lifecycle or request context, safe to cache indefinitely.
// No OnEvict callback is needed: []byte is GC'd automatically when evicted.
type overlayCacheEntry struct {
	buf    []byte
	width  int
	height int
	bands  int
}

// newOverlayCache creates a new ristretto cache for overlay images with the given byte budget.
func newOverlayCache(maxCost int64) (*overlayRistrettoCache, error) {
	return ristretto.NewCache[string, *overlayCacheEntry](&ristretto.Config[string, *overlayCacheEntry]{
		// NumCounters: 10x the expected number of unique items for accurate frequency tracking.
		// For overlay images, 1000 unique URLs is generous; 10000 counters is fine.
		NumCounters: 10000,
		MaxCost:     maxCost,
		BufferItems: 64,
	})
}

// loadOverlayImage loads an overlay image, using the overlay cache when possible.
//
// Two cases:
//
//  1. Size known (w > 0 && h > 0):
//     - If w > OverlayCacheMaxWidth || h > OverlayCacheMaxHeight: skip cache, load directly.
//     - Otherwise: load at max cache dims with SizeDown, WriteToMemory → cache.
//     - On hit: NewImageFromMemory + ThumbnailImage(w, h, size).
//
//  2. Size unknown (w == 0 || h == 0):
//     - Load normally at MaxWidth×MaxHeight with SizeDown (existing behaviour).
//     - After loading, if result fits within cache max dims: WriteToMemory → cache opportunistically.
//     - On hit: NewImageFromMemory, return as-is (already at SizeDown result).
func (v *Processor) loadOverlayImage(
	ctx context.Context, load imagor.LoadFunc,
	url string, w, h, n int, size vips.Size,
) (*vips.Image, error) {
	// Cache disabled or requested size exceeds max — load directly, no caching.
	if v.overlayCache == nil ||
		(w > 0 && w > v.OverlayCacheMaxWidth) ||
		(h > 0 && h > v.OverlayCacheMaxHeight) {
		blob, err := load(url)
		if err != nil {
			return nil, err
		}
		// For unknown size (w==0 || h==0), use MaxWidth/MaxHeight with SizeDown
		// to match the original watermark behaviour.
		if w == 0 || h == 0 {
			return v.NewThumbnail(ctx, blob, v.MaxWidth, v.MaxHeight, vips.InterestingNone, vips.SizeDown, n, 1, 0)
		}
		return v.NewThumbnail(ctx, blob, w, h, vips.InterestingNone, size, n, 1, 0)
	}

	sizeKnown := w > 0 && h > 0

	// Cache hit path
	if entry, ok := v.overlayCache.Get(url); ok {
		return overlayFromCacheEntry(entry, w, h, size, sizeKnown)
	}

	// Cache miss — use singleflight to coalesce concurrent loads for the same URL.
	// Use context.Background() for the decode so that cancellation of the
	// caller's request context does not race with libvips source cleanup
	// (contextDefer registers src.Close on the ctx passed to NewThumbnail).
	result, err, _ := v.overlaySF.Do(url, func() (interface{}, error) {
		// Double-check after acquiring the singleflight slot.
		if entry, ok := v.overlayCache.Get(url); ok {
			return entry, nil
		}
		blob, err := load(url)
		if err != nil {
			return nil, err
		}
		// Use a fresh background context so that the source reader registered
		// via contextDefer is not closed by the caller's request context
		// cancellation while libvips is still decoding.
		decodeCtx := context.Background()
		if sizeKnown {
			// Load at max cache dims with SizeDown so one entry serves all sizes.
			return v.loadAndCacheOverlay(decodeCtx, blob, n, v.OverlayCacheMaxWidth, v.OverlayCacheMaxHeight, url)
		}
		// Unknown size: load at processor MaxWidth/MaxHeight with SizeDown (existing behaviour).
		return v.loadAndCacheOverlay(decodeCtx, blob, n, v.MaxWidth, v.MaxHeight, url)
	})
	if err != nil {
		return nil, err
	}
	return overlayFromCacheEntry(result.(*overlayCacheEntry), w, h, size, sizeKnown)
}

// loadAndCacheOverlay loads the overlay at the given max dimensions, writes raw pixels
// to a Go-owned []byte via WriteToMemory, and stores the entry in the ristretto cache.
// The loaded *vips.Image is closed after WriteToMemory — the caller reconstructs from
// the cache entry via NewImageFromMemory.
// No OnEvict callback is needed: []byte is GC'd automatically when ristretto evicts the entry.
func (v *Processor) loadAndCacheOverlay(
	ctx context.Context, blob *imagor.Blob, n, maxW, maxH int, url string,
) (*overlayCacheEntry, error) {
	img, err := v.NewThumbnail(ctx, blob, maxW, maxH, vips.InterestingNone, vips.SizeDown, n, 1, 0)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	buf, err := img.WriteToMemory()
	if err != nil {
		return nil, err
	}
	entry := &overlayCacheEntry{
		buf:    buf,
		width:  img.Width(),
		height: img.PageHeight(),
		bands:  img.Bands(),
	}

	// Only cache if the result fits within the configured max dims.
	// This handles Case 2C: SizeDown result still exceeds OverlayCacheMaxWidth/Height.
	if img.Width() <= v.OverlayCacheMaxWidth && img.PageHeight() <= v.OverlayCacheMaxHeight {
		// cost = bytes used; ristretto enforces MaxCost (OverlayCacheSize) via LRU eviction.
		v.overlayCache.Set(url, entry, int64(len(buf)))
		v.overlayCache.Wait()
	}
	return entry, nil
}

// overlayFromCacheEntry reconstructs a *vips.Image from a cache entry.
// For known size: wraps the cached raw pixels and resizes to (w, h).
// For unknown size: wraps the cached raw pixels and returns as-is.
func overlayFromCacheEntry(e *overlayCacheEntry, w, h int, size vips.Size, sizeKnown bool) (*vips.Image, error) {
	// NewImageFromMemory wraps the []byte pointer — Image.buf pins it for the image's lifetime.
	// The cached entry's buf is a separate Go reference, so it stays alive after Image.Close().
	// When ristretto evicts the entry, buf is GC'd — no OnEvict close needed.
	img, err := vips.NewImageFromMemory(e.buf, e.width, e.height, e.bands)
	if err != nil {
		return nil, err
	}
	if sizeKnown && (img.Width() != w || img.PageHeight() != h) {
		if err := img.ThumbnailImage(w, &vips.ThumbnailImageOptions{
			Height: h,
			Size:   size,
		}); err != nil {
			img.Close()
			return nil, err
		}
	}
	return img, nil
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
