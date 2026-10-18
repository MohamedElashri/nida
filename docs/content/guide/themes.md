+++
title = "Themes"
description = "Build, extend, and share themes: what a theme can contain, how overrides work, and a minimal starter."
weight = 35
template = "page"
+++

A theme is a directory of templates and assets that a site can use instead of writing its own. Themes can extend other themes, and a site can override any part of a theme without copying the rest. This documentation site uses a theme, [`nida-docs`](https://github.com/MohamedElashri/nida/tree/main/docs/themes/nida-docs), which makes a good reference.

## Using a theme

Put the theme under `themes/` and select it:

```toml {title="config.toml"}
theme = "ink"
```

```text
my-site/
├── config.toml
├── content/
└── themes/
    └── ink/
```

Themes are usually added as a copy or as a git submodule:

```bash
git submodule add https://github.com/someone/ink.git themes/ink
```

With a theme, the site does not need its own `templates/` directory.

## What a theme can contain

```text
themes/ink/
├── config.toml          name, parent theme, default [extra] values
├── templates/           page templates, base.html, partials
│   ├── partials/
│   └── shortcodes/      shortcode templates
├── static/              CSS, JS, fonts, images
├── scss/                SCSS compiled when pipeline.scss is enabled
├── data/                data files available as .Data
├── i18n/                translated UI strings for the t function
└── archetypes/          templates for nida new
```

Every directory is optional. `config.toml` is optional too:

```toml {title="themes/ink/config.toml"}
name = "Ink"
description = "A quiet theme for long-form writing."
extends = "base"

[extra]
date_format = "%B %d, %Y"
main_menu = [{ name = "Posts", url = "/posts/" }]
```

## Override order

Nida layers files from least to most specific:

1. the root ancestor theme, through any themes in between
2. the selected theme
3. the site itself

A later layer wins wherever names overlap:

| Kind | Overridden by |
| --- | --- |
| Page templates (`post.html`, `page.html`, ...) | A file with the same name. |
| Partials and `base.html` | A later `{{ define "name" }}` with the same name, in any file. |
| Shortcodes | A file with the same name in `templates/shortcodes/`. Built-in shortcodes are the lowest layer. |
| Static files and SCSS | A file at the same relative path. |
| Data files | A file with the same name; the whole file is replaced. |
| Translations | Individual keys; missing keys fall back to earlier layers. |
| `[extra]` | Individual top-level keys; the site's `[extra]` wins. |

So to change one part of a theme, add only that part to your site. For example, to change a theme's footer, find which template defines it and add your own definition under the same name:

```html {title="templates/partials/footer.html"}
{{ define "site-footer" }}
<footer>Published by {{ .Config.Author }}</footer>
{{ end }}
```

## Extending a theme

A child theme names its parent with `extends`. Chains can be several themes deep; Nida rejects cycles.

```toml {title="themes/ink-dark/config.toml"}
name = "Ink Dark"
extends = "ink"
```

The child can then ship just a stylesheet with the same path as the parent's, or a few templates, and inherit everything else.

## A minimal theme

These five files make a complete theme for a small blog.

```html {title="themes/minimal/templates/base.html"}
{{ define "base" }}
<!doctype html>
<html lang="{{ default .Config.Language "en" }}" dir="{{ documentDirection .Config.Language }}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  {{ template "nida-seo" . }}
  <link rel="stylesheet" href="{{ asset "style.css" }}">
  {{ if .Config.Feed.Enabled }}<link rel="alternate" type="application/atom+xml" href="{{ .BasePath }}/{{ .Config.Feed.Filename }}">{{ end }}
</head>
<body>
  <header><a href="{{ .BasePath }}/">{{ .Config.Title }}</a></header>
  <main>{{ template "content" . }}</main>
</body>
</html>
{{ end }}
```

```html {title="themes/minimal/templates/index.html"}
{{ define "index" }}{{ template "base" . }}{{ end }}
{{ define "content" }}
  {{ range .Pages }}
  <article>
    <h2><a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a></h2>
    {{ unsafeHTML .Summary }}
  </article>
  {{ end }}
{{ end }}
```

```html {title="themes/minimal/templates/post.html"}
{{ define "post" }}{{ template "base" . }}{{ end }}
{{ define "content" }}
  <article>
    <h1>{{ .Page.Title }}</h1>
    <p>{{ formatDateWith .Page.Date "%B %d, %Y" }} · {{ t "reading_time" (dict "minutes" .Page.ReadingTime) }}</p>
    {{ unsafeHTML .Page.BodyHTML }}
  </article>
  {{ with .Index.Related .Page 3 }}
  <aside><h2>{{ t "related" }}</h2>
    {{ range . }}<a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a>{{ end }}
  </aside>
  {{ end }}
{{ end }}
```

```html {title="themes/minimal/templates/page.html"}
{{ define "page" }}{{ template "base" . }}{{ end }}
{{ define "content" }}<article><h1>{{ .Page.Title }}</h1>{{ unsafeHTML .Page.BodyHTML }}</article>{{ end }}
```

```html {title="themes/minimal/templates/section.html"}
{{ define "section" }}{{ template "base" . }}{{ end }}
{{ define "content" }}
  <h1>{{ .Section.Title }}</h1>
  {{ range groupByYear .Pages }}
  <h2>{{ .Year }}</h2>
  <ul>{{ range .Pages }}<li><a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a></li>{{ end }}</ul>
  {{ end }}
{{ end }}
```

Add `static/style.css` and an `i18n/en.toml` with the strings the templates use:

```toml {title="themes/minimal/i18n/en.toml"}
reading_time = "{minutes} min read"
related = "Related posts"
```

A site using this theme can translate it by adding `i18n/ar.toml` with the same keys.

## Required templates

A theme must provide `base.html` and the `index`, `post`, `page`, and `section` templates, unless it expects sites to supply some of them. See [required templates](@/guide/templates.md#required-templates) for the full list. Pages can choose another template with `template` front matter, and sections can set `page_template` for their pages.

## Theme settings

Read theme options from `.Theme.Extra`, which merges the theme's `[extra]` with the site's. That keeps defaults in the theme and lets sites change them:

```html
{{ with dig .Theme.Extra "analytics.src" }}<script defer src="{{ . }}"></script>{{ end }}
```

Document the keys your theme reads in its README. For anything larger than a few settings, such as navigation menus or author profiles, ship a sample data file in the theme's `data/` directory and let sites replace it.

## Building docs themes

For documentation sites, these helpers do most of the work:

- `.Index.Nav ""` returns the whole content tree in weight order for a sidebar.
- `.Index.Neighbors "" .CurrentURL` returns the previous and next pages.
- `.Index.Ancestors .Page.SectionPath` gives breadcrumbs.
- `.Page.TOC` lists the page's headings.
- `.EditURL` links to the source file when `edit_url` is set.

See [Templates](@/guide/templates.md#navigation-for-docs-sites) for examples. The `nida-docs` theme uses all of them.

## Sharing a theme

- Keep the theme self-contained: everything it needs lives under its directory.
- Use `asset` and `.BasePath` for every link so the theme works under a subpath.
- Put UI text in `i18n/en.toml` so sites can translate it.
- Use logical CSS properties (`margin-inline-start`, `border-inline-end`) and `dir` so right-to-left languages work.
- Include an example `config.toml` showing the `[extra]` keys the theme reads.
- Run `nida check` against an example site before each release.
