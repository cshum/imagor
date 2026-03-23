# Storage

imagor `Loader`, `Storage` and `Result Storage` are the building blocks for loading and saving images from various sources:

- `Loader` loads image. Enable `Loader` where you wish to load images from, but without modifying it e.g. static directory.
- `Storage` loads and saves image. This allows subsequent requests for the same image loads directly from the storage, instead of HTTP source.
- `Result Storage` loads and saves the processed image. This allows subsequent request of the same parameters loads from the result storage, saving processing resources.

imagor provides built-in adaptors that support HTTP(s), Proxy, File System, AWS S3 and Google Cloud Storage. By default, `HTTP Loader` is used as fallback. You can choose to enable additional adaptors that fit your use cases.

## File System

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
      FILE_STORAGE_MKDIR_PERMISSION: 0755 # optional
      FILE_STORAGE_WRITE_PERMISSION: 0666 # optional

      FILE_RESULT_STORAGE_BASE_DIR: /mnt/data/result # enable file result storage by specifying base dir
      FILE_RESULT_STORAGE_MKDIR_PERMISSION: 0755 # optional
      FILE_RESULT_STORAGE_WRITE_PERMISSION: 0666 # optional
      
    ports:
      - "8000:8000"
```

## AWS S3

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
      S3_STORAGE_ACL: public-read # optional - see https://docs.aws.amazon.com/AmazonS3/latest/userguide/acl-overview.html#canned-acl

      S3_RESULT_STORAGE_BUCKET: mybucket # enable S3 result storage by specifying bucket
      S3_RESULT_STORAGE_BASE_DIR: images/result # optional
      S3_RESULT_STORAGE_ACL: public-read # optional
    ports:
      - "8000:8000"
```

### Custom S3 Endpoint

Configure custom S3 endpoint for S3 compatible such as MinIO, DigitalOcean Space:

```yaml
      S3_ENDPOINT: http://minio:9000
      S3_FORCE_PATH_STYLE: 1
```

By default, S3 prepends bucket name as subdomain to the request URL:

```
http://mybucket.minio:9000/image.jpg
```

