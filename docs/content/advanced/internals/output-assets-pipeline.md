+++
title = "Output, Assets, And Pipeline"
description = "How rendered pages, generated files, static assets, and processed assets reach public output."
weight = 60
template = "page"
+++

Nida separates rendered pages, generated artifacts, static asset copying, page-bundle resources, and optional asset processing. That separation keeps output writes easier to reason about.

## Output Paths

`internal/output` maps routes to files:

- `/` becomes `index.html`
- `/posts/hello/` becomes `posts/hello/index.html`
- `/404.html` becomes `404.html`

Routes must start with `/`. Cleaned output paths cannot escape the output directory.

## Full Writes

`WriteSite` resolves the output directory, removes it, recreates it, and calls `WritePages`.

`WritePages` sorts rendered pages by URL before writing. Deterministic ordering matters for reproducible builds and tests.

## Incremental Writes

Serve mode does not clear the output directory after every change. It compares previous and next rendered pages by URL and writes changed pages. It also removes routes that disappeared.

Generated artifacts are rewritten after incremental rebuilds because feeds, sitemap, robots, and search can depend on more than one changed page.

## Write Plan Validation

`ValidateWritePlan` checks page output paths and artifact output paths before writes happen.

This catches conflicts such as a page route that writes `rss.xml` or two routes that collapse to the same `index.html` path.

## Static Assets

`internal/assets.Copy` copies static files from layered static roots:

1. theme static files
2. site static files

The copy operation skips a file if the destination already exists. Because theme static files copy first, site static files take precedence on conflicts.

Static symlinks are rejected. Missing static directories are allowed.

## Page Bundles

A page bundle is a directory with `index.md`. Non-Markdown files beside that index are copied next to the rendered page.

For example:

```text
content/posts/launching-nida/
├── index.md
└── screenshot.png
```

If the page route is `/posts/launching-nida/`, `screenshot.png` is copied into `public/posts/launching-nida/screenshot.png`.

## Asset Pipeline

`internal/pipeline.Process` runs only when at least one of these is enabled:

- `pipeline.fingerprint`
- `pipeline.images.enabled`
- `pipeline.scss.enabled`

It resolves the static root and output root, creates output as needed, then:

1. compiles SCSS when enabled
2. processes static files
3. writes `manifest.json` when fingerprinted assets exist

## Fingerprinting

Fingerprinting reads a file, hashes it with SHA-256, uses the first eight hex characters, and writes a sibling path like:

```text
style.css -> style.1a2b3c4d.css
```

The manifest maps original relative paths to fingerprinted relative paths. After processing, `RewriteOutputFiles` rewrites matching references in generated HTML.

The HTML rewrite handles common `href`, `src`, and `srcset` attributes for CSS, JS, images, and SVG links. It is intentionally conservative and regex-based, so changes should include tests for the exact markup shape being supported.

## SCSS

SCSS compilation requires the external `sass` CLI.

Nida compiles non-partial `.scss` and `.sass` files from:

- theme `scss/`
- site static SCSS entry directory, usually `static/css`

Output is written under the configured SCSS entry directory in the output root. Compilation uses compressed output and no source maps.

## Image Processing

Image processing supports PNG, JPEG, GIF, and WebP detection, with resized output currently encoded for JPEG, PNG, and GIF.

The pipeline guards image processing with maximum file size, maximum dimensions, and maximum pixel count. Target widths greater than or equal to the original width are skipped.

When fingerprinting is also enabled, both original and resized image paths can be fingerprinted and added to the manifest.

## Adding Pipeline Behavior

Add tests for:

- missing static directories
- symlink rejection
- output paths staying inside the output root
- manifest contents
- HTML rewrite behavior
- interaction with incremental rebuilds

Pipeline behavior often touches filesystem and rendered HTML assumptions, so test package behavior directly before relying on CLI coverage.
