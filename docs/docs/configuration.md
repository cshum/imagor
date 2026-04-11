# Configuration

imagor supports command-line arguments and environment variables. Environment variables are the flag name in uppercase snake case — `-imagor-secret` becomes `IMAGOR_SECRET`:

```bash
# both are equivalent
imagor -debug -imagor-secret 1234

DEBUG=1 IMAGOR_SECRET=1234 imagor
```

Configuration can also be loaded from a `.env` file using `-config`:

```bash
imagor -config path/to/config.env
```

```dotenv
PORT=8000
IMAGOR_SECRET=mysecret
DEBUG=1
```

Run `imagor -h` to print all available options with their defaults.

---

## General

```dotenv
PORT=8000                  # Server port (default 8000)
DEBUG=1                    # Debug mode
VERSION=1                  # Print imagor version
CONFIG=path/to/config.env  # Load configuration from file (default .env)
GOMAXPROCS=                # Go runtime CPU limit (default: all cores)
```

## Server

```dotenv
BIND=myhost:8888           # Combined address:port shortcut, overrides SERVER_ADDRESS + PORT
SERVER_ADDRESS=            # Server bind address (default all interfaces)
SERVER_CORS=1              # Enable CORS
SERVER_STRIP_QUERY_STRING=1  # Redirect stripping query string
SERVER_PATH_PREFIX=        # Server path prefix
SERVER_ACCESS_LOG=1        # Enable access log
```

## imagor Core

```dotenv
IMAGOR_SECRET=             # Secret key for URL signing
IMAGOR_UNSAFE=1            # Disable URL signature check (development only)

IMAGOR_AUTO_WEBP=1         # Serve WebP automatically if browser supports
IMAGOR_AUTO_AVIF=1         # Serve AVIF automatically if browser supports (experimental)
IMAGOR_AUTO_JPEG=1         # Serve JPEG automatically if JPEG or no format requested

IMAGOR_BASE_PARAMS=        # Base params applied to all images e.g. filters:watermark(logo.png)
IMAGOR_SIGNER_TYPE=sha1    # URL signature algorithm: sha1, sha256, sha512 (default sha1)
IMAGOR_SIGNER_TRUNCATE=    # Truncate URL signature to this length

IMAGOR_RESULT_STORAGE_PATH_STYLE=original  # Result storage path style: original, digest, suffix
IMAGOR_STORAGE_PATH_STYLE=original         # Loader storage path style: original, digest

IMAGOR_CACHE_HEADER_TTL=168h              # HTTP Cache-Control max-age for successful responses (default 7d)
IMAGOR_CACHE_HEADER_SWR=24h              # HTTP Cache-Control stale-while-revalidate (default 24h)
IMAGOR_CACHE_HEADER_NO_CACHE=1           # Set Cache-Control: no-cache on responses

IMAGOR_REQUEST_TIMEOUT=30s    # Overall request timeout (default 30s)
IMAGOR_LOAD_TIMEOUT=          # Loader fetch timeout (should be < request timeout)
IMAGOR_SAVE_TIMEOUT=          # Storage save timeout
IMAGOR_PROCESS_TIMEOUT=       # Image processing timeout

IMAGOR_PROCESS_CONCURRENCY=-1   # Max concurrent image operations (-1 = unlimited)
IMAGOR_PROCESS_QUEUE_SIZE=0     # Max queued requests before returning 429 (0 = unlimited)

IMAGOR_BASE_PATH_REDIRECT=     # Redirect / base path to this URL
IMAGOR_MODIFIED_TIME_CHECK=1   # Compare result vs source modified time to avoid stale results
IMAGOR_DISABLE_PARAMS_ENDPOINT=1  # Disable /params debug endpoint
IMAGOR_DISABLE_ERROR_BODY=1    # Omit response body on errors
IMAGOR_RESPONSE_RAW_ON_ERROR=1 # Return raw source image on processing error
```

## HTTP Loader

