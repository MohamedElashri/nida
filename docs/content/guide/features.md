+++
title = "Features"
description = "A tour of what Nida does, with links to the details."
weight = 25
template = "page"
+++

Nida is small, but it covers what a blog, personal site, or documentation site needs. Each section links to the full documentation.

## Content

Markdown files under `content/` become pages; `_index.md` files make sections, and a directory with `index.md` is a page bundle whose images and files are published next to it. Sections nest, and their `_index.md` controls sorting, pagination, and templates. See [Front Matter](@/reference/front-matter.md).

Pages can link to each other by source path, such as `[Templates](@/guide/templates.md)`, and Nida fills in the published URL. Front matter can be TOML or YAML.

## Writing

GitHub-flavored Markdown with footnotes, plus summaries, tables of contents, heading anchors, callouts, code block titles and line highlighting, math, Mermaid diagrams, and shortcodes. Images get their dimensions and, with the image pipeline, responsive variants. See [Writing Content](@/guide/writing.md).

## Templates and themes

Templates are Go HTML templates with Nida's [template functions](@/guide/templates.md#template-functions) and a built-in [SEO partial](@/guide/templates.md#seo-metadata). Themes can extend other themes, and a site can override any part of its theme. See [Themes](@/guide/themes.md).

## Blogs

[Taxonomies](@/guide/configuration.md#taxonomies) such as tags and categories, [related posts and series](@/guide/templates.md#related-posts-and-series), [archives](@/guide/templates.md#archives) by year or month, Atom and RSS [feeds](@/reference/config.md#feed) for the site, sections, and tags, and [`nida new`](@/reference/cli.md#new) for new posts.

## Documentation sites

[Generated navigation, breadcrumbs, and edit links](@/guide/templates.md#navigation-for-docs-sites), [data files](@/guide/templates.md#data-files), [translations](@/guide/templates.md#translations) for theme text, and client-side [search](@/reference/search.md). This site is built that way.

## Checks and publishing

[`nida check`](@/reference/cli.md#check) finds broken links, missing images, template errors, and config typos without writing files. [`nida serve`](@/reference/cli.md#serve) previews the site with live reload. [`nida build`](@/reference/cli.md#build) writes the site, a sitemap, and `robots.txt`, with optional [fingerprinting and minification](@/reference/config.md#pipeline). See [Deployment](@/guide/deployment.md) for hosting.

## Moving from another generator

[`nida migrate`](@/guide/migrating.md) converts a Hugo, Jekyll, Zola, Eleventy, or Pelican site into a Nida site. It converts the config, content, front matter, shortcodes, and static files, keeps every page at its old URL, and writes a report of what is left to do by hand.
