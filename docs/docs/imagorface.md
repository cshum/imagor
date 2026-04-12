---
description: Extends imagor with face detection — enables face-centred smart crop and privacy redaction for images.
---

# imagorface

[![Test Status](https://github.com/cshum/imagorface/workflows/test/badge.svg)](https://github.com/cshum/imagorface/actions/workflows/test.yml)
[![Codecov](https://img.shields.io/codecov/c/github/cshum/imagorface)](https://codecov.io/gh/cshum/imagorface)
[![Docker Hub](https://img.shields.io/badge/docker-shumc/imagorface-blue.svg)](https://hub.docker.com/r/shumc/imagorface/)

imagorface brings fast, on-the-fly face detection to imagor. Built on [pigo](https://github.com/esimov/pigo) PICO cascade classifier, it detects faces in images and wires into the imagor pipeline for face-centred smart crop and privacy redaction — all self-hosted with no third-party API calls.

- **Face-centred smart crop** — detected faces anchor the smart crop, no more headless bodies
- **Privacy redaction** — blur, pixelate, or solid-fill detected faces for content moderation
- **Metadata API** — detected face regions exposed through imagor `/meta` endpoint
- **Self-hosted** — no third-party API, no per-call cost, no data egress

imagorface implements the imagor [`Detector` interface](https://github.com/cshum/imagor/blob/master/detector.go), integrating with imagor [loader, storage and result storage](./storage.md), and supporting all [image endpoint](./image-endpoint.md) operations and [filters](./filters.md) out of the box.

:::info
**GitHub:** [cshum/imagorface](https://github.com/cshum/imagorface)  
**Docker:** [shumc/imagorface](https://hub.docker.com/r/shumc/imagorface)
:::

## Quick Start

```bash
docker run -p 8000:8000 shumc/imagorface -imagor-unsafe -face-detector
```

Original image:
```
https://raw.githubusercontent.com/cshum/imagorface/refs/heads/main/testdata/people.jpg
```

<img src="/img/imagorface/people.jpg" width="500" />

Try these URLs:
```
http://localhost:8000/unsafe/500x250/smart/IMAGE
http://localhost:8000/unsafe/500x250/smart/filters:draw_detections()/IMAGE
http://localhost:8000/unsafe/500x250/smart/filters:redact()/IMAGE
http://localhost:8000/unsafe/500x250/smart/filters:redact(pixelate)/IMAGE
http://localhost:8000/unsafe/500x250/smart/filters:redact(black)/IMAGE
http://localhost:8000/unsafe/500x250/smart/filters:redact_oval()/IMAGE
```

## Docker Compose Example

```yaml
version: "3"
services:
  imagor:
    image: shumc/imagorface:latest
    environment:
      PORT: 8000
      IMAGOR_SECRET: mysecret

      FACE_DETECTOR: 1                  # enable face detection for smart crop
      FACE_DETECTOR_MIN_SIZE: 20        # min face size in pixels on probe image
      FACE_DETECTOR_MIN_QUALITY: 5.0    # detection quality threshold
      FACE_DETECTOR_CACHE_SIZE: 500     # cache detection results per source image
      FACE_DETECTOR_CACHE_TTL: 1h       # cache TTL (0 = no expiry)
    ports:
      - "8000:8000"
```

## Smart Crop

When `-face-detector` is enabled, imagorface runs face detection before the crop step. If one or more faces are detected, their bounding boxes become the focal region for [smart crop](./image-endpoint.md#smart-crop), replacing the default libvips attention heuristic. When no faces are found, imagor falls back to standard attention-based crop.

Face detection runs on a downscaled greyscale probe derived from raw decoded pixels, keeping overhead minimal.

<table>
  <tr>
    <th width="50%"><code>smart</code></th>
    <th width="50%"><code>smart/filters:draw_detections()</code></th>
  </tr>
  <tr>
    <td><img src="/img/imagorface/demo-smart-crop.jpg" /></td>
    <td><img src="/img/imagorface/demo-draw-detections.jpg" /></td>
  </tr>
</table>

---

## Filters

imagorface adds the following filters to the imagor pipeline. See [Filters](./filters.md) for the full filter reference.

### `draw_detections()`

Draws colour-coded bounding boxes for all detected face regions. Each class is automatically assigned a distinct colour via hash-based palette — useful for visual inspection and debugging.

```
http://localhost:8000/unsafe/500x250/smart/filters:draw_detections()/IMAGE
```

### `redact([mode[, strength]])`

Obscures all detected face regions for privacy and anonymisation (GDPR face blurring, legal document redaction). No-op when no faces are detected. Does not apply to animated images.

- `mode` — `blur` (default), `pixelate`, or any colour name/hex for solid fill (e.g. `black`, `white`, `ff0000`)
- `strength` — blur sigma (default `15`) or pixelate block size in pixels (default `10`). Not used for solid colour mode.

```
http://localhost:8000/unsafe/500x250/smart/filters:redact()/IMAGE                 # blur (default)
http://localhost:8000/unsafe/500x250/smart/filters:redact(pixelate)/IMAGE         # pixelate
http://localhost:8000/unsafe/500x250/smart/filters:redact(blur,25)/IMAGE          # blur with custom strength
http://localhost:8000/unsafe/500x250/smart/filters:redact(black)/IMAGE            # solid black fill
http://localhost:8000/unsafe/500x250/smart/filters:redact(white)/IMAGE            # solid white fill
http://localhost:8000/unsafe/500x250/smart/filters:redact(ff0000)/IMAGE           # custom colour fill
```

<table>
  <tr>
    <th width="50%"><code>redact()</code> — blur</th>
    <th width="50%"><code>redact(pixelate)</code></th>
  </tr>
  <tr>
    <td><img src="/img/imagorface/demo-redact-blur.jpg" /></td>
    <td><img src="/img/imagorface/demo-redact-pixelate.jpg" /></td>
  </tr>
  <tr>
    <th><code>redact(black)</code></th>
    <th><code>redact_oval()</code> — oval blur</th>
  </tr>
  <tr>
    <td><img src="/img/imagorface/demo-redact-black.jpg" /></td>
    <td><img src="/img/imagorface/demo-redact-oval.jpg" /></td>
  </tr>
</table>

### `redact_oval([mode[, strength]])`

Identical to `redact()` but applies an **elliptical mask** to each region, producing a rounded redaction shape that closely follows the natural contour of a face. Same arguments and defaults as `redact()`.

```
http://localhost:8000/unsafe/500x250/smart/filters:redact_oval()/IMAGE            # oval blur (default)
http://localhost:8000/unsafe/500x250/smart/filters:redact_oval(pixelate)/IMAGE    # oval pixelate
http://localhost:8000/unsafe/500x250/smart/filters:redact_oval(black)/IMAGE       # oval solid black
```

---

## Metadata

imagorface exposes detected face regions through imagor's [metadata](./metadata-and-exif.md) endpoint. Each region is returned in absolute pixel coordinates relative to the output image, along with a detection score and label.

Detection only runs when the URL semantically requests it — via `smart`, `draw_detections()`, or `redact()`.

Add `/meta` right after the URL signature hash:

```
http://localhost:8000/unsafe/meta/500x250/smart/filters:draw_detections()/IMAGE
http://localhost:8000/unsafe/meta/500x250/smart/IMAGE
```

Response includes a `detected_regions` array:

```json
{
  "format": "jpeg",
  "content_type": "image/jpeg",
  "width": 500,
  "height": 250,
  "detected_regions": [
    { "left": 120, "top": 45, "right": 280, "bottom": 205, "score": 12.34, "name": "face" },
    { "left": 350, "top": 60, "right": 490, "bottom": 200, "score": 9.10,  "name": "face" }
  ]
}
```

`score` is the raw pigo detection quality (higher = more confident). `name` is `"face"` for all regions.

---

## Face Detect Cache

imagorface maintains an in-memory cache of detection results keyed by source image path, avoiding repeated pigo runs for the same source. Backed by [ristretto](https://github.com/dgraph-io/ristretto) with LRU eviction.

```dotenv
FACE_DETECTOR_CACHE_SIZE=500  # max cached entries (one per source image path). 0 = disabled (default)
FACE_DETECTOR_CACHE_TTL=1h    # cache TTL. 0 = no expiry (LRU eviction only)
```

Enable when the same source images are frequently requested at different crop sizes. Set `FACE_DETECTOR_CACHE_TTL` if source images may change at the same path. Leave disabled for highly varied or user-supplied image paths.

---

## Configuration

Configuration options specific to imagorface. See [Configuration](./configuration.md) for all imagor options.
The `-vips-detector-probe-size` option (default `400`) controls the maximum dimension of the downscaled probe image passed to any detector.

```
  -face-detector
        enable face detection for smart crop
  -face-detector-min-size int
        minimum face size in pixels on the probe image (default 20)
  -face-detector-max-size int
        maximum face size in pixels on the probe image (default 400)
  -face-detector-min-quality float
        minimum detection quality threshold; lower = more candidates, higher = fewer false positives (default 5.0)
  -face-detector-iou-threshold float
        intersection-over-union threshold for non-maxima suppression (default 0.2)
  -face-detector-cache-size int
        face detect cache size in number of entries (one per unique source image path). 0 = disabled (default)
  -face-detector-cache-ttl duration
        face detect cache TTL. 0 = no expiry (default)
```
