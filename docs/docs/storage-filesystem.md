# File System

imagor supports local file system storage using mounted volumes for Loader, Storage, and Result Storage.

Enable each role by setting the corresponding base directory environment variable:

- `FILE_LOADER_BASE_DIR` — load source images from the local file system
- `FILE_STORAGE_BASE_DIR` — cache source images to the local file system
- `FILE_RESULT_STORAGE_BASE_DIR` — store processed results to the local file system

## Docker Compose Example

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
