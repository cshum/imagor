---
slug: /mcp
description: Extends imagor with template capability for agents — a layout compiles to one imagor URL, at any canvas size.
keywords:
  - imagor-mcp
  - Imagor MCP
  - imagor templates
  - imagor MCP server
  - imagor agent skill
  - imagor image composition
---

# Imagor MCP

[![Test Status](https://github.com/cshum/imagor-mcp/workflows/ci/badge.svg)](https://github.com/cshum/imagor-mcp/actions/workflows/ci.yml)
[![MCP Registry](https://img.shields.io/badge/registry-io.github.cshum%2Fimagor--mcp-white)](https://registry.modelcontextprotocol.io/)

An MCP server and a skill that add template capability to
[imagor](https://github.com/cshum/imagor).

An agent writes a layout — OG cards, product grids, quotes, testimonials — that compiles to one
imagor URL.

:::info
**GitHub:** [github.com/cshum/imagor-mcp](https://github.com/cshum/imagor-mcp)  
**Registry:** [registry.modelcontextprotocol.io](https://registry.modelcontextprotocol.io/) — install as `io.github.cshum/imagor-mcp`  
**Docker:** [ghcr.io/cshum/imagor-mcp](https://github.com/cshum/imagor-mcp/pkgs/container/imagor-mcp)  
**Releases:** [macOS, Linux and Windows binaries](https://github.com/cshum/imagor-mcp/releases)  
**Requires:** imagor **1.9.10**
:::

[![Cards, each one a single imagor URL](https://raw.githubusercontent.com/cshum/imagor-mcp/main/docs/gallery/contact-sheet.png)](https://github.com/cshum/imagor-mcp/blob/main/docs/cookbook.md)

*[See the recipes »](https://github.com/cshum/imagor-mcp/blob/main/docs/cookbook.md)*

## Quick start

**1. Run an imagor.**

```sh
docker run --rm -p 8000:8000 shumc/imagor -imagor-unsafe
```

Leave it running, and use a second terminal for the rest.

Unsafe mode is for trying things out. Production runs on `IMAGOR_SECRET` — see
[imagor's security docs](./security.mdx).

**2. Install the MCP server.**

```sh
brew trust cshum/tap && brew install imagor-mcp
```

**3. Point a client at it.**

| your client | what to do |
|---|---|
| **VS Code** | [add it from here](https://vscode.dev/redirect?url=vscode%3Amcp%2Finstall%3F%257B%2522name%2522%253A%2522imagor%2522%252C%2522command%2522%253A%2522imagor-mcp%2522%252C%2522env%2522%253A%257B%2522IMAGOR_UNSAFE%2522%253A%25221%2522%252C%2522IMAGOR_BASE_URL%2522%253A%2522http%253A%252F%252Flocalhost%253A8000%2522%257D%257D) — one click |
| **Claude Desktop** | open [`imagor-mcp.mcpb`](https://github.com/cshum/imagor-mcp/releases/latest/download/imagor-mcp.mcpb) — one click |
| **Cursor, Claude Code, Windsurf** | add the block below to its config file |
| **Codex** | the same values in TOML, at `~/.codex/config.toml` |

```json
{
  "mcpServers": {
    "imagor": {
      "command": "imagor-mcp",
      "env": { "IMAGOR_UNSAFE": "1", "IMAGOR_BASE_URL": "http://localhost:8000" }
    }
  }
}
```

Those two settings have to match what you started in step 1: `IMAGOR_UNSAFE` puts this server
in the same mode, and `IMAGOR_BASE_URL` is where imagor is.

VS Code keys its servers under `servers` and Codex reads TOML, so the block above fits
neither. Every client's config file is in [Setup](https://github.com/cshum/imagor-mcp/blob/main/docs/setup.md).

**4. Ask it for something.** [See the recipes »](https://github.com/cshum/imagor-mcp/blob/main/docs/cookbook.md) — each one carries a prompt
to paste — or describe the image you want. The agent composes through the tools: it writes the layout, renders it, looks at
the result, and hands you back the url. Neither of you needs imagor's path syntax — the tools
build the path, and refuse one that cannot work.

That is the whole install. It is also on the
[official MCP Registry](https://registry.modelcontextprotocol.io/), where a client can install
it by name, and as `ghcr.io/cshum/imagor-mcp` in a container — which needs a different base
URL, in [Setup](https://github.com/cshum/imagor-mcp/blob/main/docs/setup.md). Windows binaries are on the
[releases page](https://github.com/cshum/imagor-mcp/releases).

Before anything is public, read [Security](https://github.com/cshum/imagor-mcp/blob/main/docs/setup.md#security) — what this server is
trusted with, and what it refuses.

## What an agent does with it

A brief like this one:

> *Build a 1200×630 launch card: a deep green ground, a serif headline, a product photo
> on the right, and the price below it. Show me what it looks like.*

It writes the layout, renders it, looks at the image, and revises, from a brief that names no
position and no size.

A brief is a search, so each run gives a different layout. Given the same brief three times, an
agent produced three distinct banners — hierarchy, colour, type pairing, invented copy — and
zoomed into regions to read the small text it had drawn. Like a result? `parse_template` turns it
into a recipe.

A recipe is a result. The layout is fixed and the agent fills the values, so the same recipe looks
the same every time. [See the recipes »](https://github.com/cshum/imagor-mcp/blob/main/docs/cookbook.md)

## What this adds

The template layer runs inside your imagor, so rendering stays in the instance you run, at the
throughput it already has.

A hosted template API returns an image; a layout here compiles to a URL. Those bill and transfer
bytes on every render. A URL is cached by any browser, proxy or CDN, so the bytes transfer once.

## Tools

Each tool maps onto a concept in the imagor path grammar. The list, and what each is for, is in
[Reference](https://github.com/cshum/imagor-mcp/blob/main/docs/reference.md).

## The cookbook

`cookbook/` holds ready-made layouts — OG cards, a product grid, a profile card, quotes,
testimonials, an event, a comparison table, a price ladder. Take the closest, change the values,
render it.

[See the recipes »](https://github.com/cshum/imagor-mcp/blob/main/docs/cookbook.md) The template is not on that page: a recipe travels in
the server and arrives by name.

Every recipe renders at three canvas sizes on every commit, and the contact sheet above is
generated by that same job, so both stay current.

## Going deeper

The rest of the documentation:

- [Setup](https://github.com/cshum/imagor-mcp/blob/main/docs/setup.md) — every setting, the config file each client reads, and what to do
  before it is public.
- [Reference](https://github.com/cshum/imagor-mcp/blob/main/docs/reference.md) — every tool the server serves, the layout format, and the
  skill.
- [Contributing](https://github.com/cshum/imagor-mcp/blob/main/CONTRIBUTING.md) — building, testing, and the CLI.

Everything above is the [imagor-mcp README](https://github.com/cshum/imagor-mcp#readme), kept in step with it.
