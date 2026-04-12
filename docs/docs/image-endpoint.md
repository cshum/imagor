# Image Endpoint

imagor endpoint is a series of URL parts which defines the image operations, followed by the image URI:

```
/HASH|unsafe/trim/AxB:CxD/(adaptive-)(full-)fit-in/stretch/-Ex-F/GxH:IxJ/HALIGN/VALIGN/smart/filters:NAME(ARGS):NAME(ARGS):.../IMAGE
```

- [`HASH`](./security.mdx#url-signature) is the URL signature hash, or `unsafe` if unsafe mode is used
- [`trim`](#trim) removes surrounding space in images using top-left pixel color
- [`AxB:CxD`](#manual-crop) means manually crop the image at left-top point `AxB` and right-bottom point `CxD`. Coordinates can also be provided as float values between 0 and 1 (percentage of image dimensions)
- [`fit-in`](#fit-in) means that the generated image should not be auto-cropped and otherwise just fit in an imaginary box specified by `WxH`. If [`full-fit-in`](#full-fit-in) is specified, then the largest size is used for cropping. If [`adaptive-fit-in`](#adaptive-fit-in) is specified, it inverts requested width and height if it would get a better image definition
- [`stretch`](#stretch) means resize the image to `WxH` without keeping its aspect ratio
- [`-Ex-F`](#resize--crop) means resize the image to be `ExF` of width per height size. The minus signs mean [flip](#flip) horizontally and vertically
- [`GxH:IxJ`](#padding) add left-top padding `GxH` and right-bottom padding `IxJ`, placed **after** the [resize](#resize--crop) dimensions in the URL
- [`HALIGN`](#alignment) is horizontal alignment of crop. Accepts `left`, `right` or `center`, defaults to `center`
- [`VALIGN`](#alignment) is vertical alignment of crop. Accepts `top`, `bottom` or `middle`, defaults to `middle`
- [`smart`](#smart-crop) means using smart detection of focal points
- [`filters`](./filters.md) a pipeline of image filter operations to be applied, see [Filters](./filters.md) section
- [`IMAGE`](#image-uri) is the image path or URI
  - For image URI that contains `?` character, this will interfere the URL query and should be encoded with [`encodeURIComponent`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent) or equivalent
  - Base64 URLs: Use `b64:` prefix to encode image URLs with special characters as [base64url](https://developer.mozilla.org/en-US/docs/Glossary/Base64#url_and_filename_safe_base64)
  - Color image: Use `color:<color>` to generate a solid color or transparent image without loading from a source. See [Color Image](./color-image.md) section.

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

<table width="33%">
  <tr><th><code>400x400/IMAGE</code></th></tr>
  <tr><td><img src="/img/endpoint/resize-crop.jpg" /></td></tr>
</table>

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

<table width="33%">
  <tr><th><code>400x400/smart/IMAGE</code></th></tr>
  <tr><td><img src="/img/endpoint/smart-crop.jpg" /></td></tr>
</table>

---

## Fit-in

```
/unsafe/fit-in/WxH/IMAGE
```

Resizes the image to fit **within** the given dimensions without cropping. The result may be letterboxed; use the [`fill()`](./filters.md#fillcolor) filter to add a background.

<table width="33%">
  <tr><th><code>fit-in/400x400/IMAGE</code></th></tr>
  <tr><td><img src="/img/endpoint/fit-in.jpg" /></td></tr>
</table>

```
/unsafe/fit-in/400x400/filters:fill(red)/IMAGE
/unsafe/fit-in/400x400/filters:fill(blur)/IMAGE
/unsafe/fit-in/400x400/filters:fill(white)/IMAGE
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

<table width="33%">
  <tr><th><code>stretch/400x400/IMAGE</code></th></tr>
  <tr><td><img src="/img/endpoint/stretch.jpg" /></td></tr>
</table>

```
/unsafe/stretch/400x400/IMAGE
```

---

## Flip

Prefix width or height with `-` to flip the image:

<table>
  <tr>
    <th width="50%"><code>-400x400/IMAGE</code></th>
    <th width="50%"><code>400x-400/IMAGE</code></th>
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

<table width="33%">
  <tr><th><code>100x50:1800x1200/400x400/IMAGE</code></th></tr>
  <tr><td><img src="/img/endpoint/manual-crop.jpg" /></td></tr>
</table>

```
/unsafe/100x50:1800x1200/400x400/IMAGE   → crop region then resize to 400×400
/unsafe/0.1x0.1:0.9x0.9/IMAGE           → crop inner 80% using relative coordinates
```

---

## Padding

```
/unsafe/fit-in/WxH/GxH:IxJ/IMAGE
```

Adds padding around the image after resizing, where:

- `GxH` — left and top padding in pixels
- `IxJ` — right and bottom padding in pixels

Combined with `fit-in` and [`fill()`](./filters.md#fillcolor), padding adds a colored border around the transparent or letterboxed content:

```
/unsafe/fit-in/360x360/20x20:20x20/filters:fill(yellow)/IMAGE    → 20px yellow padding on all sides
```

<table width="33%">
  <tr><th><code>fit-in/360x360/20x20:20x20/filters:fill(yellow)/IMAGE</code></th></tr>
  <tr><td><img src="/img/endpoint/padding.jpg" /></td></tr>
</table>

---

## Trim

```
/unsafe/trim/IMAGE
/unsafe/trim:bottom-right/IMAGE
/unsafe/trim:100/IMAGE
/unsafe/trim:bottom-right:100/IMAGE
```

Removes surrounding border/whitespace by detecting the background color from a corner pixel:

- `trim` — uses the top-left corner pixel as the background color reference (default)
- `trim:bottom-right` — uses the bottom-right corner pixel instead
- `:TOLERANCE` — optional integer controlling how much color variation is still considered background

Trim is applied before any resize, so it does not affect the final output dimensions:

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
/unsafe/fit-in/400x400/filters:fill(blur):format(jpeg)/IMAGE
```

See [Filters](./filters.md) for the full list of available filters.

---

## Image URI

The `IMAGE` path at the end supports several special forms:

- **Plain URL** — `raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png`
- **URL with query string** — encode with [`encodeURIComponent`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent) if the URL contains `?`
- **Base64 URL** — use `b64:` prefix with [base64url](https://developer.mozilla.org/en-US/docs/Glossary/Base64#url_and_filename_safe_base64) encoding for URLs with special characters:
  ```
  /unsafe/400x400/b64:aHR0cHM6Ly9leGFtcGxlLmNvbS9pbWFnZS5qcGc=/
  ```
- **Color image** — use `color:<color>` to generate a solid color image without a source. See [Color Image](./color-image.md).
  ```
  /unsafe/400x400/color:ff8800
  /unsafe/400x400/color:transparent
  ```
