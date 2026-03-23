# Google Cloud Storage

## Docker Compose Example

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

## Wildcard Bucket (Dynamic Bucket from Path)

Google Cloud Storage supports the same `*` bucket paradigm as S3:

```dotenv
GCLOUD_LOADER_BUCKET=*          # enable GCS loader with dynamic bucket from path
GCLOUD_STORAGE_BUCKET=*         # enable GCS storage with dynamic bucket from path
GCLOUD_RESULT_STORAGE_BUCKET=*  # enable GCS result storage with dynamic bucket from path
```

A request for `/mysite-test/images/photo.jpg` will load `images/photo.jpg` from the `mysite-test` GCS bucket. The first path segment is always used as the bucket name and the remainder as the object key.
