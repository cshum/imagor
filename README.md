# imagor

[![Test Status](https://github.com/cshum/imagor/workflows/test/badge.svg)](https://github.com/cshum/imagor/actions/workflows/test.yml)
[![Codecov](https://img.shields.io/codecov/c/github/cshum/imagor
)](https://codecov.io/gh/cshum/imagor)
[![Docker Hub](https://img.shields.io/badge/docker-shumc/imagor-blue.svg)](https://hub.docker.com/r/shumc/imagor/)
[![Docs](https://img.shields.io/badge/docs-docs.imagor.net-white)](https://docs.imagor.net)

imagor is a fast, secure image processing server and Go library.

imagor uses one of the most efficient image processing library
[libvips](https://github.com/libvips/libvips) with Go binding [vipsgen](https://github.com/cshum/vipsgen). It is typically 4-8x [faster](https://github.com/libvips/libvips/wiki/Speed-and-memory-use) than using the quickest ImageMagick settings.
imagor implements libvips [streaming](https://www.libvips.org/2019/11/29/True-streaming-for-libvips.html) that facilitates parallel processing pipelines, achieving high network throughput.

imagor features a ton of image processing use cases, available as a HTTP server with first-class Docker support. It adopts the [thumbor](https://thumbor.readthedocs.io/en/latest/usage.html#image-endpoint) URL syntax representing a high-performance drop-in replacement.

imagor is a Go library built with speed, security and extensibility in mind. Alongside there is [imagorvideo](https://github.com/cshum/imagorvideo) bringing video thumbnail capability through ffmpeg C bindings.

### Quick Start

```bash
docker run -p 8000:8000 shumc/imagor -imagor-unsafe -imagor-auto-webp
```

Original images:

```
https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif
https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png
```

<img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png" height="100" />

Try out the following image URLs:

```
http://localhost:8000/unsafe/fit-in/200x200/filters:fill(white)/https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
http://localhost:8000/unsafe/200x200/smart/filters:fill(white):format(jpeg):quality(80)/https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
http://localhost:8000/unsafe/fit-in/-180x180/10x10/filters:hue(290):saturation(100):fill(yellow)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
http://localhost:8000/unsafe/30x40:100x150/filters:fill(cyan)/raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif
http://localhost:8000/unsafe/fit-in/200x150/filters:fill(yellow):watermark(raw.githubusercontent.com/cshum/imagor/master/testdata/gopher-front.png,repeat,bottom,0,40,40)/raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif
```

<img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo1.jpg" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo2.jpg" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo4.jpg" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo3.gif" height="100" /> <img src="https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo5.gif" height="100" />

> [!NOTE]
> Ready to dive deeper? Check out [Documentation](https://docs.imagor.net/)