this may not be desirable for a self-hosted endpoint. You can also switch to [path-style requests](https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html#path-style-access) using `S3_FORCE_PATH_STYLE=1` such that the host remains unchanged:

```
http://minio:9000/mybucket/image.jpg
```

### Different AWS Credentials for S3 Loader, Storage and Result Storage

Set the following environment variables to override the global AWS Credentials for S3 Loader, Storage and Result Storage:

```dotenv
AWS_LOADER_REGION
AWS_LOADER_ACCESS_KEY_ID
AWS_LOADER_SECRET_ACCESS_KEY
S3_LOADER_ENDPOINT

AWS_STORAGE_REGION
AWS_STORAGE_ACCESS_KEY_ID
AWS_STORAGE_SECRET_ACCESS_KEY
S3_STORAGE_ENDPOINT

AWS_RESULT_STORAGE_REGION
AWS_RESULT_STORAGE_ACCESS_KEY_ID
AWS_RESULT_STORAGE_SECRET_ACCESS_KEY
S3_RESULT_STORAGE_ENDPOINT
```

### S3 Wildcard Bucket (Dynamic Bucket from Path)

For setups where the bucket name is embedded as the first path segment of the image URL, set the bucket to `*`:

```dotenv
AWS_REGION=us-east-1

S3_LOADER_BUCKET=*          # enable S3 loader with dynamic bucket from path
S3_STORAGE_BUCKET=*         # enable S3 storage with dynamic bucket from path
S3_RESULT_STORAGE_BUCKET=*  # enable S3 result storage with dynamic bucket from path
```

A request for `/mysite-test/images/photo.jpg` will load `images/photo.jpg` from the `mysite-test` bucket. A request for `/mysite-prod/assets/logo.png` will load `assets/logo.png` from the `mysite-prod` bucket. The first path segment is always used as the bucket name and the remainder as the object key.

This works identically for loader, storage, and result storage — all three use the same `S3Storage` implementation. This allows a single imagor instance to serve images from any bucket in the same AWS account without any additional configuration.

### S3 Loader Bucket Routing

For multi-tenant or multi-bucket setups, you can route image requests to different S3 buckets based on pattern matching. Each bucket can have its own region, endpoint, and credentials. Create a YAML configuration file:

```yaml
# Regex pattern with named capture group (?P<bucket>...)
# Extracts the bucket identifier from the object key.
# Optionally, add (?P<path>...) to extract the S3 key separately from the routing prefix.
routing_pattern: "^[a-f0-9]{4}-(?P<bucket>[A-Za-z0-9]+)-"

default_bucket:
  name: imagor-default
  region: us-east-1

fallback_buckets:
  - name: imagor-archive
    region: us-west-2

rules:
  - match: SG
    bucket:
      name: imagor-singapore
      region: ap-southeast-1
  - match: US
    bucket:
      name: imagor-us
      region: us-east-1
  - match: EU
    bucket:
      name: imagor-eu
      region: eu-west-1
      endpoint: https://s3.custom-endpoint.com
      access_key_id: AKIAIOSFODNN7EXAMPLE
      secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

**Pattern Examples:**

| Use Case | `routing_pattern` | Example Key | Bucket | S3 Key |
|----------|-------------------|-------------|--------|--------|
| Random prefix with bucket code | `^[a-f0-9]{4}-(?P<bucket>[A-Z]{2})-` | `f7a3-SG-image.jpg` | `SG` | `f7a3-SG-image.jpg` |
| Simple prefix routing | `^(?P<bucket>[^/]+)/` | `users/photo.jpg` | `users` | `users/photo.jpg` |
| Region-based naming | `(?P<bucket>[a-z]{2}-[a-z]+-\d)` | `eu-west-1-img.jpg` | `eu-west-1` | `eu-west-1-img.jpg` |
| Path-prefix routing (strip prefix) | `^(?P<bucket>mysite-[a-z]+)\/(?P<path>.+)$` | `mysite-test/images/photo.jpg` | `mysite-test` | `images/photo.jpg` |
| Passthrough (any bucket, no rules) | `^(?P<bucket>[^/]+)\/(?P<path>.+)$` | `any-bucket/images/photo.jpg` | `any-bucket` | `images/photo.jpg` |

Then specify the config file path:

```dotenv
S3_LOADER_BUCKET_ROUTER_CONFIG=/path/to/bucket-routing.yaml
```

Or via command line:

```bash
imagor -s3-loader-bucket-router-config /path/to/bucket-routing.yaml
```

Routing behavior:
- The `routing_pattern` must contain a named capture group `(?P<bucket>...)` to extract the bucket identifier
- The extracted value is matched against `rules[].match` to find the target bucket
- If no rule matches, the `default_bucket` is used
- If image not found in primary bucket, `fallback_buckets` are tried in order (up to 2 fallbacks)
- Each bucket config can specify its own `region`, `endpoint`, and credentials
- If bucket-specific credentials are not provided, global AWS credentials are used
- If `S3_LOADER_BUCKET` is not set, `default_bucket.name` from the config is used
- Optionally, add a named capture group `(?P<path>...)` to the pattern to use a sub-match as the S3 key instead of the full image path. This is useful for path-prefix routing where the bucket name is embedded as the first path segment and should not be included in the object key
- **Passthrough mode:** if no `rules` and no `default_bucket` are configured, the router uses the captured `(?P<bucket>...)` value directly as the bucket name, creating S3 clients on demand. This allows routing to any bucket without pre-declaring them in the YAML

Docker Compose example with bucket routing:

```yaml
version: "3"
services:
  imagor:
    image: shumc/imagor:latest
    volumes:
      - ./bucket-routing.yaml:/etc/imagor/bucket-routing.yaml
    environment:
      PORT: 8000
      IMAGOR_SECRET: mysecret
      AWS_ACCESS_KEY_ID: ...
      AWS_SECRET_ACCESS_KEY: ...
      AWS_REGION: us-east-1
      S3_LOADER_BUCKET_ROUTER_CONFIG: /etc/imagor/bucket-routing.yaml
    ports:
      - "8000:8000"
```

## Google Cloud Storage

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
      GCLOUD_STORAGE_ACL: publicRead # optional - see https://cloud.google.com/storage/docs/json_api/v1/objects/insert

      GCLOUD_RESULT_STORAGE_BUCKET: mybucket # enable result storage by specifying bucket
      GCLOUD_RESULT_STORAGE_BASE_DIR: images/result # optional
      GCLOUD_RESULT_STORAGE_ACL: publicRead # optional
    ports:
      - "8000:8000"
```

### Google Cloud Storage Wildcard Bucket (Dynamic Bucket from Path)

Google Cloud Storage supports the same `*` bucket paradigm as S3:

```dotenv
GCLOUD_LOADER_BUCKET=*          # enable GCS loader with dynamic bucket from path
GCLOUD_STORAGE_BUCKET=*         # enable GCS storage with dynamic bucket from path
GCLOUD_RESULT_STORAGE_BUCKET=*  # enable GCS result storage with dynamic bucket from path
```

A request for `/mysite-test/images/photo.jpg` will load `images/photo.jpg` from the `mysite-test` GCS bucket. The first path segment is always used as the bucket name and the remainder as the object key.

## Storage and Result Storage Path Style

`Storage` and `Result Storage` path style enables additional hashing rules to the storage path when loading and saving images:

`IMAGOR_STORAGE_PATH_STYLE=digest`

* `foobar.jpg` becomes `e6/86/1a810ff186b4f747ef85f7c53946f0e6d8cb`

`IMAGOR_RESULT_STORAGE_PATH_STYLE=digest`

* `fit-in/16x17/foobar.jpg` becomes `61/4c/9ba1725e8cdd8263a4ad437c56b35f33deba`

`IMAGOR_RESULT_STORAGE_PATH_STYLE=suffix`

* `166x169/top/foobar.jpg` becomes `foobar.45d8ebb31bd4ed80c26e.jpg`
* `17x19/smart/example.com/foobar` becomes `example.com/foobar.ddd349e092cda6d9c729`

`IMAGOR_RESULT_STORAGE_PATH_STYLE=size`

* `166x169/top/foobar.jpg` becomes `foobar.45d8ebb31bd4ed80c26e_166x169.jpg`
* `17x19/smart/example.com/foobar` becomes `example.com/foobar.ddd349e092cda6d9c729_17x19`
