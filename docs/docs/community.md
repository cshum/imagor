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

- [**symfony/ux-imagor-image**](https://packagist.org/packages/symfony/ux-imagor-image) — Imagor integration for Symfony UX Image, through transformation parameters in the image server's URL path. *Experimental.*
- [**sandstorm/laravel-imagor**](https://github.com/sandstorm/laravel-imagor) — Laravel integration for imagor
- [**vipsgen**](https://github.com/cshum/vipsgen) — Go binding for libvips, the image processing library powering imagor. Use this for low-level image operations directly in Go.

## Projects

The imagor ecosystem includes several community-contributed projects that extend and integrate with imagor:

- [**cshum/imagor-base**](https://github.com/cshum/imagor-base) - Shared Docker base images for imagor-family projects
- [**cshum/imagor-mcp**](./imagor-mcp.md) — An MCP server and a skill that add a template layer to imagor. An agent writes a layout: OG cards, product grids, quotes, testimonials, and it compiles to one imagor URL.
- [**cshum/imagor-studio**](https://github.com/cshum/imagor-studio) — Self-hosted image workspace with built-in editing, layered compositing, and reusable templates
- [**codedoga/imagor-toy**](https://github.com/codedoga/imagor-toy) - A ReactJS based app to play with Imagor

## Blog Posts & Articles

- [**Symfony UX 3.7 released**](https://symfony.com/blog/symfony-ux-3-7-released#ux-image-two-self-hosted-image-servers) by Hugo Alliaume — UX Image adds bridges to two self-hosted image servers, Imagor and IPX. The Imagor bridge fetches the original from your site and signs every URL with the server key
- [**Image Processing Without Performance Bottlenecks: Laravel + imagor**](https://sandstorm.de/blog/posts/bildverarbeitung-ohne-performance-probleme-laravel-imagor/) by Sandstorm *(German)* — A walkthrough of integrating imagor with Laravel
