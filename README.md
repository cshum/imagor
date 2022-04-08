# Imagor

[![Build Status](https://github.com/cshum/imagor/workflows/build/badge.svg)](https://github.com/cshum/imagor/actions)
[![Docker](https://img.shields.io/badge/docker-shumc/imagor-blue.svg)](https://hub.docker.com/r/shumc/imagor/)

Imagor is a fast, Docker-ready image processing server written in Go.

Imagor uses one of the most efficient image processing library
[libvips](https://github.com/libvips/libvips). It is typically 4-8x [faster](https://github.com/libvips/libvips/wiki/Speed-and-memory-use) than using the quickest ImageMagick and GraphicsMagick settings.

Imagor is a Go application that is highly optimized for concurrent requests. It is ready to be installed and used in any Unix environment, and ready to be containerized using Docker.

Imagor adopts the [Thumbor](https://thumbor.readthedocs.io/en/latest/usage.html#image-endpoint) URL syntax and covers most of the web image processing use cases. If these fit your requirements, Imagor would be a lightweight, high performance drop-in replacement.

### Quick Start

```bash
docker run -p 8000:8000 shumc/imagor -imagor-unsafe
```

Original images:
```
https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif
https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png
```
<img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png" height="100" />

Try out the following image URLs:
```
http://localhost:8000/unsafe/fit-in/200x200/filters:fill(white):format(jpeg)/https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
http://localhost:8000/unsafe/200x200/top/filters:fill(white):format(jpeg)/https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
http://localhost:8000/unsafe/fit-in/-180x180/10x10/filters:hue(290):saturation(100):fill(yellow):format(jpeg):quality(80)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
http://localhost:8000/unsafe/30x40:100x150/filters:fill(cyan)/raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif
http://localhost:8000/unsafe/fit-in/200x150/filters:fill(yellow):watermark(raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png,repeat,bottom,0,40,40)/raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif
```
<img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo1.jpg" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo2.jpg" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo4.jpg" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo3.gif" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo5.gif" height="100" />  


### Imagor Endpoint

Imagor endpoint is a series of URL parts which defines the image operations, followed by the image URI:

```
/HASH|unsafe/trim/AxB:CxD/fit-in/stretch/-Ex-F/GxH:IxJ/HALIGN/VALIGN/smart/filters:NAME(ARGS):NAME(ARGS):.../IMAGE
```

- `HASH` is the URL Signature hash, or `unsafe` if unsafe mode is used
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
- `IMAGE` is the image URI

Imagor provides utilities for previewing and generating Imagor endpoint URI, including the [imagorpath](https://github.com/cshum/imagor/tree/master/imagorpath) Go package and the `/params` endpoint:

#### `GET /params`

Prepending `/params` to the existing endpoint returns the endpoint attributes in JSON form, useful for preview:

```
curl http://localhost:8000/params/g5bMqZvxaQK65qFPaP1qlJOTuLM=/fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png

{
  "path": "fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png",
  "image": "raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png",
  "hash": "g5bMqZvxaQK65qFPaP1qlJOTuLM=",
  "fit_in": true,
  "width": 500,
  "height": 400,
  "padding_top": 20,
  "padding_bottom": 20,
  "filters": [
    {
      "name": "fill",
      "args": "white"
    }
  ]
}
```

### Filters

Filters `/filters:NAME(ARGS):NAME(ARGS):.../` is a pipeline of image operations that will be sequentially applied to the image. Examples:

```
/filters:fill(white):format(jpeg)/
/filters:hue(290):saturation(100):fill(yellow):format(jpeg):quality(80)/
/filters:fill(white):watermark(raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png,repeat,bottom,10):format(jpeg)/
```

Imagor supports the following filters:

- `background_color(color)` sets the background color of a transparent image
  - `color` the color name or hexadecimal rgb expression without the “#” character
- `blur(sigma)` applies gaussian blur to the image
- `brightness(amount)` increases or decreases the image brightness
  - `amount` -100 to 100, the amount in % to increase or decrease the image brightness
- `contrast(amount)` increases or decreases the image contrast
  - `amount` -100 to 100, the amount in % to increase or decrease the image contrast
- `fill(color)` fill the missing area or transparent image with the specified color:
  - `color` - color name or hexadecimal rgb expression without the “#” character
    - If color is "blur" - missing parts are filled with blurred original image.
    - If color is "auto" - the top left image pixel will be chosen as the filling color
- `format(format)` specifies the output format of the image
  - `format` accepts jpeg, png, gif, webp, jp2, tiff
- `frames(n[, delay])` set the number of frames to repeat for animation with gif or webp. Otherwise, stack all the frames vertically
  - `n` number of frames to repeat
  - `delay` frames delay in milliseconds, default 100
- `grayscale()` changes the image to grayscale
- `hue(angle)` increases or decreases the image hue
  - `angle` the angle in degree to increase or decrease the hue rotation
- `proportion(percentage)` scales image to the proportion percentage of the image dimension
- `quality(amount)` changes the overall quality of the image, does nothing for png
  - `amount` 0 to 100, the quality level in %
- `rgb(r,g,b)` amount of color in each of the rgb channels in %. Can range from -100 to 100
- `rotate(angle)` rotates the given image according to the angle value passed
  - `angle` accepts 0, 90, 180, 270
- `round_corner(rx [, ry [, color]])` adds rounded corners to the image with the specified color as background
  - `rx`, `ry` amount of pixel to use as radius. ry = rx if ry is not provided
  - `color` the color name or hexadecimal rgb expression without the “#” character
- `saturation(amount)` increases or decreases the image saturation
  - `amount` -100 to 100, the amount in % to increase or decrease the image saturation
- `sharpen(sigma)` sharpens the image
- `upscale()` upscale the image if `fit-in` is used
- `watermark(image, x, y, alpha [, w_ratio [, h_ratio]])` adds a watermark to the image. It can be positioned inside the image with the alpha channel specified and optionally resized based on the image size by specifying the ratio
  - `image` watermark image URI, using the same image loader configured for Imagor
  - `x` horizontal position that the watermark will be in:
    - Positive numbers indicate position from the left and negative numbers indicate position from the right.
    - Number followed by a `p` e.g. 20p means calculating the value from the image width as percentage
    - `left`,`right`,`center` positioned left, right or centered respectively
    - `repeat` the watermark will be repeated horizontally
  - `y` vertical position that the watermark will be in:
    - Positive numbers indicate position from the top and negative numbers indicate position from the bottom.
    - Number followed by a `p` e.g. 20p means calculating the value from the image height as percentage
    - `top`,`bottom`,`center` positioned top, bottom or centered respectively
    - `repeat` the watermark will be repeated vertically
  - `alpha` watermark image transparency, a number between 0 (fully opaque) and 100 (fully transparent).
  - `w_ratio` percentage of the width of the image the watermark should fit-in
  - `h_ratio` percentage of the height of the image the watermark should fit-in

### Loader, Storage and Result Storage

Imagor `Loader`, `Storage` and `Result Storage` are the building blocks for loading and saving images from various sources:

- `Loader` loads image. Enable `Loader` where you wish to load images from, but without modifying it e.g. static directory.
- `Storage` loads and saves image. This allows subsequent requests for the same image loads directly from the storage, instead of HTTP source.
- `Result Storage` loads and saves the processed image. This allows subsequent request of the same parameters loads from the result storage, saving processing resources.

Imagor provides built-in adaptors that support HTTP(s), Proxy, File System, AWS S3 and Google Cloud Storage. By default, `HTTP Loader` is used as fallback. You can choose to enable additional adaptors that fit your use cases.

#### File System

Docker Compose example with file system, using mounted volume:
```yaml
version: "3"
services:
  imagor:
    image: shumc/imagor:latest
    volumes:
      - ./:/mnt/data
    environment:
      PORT: 8000
      IMAGOR_UNSAFE: 1 # unsafe URL for testing

      FILE_LOADER_BASE_DIR: /mnt/data # enable file loader by specifying base dir

      FILE_STORAGE_BASE_DIR: /mnt/data # enable file storage by specifying base dir

      FILE_RESULT_STORAGE_BASE_DIR: /mnt/data/result # enable file result storage by specifying base dir
    ports:
      - "8000:8000"
```

#### AWS S3

Docker Compose example with AWS S3. Also works with S3 compatible such as MinIO, DigitalOcean Space.
```yaml
version: "3"
services:
  imagor:
    image: shumc/imagor:latest
    environment:
      PORT: 8000
      IMAGOR_SECRET: mysecret # secret key for URL signature
      AWS_ACCESS_KEY_ID: ...
      AWS_SECRET_ACCESS_KEY: ...
      AWS_REGION: ...

      S3_LOADER_BUCKET: mybucket # enable S3 loader by specifying bucket
      S3_LOADER_BASE_DIR: images # optional

      S3_STORAGE_BUCKET: mybucket # enable S3 storage by specifying bucket
      S3_STORAGE_BASE_DIR: images # optional

      S3_RESULT_STORAGE_BUCKET: mybucket # enable S3 result storage by specifying bucket
      S3_RESULT_STORAGE_BASE_DIR: images/result # optional

      S3_ENDPOINT: ... # optional - alternative endpoint for S3 compatible, such as MinIO
    ports:
      - "8000:8000"
```

#### Google Cloud Storage

Docker Compose example with Google Cloud Storage:
```yaml
version: "3"
services:
  imagor:
    image: shumc/imagor:latest
    volumes:
      - ./googlesecret:/etc/secrets/google
    environment:
      PORT: 8000
      IMAGOR_SECRET: mysecret # secret key for URL signature
      GOOGLE_APPLICATION_CREDENTIALS: /etc/secrets/google/appcredentials.json # google cloud secrets file

      GCLOUD_LOADER_BUCKET: mybucket # enable loader by specifying bucket
      GCLOUD_LOADER_BASE_DIR: images # optional

      GCLOUD_STORAGE_BUCKET: mybucket # enable storage by specifying bucket
      GCLOUD_STORAGE_BASE_DIR: images # optional

      GCLOUD_RESULT_STORAGE_BUCKET: mybucket # enable result storage by specifying bucket
      GCLOUD_RESULT_STORAGE_BASE_DIR: images/result # optional
      GCLOUD_RESULT_STORAGE_ACL: publicRead # optional - see https://cloud.google.com/storage/docs/json_api/v1/objects/insert
    ports:
      - "8000:8000"
```

### Security

#### URL Signature

In production environment, it is highly recommended turning off `IMAGOR_UNSAFE` and setting up URL signature using `IMAGOR_SECRET`, to prevent DDoS attacks that abuse multiple image operations.

The URL signature hash is based on SHA digest, created by taking the URL path (excluding `/unsafe/`) with secret. The hash is then Base64 URL encoded.
An example in Node.js:

```javascript
var hmacSHA1 = require("crypto-js/hmac-sha1")
var Base64 = require("crypto-js/enc-base64")

function sign(path, secret) {
  var hash = hmacSHA1(path, secret)
  hash = Base64.stringify(hash).replace(/\+/g, '-').replace(/\//g, '_')
  return hash + '/' + path
}

console.log(sign('500x500/top/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png', 'mysecret'))
// cST4Ko5_FqwT3BDn-Wf4gO3RFSk=/500x500/top/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

### Configurations

Imagor supports command-line arguments, see available options `imagor -h`. You may check [main.go](https://github.com/cshum/imagor/blob/master/cmd/imagor/main.go) for better understanding the initialization sequences.

Imagor also supports environment variables or `.env` file for the arguments equivalent in capitalized snake case. For instances `-imagor-secret` would become `IMAGOR_SECRET`:

```bash
# both are equivalent

imagor -debug -imagor-secret 1234

DEBUG=1 IMAGOR_SECRET=1234 imagor
```

Available options:

```
imagor -h
Usage of imagor:
  -debug
        Debug mode
  -port int
        Sever port (default 8000)
  -version
        Imagor version

  -imagor-secret string
        Secret key for signing Imagor URL
  -imagor-unsafe
        Unsafe Imagor that does not require URL signature. Prone to URL tampering
  -imagor-auto-webp
        Output WebP format automatically if browser supports
  -imagor-cache-header-ttl duration
        Imagor HTTP cache header ttl for successful image response. Set -1 for no-cache (default 24h0m0s)
  -imagor-request-timeout duration
        Timeout for performing Imagor request (default 30s)
  -imagor-load-timeout duration
        Timeout for Imagor Loader request, should be smaller than imagor-request-timeout (default 20s)
  -imagor-save-timeout duration
        Timeout for saving image to Imagor Storage (default 20s)
  -imagor-process-timeout duration
        Timeout for image processing (default 20s)
  -imagor-process-concurrency int
        Imagor semaphore size for process concurrency control. Set -1 for no limit (default -1)
  -imagor-base-path-redirect string
        URL to redirect for Imagor / base path e.g. https://www.google.com

  -server-address string
        Server address
  -server-cors
        Enable CORS
  -server-strip-query-string
        Enable strip query string redirection
  -server-path-prefix string
        Server path prefix
  -server-access-log
        Enable server access log

  -http-loader-allowed-sources string
        HTTP Loader allowed hosts whitelist to load images from if set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.
  -http-loader-forward-headers string
        Forward request header to HTTP Loader request by csv e.g. User-Agent,Accept
  -http-loader-forward-client-headers
        Forward browser client request headers to HTTP Loader request
  -http-loader-insecure-skip-verify-transport
        HTTP Loader to use HTTP transport with InsecureSkipVerify true
  -http-loader-max-allowed-size int
        HTTP Loader maximum allowed size in bytes for loading images if set
  -http-loader-proxy-urls string
        HTTP Loader Proxy URLs. Enable HTTP Loader proxy only if this value present. Accept csv of proxy urls e.g. http://user:pass@host:port,http://user:pass@host:port
  -http-loader-proxy-allowed-sources string
        HTTP Loader Proxy allowed hosts that enable proxy transport, if proxy URLs are set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.
  -http-loader-default-scheme string
        HTTP Loader default scheme if not specified by image path. Set "nil" to disable default scheme. (default "https")
  -http-loader-accept string
        HTTP Loader set request Accept header and validate response Content-Type header (default "image/*")
        HTTP Loader default scheme if not specified by image path (default "https")
  -http-loader-disable
        Disable HTTP Loader

  -file-safe-chars string
        File safe characters to be excluded from image key escape
  -file-loader-base-dir string
        Base directory for File Loader. Enable File Loader only if this value present
  -file-loader-path-prefix string
        Base path prefix for File Loader
  -file-result-storage-base-dir string
        Base directory for File Result Storage. Enable File Result Storage only if this value present
  -file-result-storage-mkdir-permission string
        File Result Storage mkdir permission (default "0755")
  -file-result-storage-path-prefix string
        Base path prefix for File Result Storage
  -file-result-storage-write-permission string
        File Storage write permission (default "0666")
  -file-result-storage-expiration duration
        File Result Storage expiration duration e.g. 24h. Default no expiration
  -file-storage-base-dir string
        Base directory for File Storage. Enable File Storage only if this value present
  -file-storage-path-prefix string
        Base path prefix for File Storage
  -file-storage-mkdir-permission string
        File Storage mkdir permission (default "0755")
  -file-storage-write-permission string
        File Storage write permission (default "0666")
  -file-storage-expiration duration
        File Storage expiration duration e.g. 24h. Default no expiration

  -aws-access-key-id string
        AWS Access Key ID. Required if using S3 Loader or S3 Storage
  -aws-region string
        AWS Region. Required if using S3 Loader or S3 Storage
  -aws-secret-access-key string
        AWS Secret Access Key. Required if using S3 Loader or S3 Storage
  -s3-endpoint string
        Optional S3 Endpoint to override default
  -s3-safe-chars string
        S3 safe characters to be excluded from image key escape
  -s3-force-path-style
        S3 force the request to use path-style addressing s3.amazonaws.com/bucket/key, instead of bucket.s3.amazonaws.com/key
  -s3-loader-bucket string
        S3 Bucket for S3 Loader. Enable S3 Loader only if this value present
  -s3-loader-base-dir string
        Base directory for S3 Loader
  -s3-loader-path-prefix string
        Base path prefix for S3 Loader
  -s3-result-storage-bucket string
        S3 Bucket for S3 Result Storage. Enable S3 Result Storage only if this value present
  -s3-result-storage-base-dir string
        Base directory for S3 Result Storage
  -s3-result-storage-path-prefix string
        Base path prefix for S3 Result Storage
  -s3-result-storage-acl string
        Upload ACL for S3 Result Storage (default "public-read")
  -s3-result-storage-expiration duration
        S3 Result Storage expiration duration e.g. 24h. Default no expiration
  -s3-storage-bucket string
        S3 Bucket for S3 Storage. Enable S3 Storage only if this value present
  -s3-storage-base-dir string
        Base directory for S3 Storage
  -s3-storage-path-prefix string
        Base path prefix for S3 Storage
  -s3-storage-acl string
        Upload ACL for S3 Storage (default "public-read")
  -s3-storage-expiration duration
        S3 Storage expiration duration e.g. 24h. Default no expiration

  -gcloud-safe-chars string
        Google Cloud safe characters to be excluded from image key escape
  -gcloud-loader-base-dir string
        Base directory for Google Cloud Loader
  -gcloud-loader-bucket string
        Bucket name for Google Cloud Storage Loader. Enable Google Cloud Loader only if this value present
  -gcloud-loader-path-prefix string
        Base path prefix for Google Cloud Loader
  -gcloud-result-storage-acl string
        Upload ACL for Google Cloud Result Storage
  -gcloud-result-storage-base-dir string
        Base directory for Google Cloud Result Storage
  -gcloud-result-storage-bucket string
        Bucket name for Google Cloud Result Storage. Enable Google Cloud Result Storage only if this value present
  -gcloud-result-storage-expiration duration
        Google Cloud Result Storage expiration duration e.g. 24h. Default no expiration
  -gcloud-result-storage-path-prefix string
        Base path prefix for Google Cloud Result Storage
  -gcloud-storage-acl string
        Upload ACL for Google Cloud Storage
  -gcloud-storage-base-dir string
        Base directory for Google Cloud
  -gcloud-storage-bucket string
        Bucket name for Google Cloud Storage. Enable Google Cloud Storage only if this value present
  -gcloud-storage-expiration duration
        Google Cloud Storage expiration duration e.g. 24h. Default no expiration
  -gcloud-storage-path-prefix string
        Base path prefix for Google Cloud Storage
        
  -vips-max-animation-frames int
        VIPS maximum number of animation frames to be loaded. Set 1 to disable animation, -1 for unlimited.
  -vips-disable-blur
        VIPS disable blur operations for vips processor
  -vips-disable-filters string
        VIPS disable filters by csv e.g. blur,watermark,rgb
  -vips-max-filter-ops int
        VIPS maximum number of filter operations allowed (default 10)
  -vips-max-height int
        VIPS max image height
  -vips-max-width int
        VIPS max image width
  -vips-concurrency int
        VIPS concurrency. Set -1 to be the number of CPU cores (default 1)
  -vips-max-cache-files int
        VIPS max cache files
  -vips-max-cache-mem int
        VIPS max cache mem
  -vips-max-cache-size int
        VIPS max cache size
  -vips-mozjpeg
        VIPS enable maximum compression with MozJPEG. Requires mozjpeg to be installed
```
