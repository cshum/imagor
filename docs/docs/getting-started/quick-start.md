# Quick Start

Get up and running with imagor in minutes using Docker.

## Docker Quick Start

The fastest way to try imagor is with Docker:

```bash
docker run -p 8000:8000 shumc/imagor -imagor-unsafe -imagor-auto-webp
```

This command:

- Runs imagor on port 8000
- Enables unsafe mode (no URL signing required - for testing only)
- Automatically serves WebP format when supported by the browser

## Your First Image Transformation

Once imagor is running, you can transform images using URL parameters. Here are some examples:

### Basic Resize

Resize an image to 300x200 pixels:

```
http://localhost:8000/unsafe/300x200/https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

### Smart Crop with Filters

Resize with smart cropping and apply filters:

```
http://localhost:8000/unsafe/200x200/smart/filters:fill(white):format(jpeg):quality(80)/https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

### Advanced Transformations

Apply multiple transformations:

```
http://localhost:8000/unsafe/fit-in/-180x180/10x10/filters:hue(290):saturation(100):fill(yellow)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png
```

## Understanding the URL Structure

imagor URLs follow this pattern:

```
/HASH|unsafe/trim/AxB:CxD/fit-in/stretch/-Ex-F/GxH:IxJ/HALIGN/VALIGN/smart/filters:NAME(ARGS):NAME(ARGS):.../IMAGE
```

Key components:

- `unsafe` - Disables URL signing (use only for testing)
- `300x200` - Resize dimensions
- `smart` - Enable smart cropping
- `filters:` - Apply image filters
- `IMAGE` - The source image URL

## Next Steps

- Learn about [Image Endpoint](../api/image-endpoint) syntax
- Explore available [Filters](../api/filters)
- Set up [Production Configuration](../configuration/overview)
- Deploy with [Docker Compose](../deployment/docker-compose)

:::warning Production Security
Never use `-imagor-unsafe` in production! Always configure URL signing with `IMAGOR_SECRET` for security.
:::
