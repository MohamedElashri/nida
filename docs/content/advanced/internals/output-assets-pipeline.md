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

`WritePages` sorts rendered pages by URL, then writes them concurrently with `internal/parallel`. Each page maps to its own file, so the result does not depend on scheduling, and errors are reported for the lowest URL that failed.

## Incremental Writes

Serve mode does not clear the output directory after every change. It compares previous and next rendered pages by URL and writes changed pages. It also removes routes that disappeared.

Generated artifacts are rewritten after incremental rebuilds because feeds, sitemap, robots, and search can depend on more than one changed page.

## Write Plan Validation

`ValidateWritePlan` checks page output paths and artifact output paths before writes happen.

This catches conflicts such as a page route that writes `rss.xml` or two routes that collapse to the same `index.html` path.

## Static Assets

`internal/assets.Copy` copies static files from layered static roots:

1. site static files
2. selected theme static files
3. parent theme static files, nearest first

The copy operation skips a file if the destination already exists, so the first root that has a path wins: the site overrides its theme, and a theme overrides its parent. Files the asset pipeline already wrote are skipped the same way.

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

With `pipeline.images.enabled`, `pipeline.ResizeVariants` also writes width variants next to each copied JPEG, PNG, or GIF resource, such as `screenshot.480w.png`. Files that cannot be decoded are copied without variants.

## Asset Pipeline

`internal/pipeline.Process` runs only when `PipelineConfig.Active` reports at least one enabled step:

- `pipeline.fingerprint`
- `pipeline.minify_css` or `pipeline.minify_js`
- `pipeline.images.enabled`
- `pipeline.scss.enabled`

It resolves the static roots in the same order as `assets.Copy` (site, theme, parent themes), creates output as needed, then:

1. compiles SCSS when enabled
2. processes static files from every root, skipping relative paths an earlier root already handled
3. writes `manifest.json` when fingerprinted assets exist

## Minification

`minifyAsset` minifies `.css`, `.js`, and `.mjs` files with `github.com/tdewolff/minify/v2` when the matching option is on. Files ending in `.min.css` or `.min.js` are left alone, and a file the minifier cannot parse is copied unchanged rather than failing the build. Minification happens before fingerprinting, so the hash matches the bytes that are served.

## Fingerprinting

Fingerprinting reads a file, hashes it with SHA-256, uses the first eight hex characters, and writes a sibling path like:

```text
style.css -> style.1a2b3c4d.css
```

The manifest maps original relative paths to fingerprinted relative paths. After processing, `RewriteOutputFiles` rewrites matching references in generated HTML.

For example:

```json
{
  "style.css": "style.1a2b3c4d.css",
  "images/hero.768w.jpg": "images/hero.9f8e7d6c.jpg"
}
```

Manifest keys and values are relative to the output root and do not include `base_url` path prefixes. Template helpers such as `asset` emit base-path-aware URLs before rewriting; `RewriteOutputFiles` preserves that prefix when it swaps in fingerprinted filenames.

The HTML rewrite handles common `href`, `src`, and `srcset` attributes for CSS, JS, images, and SVG links. `srcset` candidates are rewritten one URL at a time. It is intentionally conservative and regex-based, so changes should include tests for the exact markup shape being supported.

## SCSS

SCSS compilation requires the external `sass` CLI.

Nida compiles non-partial `.scss` and `.sass` files from:

- theme `scss/`
- site static SCSS entry directory, usually `static/css`

Output is written under the configured SCSS entry directory in the output root. Compilation uses compressed output and no source maps.

## Image Processing

Image processing supports PNG, JPEG, GIF, and WebP detection, with resized output currently encoded for JPEG, PNG, and GIF. Static images and bundle resources share `scaleImage` and `encodeImage`.

The pipeline generates the union of `pipeline.images.widths` and all `pipeline.images.presets.<name>.widths`. Target widths greater than or equal to the original width are skipped.

Templates can use the same presets:

```html
<img
  src="{{ asset "images/hero.jpg" }}"
  srcset="{{ imagePresetSrcset "hero" "images/hero.jpg" }}"
  sizes="{{ imagePresetSizes "hero" }}"
  alt="">
```

The pipeline guards image processing with maximum file size, maximum dimensions, and maximum pixel count.

When fingerprinting is also enabled, both original and resized static image paths can be fingerprinted and added to the manifest. Bundle images are not fingerprinted.

Markdown images get `width`, `height`, and a matching `srcset` from the site image resolver; see [Site Index And Routing](@/advanced/internals/site-index-routing.md#images).

## Adding Pipeline Behavior

Add tests for:

- missing static directories
- symlink rejection
- output paths staying inside the output root
- manifest contents
- HTML rewrite behavior
- interaction with incremental rebuilds

Pipeline behavior often touches filesystem and rendered HTML assumptions, so test package behavior directly before relying on CLI coverage.
