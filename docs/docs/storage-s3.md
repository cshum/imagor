# AWS S3

imagor supports AWS S3 for Loader, Storage, and Result Storage. It is also compatible with S3-compatible services such as Cloudflare R2, MinIO and DigitalOcean Spaces.

Enable each role by setting the corresponding bucket environment variable:

- `S3_LOADER_BUCKET` — load source images from S3
- `S3_STORAGE_BUCKET` — store source images in S3
- `S3_RESULT_STORAGE_BUCKET` — store processed results to S3

## Key Escaping And Safe Chars

imagor normalizes object keys before using them for S3 Loader, Storage, or Result Storage. In addition to alphanumeric characters, `/`, `-`, `_`, `.`, and `~`, imagor also preserves `!"()*` for S3 by default. Other characters are escaped unless allowed with `S3_SAFE_CHARS`.

For AWS object key guidance, see the AWS documentation on [safe characters in object keys](https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-keys.html#object-key-guidelines-safe-characters).

For result storage with the default path style, the processed image key is based on the request path in normalized form.

Example:

```text
fit-in/1800x1800/filters:format(jpeg):quality(90)/test.jpg
```

becomes:

```text
fit-in/1800x1800/filters%3Aformat%28jpeg%29%3Aquality%2890%29/test.jpg
```

This is because `:()` are escaped by default.

If your object keys contain literal reserved characters such as `[` and `]`, allow them with:

```dotenv
S3_SAFE_CHARS=[]
```

Example: an object stored as `images/aa[1].gif` requires `S3_SAFE_CHARS=[]`.

To disable escaping entirely:

```dotenv
S3_SAFE_CHARS=--
```

For storage key naming options beyond safe chars, see [Storage and Result Storage Path Style](./storage-path-style.md).

## Base Directory And Path Prefix

These settings control different parts of the lookup flow:

- `S3_*_BASE_DIR` selects the key prefix inside the bucket where imagor reads or writes.
- `S3_*_PATH_PREFIX` restricts which normalized request paths that role accepts.

imagor first normalizes the request path, checks that it starts with `S3_*_PATH_PREFIX`, removes that prefix, and then joins the remaining path under `S3_*_BASE_DIR` to build the final object key.

Use them together when one bucket contains multiple logical path trees and imagor should only handle one of them.

Example:

- Request path: `avatars/user-1.jpg`
- `S3_STORAGE_PATH_PREFIX=avatars`
- `S3_STORAGE_BASE_DIR=source`
- Stored object key: `source/user-1.jpg`

Settings:

- `S3_LOADER_BASE_DIR`
- `S3_STORAGE_BASE_DIR`
- `S3_RESULT_STORAGE_BASE_DIR`
- `S3_LOADER_PATH_PREFIX`
- `S3_STORAGE_PATH_PREFIX`
- `S3_RESULT_STORAGE_PATH_PREFIX`

## Storage Class

`S3_STORAGE_CLASS` controls the storage class used when saving source and result objects.

Supported values are `STANDARD`, `REDUCED_REDUNDANCY`, `STANDARD_IA`, `ONEZONE_IA`, `INTELLIGENT_TIERING`, `GLACIER`, and `DEEP_ARCHIVE`.

## ACL

`S3_STORAGE_ACL` and `S3_RESULT_STORAGE_ACL` are optional.

By default, imagor does not send an ACL header on S3 writes. Set these only when your bucket policy and S3 backend require a specific canned ACL.

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
AWS_LOADER_SESSION_TOKEN
S3_LOADER_ENDPOINT

AWS_STORAGE_REGION
AWS_STORAGE_ACCESS_KEY_ID
AWS_STORAGE_SECRET_ACCESS_KEY
AWS_STORAGE_SESSION_TOKEN
S3_STORAGE_ENDPOINT

AWS_RESULT_STORAGE_REGION
AWS_RESULT_STORAGE_ACCESS_KEY_ID
AWS_RESULT_STORAGE_SECRET_ACCESS_KEY
AWS_RESULT_STORAGE_SESSION_TOKEN
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

For multi-tenant or multi-bucket setups, you can route image requests to different S3 buckets based on pattern matching. Each bucket can have its own region, endpoint, and credentials.

Activate bucket routing by setting:

```dotenv
S3_LOADER_BUCKET_ROUTER_CONFIG=/path/to/bucket-routing.yaml
```

Then create the YAML configuration file:

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

## Expiration

`S3_STORAGE_EXPIRATION` and `S3_RESULT_STORAGE_EXPIRATION` only make imagor treat older objects as expired during retrieval, based on the object last modified time.

They do not delete old objects from S3.

Example:

```dotenv
S3_STORAGE_EXPIRATION=24h
S3_RESULT_STORAGE_EXPIRATION=168h
```

If you want old objects removed, use [S3 lifecycle configuration](https://docs.aws.amazon.com/AmazonS3/latest/userguide/lifecycle-expire-general-considerations.html) or the equivalent retention feature in your S3-compatible storage.

## Object Tagging For S3 Writes

`S3_STORAGE_TAGGING` and `S3_RESULT_STORAGE_TAGGING` let imagor attach S3 object tags when writing source or result objects.

The value must use the same query-string format accepted by the S3 `PutObject` Tagging field.

Enable this only on AWS S3 or another backend you have verified supports S3 object tagging on `PutObject`.

Example:

```dotenv
S3_STORAGE_TAGGING=source=imagor&lifecycle=generated
S3_RESULT_STORAGE_TAGGING=source=imagor&lifecycle=derived
```

This is mainly useful when you want downstream automation or lifecycle rules to distinguish imagor-managed objects from everything else in the bucket.

For example, you can tag generated or derived objects separately from original assets and use those tags in S3 lifecycle policies.

## Docker Compose Example

This example summarizes the S3 storage settings described above in a single Docker Compose configuration.

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
      S3_SAFE_CHARS: "[]" # optional - preserve literal brackets in object keys

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
