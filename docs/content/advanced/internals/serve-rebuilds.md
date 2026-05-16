+++
title = "Serve Mode And Rebuilds"
description = "How local serving, watching, livereload, and incremental rebuilds work."
weight = 70
template = "page"
+++

`nida serve` is a full build plus a local server and watcher. It uses the same initial build as `nida build`, then keeps the previous `buildResult` in memory for rebuild decisions.

## Server

`internal/server.Start` serves the configured output directory over HTTP. It:

- uses a no-symlink filesystem wrapper
- supports custom `404.html`
- supports sites whose `base_url` contains a path prefix
- injects a livereload script into HTML responses when enabled
- exposes a server-sent-events endpoint for reload messages

The server shuts down when the command context is canceled by `SIGINT` or `SIGTERM`.

## Base Path

The base path comes from `config.BasePath(cfg.BaseURL)`. If the site is configured for `https://example.com/blog/`, serve mode can serve routes under `/blog`.

The base-path wrapper maps requests for the configured prefix back to files in the output directory.

## Watcher

`internal/watcher.Run` prefers native filesystem events on platforms with support and falls back to polling. Polling snapshots file size and modification time.

The watcher skips output directories and VCS directories so generated files do not trigger rebuild loops.

Changed paths are passed back to `internal/cli` as site-relative paths.

## Rebuild Modes

`rebuildMode` chooses one of three modes:

- `assets-only`
- `partial`
- `full`

Static-only changes use `assets-only`. Markdown content changes use `partial`. Config changes, template changes, non-Markdown content changes, and unknown paths use `full`.

## Assets-Only Rebuilds

Assets-only rebuilds call `assets.SyncChanged`. They copy changed static files or remove stale output files for deleted static assets.

This mode does not rerender pages, feeds, sitemap, robots, or search.

## Partial Content Rebuilds

Partial rebuilds are optimized for Markdown page edits.

`buildIncremental` reloads changed Markdown pages with `content.LoadPage`, rerenders their Markdown, merges them into the previous page list, removes deleted pages, rebuilds the site index, and renders the site.

Even though only changed Markdown files are reloaded, rendering still produces a complete next page set. That lets `diffRenderedPages` detect secondary changes such as updated section listings, pagination, previous/next links, and taxonomy pages.

## Full Rebuilds

Full rebuilds reload config, content, Markdown, site index, and rendered pages from scratch. Incremental output writing can still write only changed rendered pages, but the in-memory state is rebuilt fully.

Template and config changes intentionally use full mode because they can affect every page.

## Output Updates

After a partial or full rebuild, `writeIncrementalOutputs`:

1. writes changed rendered pages
2. removes stale rendered routes
3. removes generated artifacts disabled by new config
4. reruns the optional asset pipeline
5. rewrites feeds, sitemap, robots, and search
6. syncs changed static assets
7. recopies page bundle resources

After a successful rebuild, the server sends a livereload event.

## Known Boundaries

Incremental rebuilds reuse existing sections during Markdown page edits. If a section file changes, rebuild mode becomes full because section metadata can affect page rendering, routing, pagination, and template selection.

When adding a new file category, update `rebuildMode` deliberately. The key question is whether the changed file can affect routing, rendered HTML, generated artifacts, or only copied bytes.
