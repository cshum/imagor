---
description: Extract image format, resolution and Exif metadata via the /meta endpoint, and compute BlurHash, ThumbHash and average color.
keywords:
  - imagor metadata
  - imagor exif
  - imagor meta endpoint
  - imagor blurhash
  - imagor thumbhash
---

# Metadata and Exif

## Metadata Endpoint

imagor provides a metadata endpoint that extracts information such as image format, resolution and Exif metadata.
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

### Filter Report

`filters` reports what the processor did with each filter in the URL, so a client can tell a URL whose filters applied from one where they were silently skipped. Omitted when there is nothing to report.

```
http://localhost:8000/unsafe/meta/200x200/filters:blur(5):rotate()/raw.githubusercontent.com/cshum/imagor/master/testdata/Canon_40D.jpg
```

```jsonc
{
  // ...
  "filters": [
    { "name": "blur", "processed": true },
    { "name": "rotate", "processed": false }
  ]
}
```

- **`processed: true`** — the filter applied.
- **`processed: false`** — it was recognised but left the image unchanged. `blur()` without a sigma is one such case.
- **Absent** — the filter did not run, and there is no entry to say why. Either this processor does not handle the name (a later processor in the chain may, as imagorvideo does for its own filters), or the deployment disabled it, or `vips-max-filter-ops` dropped it. Compare against the filters in the URL to spot a typo; the debug log names which.

A filter can repeat, so there is one entry per occurrence, in URL order, and they can disagree.

## Params Endpoint

Prepending `/params` to the existing endpoint returns the endpoint attributes in JSON form, useful for previewing the endpoint parameters. Example:
```bash
curl 'http://localhost:8000/params/g5bMqZvxaQK65qFPaP1qlJOTuLM=/fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png'
```

## Metadata Filters

These filters add computed values to the metadata response. They require the full image to be downloaded and decoded.

### `blurhash(x,y)`

Computes a [BlurHash](https://blurha.sh) string for the image. `x` and `y` are the horizontal and vertical component counts (between 1 and 9). Higher values produce more detail at the cost of a longer hash string.

```
http://localhost:8000/unsafe/meta/filters:blurhash(4,3)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

```jsonc
{
  // ...
  "blurhash": "LGF5]+Yk^6#M@-5c,1J5@[or[Q6."
}
```

### `thumbhash()`

Computes a [ThumbHash](https://evanw.github.io/thumbhash/) string for the image, returned as a base64-encoded string. ThumbHash produces better color reproduction and supports transparency, and requires no configuration parameters.

```
http://localhost:8000/unsafe/meta/filters:thumbhash()/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

```jsonc
{
  // ...
  "thumbhash": "3OcRJYB4d3h/iIeHeEh3eIhw+j3A"
}
```

### `avgcolor()`

Computes the average color of the image as an `average_color` object with `r`, `g`, and `b` integer fields (0–255).

For images with an alpha channel, transparent pixels will be filled with black. If you'd like to control the fill color, you can use `avgcolor()` in conjunction with the `background_color()` filter.

```
http://localhost:8000/unsafe/meta/filters:avgcolor()/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

```jsonc
{
  // ...
  "average_color": { "r": 99, "g": 172, "b": 229 }
}
```