```dotenv
HTTP_LOADER_ALLOWED_SOURCES=*.github.com,*.example.com  # Allowlist of hosts (glob, csv)
HTTP_LOADER_ALLOWED_SOURCE_REGEXP=                       # Allowlist of hosts (regexp, OR-ed with glob)
HTTP_LOADER_BASE_URL=                  # Prepend this base URL to all image paths
HTTP_LOADER_DEFAULT_SCHEME=https       # Default scheme when not specified in path. Set "nil" to disable

HTTP_LOADER_FORWARD_HEADERS=           # Forward these request headers to loader (csv)
HTTP_LOADER_OVERRIDE_RESPONSE_HEADERS= # Copy these loader response headers to image response (csv)
HTTP_LOADER_FORWARD_CLIENT_HEADERS=1   # Forward all browser client headers to loader

HTTP_LOADER_ACCEPT=*/*                 # Set Accept header and validate Content-Type response
HTTP_LOADER_MAX_ALLOWED_SIZE=          # Max response size in bytes (0 = unlimited)
HTTP_LOADER_INSECURE_SKIP_VERIFY_TRANSPORT=1  # Skip TLS verification (not recommended)

HTTP_LOADER_PROXY_URLS=                # Proxy URLs for loader (csv). Proxy is only used if set
HTTP_LOADER_PROXY_ALLOWED_SOURCES=     # Hosts that use proxy transport (glob, csv)

HTTP_LOADER_BLOCK_LOOPBACK_NETWORKS=1  # Block loopback addresses (127.0.0.0/8, ::1)
HTTP_LOADER_BLOCK_LINK_LOCAL_NETWORKS=1  # Block link-local addresses (169.254.0.0/16)
HTTP_LOADER_BLOCK_PRIVATE_NETWORKS=1   # Block private network addresses (RFC 1918)
HTTP_LOADER_BLOCK_NETWORKS=::1/128,127.0.0.0/8  # Block specific CIDRs (csv)

HTTP_LOADER_DISABLE=1                  # Disable HTTP Loader entirely
```

## File System

```dotenv
FILE_SAFE_CHARS=           # Characters excluded from path escaping. Set -- for no-op

# File Loader
FILE_LOADER_BASE_DIR=      # Base directory. Enables File Loader when set
FILE_LOADER_PATH_PREFIX=   # Path prefix for File Loader

# File Storage
FILE_STORAGE_BASE_DIR=     # Base directory. Enables File Storage when set
FILE_STORAGE_PATH_PREFIX=  # Path prefix for File Storage
FILE_STORAGE_MKDIR_PERMISSION=0755   # Directory permission (default 0755)
FILE_STORAGE_WRITE_PERMISSION=0666   # File write permission (default 0666)
FILE_STORAGE_EXPIRATION=   # Expiration duration e.g. 24h. Default no expiration

# File Result Storage
FILE_RESULT_STORAGE_BASE_DIR=      # Base directory. Enables File Result Storage when set
FILE_RESULT_STORAGE_PATH_PREFIX=   # Path prefix
FILE_RESULT_STORAGE_MKDIR_PERMISSION=0755
FILE_RESULT_STORAGE_WRITE_PERMISSION=0666
FILE_RESULT_STORAGE_EXPIRATION=    # Expiration duration e.g. 24h. Default no expiration
```

## AWS / S3

```dotenv
# Global AWS credentials (used for all S3 operations unless overridden)
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_SESSION_TOKEN=           # Optional temporary session token
AWS_REGION=

S3_ENDPOINT=                 # Override default S3 endpoint (for S3-compatible storage)
S3_SAFE_CHARS=               # Characters excluded from key escaping. Set -- for no-op
S3_FORCE_PATH_STYLE=1        # Use path-style addressing (s3.amazonaws.com/bucket/key)

S3_HTTP_MAX_IDLE_CONNS_PER_HOST=100  # S3 HTTP client max idle connections per host

# S3 Loader
S3_LOADER_BUCKET=            # S3 bucket. Enables S3 Loader when set
S3_LOADER_BASE_DIR=          # Key prefix directory
S3_LOADER_PATH_PREFIX=       # URL path prefix
S3_LOADER_BUCKET_ROUTER_CONFIG=  # YAML config for multi-bucket routing by pattern
S3_LOADER_ENDPOINT=          # Override S3 endpoint for Loader only

# Per-component AWS credential overrides for Loader
AWS_LOADER_ACCESS_KEY_ID=
AWS_LOADER_SECRET_ACCESS_KEY=
AWS_LOADER_SESSION_TOKEN=
AWS_LOADER_REGION=

# S3 Storage
S3_STORAGE_BUCKET=           # S3 bucket. Enables S3 Storage when set
S3_STORAGE_BASE_DIR=
S3_STORAGE_PATH_PREFIX=
S3_STORAGE_ACL=public-read   # Upload ACL (default public-read)
S3_STORAGE_CLASS=STANDARD    # Storage class: STANDARD (default), REDUCED_REDUNDANCY, STANDARD_IA, ONEZONE_IA, INTELLIGENT_TIERING, GLACIER, DEEP_ARCHIVE
S3_STORAGE_EXPIRATION=       # Expiration duration e.g. 24h. Default no expiration
S3_STORAGE_ENDPOINT=         # Override S3 endpoint for Storage only

# Per-component AWS credential overrides for Storage
AWS_STORAGE_ACCESS_KEY_ID=
AWS_STORAGE_SECRET_ACCESS_KEY=
AWS_STORAGE_SESSION_TOKEN=
AWS_STORAGE_REGION=

# S3 Result Storage
S3_RESULT_STORAGE_BUCKET=    # S3 bucket. Enables S3 Result Storage when set
S3_RESULT_STORAGE_BASE_DIR=
S3_RESULT_STORAGE_PATH_PREFIX=
S3_RESULT_STORAGE_ACL=public-read
S3_RESULT_STORAGE_EXPIRATION=
S3_RESULT_STORAGE_ENDPOINT=

# Per-component AWS credential overrides for Result Storage
AWS_RESULT_STORAGE_ACCESS_KEY_ID=
AWS_RESULT_STORAGE_SECRET_ACCESS_KEY=
AWS_RESULT_STORAGE_SESSION_TOKEN=
AWS_RESULT_STORAGE_REGION=
```

