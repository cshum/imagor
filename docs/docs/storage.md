# Loader, Storage, Result Storage

imagor `Loader`, `Storage` and `Result Storage` are the building blocks for loading and saving images from various sources:

- `Loader` loads image. Enable `Loader` where you wish to load images from, but without modifying it e.g. static directory.
- `Storage` loads and saves image. This allows subsequent requests for the same image loads directly from the storage, instead of HTTP source.
- `Result Storage` loads and saves the processed image. This allows subsequent request of the same parameters loads from the result storage, saving processing resources.

imagor provides built-in adaptors that support HTTP(s), Proxy, File System, AWS S3 and Google Cloud Storage. By default, `HTTP Loader` is used as fallback. You can choose to enable additional adaptors that fit your use cases.

## Loader

- [HTTP Loader](./loader-http.md) — Default loader for remote HTTP/HTTPS images

## Storage

- [File System](./storage-filesystem.md) — Local file system storage using mounted volumes
- [AWS S3](./storage-s3.md) — Amazon S3 and S3-compatible storage (Cloudflare R2, MinIO, DigitalOcean Spaces)
- [Google Cloud Storage](./storage-gcloud.md) — Google Cloud Storage buckets

## Storage Key Normalization

imagor normalizes image keys before passing them to File System, S3, or Google Cloud Storage.

By default, imagor preserves alphanumeric characters, `/`, and the standard unreserved URL characters `-`, `_`, `.`, and `~`. Other characters are escaped unless you allow them with the backend-specific safe chars setting.

This keeps storage keys portable across backends, but literal filenames or object keys containing reserved characters such as `[` and `]` will not match unless those characters are configured as safe.

Use the backend-specific safe chars setting to preserve additional literal characters:

- `FILE_SAFE_CHARS`
- `S3_SAFE_CHARS`
- `GCLOUD_SAFE_CHARS`

For example, to preserve literal brackets in source keys:

```dotenv
FILE_SAFE_CHARS=[]
S3_SAFE_CHARS=[]
GCLOUD_SAFE_CHARS=[]
```

To disable escaping entirely, set the safe chars value to `--`.

## [Storage and Result Storage Path Style](./storage-path-style.md)

Enables additional hashing rules to the storage key when loading and saving images. Accepts `original` (default), `digest`, `suffix`, or `size`.
