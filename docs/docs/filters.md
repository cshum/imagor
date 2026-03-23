# Filters

Filters `/filters:NAME(ARGS):NAME(ARGS):.../` is a pipeline of image operations that will be sequentially applied to the image. Examples:

```
/filters:fill(white):format(jpeg)/
/filters:hue(290):saturation(100):fill(yellow):format(jpeg):quality(80)/
/filters:fill(white):watermark(raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png,repeat,bottom,10):format(jpeg)/
```

## Processing Filters

imagor supports the following filters:

- `background_color(color)` sets the background color of a transparent image
  - `color` the color name or hexadecimal rgb expression without the "#" character
- `blur(sigma)` applies gaussian blur to the image
- `brightness(amount)` increases or decreases the image brightness
  - `amount` -100 to 100, the amount in % to increase or decrease the image brightness
- `contrast(amount)` increases or decreases the image contrast
  - `amount` -100 to 100, the amount in % to increase or decrease the image contrast
- `crop(left,top,width,height)` crops the image after resizing
  - Absolute pixels: `crop(10,20,200,150)` - crop 200x150 box starting at (10,20)
  - Relative (0.0-1.0): `crop(0.1,0.1,0.8,0.8)` - crop using percentages
- `draw_detections()` draws color-coded bounding boxes on detected regions. Each class name is automatically assigned a distinct colour via hash-based palette. For use with detection plugins such as [imagorface](https://github.com/cshum/imagorface). No-op when no Detector is configured.
- `pixelate(block_size)` applies a pixelate effect to the whole image by downscaling to 1/`block_size` then upscaling back with nearest-neighbour interpolation
  - `block_size` pixel block size in pixels, defaults to 10
- `fill(color)` fill the missing area or transparent image with the specified color:
  - `color` - color name or hexadecimal rgb expression without the "#" character
    - If color is "blur" - missing parts are filled with blurred original image
    - If color is "auto" - the top left image pixel will be chosen as the filling color
    - If color is "none" - the filling would become fully transparent
- `focal(AxB:CxD)` or `focal(X,Y)` adds a focal region or focal point for custom transformations:
  - Coordinated by a region of left-top point `AxB` and right-bottom point `CxD`, or a point `X,Y`.
  - Also accepts float values between 0 and 1 that represents percentage of image dimensions.
- `format(format)` specifies the output format of the image
  - `format` accepts jpeg, png, gif, webp, avif, jxl, tiff, jp2
- `grayscale()` changes the image to grayscale
- `hue(angle)` increases or decreases the image hue
  - `angle` the angle in degree to increase or decrease the hue rotation
- `image(imagorpath, x, y[, alpha[, blend_mode]])` composites a processed image onto the current image with full imagor transformation support, enabling recursive image composition:
  - `imagorpath` - an imagor path with transformations e.g. `/200x200/filters:grayscale()/photo.jpg`
    - The nested path supports all imagor operations: resizing, cropping, filters, etc.
    - Enables recursive nesting - images can load other processed images
    - Use `full` (or `f`) in the `WxH` dimension segment to inherit the parent image's width or height. `full` means the full parent dimension; `full-NNN` means the parent dimension minus NNN pixels. Examples:
      - `fullxfull/overlay.png` (or `fxf`) — overlay fills the parent canvas exactly
      - `fit-in/full-20xfull-20/overlay.png` (or `fit-in/f-20xf-20`) — overlay fits within the parent canvas with a 20px inset on each side
      - `fullx200/banner.png` — overlay inherits parent width, fixed 200px height
  - `x` - horizontal position (defaults to 0 if not specified):
    - Positive number indicates position from the left, negative from the right
    - Number followed by `p` e.g. `20p` means percentage of image width
    - `left` or `l`, `right` or `r`, `center` for alignment, optionally with pixel offset e.g. `left-20`, `r-10`
    - `repeat` to tile horizontally
    - Float between 0-1 represents percentage e.g. `0.5` for center
  - `y` - vertical position (defaults to 0 if not specified):
    - Positive number indicates position from the top, negative from the bottom
    - Number followed by `p` e.g. `20p` means percentage of image height
    - `top` or `t`, `bottom` or `b`, `center` for alignment, optionally with pixel offset e.g. `top-10`, `b-20`
    - `repeat` to tile vertically
    - Float between 0-1 represents percentage e.g. `0.5` for center
  - `alpha` - transparency level, 0 (fully opaque) to 100 (fully transparent)
  - `blend_mode` - compositing blend mode, defaults to `normal`. Supported modes: `normal`, `multiply`, `screen`, `overlay`, `darken`, `lighten`, `color-dodge`, `color-burn`, `hard-light`, `soft-light`, `difference`, `exclusion`, `add`, `mask`, `mask-out`
- `max_bytes(amount)` automatically degrades the quality of the image until the image is under the specified `amount` of bytes
- `max_frames(n)` limit maximum number of animation frames `n` to be loaded
- `no_upscale()` prevents the image from being upscaled beyond its original dimensions
- `orient(angle)` rotates the image before resizing and cropping, according to the angle value
  - `angle` accepts 0, 90, 180, 270
- `page(num)` specify page number for PDF, or frame number for animated image, starts from 1
- `dpi(num)` specify the dpi to render at for PDF and SVG
- `proportion(percentage)` scales image to the proportion percentage of the image dimension
- `quality(amount)` changes the overall quality of the image, does nothing for png
  - `amount` 0 to 100, the quality level in %
- `rgb(r,g,b)` amount of color in each of the rgb channels in %. Can range from -100 to 100
- `rotate(angle)` rotates the given image according to the angle value
  - `angle` accepts 0, 90, 180, 270
- `round_corner(rx [, ry [, color]])` adds rounded corners to the image with the specified color as background
  - `rx`, `ry` amount of pixel to use as radius. ry = rx if ry is not provided
  - `color` the color name or hexadecimal rgb expression without the "#" character
- `saturation(amount)` increases or decreases the image saturation
  - `amount` -100 to 100, the amount in % to increase or decrease the image saturation
- `redact([mode[, strength]])` obscures all detected regions for privacy/anonymisation (e.g. GDPR face blurring, legal document redaction). Requires a detection plugin such as [imagorface](https://github.com/cshum/imagorface). No-op when no Detector is configured or no regions are detected. Skips animated images.
  - `mode` — `blur` (default), `pixelate`, or any color name/hex for solid fill (e.g. `black`, `white`, `ff0000`)
  - `strength` — blur sigma (default 15) or pixelate block size in pixels (default 10). Not used for solid color mode.
  - Examples: `redact()`, `redact(blur,20)`, `redact(pixelate)`, `redact(pixelate,15)`, `redact(black)`, `redact(white)`, `redact(ff0000)`
- `redact_oval([mode[, strength]])` identical to `redact` but applies an elliptical mask to each region, producing a rounded/oval redaction shape. This is the most natural shape for face anonymisation as it closely follows the contour of a face. Same arguments and defaults as `redact`.
  - Examples: `redact_oval()`, `redact_oval(blur,20)`, `redact_oval(pixelate)`, `redact_oval(pixelate,15)`, `redact_oval(black)`, `redact_oval(white)`, `redact_oval(ff0000)`
- `sharpen(sigma)` sharpens the image
- `strip_exif()` removes Exif metadata from the resulting image
- `strip_icc()` removes ICC profile information from the resulting image. The image is first converted to sRGB color space to preserve correct colors before the profile is removed.
- `text(text, x, y[, font[, color[, alpha[, blend_mode[, width[, align[, justify[, wrap[, spacing[, dpi]]]]]]]]]])` renders a text overlay onto the image with full multi-line and Pango font support:
  - `text` the text to render. Supports URL query-encoding and `b64:` prefix for [base64url](https://developer.mozilla.org/en-US/docs/Glossary/Base64#url_and_filename_safe_base64) encoding to safely pass arbitrary unicode or multi-word strings.
  - `font` Pango font description with hyphens as space separators, e.g. `sans-bold-24` for `sans bold 24`, `monospace-18` for `monospace 18`. Font size is in points; at the default 72 DPI, 1pt = 1px.
  - `x` horizontal position:
    - Positive number indicates position from the left, negative from the right
    - Number followed by `p` e.g. `20p` means percentage of image width
    - `left` or `l`, `right` or `r`, `center` for alignment, optionally with pixel offset e.g. `left-20`, `r-10`
    - Float between 0-1 represents percentage e.g. `0.5` for center
  - `y` vertical position:
    - Positive number indicates position from the top, negative from the bottom
    - Number followed by `p` e.g. `20p` means percentage of image height
    - `top` or `t`, `bottom` or `b`, `center` for alignment, optionally with pixel offset e.g. `top-10`, `b-20`
    - Float between 0-1 represents percentage e.g. `0.5` for center
  - `color` color name or hexadecimal rgb expression without the `#` character, defaults to black
  - `alpha` transparency, 0 (fully opaque) to 100 (fully transparent), defaults to 0
  - `blend_mode` compositing blend mode, defaults to `normal`. Supported modes: `normal`, `multiply`, `screen`, `overlay`, `darken`, `lighten`, `color-dodge`, `color-burn`, `hard-light`, `soft-light`, `difference`, `exclusion`, `add`, `mask`, `mask-out`
  - `width` wrap width — text wraps when a line exceeds this width. Supports the same conventions as image dimensions:
    - Plain integer for pixel count, e.g. `300`
    - Number followed by `p` for percentage of canvas width, e.g. `80p`
    - Float between 0-1 as fraction of canvas width, e.g. `0.75`
    - `f` or `full` for full canvas width; `f-N` / `full-N` for canvas width minus N pixels
    - `0` or omitted means unconstrained (Pango wraps only on explicit newlines)
  - `align` horizontal alignment of lines within the text box: `low` (default, left), `centre` / `center`, `high` / `right`
  - `justify` justify text: `true` or `1`
  - `wrap` line wrapping mode: `word` (default), `char`, `wordchar`, `none`
  - `spacing` additional line spacing in pixels
  - `dpi` render DPI, defaults to 72 (where 1pt = 1px)
- `to_colorspace(profile)` converts the image to the specified ICC color profile
  - `profile` the target color profile, defaults to `srgb` if not specified. Common values: `srgb`, `p3`, `cmyk`
- `upscale()` enables upscaling for `fit-in` and `adaptive-fit-in` modes
- `watermark(image, x, y, alpha [, w_ratio [, h_ratio]])` adds a watermark to the image. It can be positioned inside the image with the alpha channel specified and optionally resized based on the image size by specifying the ratio
  - `image` watermark image URI, using the same image loader configured for imagor.
    Use `b64:` prefix to encode image URLs with special characters as [base64url](https://developer.mozilla.org/en-US/docs/Glossary/Base64#url_and_filename_safe_base64).
  - `x` horizontal position that the watermark will be in:
    - Positive number indicate position from the left, negative number from the right.
    - Number followed by a `p` e.g. 20p means calculating the value from the image width as percentage
    - `left`,`right`,`center` positioned left, right or centered respectively
    - `repeat` the watermark will be repeated horizontally
  - `y` vertical position that the watermark will be in:
    - Positive number indicate position from the top, negative number from the bottom.
    - Number followed by a `p` e.g. 20p means calculating the value from the image height as percentage
    - `top`,`bottom`,`center` positioned top, bottom or centered respectively
    - `repeat` the watermark will be repeated vertically
  - `alpha` watermark image transparency, a number between 0 (fully opaque) and 100 (fully transparent).
  - `w_ratio` percentage of the width of the image the watermark should fit-in
  - `h_ratio` percentage of the height of the image the watermark should fit-in

## Utility Filters

These filters do not manipulate images but provide useful utilities to the imagor pipeline:

- `attachment(filename)` returns attachment in the `Content-Disposition` header, and the browser will open a "Save as" dialog with `filename`. When `filename` not specified, imagor will get the filename from the image source
- `expire(timestamp)` adds expiration time to the content. `timestamp` is the unix milliseconds timestamp, e.g. if content is valid for 30s then timestamp would be `Date.now() + 30*1000` in JavaScript.
- `preview()` skips the result storage even if result storage is enabled, and opts the request into the [in-memory cache](./in-memory-cache) when configured. Useful for preview contexts where the same source image is served at multiple transformations.
- `raw()` response with a raw unprocessed and unchecked source image. Image still loads from loader and storage but skips the result storage
