# HTTP Loader

imagor includes an HTTP Loader that is **enabled by default**. It fetches source images from remote URLs over HTTP or HTTPS, and is used as the fallback loader when no other loader (e.g. File or S3) handles the request.

## Allowed Sources

By default, the HTTP Loader accepts requests to any URL. To restrict which hosts images can be loaded from, set the allowed sources:

- `HTTP_LOADER_ALLOWED_SOURCES` — comma-separated list of allowed hosts with glob pattern support, e.g. `*.example.com,images.mysite.com`
- `HTTP_LOADER_ALLOWED_SOURCE_REGEXP` — allowed source URL regexp, combined as OR with the glob pattern sources above

When either option is set, requests to unlisted sources will be rejected.

## Base URL

Set `HTTP_LOADER_BASE_URL` to prepend a fixed base URL to all image paths. This is useful for serving images from a single origin — it trims the base URL from the image endpoint, keeping URLs shorter.

For example, without base URL the full image path is required:

```
http://localhost:8000/unsafe/fit-in/200x150/filters:fill(yellow):watermark(raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png,repeat,bottom,0,40,40)/raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif
```

With `HTTP_LOADER_BASE_URL` configured:

```dotenv
HTTP_LOADER_BASE_URL=https://raw.githubusercontent.com/cshum/imagor/master
```

The URL becomes:

```
http://localhost:8000/unsafe/fit-in/200x150/filters:fill(yellow):watermark(testdata/gopher-front.png,repeat,bottom,0,40,40)/testdata/dancing-banana.gif
```

When `HTTP_LOADER_BASE_URL` is set, it overrides the default scheme option.

## Default Scheme

The HTTP Loader defaults to `https` when no scheme is specified in the image URL. Override this with:

```dotenv
HTTP_LOADER_DEFAULT_SCHEME=http
```

Set to `nil` to disable the default scheme entirely, requiring all image paths to include a full URL:

```dotenv
HTTP_LOADER_DEFAULT_SCHEME=nil
```

## Header Forwarding

Forward selected headers from the incoming request to the upstream image request:

```dotenv
HTTP_LOADER_FORWARD_HEADERS=Accept,Accept-Language,Authorization
```

To forward all browser client request headers:

```dotenv
HTTP_LOADER_FORWARD_CLIENT_HEADERS=1
```

## Header Overrides

Override upstream image request headers with fixed values:

```dotenv
HTTP_LOADER_OVERRIDE_RESPONSE_HEADERS=Cache-Control,Expires
```

This copies the specified headers from the HTTP Loader response into the imagor response.

## Proxy Support

Route HTTP Loader requests through one or more proxy servers:

```dotenv
HTTP_LOADER_PROXY_URLS=http://user:pass@proxy1:8080,http://user:pass@proxy2:8080
```

Optionally restrict proxy usage to specific hosts only:

```dotenv
HTTP_LOADER_PROXY_ALLOWED_SOURCES=*.example.com
```

When `HTTP_LOADER_PROXY_URLS` is set, proxies are rotated randomly across requests.

## Network Security (SSRF Protection)

To protect against [Server-Side Request Forgery (SSRF)](https://owasp.org/www-community/attacks/Server_Side_Request_Forgery) attacks, block connections to internal network addresses:

```dotenv
HTTP_LOADER_BLOCK_LOOPBACK_NETWORKS=1      # block loopback IPs e.g. 127.0.0.1, ::1
HTTP_LOADER_BLOCK_PRIVATE_NETWORKS=1       # block private IPs e.g. 10.x.x.x, 192.168.x.x
HTTP_LOADER_BLOCK_LINK_LOCAL_NETWORKS=1    # block link-local IPs e.g. 169.254.x.x
```

To block specific custom networks in CIDR notation:

```dotenv
HTTP_LOADER_BLOCK_NETWORKS=::1/128,127.0.0.0/8,10.0.0.0/8
```

## Max Allowed Size

Limit the maximum image size (in bytes) the HTTP Loader will fetch:

```dotenv
HTTP_LOADER_MAX_ALLOWED_SIZE=20971520  # 20MB
```

When set, a HEAD request is made first to check `Content-Length` before downloading.

## Disabling HTTP Loader

The HTTP Loader is enabled by default as a fallback. To disable it entirely (e.g. when using only File or S3 storage as the source):

```dotenv
HTTP_LOADER_DISABLE=1
```
