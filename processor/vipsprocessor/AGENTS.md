# VIPSPROCESSOR - IMAGE PROCESSING ENGINE

**Parent:** See `../../AGENTS.md` for project overview

## OVERVIEW

libvips-based image processor. Handles resize, crop, filters, watermarks, format conversion. Uses `vipsgen` bindings (cshum/vipsgen).

## STRUCTURE

```
vipsprocessor/
├── processor.go    # Entry point, Startup/Shutdown, interface impl
├── process.go      # Main transformation pipeline (736 LOC - hotspot)
├── filter.go       # Individual filters: watermark, label, blur, etc.
├── option.go       # Functional options
├── fallback.go     # BMP/ImageMagick fallback handling
├── exif.go         # EXIF metadata extraction
└── context.go      # Context-based state (rotation flags)
```

## WHERE TO LOOK

| Task | File | Function/Area |
|------|------|---------------|
| Add new filter | `filter.go` | Add to `FilterMap` in `processor.go` |
| Modify resize logic | `process.go` | `Process()`, `thumbnail*` functions |
| Change format output | `process.go` | `export()` function |
| Handle new format | `fallback.go` | `FallbackFunc` |
| Fix EXIF issues | `exif.go` | `GetExif()` |

## FILTER IMPLEMENTATION

### Adding a New Filter
```go
// 1. In filter.go, add handler:
func myFilter(ctx context.Context, img *vips.Image, load imagor.LoadFunc, args ...string) error {
    // args[0], args[1], etc. from URL: filters:my_filter(arg0,arg1)
    return nil
}

// 2. In processor.go NewProcessor(), register:
v.Filters = FilterMap{
    "my_filter": myFilter,
    // ...existing filters
}
```

### Filter Signature
```go
type FilterFunc func(ctx context.Context, img *vips.Image, load imagor.LoadFunc, args ...string) error
```
- `img` — mutable VIPS image (modify in place)
- `load` — fetch secondary images (watermarks)
- `args` — string args from URL, parse with `strconv`

## CONVENTIONS

### Image Lifecycle
```go
img, err := v.NewImage(ctx, blob, n, page, dpi)
if err != nil { return nil, WrapErr(err) }
// ALWAYS defer close if you might return early
defer img.Close()  // or use contextDefer

// After successful processing:
return img, nil  // caller owns the image now
```

### Error Wrapping
```go
// ALWAYS wrap vips errors:
if err := img.Resize(scale); err != nil {
    return WrapErr(err)
}
```

### Resolution Check (SECURITY)
```go
// ALWAYS check before heavy processing:
if _, err := v.CheckResolution(img, nil); err != nil {
    return nil, err  // ErrMaxResolutionExceeded
}
```

### Context State
```go
// Pass rotation state between filters:
setRotate90(ctx)      // mark 90° rotation applied
if isRotate90(ctx) {  // check if applied
    // adjust calculations
}
```

## ANTI-PATTERNS

| Pattern | Why Forbidden |
|---------|---------------|
| Skip `CheckResolution()` | Image bomb DoS vulnerability |
| Forget `img.Close()` | VIPS memory leak |
| Return raw vips errors | Breaks error handling, use `WrapErr()` |
| Hardcode max dimensions | Use `v.MaxWidth`, `v.MaxHeight` |
| Ignore `contextDefer` | Resource leaks on early return |

## COMPLEXITY HOTSPOTS

### `process.go:Process()` (lines 38-362)
- Massive switch/conditional logic
- Handles shrink-on-load optimization
- Mixes parameter interpretation with execution
- **Tip**: Follow the `params` variable to understand flow

### `filter.go:watermark()` (lines 20-169)
- Complex coordinate math (pixels, %, alignment)
- Handles `repeat` mode for tiled watermarks
- **Tip**: Coordinate resolution logic is duplicated with `label()`

## NOTES

- **Startup/Shutdown**: Only ONE processor should call `vips.Startup()`
- **Concurrency**: Uses `processorLock` mutex for init safety
- **MozJPEG**: Enable via `v.MozJPEG = true` option
- **Unlimited mode**: `v.Unlimited = true` bypasses resolution checks (DANGEROUS)
- **Animation**: Multi-page handling via `n` (frame count) and `page` (start frame)
