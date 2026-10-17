+++
title = "Rendering, Templates, And Themes"
description = "How Nida loads templates, chooses template names, and exposes theme data."
weight = 50
template = "page"
+++

Rendering is split between `internal/templates` and `internal/render`.

`internal/templates` loads Go HTML templates and helper functions. `internal/render` chooses what to render, builds template context, handles theme-facing data, and returns rendered pages.

## Template Loading

Templates are loaded from, in order:

1. templates of each parent theme the selected theme `extends`, root ancestor first
2. selected theme templates, when `theme` is set
3. site templates

`base.html` and templates in nested directories are treated as shared templates. Top-level `.html` files become renderable templates by filename stem.

For example, `post.html` registers the template name `post` and should contain:

```go-html-template
{{ define "post" }}...{{ end }}
```

At least one template root must contain `base.html`, so a site can rely entirely on its theme. Rendering also requires `index`, `post`, and `page`. Files under `templates/shortcodes/` are not page templates; `internal/shortcodes` loads them.

## Override Behavior

Template roots are walked in order, and later roots replace earlier entries with the same top-level name. This gives site templates precedence over theme templates, and a child theme precedence over its parent.

Shared templates are parsed into every renderable template, so common definitions can live in `base.html` or nested partial files. They are parsed root by root in the same order, so a site `templates/partials/footer.html` that defines `footer` replaces the theme's `footer` definition.

## Template Helpers

The template function map includes helpers for:

- date formatting
- trusted HTML and CSS output
- string joining and basic math
- prefix, suffix, contains, replace, lower, trim
- default strings
- slugification
- document direction
- grouping pages by year
- current time
- base-path-aware asset URLs
- image variant, srcset, and preset helpers
- reading files
- image resizing helper support
- sorting descending
- digging into nested values

`safeHTML` and `safeCSS` are compatibility aliases. New templates should prefer `unsafeHTML` and `unsafeCSS` because the trust boundary is explicit.

## Rendering Order

`render.RenderSite` loads templates, builds theme data, and renders:

1. section pages
2. content pages
3. taxonomy pages
4. 404 page

Aliases render as redirect pages during page rendering. Paginated page-one aliases also render redirects to canonical first pages.

If `minify_html` is enabled, all rendered pages are minified after rendering and before writing.

## Page Template Selection

Page template selection follows this order:

1. page front matter `template`, if it exists and is loaded
2. section `page_template`, if it exists and is loaded
3. `post`, if loaded
4. `page`

This means `post` is the normal fallback for content pages, while `page` is the final fallback.

## Section Template Selection

Section template selection follows this order:

1. section front matter `template`, if it exists and is loaded
2. `index` for the root section
3. `section`, if loaded
4. `list`

The root section can render without pagination. Non-root sections with pagination render page one at the section URL and page two onward under the configured pagination path.

## Template Context

Render context includes site config, theme data, current URL, canonical URL, page or section data, site index, taxonomy data, paginator data, and robots metadata.

Templates should read generated HTML through fields such as `.Page.BodyHTML` or `.Section.BodyHTML`, then mark it trusted with `unsafeHTML` when intentionally inserting rendered Markdown.

## Theme Chain

Theme metadata is loaded from `themes/<name>/config.toml`. A theme can extend another theme with:

```toml
extends = "base"
```

The resolver walks parent themes first and rejects circular inheritance.

Theme `[extra]` values are merged with site `[extra]` as a shallow map merge: parent theme values first, then child theme values, then site values, so the most specific value wins. If you need nested merge behavior, add tests because theme-facing fields depend on this map.

## Theme Data

`internal/render/theme.go` turns merged `extra` data into a typed `Theme` struct. It currently exposes:

- inline CSS
- main menu
- social links
- footer text
- date format
- author name
- favicon paths
- Umami analytics settings

Inline CSS is loaded from theme or site `style.css.html` when present, with a fallback to `static/site.css`.

## Adding Render Behavior

When adding a new template context field, update render types and tests. When adding a template helper, keep it deterministic unless it is intentionally dynamic like `now`.

Rendering changes are often visible across many outputs, so update golden tests in `internal/render/testdata` when the changed HTML is expected.
