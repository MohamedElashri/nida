+++
title = "Rendering, Templates, And Themes"
description = "How Nida loads templates, chooses template names, and exposes theme data."
weight = 50
template = "page"
+++

Rendering is split between `internal/templates` and `internal/render`. Theme resolution lives in `internal/themes`, which every package that reads theme files uses: templates, shortcodes, static assets, SCSS, data files, translations, and archetypes.

`internal/templates` loads Go HTML templates and helper functions. `internal/render` chooses what to render, builds template context, handles theme-facing data, and returns rendered pages.

## Template Loading

Templates are loaded from, in order:

1. built-in partials embedded in `internal/templates/builtin`: `nida-seo` and `nida-scripts`
2. templates of each parent theme the selected theme `extends`, root ancestor first
3. selected theme templates, when `theme` is set
4. site templates

`base.html` and templates in nested directories are treated as shared templates. Top-level `.html` files become renderable templates by filename stem.

For example, `post.html` registers the template name `post` and should contain:

```go-html-template
{{ define "post" }}...{{ end }}
```

At least one template root must contain `base.html`, so a site can rely entirely on its theme; see [required templates](@/guide/templates.md#required-templates). Files under `templates/shortcodes/` are not page templates; `internal/shortcodes` loads them.

## Override Behavior

Template roots are walked in order, and later roots replace earlier entries with the same top-level name. This gives site templates precedence over theme templates, and a child theme precedence over its parent.

Shared templates are parsed into every renderable template, so common definitions can live in `base.html` or nested partial files. They are parsed root by root in the same order, so a site `templates/partials/footer.html` that defines `footer` replaces the theme's `footer` definition.

## Template Helpers

`funcMapWith` in `internal/templates` builds the function map once per template load, and `FuncMap` exposes the same map, including translations, to `internal/shortcodes`. The [template function reference](@/guide/templates.md#template-functions) lists every function.

`resizeImage` holds a package lock while it writes, because pages render concurrently and may resize the same image.

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
3. `sections.default_page_template` from config, if it exists and is loaded
4. `post`, if loaded
5. `page`

This means `post` is the normal fallback for content pages, while `page` is the final fallback.

## Section Template Selection

Section template selection follows this order:

1. section front matter `template`, if it exists and is loaded
2. `index` for the root section
3. `section`, if loaded
4. `list`

If none of these exist, rendering fails with a message asking for `section.html`.

The root section can render without pagination. Non-root sections with pagination render page one at the section URL and page two onward under the configured pagination path.

## Theme Chain

Theme metadata is loaded from `themes/<name>/config.toml`. A theme can extend another theme with:

```toml
extends = "base"
```

`themes.Chain` returns the chain parent first and rejects circular inheritance. `config.toml` is optional; a theme without one extends nothing. `themes.Dirs` returns one subdirectory, such as `templates` or `static`, for every theme in the chain that has it.

Theme `[extra]` values are merged with site `[extra]` as a shallow map merge: parent theme values first, then child theme values, then site values, so the most specific value wins. If you need nested merge behavior, add tests because theme-facing fields depend on this map.

## Theme Data

`internal/render/theme.go` merges theme and site `[extra]` with `themes.MergeExtra` and exposes it as `.Theme.Extra`. For older themes it also fills named fields: main menu, social links, footer text, date format, author name, favicon paths, and Umami settings. New themes should read `.Theme.Extra` instead of adding more named fields.

`.Theme.InlineCSS` is the first file found among `templates/style.css.html` in the site, `style.css.html` in each theme root (child first), and `static/site.css`. `{% include "file" %}` lines in it are expanded relative to the file.

## Template Context

Render context includes site config, theme data, current URL, canonical URL, page or section data, site index, taxonomy data, paginator data, and robots metadata. Templates read generated HTML through `.Page.BodyHTML` or `.Section.BodyHTML` and mark it trusted with `unsafeHTML`.

Besides `.Page`, `.Section`, `.Pages`, `.Paginator`, and the taxonomy fields, every render receives:

- `.Index`, with the helper methods described in [Site Index And Routing](@/advanced/internals/site-index-routing.md#index-helpers)
- `.Data`, loaded once per build by `internal/datafiles` and carried on `templates.Set`
- `.EditURL`, built in `renderTemplate` from `edit_url` and the source path
- `.Robots`, from `robots_meta`, with `noindex` added on the 404 page

Pages are rendered concurrently with `internal/parallel.Map`, which keeps output order and reports the first error by index. Templates must therefore not rely on side effects between pages.

## Adding Render Behavior

When adding a new template context field, update render types and tests. When adding a template helper, keep it deterministic unless it is intentionally dynamic like `now`.

Rendering changes are often visible across many outputs, so update golden tests in `internal/render/testdata` when the changed HTML is expected.
