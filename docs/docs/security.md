# Security

## URL Signature

In production environment, it is highly recommended turning off `IMAGOR_UNSAFE` and setting up URL signature using `IMAGOR_SECRET`, to prevent DDoS attacks that abuse multiple image operations.

The URL signature hash is based on SHA digest, created by taking the URL path (excluding `/unsafe/`) with secret. The hash is then Base64 URL encoded.
An example in Node.js:

```javascript
const crypto = require('crypto');

function sign(path, secret) {
  const hash = crypto.createHmac('sha1', secret)
          .update(path)
          .digest('base64')
          .replace(/\+/g, '-').replace(/\//g, '_')
  return hash + '/' + path
}

console.log(sign('500x500/top/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png', 'mysecret'))
// cST4Ko5_FqwT3BDn-Wf4gO3RFSk=/500x500/top/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

## Custom HMAC Signer

imagor uses SHA1 HMAC signer by default, the same one used by [thumbor](https://thumbor.readthedocs.io/en/latest/security.html#hmac-method). However, SHA1 is not considered cryptographically secure. If that is a concern it is possible to configure different signing method and truncate length. imagor supports `sha1`, `sha256`, `sha512` signer type:

```dotenv
IMAGOR_SIGNER_TYPE=sha256
IMAGOR_SIGNER_TRUNCATE=40
```

The Node.js example then becomes:

```javascript
const crypto = require('crypto');

function sign(path, secret) {
  const hash = crypto.createHmac('sha256', secret)
          .update(path)
          .digest('base64')
          .slice(0, 40)
          .replace(/\+/g, '-').replace(/\//g, '_')
  return hash + '/' + path
}

console.log(sign('500x500/top/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png', 'mysecret'))
// IGEn3TxngivD0jy4uuiZim2bdUCvhcnVi1Nm0xGy/500x500/top/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

## Image Bombs Prevention

imagor checks the image type and its resolution before the actual processing happens. The processing will be rejected if the image dimensions are too big, which protects from so-called "image bombs". You can set the max allowed image resolution and dimensions using `VIPS_MAX_RESOLUTION`, `VIPS_MAX_WIDTH`, `VIPS_MAX_HEIGHT`:

```dotenv
VIPS_MAX_RESOLUTION=16800000
VIPS_MAX_WIDTH=5000
VIPS_MAX_HEIGHT=5000
```

## Allowed Sources and Base URL

Whitelist specific hosts to restrict loading images only from the allowed sources using `HTTP_LOADER_ALLOWED_SOURCES` or `HTTP_LOADER_ALLOWED_SOURCE_REGEXP`.

- `HTTP_LOADER_ALLOWED_SOURCES` accepts csv wth glob pattern e.g.:

  ```dotenv
  HTTP_LOADER_ALLOWED_SOURCES=*.foobar.com,my.foobar.com,mybucket.s3.amazonaws.com
  ```

- `HTTP_LOADER_ALLOWED_SOURCE_REGEXP` accepts a regular expression matching on the full URL e.g.:

  ```dotenv
  HTTP_LOADER_ALLOWED_SOURCE_REGEXP='^https://raw\.githubusercontent\.com/cshum/imagor/.*'
  ```

Alternatively, it is possible to set a base URL for loading images strictly from one HTTP source. This also trims down the base URL from image endpoint:

Example URL:
```
http://localhost:8000/unsafe/fit-in/200x150/filters:fill(yellow):watermark(raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png,repeat,bottom,0,40,40)/raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif
```

With HTTP Loader Base URL config:
```
HTTP_LOADER_BASE_URL=https://raw.githubusercontent.com/cshum/imagor/master
```

The example URL then becomes:
```
http://localhost:8000/unsafe/fit-in/200x150/filters:fill(yellow):watermark(testdata/gopher-front.png,repeat,bottom,0,40,40)/testdata/dancing-banana.gif
