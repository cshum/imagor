---
description: Extends imagor with template capability for agents — a layout compiles to one imagor URL, at any canvas size.
keywords:
  - imagor-mcp
  - imagor templates
  - imagor MCP server
  - imagor agent skill
  - imagor image composition
---

# imagor-mcp

[![Test Status](https://github.com/cshum/imagor-mcp/workflows/ci/badge.svg)](https://github.com/cshum/imagor-mcp/actions/workflows/ci.yml)
[![MCP Registry](https://img.shields.io/badge/registry-io.github.cshum%2Fimagor--mcp-white)](https://registry.modelcontextprotocol.io/)

imagor-mcp brings template capability to imagor for agents. An agent writes a layout — slots, positions, type, colours — and the layout compiles to a single [imagor URL](./image-endpoint.md), at any canvas size. That URL is then served like any other, so it is signed, cached and delivered the way the rest of the pipeline already works.

The template layer runs inside your imagor: rendering stays in the instance you run, at the throughput it already has. A hosted template API returns an image and bills every render. A URL is cached by any browser, proxy or CDN, so the bytes transfer once.

:::info
**GitHub:** [cshum/imagor-mcp](https://github.com/cshum/imagor-mcp)  
**Registry:** [registry.modelcontextprotocol.io](https://registry.modelcontextprotocol.io/) — install as `io.github.cshum/imagor-mcp`  
**Docker:** [ghcr.io/cshum/imagor-mcp](https://github.com/cshum/imagor-mcp/pkgs/container/imagor-mcp)  
**Releases:** [github.com/cshum/imagor-mcp/releases](https://github.com/cshum/imagor-mcp/releases) — macOS, Linux and Windows binaries  
**Requires:** imagor **1.9.10** or later, for the nested [`path:`](./nested-path.md) source
:::

[![Fourteen cards, each one a single imagor URL](https://raw.githubusercontent.com/cshum/imagor-mcp/main/docs/gallery/contact-sheet.png)](https://github.com/cshum/imagor-mcp/blob/main/docs/gallery/cookbook.md)

*Every card above is one imagor URL — see the **[full gallery](https://github.com/cshum/imagor-mcp/blob/main/docs/gallery/cookbook.md)**.*

## Quick Start

Run imagor, then the MCP server alongside it:

```bash
docker run -p 8000:8000 shumc/imagor -imagor-unsafe
```

```bash
docker run -i --rm \
  -e IMAGOR_UNSAFE=1 \
  -e IMAGOR_BASE_URL=http://host.docker.internal:8000 \
  ghcr.io/cshum/imagor-mcp
```

Add that to your MCP client and ask for an image. `IMAGOR_UNSAFE` pairs with an imagor running `-imagor-unsafe`; in production both sides move to `IMAGOR_SECRET` — see [Security](./security.mdx).

## Composition, not transformation

The [image endpoint](./image-endpoint.md) grammar transforms one source. A layout composes several — a background, images placed within it, and text — with each slot sized in pixels or as a proportion of the canvas:

```json
{
  "canvas": [1200, 628],
  "background": "color:102c2a",
  "slots": [
    {"name": "number", "kind": "text", "text": "{{number}}",
     "x": "8p+4", "y": "14p", "font": "Open-Sans-bold-26", "color": "e5a15b"},
    {"name": "tip", "kind": "text", "text": "{{tip}}",
     "x": "11p+2", "y": "31p+5", "width": "80p+6", "height": "35p+4",
     "font": "Liberation-Serif", "color": "fff7e8", "max_size": 70, "min_size": 48}
  ]
}
```

One template serves every size, because positions and type resolve against the canvas. `render_template` returns the compiled path, the URL, and what each slot resolved to.

## The skill

The server carries a skill: path grammar, the order of segments, the filters, and the method for composing when there is no canvas to look at. An agent reads it from the server, so what it follows is the version you are running rather than a document that has drifted from it.

The skill is served three ways — as a resource, as the `get_imagor_skill` tool, and attached to the `compose-imagor-image` prompt.

## The cookbook

Fourteen ready-made layouts ship with the server. They render at three canvas sizes on every commit, so a recipe that stops working fails the build rather than sitting in the list.

**A recipe is a result** — the layout is fixed and the agent fills the values, so the same recipe looks the same every time. **A brief is a search** — ask for a card with no recipe and expect a different layout each run.

## Configuration

| Variable | |
|---|---|
| `IMAGOR_BASE_URL` | where imagor is, and what the returned URLs are prefixed with |
| `IMAGOR_UNSAFE` | development mode: pair with an imagor running `-imagor-unsafe` |
| `IMAGOR_SECRET` | production: sign every URL |

Configuration, clients, the full tool list and the security model are in the **[imagor-mcp repo](https://github.com/cshum/imagor-mcp)**.
