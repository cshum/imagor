---
description: Build variant adding ImageMagick support through libvips, enabling processing of additional formats like PSD and XCF that libvips cannot handle natively.
---

# ImageMagick Support

imagor uses [libvips](https://github.com/libvips/libvips) which is typically 4-8x [faster](https://github.com/libvips/libvips/wiki/Speed-and-memory-use) than ImageMagick with better memory efficiency and security. However, there are image formats that libvips cannot handle natively, such as PSD, XCF and other legacy formats.

imagor provides an ImageMagick-enabled variant that includes ImageMagick support through the libvips `magickload` operation. imagor only calls it when a load through libvips fails, so formats libvips reads never go through ImageMagick. The default image decodes BMP itself.

**ImageMagick is not recommended for speed, memory and security** but is capable of opening files that libvips won't support natively.

## Docker build `imagor-magick`

```bash
docker pull ghcr.io/cshum/imagor-magick
```

Usage:

```bash
docker run -p 8000:8000 ghcr.io/cshum/imagor-magick -imagor-unsafe -imagor-auto-webp
```

We recommend using the standard imagor image for most use cases.
