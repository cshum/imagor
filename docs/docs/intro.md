---
sidebar_position: 1
slug: /
---

# imagor

**imagor** is a fast, secure image processing server and Go library.

imagor uses one of the most efficient image processing library [libvips](https://github.com/libvips/libvips) with Go binding [vipsgen](https://github.com/cshum/vipsgen). It is typically 4-8x [faster](https://github.com/libvips/libvips/wiki/Speed-and-memory-use) than using the quickest ImageMagick settings. imagor implements libvips [streaming](https://www.libvips.org/2019/11/29/True-streaming-for-libvips.html) that facilitates parallel processing pipelines, achieving high network throughput.

### Lightning-Fast Image Processing

High-performance image transformations with libvips streaming that facilitates parallel processing pipelines, achieving exceptional network throughput.

### Thumbor-Compatible API

imagor features a ton of image processing use cases, available as a HTTP server with first-class Docker support. It adopts the [thumbor](https://thumbor.readthedocs.io/en/latest/usage.html#image-endpoint) URL syntax representing a high-performance drop-in replacement.

### Secure and Extensible

imagor is a Go library built with speed, security and extensibility in mind. Alongside there is [imagorvideo](https://github.com/cshum/imagorvideo) bringing video thumbnail capability through ffmpeg C bindings.

## Quick Demo

Try imagor with Docker:

```bash
docker run -p 8000:8000 shumc/imagor -imagor-unsafe -imagor-auto-webp
```

Original images:

<div className="demo-images">

![Gopher](https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png)
![Dancing Banana](https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif)
![Gopher Front](https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png)

</div>

Try out the following image transformation URLs:

```
http://localhost:8000/unsafe/fit-in/200x200/filters:fill(white)/https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
http://localhost:8000/unsafe/200x200/smart/filters:fill(white):format(jpeg):quality(80)/https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
http://localhost:8000/unsafe/fit-in/-180x180/10x10/filters:hue(290):saturation(100):fill(yellow)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

<div className="demo-images">

![Demo 1](https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo1.jpg)
![Demo 2](https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo2.jpg)
![Demo 4](https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo4.jpg)
![Demo 3](https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo3.gif)
![Demo 5](https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo5.gif)

</div>

## Key Features

- **High-performance image processing** powered by [libvips](https://github.com/libvips/libvips) with 4-8x faster performance than ImageMagick
- **Thumbor-compatible API** for easy migration and familiar URL syntax
- **Comprehensive format support** including JPEG, PNG, WebP, AVIF, GIF, TIFF, and more
- **Advanced image operations** including smart cropping, filters, watermarks, and transformations
- **Multiple storage backends** supporting File System, AWS S3, Google Cloud Storage
- **Built-in security** with URL signing and image bomb prevention
- **Docker-first deployment** with official Docker images and Kubernetes support
- **Extensible architecture** as a Go library for custom integrations

## What Makes imagor Special?

### Performance-First Design

Built on libvips with streaming support, imagor delivers exceptional performance for high-throughput image processing workloads.

### Production-Ready Security

Comprehensive security features including URL signing, allowed source restrictions, and image bomb prevention protect against abuse.

### Universal Storage Support

Works seamlessly with local filesystems, S3-compatible storage, Google Cloud Storage, and more. Switch between storage backends without changing your workflow.

### Thumbor Compatibility

Drop-in replacement for thumbor with the same URL syntax, making migration straightforward while gaining significant performance improvements.

## Quick Links

- [Quick Start Guide](./getting-started/quick-start) - Get up and running in minutes
- [API Reference](./api/image-endpoint) - Complete URL syntax and operations
- [Configuration](./configuration/overview) - Customize imagor for your needs
- [GitHub Repository](https://github.com/cshum/imagor) - Source code and issues

---

Ready to get started? Head over to the [Quick Start Guide](./getting-started/quick-start)!
