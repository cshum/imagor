# Storage

## Loader, Storage and Result Storage

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
