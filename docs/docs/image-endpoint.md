# Image Endpoint

imagor endpoint is a series of URL parts which defines the image operations, followed by the image URI:

```
/HASH|unsafe/trim/AxB:CxD/fit-in/stretch/-Ex-F/GxH:IxJ/HALIGN/VALIGN/smart/filters:NAME(ARGS):NAME(ARGS):.../IMAGE
```

- `HASH` is the URL signature hash, or `unsafe` if unsafe mode is used
- `trim` removes surrounding space in images using top-left pixel color
- `AxB:CxD` means manually crop the image at left-top point `AxB` and right-bottom point `CxD`. Coordinates can also be provided as float values between 0 and 1 (percentage of image dimensions)
- `fit-in` means that the generated image should not be auto-cropped and otherwise just fit in an imaginary box specified by `ExF`
- `stretch` means resize the image to `ExF` without keeping its aspect ratios
- `-Ex-F` means resize the image to be `ExF` of width per height size. The minus signs mean flip horizontally and vertically
- `GxH:IxJ` add left-top padding `GxH` and right-bottom padding `IxJ`
- `HALIGN` is horizontal alignment of crop. Accepts `left`, `right` or `center`, defaults to `center`
- `VALIGN` is vertical alignment of crop. Accepts `top`, `bottom` or `middle`, defaults to `middle`
- `smart` means using smart detection of focal points
- `filters` a pipeline of image filter operations to be applied, see filters section
- `IMAGE` is the image path or URI
  - For image URI that contains `?` character, this will interfere the URL query and should be encoded with [`encodeURIComponent`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent) or equivalent
  - Base64 URLs: Use `b64:` prefix to encode image URLs with special characters as [base64url](https://developer.mozilla.org/en-US/docs/Glossary/Base64#url_and_filename_safe_base64). This encoding is  more robust if you have special characters in your image URL, and can fix encoding/signing issues in your setup.

## URL Components Details

### HASH or unsafe

- `HASH` is the URL signature hash for security
- `unsafe` disables URL signature verification (testing only)

:::danger Security Warning
Never use `unsafe` in production environments. Always use URL signing with `IMAGOR_SECRET`.
:::

### trim

Removes surrounding space in images using top-left pixel color.

```
/unsafe/trim/image.jpg
```

### Manual Crop (AxB:CxD)

Manually crop the image at left-top point `AxB` and right-bottom point `CxD`.

```
/unsafe/10x20:300x500/image.jpg
```

Coordinates can also be provided as float values between 0 and 1 (percentage of image dimensions):

```
/unsafe/0.1x0.2:0.89x0.72/image.jpg
```

### fit-in

The generated image should not be auto-cropped and otherwise just fit in an imaginary box specified by `ExF`.

```
/unsafe/fit-in/300x200/image.jpg
```

### stretch

Resize the image to `ExF` without keeping its aspect ratios.

```
/unsafe/stretch/300x200/image.jpg
```

### Resize (-Ex-F)

Resize the image to be `ExF` of width per height size. The minus signs mean flip horizontally and vertically.

Examples:
- `300x200` - Resize to 300x200
- `-300x200` - Resize to 300x200 and flip horizontally
- `300x-200` - Resize to 300x200 and flip vertically
- `-300x-200` - Resize to 300x200 and flip both ways

```
/unsafe/300x200/image.jpg
/unsafe/-300x200/image.jpg
```

### Padding (GxH:IxJ)

Add left-top padding `GxH` and right-bottom padding `IxJ`.

```
/unsafe/10x20:30x40/300x200/image.jpg
```

### Horizontal Alignment (HALIGN)

Horizontal alignment of crop:
- `left` - Align to left
- `right` - Align to right  
- `center` - Center alignment (default)

```
/unsafe/300x200/left/image.jpg
/unsafe/300x200/right/image.jpg
/unsafe/300x200/center/image.jpg
```

### Vertical Alignment (VALIGN)

Vertical alignment of crop:
- `top` - Align to top
- `bottom` - Align to bottom
- `middle` - Middle alignment (default)

```
/unsafe/300x200/top/image.jpg
/unsafe/300x200/bottom/image.jpg
/unsafe/300x200/middle/image.jpg
```

### smart

Enable smart detection of focal points for automatic cropping.

```
/unsafe/300x200/smart/image.jpg
```

### filters

A pipeline of image filter operations to be applied. See [Filters](filters.md) for detailed documentation.

```
/unsafe/300x200/filters:blur(5):quality(80)/image.jpg
```

### IMAGE

The image path or URI:

- For image URI that contains `?` character, encode with [`encodeURIComponent`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent)
- **Base64 URLs**: Use `b64:` prefix to encode image URLs with special characters as [base64url](https://developer.mozilla.org/en-US/docs/Glossary/Base64#url_and_filename_safe_base64)

Examples:
```
/unsafe/300x200/https://example.com/image.jpg
/unsafe/300x200/b64:aHR0cHM6Ly9leGFtcGxlLmNvbS9pbWFnZS5qcGc/
```
