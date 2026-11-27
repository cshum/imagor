---
description: imagor ecosystem links including official plugins, framework and language libraries, and community projects built around imagor.
keywords:
  - imagor community
  - imagor plugins
  - imagor symfony
  - imagor laravel
  - imagorvideo
  - imagorface
  - imagor-mcp
  - vipsgen
---

# Community

## Official Plugins

- [**imagorvideo**](./imagorvideo.md) — Video thumbnail server using ffmpeg C bindings
- [**imagorface**](./imagorface.md) — Fast, on-the-fly face detection for smart crop and privacy redaction

## Libraries

- [**symfony/ux-imagor-image**](https://symfony.com/bundles/ux-image) — Imagor integration for Symfony UX Image, through transformation parameters in the image server's URL path.
- [**sandstorm/laravel-imagor**](https://github.com/sandstorm/laravel-imagor) — Laravel integration for imagor
- [**cshum/vipsgen**](https://github.com/cshum/vipsgen) — Go binding for libvips, the image processing library powering imagor. Use this for low-level image operations directly in Go.

## Projects

- [**cshum/imagor-mcp**](./imagor-mcp.md) — An MCP server and skill for composing imagor URLs from a layout description
- [**cshum/imagor-studio**](https://github.com/cshum/imagor-studio) — Self-hosted image workspace with built-in editing, layered compositing, and reusable templates
- [**codedoga/imagor-toy**](https://github.com/codedoga/imagor-toy) - A ReactJS based app to play with Imagor
- [**cshum/imagor-base**](https://github.com/cshum/imagor-base) - Shared Docker base images for imagor-family projects

## Blog Posts & Articles

- [**Responsive, signed images in a Symfony blog with UX Image and Imagor**](https://hugo.alliau.me/blog/posts/responsive-signed-images-in-a-symfony-blog-with-ux-image-and-imagor) by Hugo Alliaume — Migrating a blog from hand-written proxy URLs to Markdown object keys: a CommonMark renderer over the Imagor bridge, signed URLs with `IMAGOR_UNSAFE` off, `srcset` ladders and preload headers
- [**Symfony UX 3.7 released**](https://symfony.com/blog/symfony-ux-3-7-released#ux-image-two-self-hosted-image-servers) by Hugo Alliaume — UX Image adds bridges to two self-hosted image servers, Imagor and IPX. The Imagor bridge fetches the original from your site and signs every URL with the server key
- [**Day 55: imagor — A Fast, Self-Hosted Image Processing Server in One Container**](https://portalzine.de/day-55-imagor-a-fast-self-hosted-image-processing-server-in-one-container-7-days-of-docker/) by Alexander Gräf, portalZINE.DE — A hands-on self-hosting walkthrough: Docker Compose for development and production, the URL syntax, environment variables, file system and S3 backends, URL signing and caching
- [**Image Processing Without Performance Bottlenecks: Laravel + imagor**](https://sandstorm.de/blog/posts/bildverarbeitung-ohne-performance-probleme-laravel-imagor/) by Sandstorm *(German)* — A walkthrough of integrating imagor with Laravel
