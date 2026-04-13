# Storage and Result Storage Path Style

`Storage` and `Result Storage` path style enables additional hashing rules to the storage path when loading and saving images. By default (`original`), the image path is used as-is.

## Storage Path Style

`IMAGOR_STORAGE_PATH_STYLE` controls the key used when loading or saving the source image in storage. Accepts `original` (default) or `digest`.

### `original` (default)

The image path is used as the storage key unchanged.

### `digest`

SHA1 hashes the image path into a `xx/xx/hash` directory structure, distributing files evenly across subdirectories. Useful for high-volume storage on filesystems or object stores that benefit from directory sharding.

```
IMAGOR_STORAGE_PATH_STYLE=digest
```

- `foobar.jpg` becomes `e6/86/1a810ff186b4f747ef85f7c53946f0e6d8cb`

---

## Result Storage Path Style

`IMAGOR_RESULT_STORAGE_PATH_STYLE` controls the key used when loading or saving processed results in result storage. Accepts `original` (default), `digest`, `suffix`, or `size`.

### `original` (default)

The full request path including processing parameters is used as the result storage key unchanged.

### `digest`

SHA1 hashes the full request path (including processing parameters) into a `xx/xx/hash` directory structure.

```
IMAGOR_RESULT_STORAGE_PATH_STYLE=digest
```

- `fit-in/16x17/foobar.jpg` becomes `61/4c/9ba1725e8cdd8263a4ad437c56b35f33deba`

### `suffix`

Preserves the original image path and filename, appending a short SHA1 hash as a suffix before the file extension. This keeps result keys human-readable while remaining unique per request parameters.

If a [`format()`](./filters.md#formatformat) filter is applied, the extension reflects the output format instead of the original.

```
IMAGOR_RESULT_STORAGE_PATH_STYLE=suffix
```

- `166x169/top/foobar.jpg` becomes `foobar.45d8ebb31bd4ed80c26e.jpg`
- `17x19/smart/example.com/foobar` becomes `example.com/foobar.ddd349e092cda6d9c729`

### `size`

Like `suffix`, but also appends the output dimensions (`_WxH`) to the hash. Useful when the same source image is served at multiple sizes and the result storage key should reflect the output dimensions.

```
IMAGOR_RESULT_STORAGE_PATH_STYLE=size
```

- `166x169/top/foobar.jpg` becomes `foobar.45d8ebb31bd4ed80c26e_166x169.jpg`
- `17x19/smart/example.com/foobar` becomes `example.com/foobar.ddd349e092cda6d9c729_17x19`
