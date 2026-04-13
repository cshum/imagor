---
description: Extends imagor with video thumbnail extraction — pull frames from video files and process them through the full imagor pipeline.
---

# imagorvideo

[![Test Status](https://github.com/cshum/imagorvideo/workflows/test/badge.svg)](https://github.com/cshum/imagorvideo/actions/workflows/test.yml)
[![Codecov](https://img.shields.io/codecov/c/github/cshum/imagorvideo)](https://codecov.io/gh/cshum/imagorvideo)
[![Docker Hub](https://img.shields.io/badge/docker-shumc/imagorvideo-blue.svg)](https://hub.docker.com/r/shumc/imagorvideo/)

imagorvideo brings video thumbnail capability to imagor through ffmpeg C bindings. It extracts video thumbnails by selecting the best frame using RMSE histogram analysis, then passes the frame through the full imagor pipeline for [cropping, resizing](./image-endpoint.md) and [filters](./filters.md).

imagorvideo implements ffmpeg read and seek I/O callbacks with imagor [loader, storage and result storage](./storage.md), supporting HTTP(s), File System, AWS S3 and Google Cloud Storage out of the box. For non-seekable sources such as HTTP and S3, imagorvideo simulates seek using memory or temp file buffer.

:::info
**GitHub:** [cshum/imagorvideo](https://github.com/cshum/imagorvideo)  
**Docker:** [shumc/imagorvideo](https://hub.docker.com/r/shumc/imagorvideo)
:::

## Quick Start

```bash
docker run -p 8000:8000 shumc/imagorvideo -imagor-unsafe
```

With a sample video:
```
https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4
```

Try these thumbnail URLs:
```
http://localhost:8000/unsafe/300x0/7x7/filters:label(imagorvideo,-10,-7,15,yellow):fill(yellow)/https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4
http://localhost:8000/unsafe/300x0/0x0:0x14/filters:frame(1m59s):fill(yellow):label(imagorvideo,center,bottom,12,black,20)/https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4
http://localhost:8000/unsafe/300x0/7x7/filters:frame(0.6):label(imagorvideo,10,-7,15,yellow):fill(yellow)/https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4
```

<table>
  <tr>
    <td><img src="/img/imagorvideo/demo.jpg" /></td>
    <td><img src="/img/imagorvideo/demo2.jpg" /></td>
    <td><img src="/img/imagorvideo/demo3.jpg" /></td>
  </tr>
</table>

imagorvideo streams a limited number of frames from the video, calculates the histogram of each frame, and selects the best one based on Root Mean Square Error (RMSE). This skips black frames that commonly occur at the start of videos. The selected frame is converted to RGB and forwarded to the imagor libvips processor.

## Filters

imagorvideo adds the following filters, usable alongside all standard [imagor filters](./filters.md):

### `frame(n)`

Selects the precise frame at the specified position or time — bypasses automatic best-frame selection.

- Float `0.0`–`1.0` — position index of the video (e.g. `frame(0.5)`)
- Time duration — elapsed time from start (e.g. `frame(5m1s)`, `frame(200s)`)

### `seek(n)`

Seeks to the approximate position or time, then performs automatic best-frame selection around that point using RMSE. More forgiving than `frame()` when the target position may be a black frame.

- Float `0.0`–`1.0` — position index (e.g. `seek(0.5)`)
- Time duration — elapsed time from start (e.g. `seek(5m1s)`, `seek(200s)`)

### `frame(n)` vs `seek(n)`

`frame(n)` gives the precise frame at the specified time, but that frame may be black:

```
http://localhost:8000/unsafe/filters:frame(5m)/https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4
```

<img src="/img/imagorvideo/black.jpg" height="150" />

`seek(n)` seeks to the key frame before the target time, then applies best-frame selection — producing a useful image even when the exact frame is dark:

```
http://localhost:8000/unsafe/filters:seek(5m)/https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4
```

<img src="/img/imagorvideo/seek5m.jpg" height="150" />

### `max_frames(n)`

Restricts the maximum number of frames sampled for best-frame selection. Smaller values produce faster processing at the cost of selection quality.

---

## Metadata

imagorvideo provides a metadata endpoint that extracts video metadata — format, duration, FPS, dimensions — without extracting frame data.

Add `/meta` right after the URL signature hash:

```
http://localhost:8000/unsafe/meta/https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/1080/Big_Buck_Bunny_1080_10s_30MB.mp4
```

```json
{
  "format": "mp4",
  "content_type": "video/mp4",
  "orientation": 1,
  "duration": 10000,
  "width": 1920,
  "height": 1080,
  "title": "Big Buck Bunny, Sunflower version",
  "artist": "Blender Foundation 2008, Janus Bager Kristensen 2013",
  "fps": 30,
  "has_video": true,
  "has_audio": false
}
```

---

## Docker Compose Example

```yaml
version: "3"
services:
  imagor:
    image: shumc/imagorvideo:latest
    environment:
      PORT: 8000
      IMAGOR_SECRET: mysecret

      FFMPEG_FALLBACK_IMAGE: "path/to/fallback.jpg" # optional - fallback image on processing error
    ports:
      - "8000:8000"
```

## Configuration

Configuration options specific to imagorvideo. See [Configuration](./configuration.md) for all imagor options.

```
  -ffmpeg-fallback-image string
        FFmpeg fallback image on processing error. Supports image path enabled by loaders or storages
```
