# Nida

**Work in progress. Not yet ready for production use.**

`nida` is a Go static site generator for a focused personal publishing workflow. It builds as a single binary, keeps the public interface intentionally small, and favors a deterministic file-based pipeline over runtime extensibility.

## Status

Current v1 scope includes:

* `nida build`
* `nida serve`
* `config.toml`-based site configuration
* Markdown content with TOML front matter
* syntax-highlighted fenced code blocks
* built-in tags and categories
* RSS generation
* `sitemap.xml` generation
* local watch-based development serving

Current non-goals include:

* plugins or script hooks (**NEVER**)
* theme marketplaces or remote themes (**NEVER**)
* multilingual support (**RTL will be considered**)
* arbitrary custom content types (**will be considered**)
* image pipelines or asset bundling (**will be considered**)
* CMS or database integration (**NEVER**)

## Quick Start

Build the binary:

```bash
go build ./cmd/nida
```

Or use the project Makefile:

```bash
make build
```

Build the example site:

```bash
./nida build --site ./example-site
```

Serve the example site locally:

```bash
./nida serve --site ./example-site
```

The default local address is `http://127.0.0.1:2906`.

Common development shortcuts:

```bash
make test
make site-build
make serve
make check
```

## Commands

The public CLI surface is intentionally limited to two commands:

```bash
nida build [--site PATH] [--config PATH] [--drafts]
nida serve [--site PATH] [--config PATH] [--drafts] [--port PORT]
```

## Site Layout

Default layout:

```text
site/
├── config.toml
├── content/
│   ├── posts/
│   └── pages/
├── templates/
├── static/
└── public/
```

The bundled example site at [example-site](/home/melashri/projects/nida/example-site) is the reference site used by the test suite and the documentation.

## Content Format

Content files use TOML front matter followed by Markdown:

```markdown
+++
title = "Hello World"
date = 2026-04-12T10:00:00Z
draft = false
tags = ["intro"]
categories = ["general"]
description = "A short post"
slug = "hello-world"
+++
```

## Development Notes

Current serve-mode behavior:

* watch mode uses polling, not OS-native file notifications
* rebuilds are full rebuilds, not incremental rebuilds
* watcher snapshot failures are reported as diagnostics and do not immediately crash the server
* built output changes are ignored by the watcher to avoid rebuild loops

The repository includes a `Makefile` for day-to-day development:

* `make build` builds `./cmd/nida` into `./nida`
* `make test` runs `go test ./...`
* `make site-build` builds `nida` against the selected `SITE`
* `make example-build` builds the bundled example website in `example-site`
* `make example-serve` serves the bundled example website locally
* `make serve` runs the selected site locally
* `make check` runs formatting, tests, and a real site build
* `make clean` removes the binary, coverage file, and local Go cache directories

## Versioning

Until the first tagged release, the project should be treated as pre-`v1`.

Versioning approach:

* use semantic versioning once tagged releases begin
* treat `v0.x.y` as unstable while the implementation is still settling
* reserve `v1.0.0` for the point where command behavior, config shape, and output guarantees are considered intentionally stable

## Release Readiness

The current release checklist lives in [docs/release-checklist.md](/home/melashri/projects/nida/docs/release-checklist.md:1).


## LICENCE

The project is licensed under the MIT License. See [LICENSE](./LICENSE) for details.