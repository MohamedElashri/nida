# Development

This document is for contributors working on Nida from source.

## Requirements

* Go, using the version declared in `go.mod`
* Git
* GoReleaser, only when validating release packaging

## Common Commands

The repository includes a `Makefile`:

```bash
make build
make test
make site-build
make serve
make example-build
make example-serve
make arabic-example-build
make arabic-example-serve
make check
make clean
```

Useful direct Go commands:

```bash
go test ./...
go build ./...
go run ./cmd/nida build -s ./example-site
go run ./cmd/nida serve -s ./example-site
go run ./cmd/nida build -s ./example-site-ar
```

If the normal Go cache is not writable in your environment, use a local cache:

```bash
GOCACHE="$PWD/.gocache" go test ./...
```

## Example Sites

The repository ships two integration fixtures:

* `example-site`: English example blog
* `example-site-ar`: Arabic RTL example blog

Both examples are used by tests and release preflight checks. Keep them small,
realistic, and aligned with documented behavior.

## Serve Mode Notes

Watch mode uses native filesystem events on Linux and macOS, with polling
fallback where native watching is unavailable.

Serve mode rebuilds incrementally:

* asset-only changes sync assets
* content, template, and config changes rewrite only changed outputs
* `server.livereload` refreshes the browser after successful rebuilds

## Project Layout

```text
cmd/nida/                 CLI entrypoint
internal/assets/          static asset copying
internal/cli/             command parsing and command orchestration
internal/config/          config loading, defaults, normalization, validation
internal/content/         Markdown content discovery
internal/feeds/           RSS generation
internal/markdown/        Markdown rendering
internal/output/          output path planning and writing
internal/render/          page rendering
internal/server/          local development server
internal/site/            site index and route construction
internal/sitemap/         sitemap generation
internal/templates/       Go template loading and helpers
internal/watcher/         local watch mode
```

## Template Fixtures

Theme templates use `.html` filenames and Go template definitions:

```text
templates/base.html
templates/index.html
templates/post.html
templates/page.html
templates/list.html
templates/taxonomy.html
templates/404.html
```

The filename stem is the template name. For example, `post.html` should define
`{{ define "post" }}`.

## Theme System

Nida supports loadable themes with override chains. Themes live in the `themes/`
directory, and a site selects a theme via `config.toml`:

```toml
theme = "ink"
```

### Theme Structure

A theme is a directory under `themes/` containing:

```text
themes/ink/
├── config.toml      # theme metadata and defaults
├── templates/       # template files (override site templates)
├── static/          # static assets (copied to output)
└── scss/            # SCSS source files (compiled before site SCSS)
```

### Theme Config

```toml
name = "Ink"
description = "A minimalist theme for Nida"
extends = "base"  # optional parent theme

[extra]
main_menu = [{ name = "Home", url = "/" }]
footer = { text = "Powered by Nida" }
date_format = "%Y-%m-%d"
```

- `name` and `description` are metadata
- `extends` names a parent theme for inheritance
- `[extra]` provides default values merged with site config

### Template Override Chain

Theme templates are loaded first, site templates second. When both have a template
with the same name, the site version takes precedence. This allows themes to
provide defaults that sites can override.

For example, a theme can provide `base.html` and `post.html`, while the site
overrides only `post.html` to customize post rendering while keeping the theme's
base layout.

### Inheritance Chain

Themes can extend other themes via `extends = "parent"` in `config.toml`. The
parent theme is loaded first, then the child. Child templates override parent
templates with the same name. Circular inheritance is detected and rejected.

### SCSS Compilation

Theme SCSS files in `themes/<name>/scss/` are compiled before site SCSS in
`static/scss/`. This allows themes to provide base styles that site SCSS can
extend.

### Static Assets

Theme static files in `themes/<name>/static/` are copied to output before site
static files. Site static files take precedence for files with the same path,
allowing sites to override theme assets.

### Theme Extra Values

Theme `[extra]` values are merged with site `[extra]` values. Site values take
precedence over theme values, enabling theme customization without modifying
theme files.

## Release Work

Release process documentation lives in `docs/release.md`.

Before changing release packaging, run:

```bash
goreleaser check
GOCACHE="$PWD/.gocache" goreleaser release --snapshot --clean
```
