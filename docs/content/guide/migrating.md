+++
title = "Migrating to Nida"
description = "Move a Zola, Hugo, or Jekyll site to Nida: what carries over, what to convert, and how to keep your URLs."
weight = 45
template = "page"
+++

Most of a site's value is its content, and Nida reads content from other generators with few changes. Templates are the part you rewrite. This page lists what carries over from Zola, Hugo, and Jekyll, and what to change.

## The general approach

{{< steps >}}

### Copy content and static files

Copy your Markdown into `content/` and your images, CSS, and other assets into `static/`. Keep section directories and page bundles as they are.

### Write a config.toml

Start from your old config. Nida warns about every key it does not understand, so you can run a check and work through the list:

```bash
nida check --site .
```

```text
warning: unknown config key "generate_feed" is ignored
warning: unknown config key "highlight_code" is ignored
```

### Pick templates

Use a theme, or start from the templates `nida init` creates and adapt your old layout to them. See [Templates](@/guide/templates.md) and [Themes](@/guide/themes.md).

### Keep your URLs

Match your old URL scheme with [permalinks](@/guide/configuration.md#permalinks), and add `aliases` front matter for pages whose URLs change. Nida writes a redirect page for each alias. Compare the old and new `sitemap.xml` files to find anything you missed.

### Check everything

`nida check` fails on broken internal links, missing images, unknown shortcodes, and template errors. Add `--external` to also check outbound links.

{{< /steps >}}

## From Zola

Zola and Nida share a lot: TOML front matter between `+++`, `_index.md` sections, page bundles, `@/` internal links, and the search index format.

| Zola | Nida |
| --- | --- |
| `+++` TOML front matter | Same. |
| `[taxonomies]` in front matter | Same; terms are read as taxonomy values. |
| `[extra]` | Same. |
| `@/posts/hello.md` links | Same. |
| `sort_by`, `paginate_by`, `paginate_path`, `page_template`, `aliases`, `weight` | Same names. |
| `transparent = true` | Accepted, but pages are not passed up to the parent section. List them from the parent template with `where .Index.AllPages "SectionPath" "..."` instead. |
| `generate_feeds` on a section | Same; writes a feed for that section. |
| `{% details(summary="...") %}` blocks and `{{< rawhtml >}}` markers | Supported. |
| Other `{{ name(args) }}` shortcodes | Rewrite as `{{< name key="value" >}}` with a template in `templates/shortcodes/`. |
| Tera templates | Rewrite as Go templates. |
| `insert_anchor_links` | `[markdown] heading_anchors = true`. |
| `<!-- more -->` | Same. |

Config keys:

| Zola `config.toml` | Nida `config.toml` |
| --- | --- |
| `base_url`, `title`, `description` | Same. |
| `default_language` | `language` |
| `generate_feeds = true`, `feed_filenames` | `[feed]` (Atom) and `[rss]` |
| `build_search_index = true` | `[search] enabled = true` |
| `[[taxonomies]]` with `name`, `paginate_by`, `feed` | Same, plus `render = true` to render term pages. |
| `[markdown] external_links_*` | Same names. |
| `[markdown] highlight_code`, `highlight_theme` | Highlighting is always on; set `syntax_theme`. |

Nida's `search_index.en.js` assigns a `window.searchIndex` object with a `documentStore.docs` map, a similar shape to Zola's elasticlunr index. Nida's keys are site-relative URLs, so a search script written for Zola usually needs only small changes. See [Search](@/reference/search.md).

## From Hugo

Hugo uses Go templates too, so its shortcode syntax and much of its template logic look familiar.

| Hugo | Nida |
| --- | --- |
| TOML (`+++`) or YAML (`---`) front matter | Same, including nested YAML. |
| `tags`, `categories`, and other taxonomy keys at the top level | Same, once the taxonomy is configured. |
| `lastmod` | Rename to `updated`, or set `updated_from_git = true`. |
| `summary` front matter, `<!--more-->` | Use `description` for a hand-written summary; `<!-- more -->` works. |
| `{{< figure >}}` and `{{< youtube >}}` | Built in, with the common arguments: `src`, `alt`, `caption`, `link`, `width`, `height`, and `class` for `figure`; the video id and `start` for `youtube`. |
| Other `{{< name >}}` shortcodes | Copy the template to `templates/shortcodes/name.html` and adjust it to Nida's [shortcode context](@/guide/writing.md#writing-a-shortcode). |
| `{{% name %}}` shortcodes | Use `{{< name >}}`; the body is available as Markdown-rendered `.InnerHTML`. |
| `data/`, `static/`, `archetypes/`, `i18n/` | Same directories. Translation files are TOML with `key = "value"` entries; tables such as `[callout]` become dotted keys. |
| Page bundles, `_index.md` sections | Same. |
| `layouts/` | Rename to `templates/`, and adapt the context. |

Common template changes:

| Hugo template | Nida template |
| --- | --- |
| `.Content` | `unsafeHTML .Page.BodyHTML` |
| `.Params.foo` | `.Page.Extra.foo` |
| `.Site.Title`, `.Site.Params.foo` | `.Config.Title`, `.Config.Extra.foo` |
| `.Permalink`, `.RelPermalink` | `.CanonicalURL`, `{{ .BasePath }}{{ .Page.URL }}` |
| `.Summary` | `.Page.Summary` |
| `.TableOfContents` | Build from `.Page.TOC`; see [Writing Content](@/guide/writing.md#table-of-contents). |
| `.Site.RegularPages`, `where` | `.Index.AllPages`, `where` |
| `.Site.GetPage "/about"` | `.Index.GetPage "pages/about.md"` |
| `partial "x" .` | `template "x" .` with `{{ define "x" }}` in a partial file |
| `T "key"`, `i18n "key"` | `t "key"` |
| `relURL`, `absURL`, `dict`, `slice`, `first`, `jsonify`, `markdownify`, `truncate`, `plainify` | Available with the same names. |
| `default "x" .Foo` | `default .Foo "x"`: Nida takes the value first. |

## From Jekyll

| Jekyll | Nida |
| --- | --- |
| YAML front matter | Same. |
| `_posts/2024-05-01-my-post.md` | Move to `content/posts/my-post.md` and add `date: 2024-05-01` to the front matter. Nida does not read dates from file names. |
| `layout: post` | `template = "post"`, or set `page_template` on the section. |
| `permalink:` on a page | Use `[permalinks]` for patterns, `slug` for one page, and `aliases` for old URLs. |
| `excerpt_separator` | `<!-- more -->` |
| `tags`, `categories` | Same, once the taxonomies are configured. |
| `_data/` | `data/` |
| `_includes/` | Partials in `templates/partials/`. |
| Liquid templates and `{% include %}` | Rewrite as Go templates. |
| `_config.yml` | `config.toml` |

To keep date-based URLs, use a permalink pattern:

```toml
[permalinks]
posts = "/{year}/{month}/{day}/{slug}/"
```

> [!WARNING]
> Nida's URLs always end in `/` and are written as `index.html` files. Jekyll's default post URLs end in `.html`, such as `/2024/05/01/my-post.html`, and Nida cannot produce those paths or aliases for them. Before migrating, either switch the Jekyll site to `permalink: pretty` and let the old URLs redirect there, or add redirect rules on your web host from the `.html` URLs to the new ones.

## After the move

- Run `nida check --site . --external` and fix what it reports.
- Compare page counts: `nida build` prints how many pages and routes it wrote.
- Diff the old and new `sitemap.xml` to confirm every URL still exists or redirects.
- Subscribe to your new feed in a reader to check titles, dates, and summaries.
