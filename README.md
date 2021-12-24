# Imagor

[![Build Status](https://github.com/cshum/imagor/workflows/build/badge.svg)](https://github.com/cshum/imagor/actions)
[![Docker](https://img.shields.io/badge/docker-shumc/imagor-blue.svg)](https://hub.docker.com/r/shumc/imagor/) 

Imagor is a fast, Docker-ready image processing server written in Go.

Imagor uses one of the most efficient image processing library
[libvips](https://github.com/libvips/libvips). It is typically 4-8x [faster](https://github.com/libvips/libvips/wiki/Speed-and-memory-use) than using the quickest ImageMagick and GraphicsMagick settings.

Imagor is a Go library that is easily extensible, ready to be installed and used in any Unix environment, and ready to be containerized using Docker.

Imagor adopts the [Thumbor](https://thumbor.readthedocs.io/en/latest/usage.html#image-endpoint) URL syntax and covers most of the web image processing use cases. If these fit your requirements, Imagor would be a lightweight, high performance drop-in replacement.

### Quick Start

```bash
docker run -p 8000:8000 shumc/imagor -imagor-unsafe
```

Try out the following image URLs:

```
# original images
https://raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png
https://raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher-front.png

http://localhost:8000/unsafe/500x500/top/https://raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png
http://localhost:8000/unsafe/fit-in/500x500/filters:fill(white):format(jpeg)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png
http://localhost:8000/unsafe/fit-in/-500x500/10x10/filters:hue(290):saturation(100):fill(yellow):format(jpeg):quality(80)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png
http://localhost:8000/unsafe/fit-in/800x800/filters:fill(white):watermark(raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher-front.png,repeat,bottom,10):format(jpeg)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png
```

#### Docker Compose Example

Imagor with File Loader and Storage using mounted volume:

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
      
      FILE_RESULT_LOADER_BASE_DIR: /mnt/data/result # enable file result loader by specifying base dir
      FILE_RESULT_STORAGE_BASE_DIR: /mnt/data/result # enable file result storage by specifying base dir
    ports:
      - "8000:8000"
```

Imagor with AWS S3 Loader and Storage:

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
      
      S3_LOADER_BUCKET: mybucket # enable S3 loader by specifying loader bucket
      S3_LOADER_BASE_DIR: images # optional
      S3_STORAGE_BUCKET: mybucket # enable S3 storage by specifying storage bucket
      S3_STORAGE_BASE_DIR: images # optional
      
      S3_RESULT_LOADER_BUCKET: mybucket # enable S3 result loader by specifying loader bucket
      S3_RESULT_LOADER_BASE_DIR: images/result # optional
      S3_RESULT_STORAGE_BUCKET: mybucket # enable S3 result loader by specifying loader bucket
      S3_RESULT_STORAGE_BASE_DIR: images/result # optional
    ports:
      - "8000:8000"
```

### Imagor Endpoint

Imagor endpoint is a series of URL parts which defines the image operations, followed by the image URI:

```
/HASH|unsafe/trim/AxB:CxD/fit-in/stretch/-Ex-F/GxH/HALIGN/VALIGN/smart/filters:NAME(ARGS):NAME(ARGS):.../IMAGE
```

- `HASH` is the URL Signature hash, or `unsafe` if unsafe mode is used
- `trim` removes surrounding space in images using top-left pixel color
- `AxB:CxD` means manually crop the image at left-top point `AxB` and right-bottom point `CxD`
- `fit-in` means that the generated image should not be auto-cropped and otherwise just fit in an imaginary box specified by `ExF`
- `stretch` means resize the image to `ExF` without keeping its aspect ratios
- `-Ex-F` means resize the image to be `ExF` of width per height size. The minus signs mean flip horizontally and vertically
- `GxH` add horizontal padding `G` and vertical padding `H` under `fit-in`
- `HALIGN` is horizontal alignment of crop. Accepts `left`, `right` or `center`, defaults to `center`
- `VALIGN` is vertical alignment of crop. Accepts `top`, `bottom` or `middle`, defaults to `middle`
- `smart` means using smart detection of focal points
- `filters` a pipeline of image filter operations to be applied, see filters section
- `IMAGE` is the image URI

Imagor provides utilities for previewing and generating Imagor endpoint URI:

#### `GET /params`

Prepending `/params` to the existing endpoint returns the endpoint attributes in JSON form, useful for preview:

```
curl http://localhost:8000/params/g5bMqZvxaQK65qFPaP1qlJOTuLM=/fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png

{
  "path": "fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png",
  "image": "raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png",
  "hash": "g5bMqZvxaQK65qFPaP1qlJOTuLM=",
  "fit_in": true,
  "width": 500,
  "height": 400,
  "v_padding": 20,
  "filters": [
    {
      "name": "fill",
      "args": "white"
    }
  ]
}
```

#### `imagorpath` package

Imagor Go library provides a `imagorpath` package which
allows you to parse and generate Imagor endpoint using the `Params` struct:

```go
import "github.com/cshum/imagor/imagorpath"

...

func Test(t *testing.T) {
	params := imagorpath.Params{
		Image:    "raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png",
		FitIn:    true,
		Width:    500,
		Height:   400,
		VPadding: 20,
		Filters: imagorpath.Filters{
			{
				Name: "fill",
				Args: "white",
			},
		},
	}

	// generate signed Imagor endpoint from Params struct with secret
	path := imagorpath.Generate(params, "mysecret")

	assert.Equal(t, path, "g5bMqZvxaQK65qFPaP1qlJOTuLM=/fit-in/500x400/0x20/filters:fill(white)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png")
}

```

### URL Signature

In production environment, it is highly recommended turning off `IMAGOR_UNSAFE` and setup `IMAGOR_SECRET` to avoid DDoS attacks that abuse multiple image operations.

The hash is based on HMAC digest, created by taking the URL path (excluding /unsafe/) with secret. The hash is then base64url-encoded.
An example in Go:

```go
func SignPath(path, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(strings.TrimPrefix(path, "/")))
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return hash + "/" + path
}

func main() {
	fmt.Println(SignPath("500x500/top/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png", "mysecret"))
	// RArq3FZw_bqxLcpKo1WI0aX_q7s=/fit-in/500x500/filters:fill(white):format(jpeg)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png
}
```

### Filters

Filters `/filters:NAME(ARGS):NAME(ARGS):.../` is a pipeline of image operations that will be sequentially applied to the image. Examples:

```
/filters:fill(white):format(jpeg)/
/filters:hue(290):saturation(100):fill(yellow):format(jpeg):quality(80)/
/filters:fill(white):watermark(raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher-front.png,repeat,bottom,10):format(jpeg)/
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
  - `format` accepts jpeg, png, webp, gif, jp2, tiff
- `grayscale()` changes the image to grayscale
- `hue(angle)` increases or decreases the image hue
  - `angle` the angle in degree to increase or decrease the hue rotation
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
- `trim([tolerance [, position]])` apply trim operation as part of the filter pipeline
  - `tolerance` the euclidean distance between the colors to get trimmed within the tolerance, default 1
  - `position` default using `top-left` pixel color unless specified `bottom-right`
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

### Configurations

Imagor supports command-line arguments, see available options `imagor -h`. You may check [main.go](https://github.com/cshum/imagor/blob/master/cmd/imagor/main.go) for better understanding the initialization sequences.

Imagor also supports environment variables or `.env` file for the arguments equivalent in capitalized snake case. For instances `-imagor-secret` would become `IMAGOR_SECRET`:

```bash
# both are equivalent

imagor -debug -imagor-secret=1234

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

  -imagor-secret string
        Secret key for signing Imagor URL
  -imagor-unsafe
        Unsafe Imagor that does not require URL signature. Prone to URL tampering
  -imagor-version
        Imagor version
  -imagor-cache-header-ttl duration
        Imagor HTTP cache header ttl for successful image response. Set -1 for no-cache (default 24h0m0s)
  -imagor-load-timeout duration
        Timeout for Imagor Loader request, should be smaller than imagor-request-timeout (default 20s)
  -imagor-process-timeout duration
        Timeout for image processing (default 20s)
  -imagor-request-timeout duration
        Timeout for performing Imagor request (default 30s)
  -imagor-save-timeout duration
        Timeout for saving image to Imagor Storage (default 20s)


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
  -http-loader-default-scheme string
        HTTP Loader default scheme if not specified by image path. Set "nil" to disable default scheme. (default "https")
  -http-loader-forward-headers string
        Forward request header to HTTP Loader request by csv e.g. User-Agent,Accept
  -http-loader-forward-all-headers
        Forward all request headers to HTTP Loader request
  -http-loader-insecure-skip-verify-transport
        HTTP Loader to use HTTP transport with InsecureSkipVerify true
  -http-loader-max-allowed-size int
        HTTP Loader maximum allowed size in bytes for loading images if set
  -http-loader-disable
        Disable HTTP Loader

  -file-loader-base-dir string
        Base directory for File Loader. Will activate File Loader only if this value present
  -file-loader-path-prefix string
        Base path prefix for File Loader

  -file-storage-base-dir string
        Base directory for File Storage. Will activate File Storage only if this value present
  -file-storage-path-prefix string
        Base path prefix for File Storage
  -file-storage-mkdir-permission string
        File Storage mkdir permission (default "0755")
  -file-storage-write-permission string
        File Storage write permission (default "0666")

  -aws-access-key-id string
        AWS Access Key ID. Required if using S3 Loader or S3 Storage
  -aws-region string
        AWS Region. Required if using S3 Loader or S3 Storage
  -aws-secret-access-key string
        AWS Secret Access Key. Required if using S3 Loader or S3 Storage
  -s3-endpoint string
        Optional S3 Endpoint to override default

  -s3-loader-base-dir string
        Base directory for S3 Loader
  -s3-loader-bucket string
        S3 Bucket for S3 Loader. Will activate S3 Loader only if this value present
  -s3-loader-path-prefix string
        Base path prefix for S3 Loader

  -s3-storage-base-dir string
        Base directory for S3 Storage
  -s3-storage-bucket string
        S3 Bucket for S3 Storage. Will activate S3 Storage only if this value present
  -s3-storage-path-prefix string
        Base path prefix for S3 Storage
  -s3-storage-acl string
        Upload ACL for S3 Storage (default "public-read")
  
  -file-result-loader-base-dir string
        Base directory for File Result Loader. Will activate File Result Loader only if this value present
  -file-result-loader-path-prefix string
        Base path prefix for File Result Loader
        
  -file-result-storage-base-dir string
        Base directory for File Result Storage. Will activate File Result Storage only if this value present
  -file-result-storage-mkdir-permission string
        File Result Storage mkdir permission (default "0755")
  -file-result-storage-path-prefix string
        Base path prefix for File Result Storage
  -file-result-storage-write-permission string
        File Storage write permission (default "0666")
  
  -s3-result-loader-base-dir string
        Base directory for S3 Result Loader
  -s3-result-loader-bucket string
        S3 Bucket for S3 Result Loader. Will activate S3 Result Loader only if this value present
  -s3-result-loader-path-prefix string
        Base path prefix for S3 Result Loader
        
  -s3-result-storage-acl string
        Upload ACL for S3 Result Storage (default "public-read")
  -s3-result-storage-base-dir string
        Base directory for S3 Result Storage
  -s3-result-storage-bucket string
        S3 Bucket for S3 Result Storage. Will activate S3 Result Storage only if this value present
  -s3-result-storage-path-prefix string
        Base path prefix for S3 Result Storage


  -vips-concurrency int
        VIPS concurrency. Set -1 to be the number of CPU cores (default 1)
  -vips-disable-blur
        VIPS disable blur operations for vips processor
  -vips-disable-filters string
        VIPS disable filters by csv e.g. blur,watermark,rgb
  -vips-load-from-file
        VIPS to load from file when file loader is used. By default load from buffer, which is faster but consumes more memory
  -vips-max-cache-files int
        VIPS max cache files
  -vips-max-cache-mem int
        VIPS max cache mem
  -vips-max-cache-size int
        VIPS max cache size
  -vips-max-filter-ops int
        VIPS maximum number of filter operations allowed (default 10)
  -vips-max-height int
        VIPS max image height
  -vips-max-width int
        VIPS max image width
```
