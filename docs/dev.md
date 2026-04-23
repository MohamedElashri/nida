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
go run ./cmd/nida build --site ./example-site
go run ./cmd/nida serve --site ./example-site
go run ./cmd/nida build --site ./example-site-ar
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

## Release Work

Release process documentation lives in `docs/release.md`.

Before changing release packaging, run:

```bash
goreleaser check
GOCACHE="$PWD/.gocache" goreleaser release --snapshot --clean
```
