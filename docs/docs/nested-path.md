---
description: Use path:<imagor-path> as the image source to compose and resize a nested path in-process, without a network round trip.
---

# Nested Path Source

Use `path:<imagor-path>` as the image source to process a nested imagor path in-process, without loading it over the network. This is useful when a transformation needs an image this instance already computed — a preview, a thumbnail of a composition, or a variant of a layout.

```
/unsafe/{transformations}/path:{nested-imagor-path}
```

The nested path is parsed and processed by the same instance, then handed back as the source for the outer path. **No HTTP request is made**, so the server does not need to be able to reach itself:

```
/unsafe/fit-in/400x400/path:1200x628/filters:text(01,20,20,Open-Sans-bold-40,fff)/color:102c2a
```

The nested path is percent-encoded, because it contains the same separators as the outer path:

```
/unsafe/fit-in/400x400/path:1200x628%2Ffilters%3Atext(...)%2Fcolor%3A102c2a
```

## Why not a URL source?

Passing an absolute URL also works, and produces the same bytes:

```
/unsafe/fit-in/400x400/http://imagor.example.com/unsafe/1200x628/filters:text(...)/color:102c2a
```

but it requires `imagor.example.com` to resolve **from inside the imagor instance**, which fails whenever the service is behind a proxy, in a container, or simply does not know its own public name. `path:` removes that requirement, and skips the fetch, the transfer and the re-decode.

## Nesting

A nested path may itself contain a `path:` source, up to `MaxNestedDepth` levels (4 by default). Deeper than that is refused with `400 too many nested levels` rather than recursing further.

## Signing

A nested path is authorised by the outer request's signature and is not re-signed — one signature covers the whole tree. Enable `IMAGOR_UNSAFE` or sign the outer path as usual; nothing extra is needed for the nested part.

## Notes

- The nested path resolves first, then the outer transformations apply to its result.
- `f`-dimensions such as `f-20xf-20` resolve against the parent image, as they do inside `image()`.
- Result storage is not used for a nested path's intermediate result; only the outer request is cached.
