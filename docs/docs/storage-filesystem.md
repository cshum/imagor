# File System

imagor supports local file system storage using mounted volumes for Loader, Storage, and Result Storage.

Enable each role by setting the corresponding base directory environment variable:

- `FILE_LOADER_BASE_DIR` — load source images from the local file system
- `FILE_STORAGE_BASE_DIR` — store source images in the local file system
- `FILE_RESULT_STORAGE_BASE_DIR` — store processed results to the local file system

## Path Escaping And Safe Chars

imagor normalizes file paths before using them for File Loader, Storage, or Result Storage. Reserved characters are escaped by default.

For safety, file paths containing dotfile-style segments such as `/.git` are rejected by default.

If your filenames contain literal reserved characters, allow them with `FILE_SAFE_CHARS`.

```dotenv
FILE_SAFE_CHARS=[]
```

Example: a file named `photos/aa[1].gif` requires `FILE_SAFE_CHARS=[]`.

To disable escaping entirely:

```dotenv
FILE_SAFE_CHARS=--
```

## Base Directory And Path Prefix

These settings control different parts of the lookup flow:

- `FILE_*_BASE_DIR` selects where imagor reads or writes on disk.
- `FILE_*_PATH_PREFIX` restricts which normalized request paths that role accepts.

imagor first normalizes the request path, checks that it starts with `FILE_*_PATH_PREFIX`, removes that prefix, and then joins the remaining path under `FILE_*_BASE_DIR`.

Use path prefixes when one file system mount contains multiple logical image trees and imagor should only handle one of them.

Example:

- Request path: `avatars/user-1.jpg`
- `FILE_STORAGE_PATH_PREFIX=avatars`
- `FILE_STORAGE_BASE_DIR=/mnt/data/source`
- Stored file path: `/mnt/data/source/user-1.jpg`

Settings:

- `FILE_LOADER_PATH_PREFIX`
- `FILE_STORAGE_PATH_PREFIX`
- `FILE_RESULT_STORAGE_PATH_PREFIX`

## Expiration

`FILE_STORAGE_EXPIRATION` and `FILE_RESULT_STORAGE_EXPIRATION` only make imagor treat older files as expired during retrieval, based on file modified time.

They do not delete old files from disk.

Example:

```dotenv
FILE_STORAGE_EXPIRATION=24h
FILE_RESULT_STORAGE_EXPIRATION=168h
```

If you want old files removed, use your own cleanup process, for example a cron job or another file system retention workflow.

## Docker Compose Example

This example summarizes the file storage settings described above in a single Docker Compose configuration.

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
      FILE_SAFE_CHARS: "[]" # optional - preserve literal brackets in filenames

      FILE_STORAGE_BASE_DIR: /mnt/data # enable file storage by specifying base dir
      FILE_STORAGE_MKDIR_PERMISSION: 0755 # optional
      FILE_STORAGE_WRITE_PERMISSION: 0666 # optional

      FILE_RESULT_STORAGE_BASE_DIR: /mnt/data/result # enable file result storage by specifying base dir
      FILE_RESULT_STORAGE_MKDIR_PERMISSION: 0755 # optional
      FILE_RESULT_STORAGE_WRITE_PERMISSION: 0666 # optional
      
    ports:
      - "8000:8000"
```
