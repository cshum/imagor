# Security

In production, always set `IMAGOR_SECRET` to require URL signatures on every request. `IMAGOR_UNSAFE` bypasses signature verification and should only be used during development.

## URL Signature

Set `IMAGOR_SECRET` to require every request URL to carry a valid HMAC signature. This prevents DDoS attacks that abuse arbitrary image operations and stops unauthenticated use of your imagor instance. Do not use `IMAGOR_UNSAFE` in production — it disables signature verification entirely.

The signature hash is computed from the URL path (excluding `/unsafe/`) using the secret, then Base64 URL encoded and prepended to the path:

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

### URL Expiry

Use the [`expire`](./filters#expiretimestamp) filter to give a signed URL a hard expiry time. The timestamp is unix milliseconds.

```javascript
// URL expires in 5 minutes
const expiry = Date.now() + 5 * 60 * 1000;
const path = `500x500/filters:expire(${expiry})/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png`;
console.log(sign(path, 'mysecret'));
```

imagor rejects requests whose `expire` timestamp is in the past, even if the signature is otherwise valid.

---

## Custom HMAC Signer

imagor uses SHA1 HMAC by default, the same algorithm used by [thumbor](https://thumbor.readthedocs.io/en/latest/security.html#hmac-method). SHA1 is not considered cryptographically secure today. Use `sha256` or `sha512` and optionally truncate the hash length:

```dotenv
IMAGOR_SIGNER_TYPE=sha256
IMAGOR_SIGNER_TRUNCATE=40
```

The signing function then becomes:

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

---

## Allowed Sources

Restricting which hosts the HTTP Loader can fetch from is an important measure against SSRF and open-proxy abuse.

**Glob allowlist** — comma-separated glob patterns:

```dotenv
HTTP_LOADER_ALLOWED_SOURCES=*.mydomain.com,assets.cdn.io,s3.amazonaws.com
```

**Regex allowlist** — full regular expression:

```dotenv
HTTP_LOADER_ALLOWED_SOURCE_REGEXP=^([\w-]+\.)?mydomain\.com$
```

**Base URL lock** — restrict loading to a single origin and shorten image URLs:

```dotenv
HTTP_LOADER_BASE_URL=https://assets.mydomain.com/
```

With a base URL set the image path is appended to it, so `500x500/photo.jpg` loads `https://assets.mydomain.com/photo.jpg`.

**HTTPS only** — refuse plain-HTTP image sources:

```dotenv
HTTP_LOADER_HTTPS_ONLY=1
```

See [HTTP Loader](./loader-http) for the full configuration reference.

---

## Image Bombs Prevention

imagor checks the image type and resolution before processing begins. Requests are rejected when image dimensions exceed configured limits, protecting against "image bomb" attacks:

```dotenv
VIPS_MAX_RESOLUTION=16800000
VIPS_MAX_WIDTH=5000
VIPS_MAX_HEIGHT=5000
```

---

## Timeouts and Concurrency

Limit per-request budget and parallel processing capacity to defend against slowloris and resource exhaustion:

```dotenv
IMAGOR_REQUEST_TIMEOUT=30s          # total request budget
IMAGOR_LOAD_TIMEOUT=10s             # loader fetch must complete within this time
IMAGOR_PROCESS_CONCURRENCY=20       # max simultaneous image processing jobs
IMAGOR_PROCESS_QUEUE_SIZE=100       # max queued jobs before 429 is returned
```

---

## Production Setup

A minimal production setup bringing together the key configurations above:

```yaml
version: "3"
services:
  imagor:
    image: shumc/imagor:latest
    environment:
      PORT: 8000

      IMAGOR_SECRET: your-secret-here       # required — signs all URLs
      IMAGOR_SIGNER_TYPE: sha256            # stronger hash than the default SHA1
      IMAGOR_SIGNER_TRUNCATE: 40           # truncate to 40 chars

      HTTP_LOADER_ALLOWED_SOURCES: "*.mydomain.com,assets.cdn.io"  # restrict sources
      HTTP_LOADER_HTTPS_ONLY: 1            # refuse plain-HTTP image URLs

      VIPS_MAX_RESOLUTION: 16800000        # reject images over ~4K×4K
      VIPS_MAX_WIDTH: 5000
      VIPS_MAX_HEIGHT: 5000

      IMAGOR_REQUEST_TIMEOUT: 30s
      IMAGOR_LOAD_TIMEOUT: 10s
      IMAGOR_PROCESS_CONCURRENCY: 20       # cap simultaneous processing
    ports:
      - "8000:8000"
```
