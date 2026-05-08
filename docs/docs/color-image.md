---
description: Generate solid color or transparent images on-the-fly using color:<color> as the image path, without loading from any source.
---

# Color Image

Use `color:<color>` as the image path to generate a solid color or transparent image on-the-fly, without loading from any source. This is useful for creating background canvases, placeholder images, or base layers for further composition.

```
/unsafe/{width}x{height}/color:{color}
```

This behaves like any other imagor source image once generated, so you can still resize it, crop it, add rounded corners, composite it with `image()`, or change the output format.

Supported color values:

| Format | Example | Description |
|--------|---------|-------------|
| Named color | `color:red`, `color:blue` | CSS named colors |
| Transparent | `color:none` | Fully transparent (RGBA) |
| 3-char hex | `color:fff` | Short hex (expanded to 6-char) |
| 6-char hex | `color:ff0000` | Standard RGB hex |
| 8-char hex | `color:ff000080` | RGBA hex with alpha channel |

Examples:

```
http://localhost:8000/unsafe/200x200/color:red
http://localhost:8000/unsafe/100x100/filters:format(png)/color:none
http://localhost:8000/unsafe/300x300/filters:round_corner(20):format(png)/color:ff6600
http://localhost:8000/unsafe/50x50/filters:format(png)/color:ff000080
```

- Transparent outputs such as `color:none` usually make the most sense with a format that supports alpha, such as PNG.
- All existing filters and transformations work with color images.
- When no dimensions are specified, imagor defaults to a 1×1 image.
- When only width or height is specified, the other dimension defaults to the same value.
