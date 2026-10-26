+++
title = "Development Workflow"
description = "Local commands and project layout for contributors."
weight = 20
template = "page"
+++

This page is for working on Nida from source.

## Requirements

- Go, using the version declared in `go.mod`
- A 64-bit platform: amd64, arm64, mips64, ppc64, riscv64, s390x, or wasm. Nida's copy of goja builds nowhere else.
- Git
- GoReleaser, only when validating release packaging

Node is not needed. Pele, Temml, and goja are embedded in the repository; see [Bundled dependencies](#bundled-dependencies).

## Common Commands

The repository includes a `Makefile`:

Run `make help` for the full list. The common targets:

| Target | Does |
| --- | --- |
| `make build`, `make rebuild`, `make install` | Build the `nida` binary (`rebuild` cleans first), or install it into `GOBIN`. |
| `make run ARGS='check ./docs'` | Run Nida with any arguments. |
| `make size` | Build a stripped release binary and print its size. |
| `make test`, `make test-race`, `make test-cover` | Run the tests, with the race detector, or with coverage. |
| `make lint`, `make fmt`, `make fmt-check`, `make vet`, `make tidy` | golangci-lint, gofmt (fixing or only checking), go vet, and `go mod tidy`. |
| `make check` (or `make dev`) | Format, vet, lint, test, and build every site. |
| `make ci` | Run what CI runs, without changing files. |
| `make serve`, `make site-build` | Serve or build `SITE=...` (default `./example-site`). |
| `make example-build`, `make arabic-example-build`, `make sites` | Build the example sites; `sites` also checks the docs. |
| `make docs-build`, `make docs-serve`, `make docs-check` | Regenerate release notes, then build, serve, or check the docs. |
| `make vendor-js`, `make vendor-goja`, `make goja-update`, `make vendor-check` | Regenerate or verify the bundled dependencies. |
| `make outdated` | List Go modules and goja commits with newer versions. |
| `make release-check`, `make release-snapshot` | Validate the GoReleaser config, or build release archives into `dist/`. |
| `make clean` | Remove build output and caches. |

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

## Bundled Dependencies

Nida renders math and diagrams at build time with Pele and Temml, JavaScript libraries that run in goja, a JavaScript engine written in Go. All three are pinned in `third_party/`, and generated files are never edited by hand:

- `third_party/js/package-lock.json` pins Pele and Temml. Dependabot opens pull requests that bump it, and CI automatically vendors the updated assets and pushes them back to the PR branch.
- `third_party/goja/` is upstream goja with `third_party/goja.patch` applied and every file limited to 64-bit platforms. A weekly workflow opens an issue when upstream moves; `make goja-update` applies the patch to the new version.

`third_party/README.md` has the details.

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
internal/bundled/         math stylesheet and fonts written to vendor/
internal/cli/             command parsing, builds, check, serve mode, rebuilds
internal/config/          config loading, defaults, normalization, validation, warnings
internal/content/         Markdown content discovery and git dates
internal/datafiles/       data/ files for templates
internal/diagnostics/     broken link and missing image checks
internal/diagram/         Mermaid to SVG with Pele
internal/feeds/           Atom and RSS feeds, including scoped feeds
internal/frontmatter/     TOML and YAML front matter
internal/highlight/       Chroma syntax highlighting
internal/i18n/            translation files and the t function
internal/jsengine/        goja runtime pools for Pele and Temml
internal/linkcheck/       external link checking for nida check --external
internal/markdown/        Markdown rendering, shortcodes, callouts, math, TOC
internal/mathml/          TeX to MathML with Temml
internal/migrate/         nida migrate from Hugo, Jekyll, Zola, Eleventy, Pelican
internal/output/          output path planning and writing
internal/parallel/        ordered parallel map used for rendering and writes
internal/pipeline/        SCSS, minification, fingerprinting, image processing
internal/render/          page rendering and template context
internal/robots/          robots.txt
internal/route/           URL validation and building
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
scripts/                  release notes and bundled dependency generators
third_party/              pinned Pele, Temml, and patched goja
```

## Template Fixtures

Templates use `.html` filenames, and the file name is the template name: `post.html` defines `{{ define "post" }}`. See [required templates](@/guide/templates.md#required-templates) for the names Nida looks for.

## Theme Notes

Themes live under `themes/` and a site selects one with `theme = "name"`. A theme can extend another with `extends` in its `config.toml`. The docs site's own theme, `docs/themes/nida-docs`, is the largest real example in the repository.

Site files override theme files, and a child theme overrides its parent, for templates, partials, shortcodes, static files, SCSS, data, translations, and archetypes. See [Themes](@/guide/themes.md) for the full rules.

## CI

`.github/workflows/ci.yml` runs on branch pushes and pull requests. It checks formatting with `gofmt`, runs `go vet ./...`, checks that the bundled JavaScript and goja match their pins, runs `go test ./...`, builds both example sites, and runs `nida check` on the documentation site.

`.github/workflows/goja.yml` runs weekly and opens or updates an issue when upstream goja has new commits, saying whether `goja.patch` still applies.

`.github/workflows/pages.yml` checks out full history (for git-based "last updated" dates), regenerates the release notes page from `CHANGELOG.md`, checks and builds the documentation site from `docs/`, and deploys it to GitHub Pages when documentation or changelog files change on `main`.
