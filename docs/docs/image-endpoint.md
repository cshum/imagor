# Image Endpoint

The imagor image endpoint is a series of URL parts that define image operations, followed by the image URI:

```
/HASH|unsafe/trim/AxB:CxD/(adaptive-)(full-)fit-in/stretch/-Ex-F/GxH:IxJ/HALIGN/VALIGN/smart/filters:NAME(ARGS):NAME(ARGS):.../IMAGE
```

- `HASH` — URL signature hash, or `unsafe` if unsafe mode is used
- `IMAGE` — the image path or URI

All other parts are optional and can be combined. They are described in detail below.

---

## Resize & Crop

```
/unsafe/WxH/IMAGE
```

Resizes the image to the given width `W` and height `H`, auto-cropping the excess to fill the box. The crop is centered by default.

- Set `W` or `H` to `0` to auto-scale that dimension while preserving aspect ratio.

```
/unsafe/400x400/IMAGE    → resize to 400×400, center-crop to fill
/unsafe/400x0/IMAGE      → resize to width 400, height proportional
/unsafe/0x400/IMAGE      → resize to height 400, width proportional
```

<img src="/img/endpoint/resize-crop.jpg" width="33%" />

### Alignment

Control where the auto-crop is anchored using `HALIGN` and `VALIGN`:

- `HALIGN` — `left`, `center` (default), or `right`
- `VALIGN` — `top`, `middle` (default), or `bottom`

<table>
  <tr>
    <th width="33%"><code>left/top</code></th>
    <th width="33%"><code>center/middle</code></th>
    <th width="33%"><code>right/bottom</code></th>
  </tr>
  <tr>
    <td><img src="/img/endpoint/align-left-top.jpg" /></td>
    <td><img src="/img/endpoint/align-center.jpg" /></td>
    <td><img src="/img/endpoint/align-right-bottom.jpg" /></td>
  </tr>
</table>

```
/unsafe/400x400/left/top/IMAGE
/unsafe/400x400/center/middle/IMAGE
/unsafe/400x400/right/bottom/IMAGE
```

### Smart Crop

`smart` uses focal point detection to find the most visually important part of the image for cropping:

```
/unsafe/400x400/smart/IMAGE
```

<img src="/img/endpoint/smart-crop.jpg" width="33%" />

---

## Fit-in

```
/unsafe/fit-in/WxH/IMAGE
```

Resizes the image to fit **within** the given dimensions without cropping. The result may be letterboxed; use the [`fill()`](./filters#fillcolor) filter to add a background:

<table>
  <tr>
    <th width="33%">No fill</th>
    <th width="33%"><code>fill(eeeeee)</code></th>
    <th width="33%"><code>fill(blur)</code></th>
  </tr>
  <tr>
    <td><img src="/img/endpoint/fit-in.jpg" /></td>
    <td><img src="/img/endpoint/fit-in-fill-grey.jpg" /></td>
    <td><img src="/img/endpoint/fit-in-fill-blur.jpg" /></td>
  </tr>
</table>

```
/unsafe/fit-in/400x400/IMAGE
/unsafe/fit-in/400x400/filters:fill(eeeeee)/IMAGE
/unsafe/fit-in/400x400/filters:fill(blur)/IMAGE
```

### Full Fit-in

```
/unsafe/full-fit-in/WxH/IMAGE
```

Like `fit-in` but uses the **larger** dimension for fitting — the image always fills at least one edge of the box.

### Adaptive Fit-in

```
/unsafe/adaptive-fit-in/WxH/IMAGE
```

Like `fit-in` but automatically swaps width and height if it produces better image coverage.

---

## Stretch

```
/unsafe/stretch/WxH/IMAGE
```

Resizes the image to exactly `W×H` without preserving the aspect ratio. The image is distorted to fill the box.

<img src="/img/endpoint/stretch.jpg" width="33%" />

```
/unsafe/stretch/400x400/IMAGE
```

---

## Flip

Prefix width or height with `-` to flip the image:

<table>
  <tr>
    <th width="50%">Flip horizontal (<code>-400x400</code>)</th>
    <th width="50%">Flip vertical (<code>400x-400</code>)</th>
  </tr>
  <tr>
    <td><img src="/img/endpoint/flip-h.jpg" /></td>
    <td><img src="/img/endpoint/flip-v.jpg" /></td>
  </tr>
</table>

```
/unsafe/-400x400/IMAGE    → flip horizontally
/unsafe/400x-400/IMAGE    → flip vertically
/unsafe/-400x-400/IMAGE   → flip both
```

---

## Manual Crop

```
/unsafe/AxB:CxD/IMAGE
```

Manually crops the image at left-top point `AxB` to right-bottom point `CxD` **before** resizing.

- Coordinates can be integer pixels or float values between `0.0`–`1.0` (percentage of image dimensions).

<img src="/img/endpoint/manual-crop.jpg" width="33%" />

```
/unsafe/100x50:1800x1200/400x400/IMAGE   → crop region then resize to 400×400
/unsafe/0.1x0.1:0.9x0.9/IMAGE           → crop inner 80% using relative coordinates
```

---

## Padding

```
/unsafe/GxH:IxJ/WxH/IMAGE
```

Adds padding around the image after resizing:

- `GxH` — left and top padding in pixels
- `IxJ` — right and bottom padding in pixels

```
/unsafe/20x20:20x20/400x400/IMAGE    → 20px padding on all sides
```

---

## Trim

```
/unsafe/trim/IMAGE
/unsafe/trim:top-right/IMAGE
```

Removes surrounding border/whitespace based on the color of the corner pixel:

- `trim` uses the top-left pixel color by default
- `trim:top-right` uses the top-right pixel color instead

```
/unsafe/trim/400x400/IMAGE
```

---

## Filters

```
/unsafe/filters:NAME(ARGS):NAME(ARGS):.../IMAGE
```

Filters are a pipeline of image operations applied after resizing. Multiple filters can be chained:

```
/unsafe/400x400/filters:grayscale():quality(80)/IMAGE
/unsafe/fit-in/400x400/filters:fill(eeeeee):format(jpeg)/IMAGE
```

See [Filters](./filters) for the full list of available filters.

---

## Image URI

The `IMAGE` path at the end supports several special forms:

- **Plain URL** — `raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png`
- **URL with query string** — encode with [`encodeURIComponent`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent) if the URL contains `?`
- **Base64 URL** — use `b64:` prefix with [base64url](https://developer.mozilla.org/en-US/docs/Glossary/Base64#url_and_filename_safe_base64) encoding for URLs with special characters:
  ```
  /unsafe/400x400/b64:aHR0cHM6Ly9leGFtcGxlLmNvbS9pbWFnZS5qcGc=/
  ```
- **Color image** — use `color:<color>` to generate a solid color image without a source. See [Color Image](./color-image).
  ```
  /unsafe/400x400/color:ff8800
  /unsafe/400x400/color:transparent
  ```
