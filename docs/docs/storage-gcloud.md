# Google Cloud Storage

imagor supports Google Cloud Storage for Loader, Storage, and Result Storage.

Enable each role by setting the corresponding bucket environment variable:

- `GCLOUD_LOADER_BUCKET` — load source images from GCS
- `GCLOUD_STORAGE_BUCKET` — store source images in GCS
- `GCLOUD_RESULT_STORAGE_BUCKET` — store processed results to GCS

## Base Directory And Path Prefix

These settings control different parts of the lookup flow:

- `GCLOUD_*_BASE_DIR` selects the key prefix inside the bucket where imagor reads or writes.
- `GCLOUD_*_PATH_PREFIX` restricts which normalized request paths that role accepts.

imagor first normalizes the request path, checks that it starts with `GCLOUD_*_PATH_PREFIX`, removes that prefix, and then joins the remaining path under `GCLOUD_*_BASE_DIR` to build the final object key.

Use them together when one bucket contains multiple logical path trees and imagor should only handle one of them.

Example:

- Request path: `avatars/user-1.jpg`
- `GCLOUD_STORAGE_PATH_PREFIX=avatars`
- `GCLOUD_STORAGE_BASE_DIR=source`
- Stored object key: `source/user-1.jpg`

Settings:

- `GCLOUD_LOADER_BASE_DIR`
- `GCLOUD_STORAGE_BASE_DIR`
- `GCLOUD_RESULT_STORAGE_BASE_DIR`
- `GCLOUD_LOADER_PATH_PREFIX`
- `GCLOUD_STORAGE_PATH_PREFIX`
- `GCLOUD_RESULT_STORAGE_PATH_PREFIX`

## Key Escaping And Safe Chars

imagor normalizes object keys before using them for Google Cloud Loader, Storage, or Result Storage. Reserved characters are escaped by default.

If your object keys contain literal reserved characters such as `[` and `]`, allow them with `GCLOUD_SAFE_CHARS`:

```dotenv
GCLOUD_SAFE_CHARS=[]
```

Example: an object stored as `images/aa[1].gif` requires `GCLOUD_SAFE_CHARS=[]`.

To disable escaping entirely:

```dotenv
GCLOUD_SAFE_CHARS=--
```

## ACL

`GCLOUD_STORAGE_ACL` and `GCLOUD_RESULT_STORAGE_ACL` are optional.

Set these only when your Google Cloud Storage bucket policy requires a specific predefined ACL on writes.

## Expiration

`GCLOUD_STORAGE_EXPIRATION` and `GCLOUD_RESULT_STORAGE_EXPIRATION` only make imagor treat older objects as expired during retrieval, based on the object updated time.

They do not delete old objects from the bucket.

Example:

```dotenv
GCLOUD_STORAGE_EXPIRATION=24h
GCLOUD_RESULT_STORAGE_EXPIRATION=168h
```

If you want old objects removed, use Google Cloud Storage lifecycle management or your own bucket retention workflow.

## Wildcard Bucket (Dynamic Bucket from Path)

Google Cloud Storage supports the same `*` bucket paradigm as S3:

```dotenv
GCLOUD_LOADER_BUCKET=*          # enable GCS loader with dynamic bucket from path
GCLOUD_STORAGE_BUCKET=*         # enable GCS storage with dynamic bucket from path
GCLOUD_RESULT_STORAGE_BUCKET=*  # enable GCS result storage with dynamic bucket from path
```

A request for `/mysite-test/images/photo.jpg` will load `images/photo.jpg` from the `mysite-test` GCS bucket. The first path segment is always used as the bucket name and the remainder as the object key.

## Docker Compose Example

This example summarizes the Google Cloud Storage settings described above in a single Docker Compose configuration.

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
      GCLOUD_SAFE_CHARS: "[]" # optional - preserve literal brackets in object keys

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
