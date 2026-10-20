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
| `data_dir` | string | `data` | Directory of TOML, JSON, and YAML files exposed to templates as `.Data`. |
| `theme` | string | empty | Theme name under `themes_dir`. Empty means use site templates/static directly. |
| `paginate` | integer | `10` | Default pagination size. Must be greater than zero. |
| `drafts` | boolean | `false` | Include draft content when building. |
| `future` | boolean | `false` | Include pages whose `date` is in the future. Future-dated pages are skipped by default so scheduled posts stay hidden until the next build after their date. |
| `updated_from_git` | boolean | `false` | Set each page's `updated` date from the last git commit that changed its file, unless front matter sets `updated`. Uses one `git log` call per build; skipped outside a git repository. |
| `edit_url` | string | empty | Prefix for `.EditURL`, the link to a page's source. For GitHub, use `https://github.com/user/repo/edit/main/content/`. |
| `robots_meta` | string | empty | Value exposed to templates as `.Robots` for a `<meta name="robots">` tag, such as `"noai, noimageai"`. The 404 page always adds `noindex`. |
| `minify_html` | boolean | `false` | Minify rendered HTML output. |
| `syntax_theme` | string | `github` | [Chroma style](https://xyproto.github.io/splash/docs/) for the `syntaxCSS` template function. Code blocks are rendered with `z-` CSS classes, so a theme must include highlighting CSS, either a static stylesheet or `<style>{{ syntaxCSS }}</style>`. |
| `extra` | table | empty | Custom values available to templates as `.Config.Extra`. |

## Markdown

```toml
[markdown]
hard_wraps = false
heading_anchors = false
math = false
math_font = "latin-modern"
diagram_width = 720
external_links_target_blank = false
external_links_no_follow = false
external_links_no_referrer = false
unsafe_html = false
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `markdown.hard_wraps` | boolean | `false` | Render single newlines inside a paragraph as `<br>`. Off by default, matching CommonMark: a single newline is a space and a blank line starts a new paragraph. |
| `markdown.heading_anchors` | boolean | `false` | Append a `#` self-link to each heading. See [Writing Content](@/guide/writing.md#table-of-contents). |
| `markdown.math` | boolean | `false` | Render `$...$` and `$$...$$` TeX to MathML at build time. While off, TeX in content fails the build. See [Writing Content](@/guide/writing.md#math). |
| `markdown.math_font` | string | `latin-modern` | Site-wide math stylesheet: `latin-modern` ships Latin Modern Math (390 KB), `local` uses the reader's installed math fonts (10 KB), and `none` ships nothing. See [Math fonts](@/guide/writing.md#math-fonts). |
| `markdown.diagram_width` | integer | `720` | Width in pixels that Mermaid diagrams are laid out for, usually your content column. Wider diagrams wrap labels or switch to a vertical layout instead of shrinking. `0` keeps each diagram's natural layout. See [Diagrams](@/guide/writing.md#diagrams). |
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
| `sections.default_sort_by` | string | `date` | Default section sort mode: `date` (newest first), `title`, `weight` (lowest first), or `none` (file order). Ties sort by slug. |
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
| `home.limit` | integer | `0` | Most recent pages the homepage lists, split into pages of `paginate`. `0` lists all of them. |

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
| `taxonomies[].feed` | boolean | `false` | Write a feed for every term, such as `tags/go/feed.xml` (and `tags/go/rss.xml` when `[rss]` is enabled). |
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

A route normally ends in `/` and is written as `index.html` inside that directory. When the pattern ends in a page or feed extension such as `.html`, `.php`, or `.xml`, as in `/{year}/{slug}.html`, the page is written as that file instead. Other dots do not count, so a slug like `node.js` still gets a directory. A page's `path` front matter overrides its section's pattern.

A key named after a taxonomy sets that taxonomy's term URLs instead. The defaults are:

```toml
[permalinks]
tags = "/tags/{slug}/"
categories = "/categories/{slug}/"
```

The taxonomy list page is the pattern with `{slug}` removed, here `/tags/`. Sections always render at `/<section>/`; permalinks apply only to the pages inside them.

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

Sections with `generate_feeds = true` in their `_index.md` get their own feed at `<section>/feed.xml`, which includes pages in nested sections. Taxonomies with `feed = true` get one feed per term. Scoped feeds use the same filename and limit as the site feed.

Entries use the page `description` as the summary, falling back to the [page summary](@/guide/writing.md#summaries), and include the full rendered HTML with links made absolute. An entry's `updated` time comes from the page's `updated` front matter when that is newer than `date`.

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
sections = []
body_limit = 0
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `search.enabled` | boolean | `false` | Generate a JavaScript search index. |
| `search.filename` | string | `search_index.en.js` | Output filename for the search index. |
| `search.sections` | array of strings | empty | Root sections to index, such as `["guide", "reference"]`. Empty indexes everything. |
| `search.body_limit` | integer | `0` | Keep at most this many characters of each body. `0` keeps full text. Lowering it shrinks the index for larger sites. |

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
| `pipeline.scss.enabled` | boolean | `false` | Compile SCSS with the `sass` command. Sources are each theme's `scss/` directory and the site's `static/<entry_dir>/`; files starting with `_` are partials and are not compiled on their own. |
| `pipeline.scss.entry_dir` | string | `css` | Static subdirectory with the site's SCSS files, and the output directory for compiled CSS. |

Default image presets are `thumb`, `content`, and `hero`. They are intended for small cards, article body images, and full-width hero images.

## Extra

```toml
[extra]
footer = { text = "Built with Nida" }
main_menu = [
  { name = "Home", url = "/" },
]
```

`extra` is intentionally open-ended. Templates can read values with `.Config.Extra`.

## Translations

Theme and template UI strings can be translated with `i18n/<language>.toml` files in the site or a theme:

```toml
# i18n/ar.toml
read_more = "اقرأ المزيد"
posts_count = "{count} مقالات"

[callout]
note = "ملاحظة"
warning = "تحذير"
```

Nida picks files from the site `language`. For `ar-EG` it layers `en.toml`, then `ar.toml`, then `ar-eg.toml`, so missing keys fall back to broader languages. Site files override theme files. Templates use the `t` function, and `callout.<type>` keys translate default callout titles.

