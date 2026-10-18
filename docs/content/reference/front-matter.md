+++
title = "Front Matter Reference"
description = "Metadata fields supported by Nida pages and sections."
weight = 20
template = "page"
+++

Nida supports TOML front matter with `+++` delimiters and YAML front matter with `---` delimiters. YAML supports the full syntax, including block lists and nested maps, so front matter from Jekyll or Hugo usually works unchanged.

The implementation lives in [`internal/frontmatter`](https://github.com/MohamedElashri/nida/tree/main/internal/frontmatter) and feeds the page and section types in [`internal/content`](https://github.com/MohamedElashri/nida/tree/main/internal/content).

A typical page starts like this:

```toml
+++
title = "About"
description = "What this page explains."
date = 2026-04-13T08:30:00Z
draft = false
slug = "about"
template = "page"
aliases = ["/old-about/"]

[extra]
tags = ["docs"]
+++
```

## Page fields

`title` sets the page title.

`description` sets the page summary used by templates, metadata, and list pages.

`date` sets the publication date. Pages dated in the future are skipped unless the site sets `future = true` or the build uses `--future`.

`updated` sets the update date.

`draft` excludes the page unless drafts are enabled.

`weight` provides a manual ordering value.

`slug` overrides the route slug derived from the file name.

`path` sets the page's URL exactly, overriding `slug` and the section's [permalinks](@/guide/configuration.md#permalinks). It is the same field as in Zola. `path = "about/team"` publishes the page at `/about/team/`. A path that ends in a page or feed extension (`.html`, `.htm`, `.php`, `.xml`, `.json`, `.txt`, and similar) names that file: `path = "/2024/05/01/launch.html"` writes `launch.html` instead of a directory with an `index.html`, which keeps URLs from generators that end them in `.html`.

`template` selects a page template.

`aliases` creates redirect pages for old URLs.

`extra` stores custom metadata, including taxonomy values. A few `extra` keys have built-in meaning:

- `image`: the social preview image used by the [`nida-seo` partial](@/guide/templates.md#seo-metadata). Relative paths resolve against the page URL, so a bundle can use `image = "cover.png"`.
- `series`: groups posts into a [series](@/guide/templates.md#related-posts-and-series).
- `authors`: names, or tables with `name`, `email`, and `uri`, for [feed](@/reference/config.md#feed) entries.

Keys Nida does not recognize at the top level are added to `extra`, so `tags = ["go"]` and `[extra] tags = ["go"]` mean the same thing. Zola's `[taxonomies]` table is supported as well: its entries are moved into `extra`, so this works unchanged:

```toml
[taxonomies]
tags = ["rust", "zola"]
```

## Section fields

Sections use `_index.md` files.

```toml
+++
title = "Guide"
description = "How to use Nida."
sort_by = "weight"
page_template = "page"
paginate_by = 20
+++
```

| Field | Description |
| --- | --- |
| `title`, `description`, `extra` | As for pages. |
| `template` | Template for the section page itself. |
| `page_template` | Default template for pages in this section. |
| `sort_by` | Page order: `date` (newest first, the default), `title`, `weight` (lowest first), or `none` (file order). |
| `paginate_by` | Pages per list page. `0` uses `[sections] paginate_by`, then the top-level `paginate`. |
| `paginate_path` | Path segment for later pages, such as `/posts/page/2/`. Defaults to `[sections] paginate_path`. |
| `generate_feeds` | Write a feed for the section and its nested sections at `<section>/feed.xml`. See [Feed](@/reference/config.md#feed). |
| `transparent` | Accepted for Zola compatibility. It lists the section in `.Index.Sections` alongside the root, but pages are not moved into the parent section as Zola does. |
| `weight` | Order among sibling sections, lowest first, then by title. |

`draft` applies to pages only; a section's own `draft` value does not hide it.

## Page bundles

A directory with `index.md` is treated as a page bundle.

```text
content/posts/my-post/
  index.md
  screenshot.png
```

Non-Markdown files in the bundle are copied with the page so related assets can live beside the content.

## Aliases

Aliases are useful when changing URLs:

```toml
aliases = ["/old-path/", "/another-old-path/"]
```

Nida emits redirect pages for each alias. An alias ending in one of those extensions, such as `/2024/05/01/launch.html`, is written as that file, so old `.html` URLs can redirect too.
