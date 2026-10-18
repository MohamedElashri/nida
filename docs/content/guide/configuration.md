+++
title = "Configuration"
description = "Set up config.toml: site identity, content options, feeds, Markdown, and the asset pipeline."
weight = 20
template = "page"
+++

Every Nida site has a `config.toml` in its root. This page walks through the settings most sites change. The [Config Reference](@/reference/config.md) lists every key.

## A typical blog

```toml {title="config.toml"}
base_url = "https://example.com/"
title = "My Site"
description = "Notes from a quiet corner of the web."
language = "en"
author = "Your Name"

[[taxonomies]]
name = "tags"
render = true
feed = true

[feed]
sections = ["posts"]

[permalinks]
posts = "/posts/{year}/{slug}/"
pages = "/{slug}/"

[markdown]
heading_anchors = true
external_links_target_blank = true
external_links_no_referrer = true

[search]
enabled = true
```

Only `base_url` and `title` are required. Everything else has a default.

Nida warns about keys it does not recognize, so typos show up in the build output. See [Config Reference](@/reference/config.md) for every key.

## Site identity

`base_url` must be an absolute `http` or `https` URL. Nida uses it for canonical URLs, feeds, the sitemap, and the `.BasePath` template value. When it contains a path, such as `https://example.com/blog/`, every generated link includes that path.

`language` sets the page language and text direction: `ar`, `fa`, `he`, `ur`, and a few others render right-to-left. It also picks the [translation files](@/reference/config.md#translations) for theme strings.

`author` is used in feeds and the built-in SEO metadata.

## Content

```toml
paginate = 10
future = false
drafts = false
updated_from_git = false

[home]
sections = ["posts"]
limit = 5

[sections]
default_sort_by = "date"
paginate_by = 10
paginate_path = "page"
```

Pages with `draft = true` and pages dated in the future are skipped. Use `--drafts` and `--future` on the command line to preview them.

`[home]` controls which pages the homepage lists in `.Pages`.

`updated_from_git` sets each page's `updated` date from its last commit, so "last updated" lines stay accurate without front matter edits.

Sections can override sorting and pagination in their `_index.md`. See [Front Matter](@/reference/front-matter.md#section-fields).

## Permalinks

Permalinks set page URLs per section:

```toml
[permalinks]
posts = "/posts/{year}/{month}/{slug}/"
notes = "/{section}/{slug}/"
```

Placeholders are `{section}`, `{slug}`, `{year}`, `{month}`, `{day}`, and any taxonomy name, such as `{tags}`, which uses the page's first term. Sections themselves always live at `/<section>/`.

## Taxonomies

```toml
[[taxonomies]]
name = "tags"
render = true
paginate_by = 20
feed = true

[[taxonomies]]
name = "series"
```

Pages assign terms in front matter with `tags = ["go", "web"]`. Term URLs come from [permalinks](@/reference/config.md#permalinks). With `render = true`, Nida renders a list page at `/tags/` and a page per term. With `feed = true`, each term also gets a feed. Taxonomies power `.Index.Related` for related posts.

## Feeds

Nida writes an Atom feed by default. Add RSS if your readers need it:

```toml
[feed]
sections = ["posts"]

[rss]
enabled = true
```

A section with `generate_feeds = true` in its `_index.md`, or a taxonomy with `feed = true`, gets its own feed. See [Feed](@/reference/config.md#feed) for filenames, limits, and summaries.

## Markdown

```toml
[markdown]
heading_anchors = true
math = true
```

`heading_anchors` adds `#` links to headings and `math` passes TeX through for KaTeX or MathJax. See [Writing Content](@/guide/writing.md) for the syntax and [Markdown](@/reference/config.md#markdown) for every option, including external link attributes and hard wraps.

> [!WARNING]
> Raw HTML in Markdown is off by default. Turn on `unsafe_html` only when you trust everyone who writes content. Shortcodes are the safer way to add custom HTML.

## Search, sitemap, and robots

A sitemap and `robots.txt` are written by default. Client-side search is opt-in:

```toml
[search]
enabled = true
```

See [Search](@/reference/search.md) for the index format and a search UI, and `robots_meta` in the [Config Reference](@/reference/config.md#top-level-settings) for a site-wide robots meta tag.

## Asset pipeline

```toml
[pipeline]
fingerprint = true
minify_css = true
minify_js = true

[pipeline.images]
enabled = true
```

Every step is optional. Fingerprinting adds a content hash to file names so browsers can cache aggressively, and image processing writes resized variants used by Markdown images and the [image helpers](@/guide/templates.md). See [Pipeline](@/reference/config.md#pipeline) for widths, presets, quality, and SCSS.

## Development server

`nida serve` listens on `127.0.0.1:1702`. Change it with `[server] port` or `nida serve --port 4000`; see [Server](@/reference/config.md#server).

## Custom values

```toml
[extra]
twitter = "@example"
image = "/images/social.png"

[[extra.main_menu]]
name = "Posts"
url = "/posts/"
```

`[extra]` is free-form. Templates read it as `.Config.Extra`, or as `.Theme.Extra` merged with the theme's defaults. For larger structured data, use [data files](@/guide/templates.md#data-files).