## Google Cloud Storage

```dotenv
GCLOUD_SAFE_CHARS=           # Characters excluded from key escaping. Set -- for no-op

# Google Cloud Loader
GCLOUD_LOADER_BUCKET=        # Bucket name. Enables Google Cloud Loader when set
GCLOUD_LOADER_BASE_DIR=
GCLOUD_LOADER_PATH_PREFIX=

# Google Cloud Storage
GCLOUD_STORAGE_BUCKET=       # Bucket name. Enables Google Cloud Storage when set
GCLOUD_STORAGE_BASE_DIR=
GCLOUD_STORAGE_PATH_PREFIX=
GCLOUD_STORAGE_ACL=          # Upload ACL
GCLOUD_STORAGE_EXPIRATION=   # Expiration duration e.g. 24h. Default no expiration

# Google Cloud Result Storage
GCLOUD_RESULT_STORAGE_BUCKET=  # Bucket name. Enables Google Cloud Result Storage when set
GCLOUD_RESULT_STORAGE_BASE_DIR=
GCLOUD_RESULT_STORAGE_PATH_PREFIX=
GCLOUD_RESULT_STORAGE_ACL=
GCLOUD_RESULT_STORAGE_EXPIRATION=
```

## Upload Loader

```dotenv
UPLOAD_LOADER_ENABLE=1                  # Enable POST upload (opt-in, trusted environments only)
UPLOAD_LOADER_MAX_ALLOWED_SIZE=33554432 # Max upload size in bytes (default 32 MiB)
UPLOAD_LOADER_ACCEPT=image/*            # Accepted Content-Type (default image/*)
UPLOAD_LOADER_FORM_FIELD_NAME=image     # Multipart form field name (default image)
```

## VIPS / Image Processing

```dotenv
VIPS_CONCURRENCY=1           # libvips thread count per operation. -1 = auto (all CPU cores)

# Safety limits
VIPS_MAX_WIDTH=              # Maximum image width in pixels
VIPS_MAX_HEIGHT=             # Maximum image height in pixels
VIPS_MAX_RESOLUTION=         # Maximum image resolution (width × height)
VIPS_UNLIMITED=1             # Bypass all resolution limits (not recommended for public endpoints)
VIPS_MAX_ANIMATION_FRAMES=   # Max animation frames to load. 1 = disable animation, -1 = unlimited
VIPS_MAX_FILTER_OPS=-1       # Max filter operations per request (-1 = unlimited)
VIPS_DISABLE_BLUR=1          # Disable all blur operations
VIPS_DISABLE_FILTERS=blur,watermark  # Disable specific filters (csv)

# Output
VIPS_MOZJPEG=1               # Use MozJPEG for JPEG encoding (requires imagor-mozjpeg build)
VIPS_AVIF_SPEED=5            # AVIF encode speed: 0 (smallest) to 9 (fastest). Default 5
VIPS_STRIP_METADATA=1        # Strip all metadata from output images

# Smart crop
VIPS_DETECTOR_PROBE_SIZE=400  # Probe image size for smart crop detection (default 400)

# libvips operation cache (rarely useful for imagor workloads — leave at defaults)
VIPS_MAX_CACHE_MEM=0         # libvips op cache max memory (0 = disabled)
VIPS_MAX_CACHE_SIZE=0        # libvips op cache max entries (0 = disabled)
VIPS_MAX_CACHE_FILES=0       # libvips op cache max open files (0 = disabled)

# In-memory pixel cache (see In-Memory Cache docs for details)
VIPS_CACHE_SIZE=0            # Cache budget in bytes. 0 = disabled (default)
VIPS_CACHE_MAX_WIDTH=2400    # Max image width to cache (default 2400)
VIPS_CACHE_MAX_HEIGHT=2000   # Max image height to cache (default 2000)
VIPS_CACHE_TTL=              # Cache entry TTL. 0 = no expiry (LRU eviction only)
VIPS_CACHE_FORMAT=pixel      # Cache format: pixel (default), png (lossless), webp (lossy)
```

## Monitoring

```dotenv
# Prometheus metrics
PROMETHEUS_BIND=:5000        # Address to expose Prometheus metrics. Disabled if not set
PROMETHEUS_PATH=/            # Metrics path (default /)

# Sentry error tracking
SENTRY_DSN=                  # Sentry DSN. Enables Sentry integration when set

# Logging
LOG_ECS=1                    # Use Elastic Common Schema (ECS) log format
```
