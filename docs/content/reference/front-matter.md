+++
title = "Front Matter Reference"
description = "Metadata fields supported by Nida pages and sections."
weight = 20
template = "page"
+++

Nida supports TOML front matter with `+++` delimiters and simple YAML front matter with `---` delimiters.

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

`date` sets the publication date.

`updated` sets the update date.

`draft` excludes the page unless drafts are enabled.

`weight` provides a manual ordering value.

`slug` overrides the route slug derived from the file name.

`template` selects a page template.

`aliases` creates redirect pages for old URLs.

`extra` stores custom metadata, including taxonomy values.

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

Useful section fields include:

`title`, `description`, `template`, `page_template`, `sort_by`, `paginate_by`, `paginate_path`, `transparent`, `generate_feeds`, and `extra`.

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

Nida emits redirect pages for each alias.
