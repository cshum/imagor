# POST Upload

imagor supports POST uploads for direct image processing and transformation. 

Upload functionality is an **opt-in feature** designed for **internal use** where imagor serves as a backend service in trusted environments with proper access controls, not for public-facing endpoints. 
When enabled, it requires both flags to be explicitly set:

- **Unsafe mode** (`IMAGOR_UNSAFE=1`) - disables URL signature verification
- **Upload Loader** (`UPLOAD_LOADER_ENABLE=1`) - enables POST upload functionality

Usage:

```bash
docker run -p 8000:8000 shumc/imagor -imagor-unsafe -upload-loader-enable
```

```dotenv
IMAGOR_UNSAFE=1
UPLOAD_LOADER_ENABLE=1
```

Upload an image using POST request to any imagor endpoint. The URL path defines the image operations to apply:

```bash
# Upload and resize to 300x200
curl -X POST -F "image=@photo.jpg" http://localhost:8000/unsafe/300x200/

# Upload with filters applied
curl -X POST -F "image=@photo.jpg" \
  http://localhost:8000/unsafe/fit-in/400x300/filters:quality(80):format(webp)/
```

When upload is enabled, visiting processing paths in a browser shows a built-in upload form:

- `http://localhost:8000/unsafe/200x200/` - Upload form with 200x200 resize
- `http://localhost:8000/unsafe/filters:blur(5)/` - Upload form with blur filter

The upload form includes debug information showing how imagor parses the URL parameters, useful for testing and development.

## Security Considerations

:::danger Important Security Notice
Upload functionality should only be enabled in trusted, internal environments. Never expose upload endpoints to public internet without proper authentication and authorization.
:::

### Required Configuration

Both settings must be explicitly enabled:

```bash
# Command line
imagor -imagor-unsafe -upload-loader-enable

# Environment variables
IMAGOR_UNSAFE=1
UPLOAD_LOADER_ENABLE=1
```

### Upload Limits

Configure upload size limits and accepted file types:

```dotenv
UPLOAD_LOADER_MAX_ALLOWED_SIZE=33554432  # 32MB default
UPLOAD_LOADER_ACCEPT=image/*             # Accept all image types
UPLOAD_LOADER_FORM_FIELD_NAME=image     # Form field name
```
