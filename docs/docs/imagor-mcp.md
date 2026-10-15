---
slug: /mcp
description: Extends imagor with template capability for agents, compiling a layout to one imagor URL at any canvas size.
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

An MCP server and a skill that add template capability to
[imagor](https://github.com/cshum/imagor).

An agent writes a layout: OG cards, product grids, quotes, testimonials. It compiles to one
imagor URL, rendered by the imagor you already run.

:::info
**GitHub:** [github.com/cshum/imagor-mcp](https://github.com/cshum/imagor-mcp)  
**Registry:** install as `io.github.cshum/imagor-mcp`  
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

Unsafe mode is for trying things out. Production runs on `IMAGOR_SECRET`. See
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

**4. Ask it for something.** [See the recipes](https://github.com/cshum/imagor-mcp/blob/main/docs/cookbook.md) — each one carries a prompt
to paste, or describe the image you want. The agent composes through the tools: it writes the layout, renders it, looks at
the result, and hands you back the url. Neither of you needs imagor's path syntax: the tools
build the path, and refuse one that cannot work.

That is the whole install. It is also in the official MCP Registry as
`io.github.cshum/imagor-mcp`, which some clients install by name, and as
`ghcr.io/cshum/imagor-mcp` in a container, which needs a different base
URL, in [Setup](https://github.com/cshum/imagor-mcp/blob/main/docs/setup.md). Windows binaries are on the
[releases page](https://github.com/cshum/imagor-mcp/releases).

Before anything is public, read [Security](https://github.com/cshum/imagor-mcp/blob/main/docs/setup.md#security): what this server is
trusted with, and what it refuses.

## Going deeper

- [**Setup**](https://github.com/cshum/imagor-mcp/blob/main/docs/setup.md) — every setting, the config file each client reads, and what to do
  before it is public.
- [**Reference**](https://github.com/cshum/imagor-mcp/blob/main/docs/reference.md) — every tool the server serves, the layout format, and the
  skill.
- [**Cookbook**](https://github.com/cshum/imagor-mcp/blob/main/docs/cookbook.md) — the ready-made layouts, and the url each one resolves to.
- [**Contributing**](https://github.com/cshum/imagor-mcp/blob/main/CONTRIBUTING.md) — building, testing, and the CLI.
