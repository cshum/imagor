# Filters

Filters `/filters:NAME(ARGS):NAME(ARGS):.../` is a pipeline of image operations that will be sequentially applied to the image. Examples:

```
/filters:fill(white):format(jpeg)/
/filters:hue(290):saturation(100):fill(yellow):format(jpeg):quality(80)/
/filters:fill(white):watermark(raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png,repeat,bottom,10):format(jpeg)/
```

imagor supports the following filters:

- `background_color(color)` sets the background color of a transparent image
  - `color` the color name or hexadecimal rgb expression without the "#" character
- `blur(sigma)` applies gaussian blur to the image
- `brightness(amount)` increases or decreases the image brightness
  - `amount` -100 to 100, the amount in % to increase or decrease the image brightness
- `contrast(amount)` increases or decreases the image contrast
  - `amount` -100 to 100, the amount in % to increase or decrease the image contrast
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
- `label(text, x, y, size, color[, alpha[, font]])` adds a text label to the image. It can be positioned inside the image with the alignment specified, color and transparency support:
  - `text` text label, also support url encoded text.
  - `x` horizontal position that the text label will be in:
    - Positive number indicate position from the left, negative number from the right.
    - Number followed by a `p` e.g. 20p means calculating the value from the image width as percentage
    - `left`,`right`,`center` align left, right or centered respectively
  - `y` vertical position that the text label will be in:
    - Positive number indicate position from the top, negative number from the bottom.
    - Number followed by a `p` e.g. 20p means calculating the value from the image height as percentage
    - `top`,`bottom`,`center` vertical align top, bottom or centered respectively
  - `size` - text label font size
  - `color` - color name or hexadecimal rgb expression without the "#" character
  - `alpha` - text label transparency, a number between 0 (fully opaque) and 100 (fully transparent).
  - `font` - text label font type
- `max_bytes(amount)` automatically degrades the quality of the image until the image is under the specified `amount` of bytes
- `max_frames(n)` limit maximum number of animation frames `n` to be loaded
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
- `sharpen(sigma)` sharpens the image
- `strip_exif()` removes Exif metadata from the resulting image
- `strip_icc()` removes ICC profile information from the resulting image
- `strip_metadata()` removes all metadata from the resulting image
- `upscale()` upscale the image if `fit-in` is used
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
- `preview()` skips the result storage even if result storage is enabled. Useful for conditional caching
- `raw()` response with a raw unprocessed and unchecked source image. Image still loads from loader and storage but skips the result storage
