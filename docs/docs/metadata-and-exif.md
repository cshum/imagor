# Metadata and Exif

imagor provides metadata endpoint that extracts information such as image format, resolution and Exif metadata.
Under the hood, it tries to retrieve data just enough to extract the header, without reading and processing the whole image in memory.

To use the metadata endpoint, add `/meta` right after the URL signature hash before the image operations. Example:

```
http://localhost:8000/unsafe/meta/fit-in/50x50/raw.githubusercontent.com/cshum/imagor/master/testdata/Canon_40D.jpg
```

```jsonc
{
  "format": "jpeg",
  "content_type": "image/jpeg",
  "width": 50,
  "height": 34,
  "orientation": 1,
  "pages": 1,
  "bands": 3,
  "exif": {
    "ApertureValue": "368640/65536",
    "ColorSpace": 1,
    "ComponentsConfiguration": "Y Cb Cr -",
    "Compression": 6,
    "DateTime": "2008:07:31 10:38:11",
    "ISOSpeedRatings": 100,
    "Make": "Canon",
    "MeteringMode": 5,
    "Model": "Canon EOS 40D",
    //...
  }
}
```

Prepending `/params` to the existing endpoint returns the endpoint attributes in JSON form, useful for previewing the endpoint parameters. Example:
```bash
curl 'http://localhost:8000/params/g5bMqZvxaQK65qFPaP1qlJOTuLM=/fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png'
```

## Metadata Endpoint Details

### Basic Usage

The metadata endpoint provides comprehensive image information without processing the entire image:

- **Format detection**: Identifies image format (JPEG, PNG, WebP, etc.)
- **Dimensions**: Width and height in pixels
- **Color information**: Number of bands/channels
- **Orientation**: EXIF orientation value
- **Page count**: For multi-page formats like PDF or animated GIFs

### EXIF Data

When available, the metadata endpoint extracts EXIF data including:

- **Camera information**: Make, model, lens details
- **Shooting parameters**: ISO, aperture, shutter speed
- **Date and time**: When the photo was taken
- **GPS coordinates**: Location data (if present)
- **Technical details**: Color space, compression, etc.

### Performance Benefits

The metadata endpoint is optimized for performance:

- Only reads image headers, not full image data
- Minimal memory usage
- Fast response times
- Suitable for batch processing and image cataloging

### Params Endpoint

The `/params` endpoint is useful for debugging and development:

- Shows how imagor parses the URL
- Displays all transformation parameters
- Helps verify URL signature and structure
- Returns JSON format for easy integration
