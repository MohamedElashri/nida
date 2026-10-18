# Nida

[![CI](https://github.com/MohamedElashri/nida/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/MohamedElashri/nida/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/MohamedElashri/nida)](https://github.com/MohamedElashri/nida/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/MohamedElashri/nida.svg)](https://pkg.go.dev/github.com/MohamedElashri/nida)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MohamedElashri/nida)](./go.mod)
[![Docs](https://img.shields.io/badge/docs-melashri.net%2Fnida-blue)](https://melashri.net/nida/)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

Nida (نداء, Arabic for "call") is a small static site generator written in Go
for blogs, personal sites, and documentation. It reads Markdown with TOML or
YAML front matter, renders it with Go HTML templates, and writes a static site
you can host anywhere.

## Install

```bash
curl -fsSL https://melashri.net/nida/install.sh | sh   # Linux and macOS
brew install MohamedElashri/nida/nida                   # Homebrew
go install github.com/MohamedElashri/nida/cmd/nida@latest
```

Prebuilt archives for Linux, macOS, and Windows are on
[GitHub Releases](https://github.com/MohamedElashri/nida/releases).

## Quick start

```bash
nida init ./my-site                       # create an example site
nida serve ./my-site                      # preview at http://127.0.0.1:1702
nida new -s ./my-site posts/hello.md      # add a post
nida check ./my-site                      # find broken links and errors
nida build ./my-site                      # write the site to my-site/public
```

## Documentation

The full documentation lives at [melashri.net/nida](https://melashri.net/nida/).

| Guide | What it covers |
| --- | --- |
| [Getting Started](https://melashri.net/nida/guide/getting-started/) | Installing Nida and building your first site |
| [Features](https://melashri.net/nida/guide/features/) | Everything Nida supports, at a glance |
| [Writing Content](https://melashri.net/nida/guide/writing/) | Front matter, sections, shortcodes, math, and diagrams |
| [Templates](https://melashri.net/nida/guide/templates/) | Layouts, partials, and template functions |
| [Themes](https://melashri.net/nida/guide/themes/) | Using and writing themes |
| [CLI Reference](https://melashri.net/nida/reference/cli/) | Every command and flag |
| [Config Reference](https://melashri.net/nida/reference/config/) | Every `config.toml` option |
| [Migrating](https://melashri.net/nida/guide/migrating/) | Moving a site from Zola, Hugo, or Jekyll |

Release history is in the [Changelog](./CHANGELOG.md).

## Examples

| Site | Description |
| --- | --- |
| [`example-site`](./example-site) | English blog |
| [`example-site-ar`](./example-site-ar) | Arabic blog with right-to-left layout |
| [`docs`](./docs) | The Nida documentation site, with its own theme |

Preview any of them with `nida serve ./example-site`.

## License

[MIT](./LICENSE)
