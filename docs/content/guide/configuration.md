+++
title = "Configuration"
description = "Configure the site title, base URL, directories, server, feeds, and output behavior."
weight = 20
template = "page"
+++

Every Nida site starts with `config.toml` in the site root.

```toml
config_version = "0.4"
base_url = "https://example.com/"
title = "My Site"
description = "Notes from a quiet corner of the web."
language = "en"
author = "Your Name"

content_dir = "content"
template_dir = "templates"
static_dir = "static"
output_dir = "public"
```

## Required values

`base_url` must be an absolute `http` or `https` URL. Nida uses it for canonical URLs, feeds, sitemaps, robots.txt, and the template `.BasePath` value.

`title` names the site.

`config_version` tells Nida which config shape the site expects.

## Directory values

`content_dir` points to Markdown content.

`template_dir` points to Go HTML templates.

`static_dir` points to static assets copied into the generated site.

`output_dir` points to the generated output directory. The default is `public`.

`themes_dir` points to available themes. The default is `themes`.

## Sections and pagination

```toml
paginate = 10

[sections]
default_sort_by = "date"
paginate_by = 10
paginate_path = "page"
```

Sections can override sorting and pagination in their `_index.md` front matter.

## Permalinks

Permalinks let you customize generated routes by section.

```toml
[permalinks]
posts = "/posts/{slug}/"
pages = "/{slug}/"
```

Supported placeholders include `{section}`, `{slug}`, `{year}`, `{month}`, `{day}`, and taxonomy placeholders such as `{tags}` when a matching taxonomy exists.

## Development server

```toml
[server]
host = "127.0.0.1"
port = 1307
livereload = true
```

Use `nida serve --site ./my-site` to build, watch, and serve the output directory.

## Generated files

```toml
[rss]
enabled = true
filename = "rss.xml"
limit = 20
sections = ["post"]

[atom]
enabled = false
filename = "atom.xml"
limit = 20
sections = ["post"]

[sitemap]
enabled = true
filename = "sitemap.xml"

[robots]
enabled = true
filename = "robots.txt"
```

Omit `rss.sections` or `atom.sections`, or set them to an empty array, to include pages from every section. When set, feed entries are limited to pages whose root section matches one of the configured names; for example, `post/notes` matches `post`.

You can also set custom robots content:

```toml
[robots]
enabled = true
content = "User-agent: *\nAllow: /"
```

## Markdown safety

Raw HTML in Markdown is disabled by default.

```toml
[markdown]
unsafe_html = false
external_links_target_blank = true
external_links_no_referrer = true
```

Only enable `unsafe_html = true` for trusted-author sites.

## Asset pipeline

```toml
[pipeline]
fingerprint = true
minify_css = true
minify_js = true

[pipeline.scss]
enabled = true
entry_dir = "css"

[pipeline.images]
enabled = true
widths = [480, 768, 1200]
quality = 85
```

The pipeline is optional. Keep it disabled when plain static asset copying is enough.

Next: see the broader feature tour in [Features](../features/).
