# Configuration

imagor supports command-line arguments and environment variables for the arguments equivalent in capitalized snake case, see available options `imagor -h`.
For instances `-imagor-secret` would become `IMAGOR_SECRET`:

```bash
# both are equivalent

imagor -debug -imagor-secret 1234

DEBUG=1 IMAGOR_SECRET=1234 imagor
```

Configuration can also be specified in a `.env` environment variable file and referenced with the `-config` flag:

```bash
imagor -config path/to/config.env
```

config.env:

```dotenv
PORT=8000
IMAGOR_SECRET=mysecret
DEBUG=1
```

## Available options

```
imagor -h
Usage of imagor:
  -debug
        Debug mode
  -port int
        Server port (default 8000)
  -version
        imagor version
  -config string
        Retrieve configuration from the given file (default ".env")

  -imagor-secret string
        Secret key for signing imagor URL
  -imagor-unsafe
        Unsafe imagor that does not require URL signature. Prone to URL tampering
  -imagor-auto-webp
        Output WebP format automatically if browser supports
  -imagor-auto-avif
        Output AVIF format automatically if browser supports (experimental)
  -imagor-auto-jpeg
        Output JPEG format automatically if JPEG or no specific format is requested
  -imagor-base-params string
        imagor endpoint base params that applies to all resulting images e.g. filters:watermark(example.jpg)
  -imagor-signer-type string
        imagor URL signature hasher type: sha1, sha256, sha512 (default "sha1")
  -imagor-signer-truncate int
        imagor URL signature truncate at length
  -imagor-result-storage-path-style string
        imagor result storage path style: original, digest, suffix (default "original")
  -imagor-storage-path-style string
        imagor storage path style: original, digest (default "original")
  -imagor-cache-header-ttl duration
        imagor HTTP cache header ttl for successful image response (default 168h0m0s)
  -imagor-cache-header-swr duration
        imagor HTTP Cache-Control header stale-while-revalidate for successful image response (default 24h0m0s)
  -imagor-cache-header-no-cache
        imagor HTTP Cache-Control header no-cache for successful image response
  -imagor-request-timeout duration
        Timeout for performing imagor request (default 30s)
  -imagor-load-timeout duration
        Timeout for imagor Loader request, should be smaller than imagor-request-timeout
  -imagor-save-timeout duration
        Timeout for saving image to imagor Storage
  -imagor-process-timeout duration
        Timeout for image processing
  -imagor-process-concurrency int
        Maximum number of image process to be executed simultaneously. Requests that exceed this limit are put in the queue. Set -1 for no limit (default -1)
  -imagor-process-queue-size int
        Maximum number of image process that can be put in the queue. Requests that exceed this limit are rejected with HTTP status 429
  -imagor-base-path-redirect string
        URL to redirect for imagor / base path e.g. https://www.google.com
  -imagor-modified-time-check
        Check modified time of result image against the source image. This eliminates stale result but require more lookups
  -imagor-disable-params-endpoint
        imagor disable /params endpoint
  -imagor-disable-error-body
        imagor disable response body on error

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

  -prometheus-bind string
        Specify address and port to enable Prometheus metrics, e.g. :5000, prom:7000
  -prometheus-path string
        Prometheus metrics path (default "/")
        
  -http-loader-allowed-sources string
        HTTP Loader allowed hosts whitelist to load images from if set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.
  -http-loader-base-url string
        HTTP Loader base URL that prepends onto existing image path. This overrides the default scheme option.
  -http-loader-forward-headers string
        Forward request header to HTTP Loader request by csv e.g. User-Agent,Accept
  -http-loader-override-response-headers string
        Override HTTP Loader response header to image response by csv e.g. Cache-Control,Expires
  -http-loader-forward-client-headers
        Forward browser client request headers to HTTP Loader request
  -http-loader-insecure-skip-verify-transport
        HTTP Loader to use HTTP transport with InsecureSkipVerify true
  -http-loader-max-allowed-size int
        HTTP Loader maximum allowed size in bytes for loading images if set
  -http-loader-proxy-urls string
        HTTP Loader Proxy URLs. Enable HTTP Loader proxy only if this value present. Accept csv of proxy urls e.g. http://user:pass@host:port,http://user:pass@host:port
  -http-loader-allowed-source-regexp string
        HTTP Loader allowed hosts regexp to load images from if set. Combines as OR with allowed host glob pattern sources.
  -http-loader-proxy-allowed-sources string
        HTTP Loader Proxy allowed hosts that enable proxy transport, if proxy URLs are set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.
  -http-loader-default-scheme string
        HTTP Loader default scheme if not specified by image path. Set "nil" to disable default scheme. (default "https")
  -http-loader-accept string
        HTTP Loader set request Accept header and validate response Content-Type header (default "*/*") 
  -http-loader-block-link-local-networks
        HTTP Loader rejects connections to link local network IP addresses.
  -http-loader-block-loopback-networks
        HTTP Loader rejects connections to loopback network IP addresses.
  -http-loader-block-private-networks
        HTTP Loader rejects connections to private network IP addresses.
  -http-loader-block-networks string
        HTTP Loader rejects connections to link local network IP addresses. This options takes a comma separated list of networks in CIDR notation e.g ::1/128,127.0.0.0/8.
  -http-loader-disable
        Disable HTTP Loader

  -upload-loader-enable
        Enable Upload Loader for POST uploads
  -upload-loader-max-allowed-size int
        Upload Loader maximum allowed size in bytes for uploaded images (default 33554432)
  -upload-loader-accept string
        Upload Loader accepted Content-Type for uploads (default "image/*")
  -upload-loader-form-field-name string
        Upload Loader form field name for multipart uploads (default "image")

  -file-safe-chars string
        File safe characters to be excluded from image key escape. Set -- for no-op
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
  -aws-session-token string
        AWS Session Token. Optional temporary credentials token
  -s3-endpoint string
        Optional S3 Endpoint to override default
  -s3-safe-chars string
        S3 safe characters to be excluded from image key escape. Set -- for no-op
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
        
  -aws-loader-access-key-id string
        AWS Access Key ID for S3 Loader to override global config
  -aws-loader-region string
        AWS Region for S3 Loader to override global config
  -aws-loader-secret-access-key string
        AWS Secret Access Key for S3 Loader to override global config
  -aws-loader-session-token string
        AWS Session Token for S3 Loader to override global config
  -s3-loader-endpoint string
        Optional S3 Loader Endpoint to override default
  -aws-storage-access-key-id string
        AWS Access Key ID for S3 Storage to override global config
  -aws-storage-region string
        AWS Region for S3 Storage to override global config
  -aws-storage-secret-access-key string
        AWS Secret Access Key for S3 Storage to override global config
  -aws-storage-session-token string
        AWS Session Token for S3 Storage to override global config
  -s3-storage-endpoint string
        Optional S3 Storage Endpoint to override default
  -aws-result-storage-access-key-id string
        AWS Access Key ID for S3 Result Storage to override global config
  -aws-result-storage-region string
        AWS Region for S3 Result Storage to override global config
  -aws-result-storage-secret-access-key string
        AWS Secret Access Key for S3 Result Storage to override global config
  -aws-result-storage-session-token string
        AWS Session Token for S3 Result Storage to override global config
  -s3-result-storage-endpoint string
        Optional S3 Storage Endpoint to override default

  -gcloud-safe-chars string
        Google Cloud safe characters to be excluded from image key escape. Set -- for no-op
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
        VIPS maximum number of animation frames to be loaded. Set 1 to disable animation, -1 for unlimited
  -vips-disable-blur
        VIPS disable blur operations for vips processor
  -vips-disable-filters string
        VIPS disable filters by csv e.g. blur,watermark,rgb
  -vips-max-filter-ops int
        VIPS maximum number of filter operations allowed. Set -1 for unlimited (default -1)
  -vips-max-width int
        VIPS max image width
  -vips-max-height int
        VIPS max image height
  -vips-max-resolution int
        VIPS max image resolution
  -vips-mozjpeg
        VIPS enable maximum compression with MozJPEG. Requires mozjpeg to be installed
  -vips-avif-speed int
        VIPS avif speed, the lowest is at 0 and the fastest is at 9 (Default 5).
  -vips-strip-metadata
        VIPS strips all metadata from the resulting image
  -vips-unlimited
    	VIPS bypass image max resolution check and remove all denial of service limits
        
  -sentry-dsn
        include sentry dsn to integrate imagor with sentry
