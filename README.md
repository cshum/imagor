# Imagor

Imagor is a fast, Docker-ready image processing server written in Go. 

Imagor uses one of the most efficient image processing library 
[libvips](https://github.com/libvips/libvips) (with [govips](https://github.com/davidbyttow/govips)). It is typically 4-8x faster than using the quickest ImageMagick and GraphicsMagick settings.

Imagor is a Go library that is easily extensible, ready to be installed and used in any Unix environment, and ready to be containerized using Docker.

Imagor adopts the [Thumbor](https://thumbor.readthedocs.io/en/latest/usage.html#image-endpoint) URL syntax and covers most of the common web image operations use cases. If these fits your requirements, then Imagor would be a lightweight, high performance drop-in replacement.

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
http://localhost:8000/unsafe/fit-in/500x500/filters:hue(290):saturation(100):fill(yellow):rotate(90):format(jpeg):quality(80)/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png
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
      IMAGOR_UNSAFE: 1 # unsafe URL
      FILE_LOADER_BASE_DIR: /mnt/data # enable file loader by specifying base dir
      FILE_STORAGE_BASE_DIR: /mnt/data # enable file storage by specifying base dir
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
      HTTP_LOADER_FORWARD_ALL_HEADERS: 1 # Forward all request headers to HTTP Loader
      AWS_ACCESS_KEY_ID: ... 
      AWS_SECRET_ACCESS_KEY: ...
      AWS_REGION: ...
      S3_LOADER_BUCKET: mybucket # enable S3 loader by specifying loader bucket
      S3_LOADER_BASE_DIR: images # optional
      S3_STORAGE_BUCKET: mybucket # enable S3 loader by specifying storage bucket
      S3_STORAGE_BASE_DIR: images # optional
    ports:
      - "8000:8000"
```

### Imagor Endpoint

Imagor endpoint is a series of URL parts which defines the image operations, followed by the image URI:

```
/HASH|unsafe/trim/AxB:CxD/fit-in/stretch/-Ex-F/HALIGN/VALIGN/smart/filters:NAME(ARGS):.../IMAGE
```

* `HASH` is the URL Signature hash, or `unsafe` if unsafe mode is used
* `trim` removes surrounding space in images using top-left pixel color unless specified otherwise
* `AxB:CxD` means manually crop the image at left-top point `AxB` and right-bottom point `CxD`
* `fit-in` means that the generated image should not be auto-cropped and otherwise just fit in an imaginary box specified by `ExF`
* `stretch` means resize the image to `ExF` without keeping its aspect ratios
* `-Ex-F` means resize the image to be `ExF` of width per height size. The minus signs mean flip horizontally and vertically
* `HALIGN` is horizontal alignment of crop. Accepts `left`, `right` or `center`, defaults to `center`
* `VALIGN` is vertical alignment of crop. Accepts `top`, `bottom` or `middle`, defaults to `middle`
* `smart` means using smart detection of focal points
* `filters` a series of image filter operations to be applied, see filters section
* `IMAGE` is the image URI

In addition, prepending `/params` to the existing endpoint returns the endpoint params in JSON form for preview:
```
curl http://localhost:8000/params/unsafe/500x500/top/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png

{
  "path": "500x500/top/raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png",
  "image": "raw.githubusercontent.com/golang-samples/gopher-vector/master/gopher.png",
  "unsafe": true,
  "width": 500,
  "height": 500,
  "valign": "top"
}
```

### URL Signature

In production environment, it is highly recommended turning off `IMAGOR_UNSAFE` and setup `IMAGOR_SECRET` 
in order to avoid DDoS attacks using multiple image operations.

The HMAC-SHA256 hash is created by taking the URL path (excluding /unsafe/) with secret. The hash is then base64url-encoded.
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

### Configurations

Imagor supports command-line arguments, see available options `imagor -h`. You may check [main.go](https://github.com/cshum/imagor/blob/master/cmd/imagor/main.go) for better understanding the initialization sequences.

Imagor also supports environment variables or `.env` file for the arguments equivalent in capitalized snake case. For instances `-imagor-secret` would become `IMAGOR_SECRET`:
```bash
# both are equivalent
imagor -debug -imagor-scret=1234

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
  -imagor-request-timeout duration
        Timeout for performing imagor request (default 30s)
  -imagor-save-timeout duration
        Timeout for saving requesting image for storage (default 1m0s)
        
  -server-address string
        Server address
  -server-cors
        Enable CORS
  -server-path-prefix string
        Server path prefix
        
  -vips-concurrency-level int
        VIPS concurrency level. Default to the number of CPU cores.
  -vips-disable-blur
        VIPS disable blur operations for vips processor
  -vips-disable-filters string
        VIPS disable filters by csv e.g. blur,watermark,rgb
  -vips-max-filter-ops int
        VIPS maximum number of filter operations allowed (default 10)
        
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
        
  -aws-access-key-id string
        AWS Access Key ID. Required if using S3 Loader or S3 Storage
  -aws-region string
        AWS Region. Required if using S3 Loader or S3 Storage
  -aws-secret-access-key string
        AWS Secret Access Key. Required if using S3 Loader or S3 Storage
        
  -s3-loader-base-dir string
        Base directory for S3 Loader (default "/")
  -s3-loader-bucket string
        S3 Bucket for S3 Loader. Will activate S3 Loader only if this value present
  -s3-loader-path-prefix string
        Base path prefix for S3 Loader (default "/")
        
  -s3-storage-base-dir string
        Base directory for S3 Storage
  -s3-storage-bucket string
        S3 Bucket for S3 Storage. Will activate S3 Storage only if this value present
  -s3-storage-path-prefix string
        Base path prefix for S3 Storage
```


