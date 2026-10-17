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
internal/archetype/       nida new and archetype templates
internal/assets/          static asset copying and page bundles
internal/buildinfo/       version and build metadata
internal/cli/             command parsing, builds, check, serve mode, rebuilds
internal/config/          config loading, defaults, normalization, validation, warnings
internal/content/         Markdown content discovery and git dates
internal/datafiles/       data/ files for templates
internal/diagnostics/     broken link and missing image checks
internal/feeds/           Atom and RSS feeds, including scoped feeds
internal/frontmatter/     TOML and YAML front matter
internal/highlight/       Chroma syntax highlighting
internal/i18n/            translation files and the t function
internal/linkcheck/       external link checking for nida check --external
internal/markdown/        Markdown rendering, shortcodes, callouts, math, TOC
internal/output/          output path planning and writing
internal/parallel/        ordered parallel map used for rendering and writes
internal/pipeline/        SCSS, minification, fingerprinting, image processing
internal/render/          page rendering and template context
internal/robots/          robots.txt
internal/safepath/        path joining and symlink checks
internal/scaffold/        nida init example site
internal/searchindex/     client-side search index
internal/server/          local development server and livereload
internal/shortcodes/      shortcode templates and built-in shortcodes
internal/site/            site index, routes, navigation, related posts, images
internal/sitemap/         sitemap generation
internal/taxonomies/      taxonomy collections and terms
internal/templates/       template loading, helpers, built-in partials
internal/themes/          theme chain resolution
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

Themes live under `themes/` and a site selects one with `theme = "name"`. A theme can extend another with `extends` in its `config.toml`. The docs site's own theme, `docs/themes/nida-docs`, is the largest real example in the repository.

Site files override theme files, and a child theme overrides its parent, for templates, partials, shortcodes, static files, SCSS, data, translations, and archetypes. See [Themes](@/guide/themes.md) for the full rules.

## CI

`.github/workflows/ci.yml` runs on branch pushes and pull requests. It checks formatting with `gofmt`, runs `go vet ./...`, runs `go test ./...`, builds both example sites, and runs `nida check` on the documentation site.

`.github/workflows/pages.yml` checks out full history (for git-based "last updated" dates), regenerates the release notes page from `CHANGELOG.md`, checks and builds the documentation site from `docs/`, and deploys it to GitHub Pages when docs, changelog, or source files change on `main`.
