# MozJPEG Support

By default, imagor uses libjpeg-turbo for JPEG encoding, which provides fast compression. For enhanced JPEG compression at the cost of slower encoding speed, imagor provides a MozJPEG-enabled variant that includes [MozJPEG](https://github.com/mozilla/mozjpeg) support through libvips.

MozJPEG improves JPEG compression efficiency while maintaining compatibility with existing JPEG decoders. It can reduce JPEG file sizes up to 30% compared to libjpeg-turbo while maintaining the same visual quality, but with slower encoding performance.

## Docker build `imagor-mozjpeg`

```bash
docker pull ghcr.io/cshum/imagor-mozjpeg
```

Usage:

```bash
docker run -p 8000:8000 ghcr.io/cshum/imagor-mozjpeg -imagor-unsafe -vips-mozjpeg
```

## Enabling MozJPEG

MozJPEG can be enabled using the `-vips-mozjpeg` command-line argument, or the equivalent environment variable:

```bash
VIPS_MOZJPEG=1
```

Docker Compose example:

```yaml
version: "3"
services:
  imagor:
    image: ghcr.io/cshum/imagor-mozjpeg:latest
    environment:
      PORT: 8000
      IMAGOR_UNSAFE: 1
      VIPS_MOZJPEG: 1  # Enable MozJPEG compression
    ports:
      - "8000:8000"
```

When enabled, MozJPEG will be used for JPEG output, providing better compression efficiency for JPEG images.
