# pigoprocessor — face-detection-aware smart crop

`pigoprocessor` implements the `vipsprocessor.Detector` interface using
[pigo](https://github.com/esimov/pigo), a pure-Go face detection library based
on the PICO algorithm (Pixel Intensity Comparison-based Object detection).

When wired into the imagor server it upgrades the `smart` URL token from
libvips' general-purpose attention heuristic into face-aware cropping: the
crop window is centred on the weighted centroid of all detected faces rather
than on the most visually salient region.

---

## Background — how smart crop works today

The imagor URL format accepts a `smart` token:

```
/smart/400x300/https://example.com/portrait.jpg
```

Without face detection this sets `Params.Smart = true`, which tells
vipsprocessor to pass `vips.InterestingAttention` to libvips'
`ThumbnailImage`. libvips uses an entropy/saliency heuristic to pick the crop
centre — it works well for general scenes but has no concept of faces.

The existing focal-point pipeline, used by the `focal()` filter, is strictly
more precise: absolute bounding boxes are aggregated into a single weighted
centroid (`parseFocalPoint`), and `FocalThumbnail` performs a two-step
scale-then-extract crop centred on that point. This implementation plugs
directly into that pipeline.

---

## Architecture

```
Request: /smart/400x300/image.jpg
               │
               ▼
    vipsprocessor.Process()
               │
    loadAndProcess() decodes image
               │
    origWidth, origHeight established
               │
    p.Smart && v.Detector != nil && len(focalRects) == 0 ?
               │ yes
               ▼
    detectRegions(ctx, img)
      ├─ Copy image → ThumbnailImage(max 400px, SizeDown)
      ├─ normalizeSrgb
      ├─ WriteToMemory → []uint8 (raw sRGB bands)
      └─ Detector.Detect(ctx, buf, probeW, probeH, bands)
               │
               │  pigoprocessor.Detect()
               │    ├─ toGrayscale (ITU-R BT.601)
               │    ├─ pigo.RunCascade(CascadeParams)
               │    ├─ pigo.ClusterDetections (NMS, IoU=0.2)
               │    └─ filter by minQuality (default 5.0)
               │    └─ return []Region  ← normalised [0.0, 1.0] ratios
               │
    multiply Region by origWidth/origHeight → []focal (absolute px)
               │
    parseFocalPoint(focalRects...) → weighted centroid (focalX, focalY)
               │
    FocalThumbnail(img, w, h, fx, fy) → scale + extract
               │
    No faces found? → InterestingAttention fallback (unchanged behaviour)
```

### Key design decisions

| Decision | Rationale |
|---|---|
| `Detector` not `FaceDetector` | Interface is generic — any region detector (faces, objects, text) can implement it |
| Regions as normalised ratios `[0.0, 1.0]` | Probe size stays internal to the implementation; caller never needs to know it |
| 400 px probe cap | ~10× speed-up vs full-res; PICO cascade is designed for small fixed windows, precision loss is negligible for crop purposes |
| Errors are non-fatal | `detectRegions` returns nil on any error; code falls through to `InterestingAttention`, so a broken detector degrades gracefully |
| No detection cache | V1 scope; each request runs detection independently. Future: cache results keyed by image URL + TTL |
| Force full-res load when detector active | When `p.Smart` is true and a detector is configured, `thumbnailNotSupported = true` is set. This prevents libvips from decoding + attention-cropping in one shot via `NewThumbnail`, which would otherwise leave only the already-cropped thumbnail available for detection. |

---

## File structure

```
processor/pigoprocessor/
  pigoprocessor.go       # Detector implementation
  cascade/
    facefinder           # Pre-trained PICO face cascade (~240 KB), go:embed target
```

### cascade/facefinder

This is a **pre-trained binary model** — a serialised ensemble of binary pixel
comparison decision trees trained on a large labelled face dataset. It was
produced by the pigo author and is committed to the pigo repository at
`github.com/esimov/pigo/cascade/facefinder`.

**It must be committed to git.** The file is loaded at compile time via
`//go:embed cascade/facefinder`; without it the build fails. It cannot be
regenerated from any code in this repository — treat it like a vendored asset
(font file, TLS root cert, etc.). To upgrade the model, copy a newer version
of the file from the pigo module cache:

```sh
cp $(go env GOPATH)/pkg/mod/github.com/esimov/pigo@<version>/cascade/facefinder \
   processor/pigoprocessor/cascade/facefinder
```

---

## Integration points

### vipsprocessor.Detector (interface)

Defined in [processor/vipsprocessor/detector.go](../vipsprocessor/detector.go).

```go
type Region struct {
    Left, Top, Right, Bottom float64 // normalised [0.0, 1.0]
}

type Detector interface {
    Detect(ctx context.Context, buf []uint8, width, height, bands int) ([]Region, error)
}
```

`buf` is a row-major raw pixel buffer from `vips.WriteToMemory` — sRGB with
`bands` channels per pixel (3 = RGB, 4 = RGBA). `len(buf) == width * height * bands`.

### vipsprocessor.Processor

```go
// Field on Processor struct (processor/vipsprocessor/processor.go)
Detector Detector

// Functional option (processor/vipsprocessor/option.go)
vipsprocessor.WithDetector(d Detector)
```

### Detection hook in process.go

The hook lives in `loadAndProcess` immediately after the `focal()` filter
argument loop and before `applyTransformations`
([processor/vipsprocessor/process.go](../vipsprocessor/process.go)):

```go
if p.Smart && v.Detector != nil && len(focalRects) == 0 {
    for _, r := range v.detectRegions(ctx, img) {
        focalRects = append(focalRects, focal{
            Left:   r.Left * origWidth,
            Top:    r.Top * origHeight,
            Right:  r.Right * origWidth,
            Bottom: r.Bottom * origHeight,
        })
    }
}
```

Gate conditions:
- `p.Smart` — only runs for `smart` URL token
- `v.Detector != nil` — only when a detector is configured
- `len(focalRects) == 0` — explicit `focal()` filter takes precedence; detection is skipped

---

## Enabling face detection

### Server flag

```sh
imagor --vips-face-detect ...
```

Defined in [config/vipsconfig/vipsconfig.go](../../config/vipsconfig/vipsconfig.go).
If pigo fails to initialise (corrupt cascade file, etc.) a warning is logged
and the server starts normally without face detection.

### Programmatic

```go
import (
    "github.com/cshum/imagor/processor/pigoprocessor"
    "github.com/cshum/imagor/processor/vipsprocessor"
)

detector, err := pigoprocessor.New()
if err != nil {
    log.Fatal(err)
}

processor := vipsprocessor.NewProcessor(
    vipsprocessor.WithDetector(detector),
    // ...other options
)
```

### Custom cascade or tuning

```go
detector, err := pigoprocessor.NewWithCascade(myCascadeBytes,
    pigoprocessor.WithMinSize(30),       // minimum face px on probe image (default 20)
    pigoprocessor.WithMaxSize(300),      // maximum face px on probe image (default 400)
    pigoprocessor.WithMinQuality(8.0),   // raise to reduce false positives (default 5.0)
    pigoprocessor.WithIoUThreshold(0.3), // NMS aggressiveness (default 0.2)
)
```

---

## Full-resolution load requirement

When a detector is configured, vipsprocessor sets `thumbnailNotSupported = true`
for `smart` requests. This forces `NewImage` (full-res decode) instead of
`NewThumbnail`. Without this, libvips' shrink-on-load path would decode and
attention-crop to the target size in a single C-library call — by the time the
Go detection hook runs, `img` would already be the final cropped output and
`origWidth`/`origHeight` would reflect the output size, not the source.
Detection would then see a tiny already-cropped image and return `regions: 0`
every time (confirmed during development: `orig_width: 600, orig_height: 600`).

The performance cost is a full-resolution decode instead of a shrink-on-load
decode. This is partially offset by detection running on a cheap 400 px probe
copy, not the full-res image.

---

## Debugging detected regions

### Meta endpoint — JSON output (Option 1)

When a detector is configured, the imagor `meta` endpoint includes a
`detected_regions` array in its JSON response.  Each element contains absolute
pixel coordinates of a detected face in the **original source image**.

```sh
# example
curl "http://localhost:8000/meta/smart/0x0/https://example.com/portrait.jpg"
```

```json
{
  "format": "jpeg",
  "content_type": "image/jpeg",
  "width": 1200,
  "height": 900,
  "orientation": 1,
  "detected_regions": [
    { "left": 312, "top": 95, "right": 488, "bottom": 271 },
    { "left": 680, "top": 120, "right": 830, "bottom": 270 }
  ]
}
```

The `meta` endpoint does not transform the image, so `detected_regions` always
refers to the source dimensions. The array is omitted (`omitempty`) when the
detector returns no regions or no detector is configured.

### Visual overlay filter — `detect_regions()` (Option 2)

The `detect_regions()` filter draws semi-transparent filled rectangles and
a solid 2 px outline around each detected face on the output image. It is
intended for visual debugging only.

```
filters:detect_regions()
filters:detect_regions(color)
filters:detect_regions(color,opacity)
```

| Parameter | Default | Description |
|---|---|---|
| `color` | `ff0000` | Any hex colour string accepted by other imagor filters (e.g. `00ff00`, `blue`) |
| `opacity` | `40` | Fill opacity 0–100. `0` draws the outline only with no fill. Outline is always fully opaque. |

Example URLs:

```
# Red boxes at 40 % fill opacity (default)
/filters:detect_regions()/smart/400x300/portrait.jpg

# Green outline only, no fill
/filters:detect_regions(00ff00,0)/smart/400x300/portrait.jpg

# Blue at 60 % fill
/filters:detect_regions(0000ff,60)/smart/400x300/portrait.jpg
```

The filter is a no-op when no detector is configured. Detection runs on the
400 px probe copy (same as the smart crop path), so adding `detect_regions()`
does not materially affect performance.

---

## Cache behaviour

The image cache (`--vips-cache-size`) presents no special concern for face
detection. The cache may return a downscaled blob instead of the original
source, so `origWidth`/`origHeight` in `loadAndProcess` reflect the decoded
(possibly cached) dimensions — not necessarily the original source dimensions.
Because detection results are returned as normalised ratios and are immediately
multiplied by `origWidth`/`origHeight`, the focal point is always correct
relative to whatever image was decoded. No `HasCacheBypass` is required for
`smart`.

The `focal()` filter already sets `HasCacheBypass = true` (see
[imagorpath/params.go](../../imagorpath/params.go)), so any explicit focal
point bypasses the cache anyway; the detector is not called in that case
(`len(focalRects) == 0` gate).

---

## image() filter overlays

The `image()` filter loads and processes an overlay image through its own
`loadAndProcess` call with the overlay's own parsed params. The overlay path
would need to explicitly contain the `smart` token (e.g.
`image(smart/200x200/face.jpg)`) for detection to run on it. By default
`Smart = false` for overlay params, so the detector is never triggered for
overlay images unless the user opts in. This is correct behaviour.

---

## Implementing a custom Detector

Any type that satisfies `vipsprocessor.Detector` can be passed to
`WithDetector`. Example skeleton:

```go
type MyDetector struct{}

func (d *MyDetector) Detect(
    ctx context.Context, buf []uint8, width, height, bands int,
) ([]vipsprocessor.Region, error) {
    // buf is row-major sRGB(A), len(buf) == width*height*bands
    // Return regions as normalised [0.0, 1.0] ratios.
    // Return nil, nil for "no regions" — do not return an error for empty results.
    return []vipsprocessor.Region{
        {Left: 0.3, Top: 0.1, Right: 0.7, Bottom: 0.6},
    }, nil
}
```

Multiple regions are supported. `parseFocalPoint` computes a weighted centroid
where each region's weight is proportional to its area, so larger detected
faces have more influence over the crop centre.

---

## Future work

- **Detection result cache** — cache `[]Region` keyed by image URL + TTL to
  avoid re-running detection on repeated `smart` requests for the same source
  image. This is the main remaining performance optimisation for high-traffic
  deployments.
- **Pupil / landmark detection** — pigo also provides pupil locator and facial
  landmark cascades (`puploc`, `lps` in the cascade directory). These could
  be used to produce a tighter, more accurate crop centre on the eye region
  rather than the full face bounding box.
- **Alternative detectors** — wire in a different backend (e.g. a TFLite or
  ONNX model via CGO) by implementing `vipsprocessor.Detector`. The pigo
  implementation is one choice; the interface is intentionally generic.
