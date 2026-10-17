+++
title = "Config Reference"
description = "All supported config keys for Nida sites."
weight = 30
template = "page"
+++

This page documents the `config.toml` settings Nida understands.

The implementation lives in [`internal/config`](https://github.com/MohamedElashri/nida/tree/main/internal/config).

## Minimal config

```toml

base_url = "https://example.com/"
title = "My Site"
```

`base_url` and `title` are the important starting points. Most other settings have defaults.

Nida warns about keys it does not recognize, for example `warning: unknown config key "paginte" is ignored`. Unknown keys never fail the build, but the warning catches typos and settings from other generators that Nida does not support. Keys under `extra` are never reported.

## Top-level settings

| Key | Type | Default | Description |
| --- | --- | --- | --- |

| `base_url` | string | required | Absolute `http` or `https` URL for the site. May include a path such as `/nida/`. |
| `title` | string | required | Site title. |
| `description` | string | empty | Site description used by templates and generated metadata. |
| `language` | string | `en` | Site language. Also used for document direction detection. |
| `author` | string | empty | Site author. |
| `content_dir` | string | `content` | Directory containing Markdown content. |
| `template_dir` | string | `templates` | Directory containing site templates. |
| `static_dir` | string | `static` | Directory containing static assets. |
| `output_dir` | string | `public` | Directory where generated output is written. |
| `themes_dir` | string | `themes` | Directory containing available themes. |
| `theme` | string | empty | Theme name under `themes_dir`. Empty means use site templates/static directly. |
| `paginate` | integer | `10` | Default pagination size. Must be greater than zero. |
| `drafts` | boolean | `false` | Include draft content when building. |
| `future` | boolean | `false` | Include pages whose `date` is in the future. Future-dated pages are skipped by default so scheduled posts stay hidden until the next build after their date. |
| `robots_meta` | string | empty | Value exposed to templates as `.Robots` for a `<meta name="robots">` tag, such as `"noai, noimageai"`. The 404 page always adds `noindex`. |
| `minify_html` | boolean | `false` | Minify rendered HTML output. |
| `syntax_theme` | string | `github` | Syntax highlighting theme name. |
| `extra` | table | empty | Custom values available to templates as `.Config.Extra`. |

## Markdown

```toml
[markdown]
hard_wraps = false
external_links_target_blank = false
external_links_no_follow = false
external_links_no_referrer = false
unsafe_html = false
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `markdown.hard_wraps` | boolean | `false` | Render single newlines inside a paragraph as `<br>`. Off by default, matching CommonMark: a single newline is a space and a blank line starts a new paragraph. |
| `markdown.external_links_target_blank` | boolean | `false` | Add `target="_blank"` to external HTTP(S) links. |
| `markdown.external_links_no_follow` | boolean | `false` | Add `nofollow` to external link `rel` values. |
| `markdown.external_links_no_referrer` | boolean | `false` | Add `noreferrer` to external link `rel` values. |
| `markdown.unsafe_html` | boolean | `false` | Allow raw HTML in Markdown. Enable only for trusted-author sites. |

## Sections

```toml
[sections]
default_page_template = "page"
default_sort_by = "date"
paginate_by = 10
paginate_path = "page"
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sections.default_page_template` | string | empty | Default template for pages when neither the page `template` nor the section `page_template` is set. Falls back to `post`, then `page`. |
| `sections.default_sort_by` | string | `date` | Default section sort mode. Common values include `date`, `title`, `weight`, and `slug`. |
| `sections.paginate_by` | integer | `0` | Section pagination size. Falls back to top-level `paginate` when unset. |
| `sections.paginate_path` | string | `page` | Path segment used for paginated section pages. A section's own `paginate_path` front matter takes precedence. |

## Home

```toml
[home]
sections = ["posts"]
limit = 5
```

The homepage template receives recent pages in `.Pages`.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `home.sections` | array of strings | empty | Root sections to list on the homepage. Empty includes every section. Nested sections match by root, so `posts/2026` matches `posts`. |
| `home.limit` | integer | `5` | Number of pages in `.Pages` on the homepage. `0` lists all of them. When the homepage paginates, `home.sections` also filters the paginated list. |

## Taxonomies

Taxonomies are configured with repeated `[[taxonomies]]` tables.

```toml
[[taxonomies]]
name = "tags"
paginate_by = 20
paginate_path = "page"
feed = false
render = true
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `taxonomies[].name` | string | required | Taxonomy name. Values are read from page `extra` metadata with the same key. |
| `taxonomies[].paginate_by` | integer | `0` | Number of items per taxonomy term page. `0` disables taxonomy pagination. |
| `taxonomies[].paginate_path` | string | empty | Path segment for paginated taxonomy pages. Falls back to `page`. |
| `taxonomies[].feed` | boolean | `false` | Reserved for taxonomy feed behavior. |
| `taxonomies[].render` | boolean | `false` | Whether to render taxonomy list and term pages. |

## Permalinks

Permalinks are configured as a table keyed by section path.

```toml
[permalinks]
posts = "/posts/{slug}/"
pages = "/{slug}/"
```

| Placeholder | Description |
| --- | --- |
| `{section}` | Page section path. |
| `{slug}` | Page slug. |
| `{year}` | Four-digit year from the page date. |
| `{month}` | Two-digit month from the page date. |
| `{day}` | Two-digit day from the page date. |
| `{taxonomy-name}` | First value for a configured taxonomy, when available. |

Permalink routes must stay site-relative and safe. Nida validates generated routes before writing output.

## Feed

Nida generates an Atom feed by default.

```toml
[feed]
enabled = true
filename = "feed.xml"
limit = 20
sections = ["posts"]
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `feed.enabled` | boolean | `true` | Generate the Atom feed. |
| `feed.filename` | string | `feed.xml` | Output filename for the Atom feed. |
| `feed.limit` | integer | `20` | Maximum number of feed entries. |
| `feed.sections` | array of strings | empty | Root content sections to include, such as `["posts"]`. Omit or leave empty to include all pages. Nested section paths match by root section, so `posts/notes` matches `posts`. |

Entries use the page `description` as the summary and include the full rendered HTML with links made absolute. An entry's `updated` time comes from the page's `updated` front matter when that is newer than `date`.

## RSS

Some feed readers and services still expect RSS 2.0. Nida can write an RSS feed next to the Atom feed. It is off by default.

```toml
[rss]
enabled = true
filename = "rss.xml"
limit = 20
sections = ["posts"]
```

The keys mean the same as in `[feed]`. The defaults are `enabled = false` and `filename = "rss.xml"`. `rss.filename` must differ from `feed.filename`.

## Sitemap

```toml
[sitemap]
enabled = true
filename = "sitemap.xml"
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sitemap.enabled` | boolean | `true` | Generate a sitemap. |
| `sitemap.filename` | string | `sitemap.xml` | Output filename for the sitemap. |

## Robots

```toml
[robots]
enabled = true
filename = "robots.txt"
content = ""
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `robots.enabled` | boolean | `true` | Generate `robots.txt`. |
| `robots.filename` | string | `robots.txt` | Output filename for robots content. |
| `robots.content` | string | empty | Custom robots content. When empty, Nida generates a default file. |

## Search

```toml
[search]
enabled = false
filename = "search_index.en.js"
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `search.enabled` | boolean | `false` | Generate a JavaScript search index. |
| `search.filename` | string | `search_index.en.js` | Output filename for the search index. |

See [Search](@/reference/search.md) for the generated index format and a small
client-side template pattern.

## Diagnostics

```toml
[diagnostics]
enabled = false
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `diagnostics.enabled` | boolean | `false` | Check content before rendering and fail builds for broken `@/` internal links or missing Markdown image assets. |

## Server

```toml
[server]
host = "127.0.0.1"
port = 1702
livereload = true
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `server.host` | string | `127.0.0.1` | Host for `nida serve`. |
| `server.port` | integer | `1702` | Port for `nida serve`. Must be between 1 and 65535. |
| `server.livereload` | boolean | `true` | Inject live reload during local development. |

## Pipeline

```toml
[pipeline]
fingerprint = false
minify_css = false
minify_js = false

[pipeline.images]
enabled = false
widths = [480, 768, 1200]
quality = 85

[pipeline.images.presets.content]
widths = [480, 768, 1200]
sizes = "(max-width: 760px) 100vw, 760px"

[pipeline.scss]
enabled = false
entry_dir = "css"
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `pipeline.fingerprint` | boolean | `false` | Fingerprint eligible static assets and rewrite matching HTML references. |
| `pipeline.minify_css` | boolean | `false` | Minify CSS assets. |
| `pipeline.minify_js` | boolean | `false` | Minify JavaScript assets. |
| `pipeline.images.enabled` | boolean | `false` | Process image assets. |
| `pipeline.images.widths` | integer array | `[480, 768, 1200]` | Widths for generated image variants. |
| `pipeline.images.quality` | integer | `85` | Output image quality. |
| `pipeline.images.presets.<name>.widths` | integer array | see below | Widths for a named image preset. Preset widths are also generated by the pipeline. |
| `pipeline.images.presets.<name>.sizes` | string | see below | HTML `sizes` value returned by `imagePresetSizes`. |
| `pipeline.scss.enabled` | boolean | `false` | Compile SCSS from the configured static entry directory. |
| `pipeline.scss.entry_dir` | string | `css` | Static subdirectory containing SCSS entry files. |

Default image presets are `thumb`, `content`, and `hero`. They are intended for small cards, article body images, and full-width hero images.

## Extra

```toml
[extra]
footer = "Built with Nida"
main_menu = [
  { name = "Home", url = "/" },
]
```

`extra` is intentionally open-ended. Templates can read values with `.Config.Extra`.
