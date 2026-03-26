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

## Metadata Options

These optional filters add computed values to the metadata response. Because all of them require downloading and fully decoding the image, they are **noticeably slower** than a standard metadata request.

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

Computes the average color of the image as an `average_color` object with `r`, `g`, `b`, and `a` integer fields (0–255).

Fully transparent pixels (`a == 0`) are excluded from the average. `r`, `g`, and `b` reflect the mean color of visible pixels only. `a` is the mean alpha of those same visible pixels. For images without an alpha channel `a` is always `255`.

```
http://localhost:8000/unsafe/meta/filters:avgcolor()/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

```jsonc
{
  // ...
  "average_color": { "r": 99, "g": 172, "b": 229, "a": 226 }
}
```
