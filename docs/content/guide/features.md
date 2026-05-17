+++
title = "Features"
description = "A tour of the major capabilities Nida already supports."
weight = 25
template = "page"
+++

Nida is small, but it already covers the pieces needed for a practical static site.

## Content model

Nida discovers Markdown under `content/`.

- `_index.md` creates a section.
- `page.md` creates a regular page.
- `some-page/index.md` creates a page bundle.

Sections can be nested, and pages inherit behavior from their section when a section defines things like `page_template` or sorting.

## Markdown rendering

Nida uses GitHub-flavored Markdown, footnotes, heading IDs, and syntax-highlighted fenced code blocks.

Internal links can use Zola-style `@/` paths:

```markdown
Read [Templates](@/guide/templates.md).
```

Nida resolves those paths to generated URLs during rendering.

## Templates and themes

Templates are Go HTML templates. Sites can provide local templates directly, or use a theme from `themes/name/`.

Themes can extend parent themes, and site templates override theme templates with the same name.

## Taxonomies

Nida can build arbitrary taxonomies, not just tags and categories.

```toml
[[taxonomies]]
name = "tags"
render = true
paginate_by = 20
```

Content can assign taxonomy values in front matter through `extra`:

```toml
[extra]
tags = ["docs", "release"]
```

## Generated files

Nida can generate RSS, Atom, sitemap, robots.txt, and a JavaScript search index.

These features are controlled from `config.toml`, so small sites can disable what they do not need. The generated search index is a plain `window.searchIndex` script that templates can load for client-side search.

## Content diagnostics

Nida can check content before rendering and report broken `@/` internal links and missing Markdown image assets.

## Static assets and pipeline

Files under `static/` are copied to the output directory. Nida also includes optional pipeline features for fingerprinting, CSS/JS minification, SCSS compilation, and image processing.

## Local development server

`nida serve` builds the site, serves the output directory, watches for changes, and injects live reload when enabled.

When `base_url` contains a path such as `/nida/`, the dev server also serves that base path locally so project-hosted sites can be previewed accurately.
