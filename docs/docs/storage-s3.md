# AWS S3

imagor supports AWS S3 for Loader, Storage, and Result Storage. It is also compatible with S3-compatible services such as Cloudflare R2, MinIO and DigitalOcean Spaces.

Enable each role by setting the corresponding bucket environment variable:

- `S3_LOADER_BUCKET` — load source images from S3
- `S3_STORAGE_BUCKET` — cache source images to S3
- `S3_RESULT_STORAGE_BUCKET` — store processed results to S3

## Docker Compose Example

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

## Custom S3 Endpoint

Configure custom S3 endpoint for S3-compatible services such as Cloudflare R2, MinIO, DigitalOcean Spaces:

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

## Different AWS Credentials for S3 Loader, Storage and Result Storage

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

## S3 Wildcard Bucket (Dynamic Bucket from Path)

For setups where the bucket name is embedded as the first path segment of the image URL, set the bucket to `*`:

```dotenv
AWS_REGION=us-east-1

S3_LOADER_BUCKET=*          # enable S3 loader with dynamic bucket from path
S3_STORAGE_BUCKET=*         # enable S3 storage with dynamic bucket from path
S3_RESULT_STORAGE_BUCKET=*  # enable S3 result storage with dynamic bucket from path
```

A request for `/mysite-test/images/photo.jpg` will load `images/photo.jpg` from the `mysite-test` bucket. A request for `/mysite-prod/assets/logo.png` will load `assets/logo.png` from the `mysite-prod` bucket. The first path segment is always used as the bucket name and the remainder as the object key.

This works identically for loader, storage, and result storage — all three use the same `S3Storage` implementation. This allows a single imagor instance to serve images from any bucket in the same AWS account without any additional configuration.

## S3 Loader Bucket Routing

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
