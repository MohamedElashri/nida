+++
title = "Development Workflow"
description = "Local commands and project layout for contributors."
weight = 20
template = "page"
+++

This page is for working on Nida from source.

## Requirements

- Go, using the version declared in `go.mod`
- Git
- GoReleaser, only when validating release packaging

## Common Commands

The repository includes a `Makefile`:

```bash
make build
make test
make site-build
make docs-build
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

Use `make docs-build` for the documentation site. It regenerates
`docs/content/release-notes.md` from `CHANGELOG.md` before running the Nida
build.

If the normal Go cache is not writable, keep the cache inside the repository:

```bash
GOCACHE="$PWD/.gocache" go test ./...
```

## Example Sites

The repository ships two integration fixtures:

- `example-site`: English example blog
- `example-site-ar`: Arabic RTL example blog

Both examples are used by tests, CI, docs checks, and release preflight checks. Keep them small, realistic, and aligned with documented behavior.

## Project Layout

```text
cmd/nida/                 CLI entrypoint
internal/assets/          static asset copying and page bundles
internal/cli/             command parsing, builds, serve mode, rebuilds
internal/config/          config loading, defaults, normalization, validation
internal/content/         Markdown content discovery
internal/feeds/           RSS and Atom generation
internal/markdown/        Markdown rendering, shortcodes, internal links
internal/output/          output path planning and writing
internal/pipeline/        SCSS, minification, fingerprinting, image processing
internal/render/          page rendering and theme-facing data
internal/server/          local development server and livereload
internal/site/            site index, routes, sections, taxonomies
internal/sitemap/         sitemap generation
internal/templates/       Go template loading and helpers
internal/watcher/         local watch mode
```

## Template Fixtures

Templates use `.html` filenames and Go template definitions. The filename stem is the template name. For example, `post.html` should define `{{ define "post" }}`.

Common template names:

```text
templates/base.html
templates/index.html
templates/post.html
templates/page.html
templates/section.html
templates/list.html
templates/taxonomy.html
templates/404.html
```

## Theme Notes

Themes live under `themes/` and a site selects one with:

```toml
theme = "ink"
```

A typical theme contains:

```text
themes/ink/
├── config.toml
├── templates/
├── static/
└── scss/
```

Theme templates load before site templates, so site templates override theme defaults. Theme static files copy before site static files, so site assets override theme assets. Theme `[extra]` values merge with site `[extra]`, with site values taking precedence.

## CI

`.github/workflows/ci.yml` runs on branch pushes and pull requests. It checks formatting with `gofmt`, runs `go vet ./...`, runs `go test ./...`, and builds both example sites.

`.github/workflows/pages.yml` regenerates the release notes page from `CHANGELOG.md`, builds the documentation site from `docs/`, and deploys it to GitHub Pages when docs, changelog, or source files change on `main`.
