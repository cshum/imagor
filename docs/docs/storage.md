# Loader, Storage, Result Storage

imagor `Loader`, `Storage` and `Result Storage` are the building blocks for loading and saving images from various sources:

- `Loader` loads image. Enable `Loader` where you wish to load images from, but without modifying it e.g. static directory.
- `Storage` loads and saves image. This allows subsequent requests for the same image loads directly from the storage, instead of HTTP source.
- `Result Storage` loads and saves the processed image. This allows subsequent request of the same parameters loads from the result storage, saving processing resources.

imagor provides built-in adaptors that support HTTP(s), Proxy, File System, AWS S3 and Google Cloud Storage. By default, `HTTP Loader` is used as fallback. You can choose to enable additional adaptors that fit your use cases.

## Loader

- [HTTP Loader](./loader-http) — Default loader for remote HTTP/HTTPS images

## Storage

- [File System](./storage-filesystem) — Local file system storage using mounted volumes
- [AWS S3](./storage-s3) — Amazon S3 and S3-compatible storage (MinIO, DigitalOcean Spaces)
- [Google Cloud Storage](./storage-gcloud) — Google Cloud Storage buckets

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
