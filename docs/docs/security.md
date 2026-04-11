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

Restricting which hosts the HTTP Loader can fetch from is an important security measure. Use `HTTP_LOADER_ALLOWED_SOURCES` (glob pattern) or `HTTP_LOADER_ALLOWED_SOURCE_REGEXP` (regex) to whitelist allowed sources. You can also set `HTTP_LOADER_BASE_URL` to lock loading to a single origin and keep image URLs shorter.

See [HTTP Loader](./loader-http) for full configuration details.
