+++
title = "Templates"
description = "Understand the template files Nida expects and the context they receive."
weight = 30
template = "page"
+++

Nida templates are Go HTML templates. A site needs a shared `base.html` template and page templates for the content types it renders.

A small template directory can look like this:

```text
templates/
  base.html
  index.html
  page.html
  post.html
  section.html
  404.html
```

## Required templates

Nida requires these template names somewhere in the site or its theme:

- `base.html`, shared by every page (it usually defines `base`)
- `index`, for the home page
- `post` and `page`, for content pages
- `section` (or `list`), once the site has sections such as `content/posts/`

Taxonomy pages use `taxonomy_list` and `taxonomy_single`, or a single `taxonomy` template, and `404.html` renders the not-found page. A docs site can make `post.html` reuse the same layout as `page.html`. Templates can come from a theme instead of `templates/`; see [Themes](@/guide/themes.md#override-order) for how the two combine.

## Template context

Every template receives:

- `.Title` and `.Description` for the current page, section, or term
- `.Config` for site configuration
- `.Page` for the current page
- `.Section` for the current section
- `.Pages` for page lists
- `.Index` for the site index
- `.CurrentURL` for the current route, and `.HomeURL` (`/`)
- `.CanonicalURL` for the absolute canonical URL
- `.BasePath` for the path prefix derived from `base_url`
- `.Paginator` on paginated section and taxonomy pages, with `.CurrentIndex` and `.NumberPagers` (1-based page and page count), `.Previous` and `.Next` URLs, `.PageLinks` (each with `.Number`, `.URL`, `.Current`), and `.Pages`
- `.Taxonomy`, `.Term`, and `.Terms` on taxonomy pages; see [Taxonomy pages](#taxonomy-pages)
- `.Robots` for the configured `robots_meta` value
- `.Theme.Extra` for the theme `[extra]` table merged with the site's `[extra]` (site values win)
- `.Data` for values from the data directory
- `.EditURL` when `edit_url` is configured

```html
{{ with .Robots }}<meta name="robots" content="{{ . }}">{{ end }}
{{ with dig .Theme.Extra "analytics.src" }}<script defer src="{{ . }}"></script>{{ end }}
```

`.Theme` also has older shortcut fields filled from `[extra]`: `MainMenu` (`main_menu`), `Social` (`social_icons`), `FooterText` (`footer.text`), `DateFormat` (`date_format`), `AuthorName` (`author.name`), `Favicon` (`favicon.webmanifest`, `favicon_16x16`, `favicon_32x32`, `apple_touch_icon`), `Umami` (`umami.enabled`, `src`, `website_id`), and `InlineCSS`, the contents of a `style.css.html` template. They still work, but new themes should read `.Theme.Extra` so the engine doesn't need to know about each theme setting.

Rendered Markdown is available as `.Page.BodyHTML` or `.Section.BodyHTML`.

```html
<article>
  <h1>{{ .Page.Title }}</h1>
  <div>{{ unsafeHTML .Page.BodyHTML }}</div>
</article>
```

Use `asset` when linking static files. It keeps asset URLs under the path prefix from `base_url`, and fingerprinting can rewrite the rendered URL during the asset pipeline step:

```html
<link rel="stylesheet" href="{{ asset "style.css" }}">
<link rel="preload" href="{{ asset "fonts/site.woff2" }}" as="font" type="font/woff2" crossorigin>
<script src="{{ asset "js/site.js" }}" defer></script>
```

Use `.BasePath` for route links that need to work under a project path such as GitHub Pages:

```html
<a href="{{ .BasePath }}/guide/">Guide</a>
```

For processed responsive images, `imageSrcset` uses `pipeline.images.widths`, while `imagePresetSrcset` and `imagePresetSizes` use a named image preset:

```html
<img
  src="{{ asset "images/hero.jpg" }}"
  srcset="{{ imagePresetSrcset "hero" "images/hero.jpg" }}"
  sizes="{{ imagePresetSizes "hero" }}"
  alt="A writing desk">
```

## Page fields

Besides front matter values, `.Page` has fields Nida computes:

| Field | Description |
| --- | --- |
| `.BodyHTML` | Rendered Markdown. |
| `.Summary` | HTML before `<!-- more -->`, or the first paragraph. See [Writing Content](@/guide/writing.md#summaries). |
| `.TOC` | Nested headings. See [Writing Content](@/guide/writing.md#table-of-contents). |
| `.WordCount`, `.ReadingTime` | Word count and estimated minutes to read. |
| `.URL` | Site-relative URL. Prefix with `.BasePath` in links. |
| `.Slug`, `.SectionPath`, `.RelativePath` | The URL slug, the section directory (`posts/2026`), and the content path (`posts/2026/hello.md`). |
| `.Date`, `.Updated` | Publication and update times. `.Updated` can come from git with `updated_from_git`. |
| `.IsBundle`, `.BundleDir`, `.Resources` | For page bundles: the bundle directory and the file names copied next to the page. |
| `.PrevURL`, `.PrevTitle`, `.NextURL`, `.NextTitle` | Neighbors within the section. |
| `.Extra` | Custom front matter values. |

## Taxonomy pages

With `render = true`, each taxonomy gets a list page (`/tags/`) and a page per term (`/tags/go/`). Both receive `.Taxonomy`, the collection with `.Name`, `.URL`, and `.Terms`. The list page also gets `.Terms`; a term page gets `.Term` and its pages in `.Pages`. Each term has `.Name`, `.Slug`, `.URL`, `.CanonicalURL`, and `.Items` (all of its pages).

```html
{{ define "taxonomy_list" }}{{ template "base" . }}{{ end }}
{{ define "content" }}
<h1>{{ title .Taxonomy.Name }}</h1>
<ul>
  {{ range .Terms }}<li><a href="{{ $.BasePath }}{{ .URL }}">{{ .Name }}</a> ({{ len .Items }})</li>{{ end }}
</ul>
{{ end }}
```

## Navigation for docs sites

Every section has `.Sections` (child sections, ordered by `weight` and then title) and `.Pages` (ordered by the section's `sort_by`), along with `.URL` and `.Title`. Together they give the whole content tree, so a sidebar can be generated instead of written by hand:

```html
{{ define "nav-items" }}
  {{ $ctx := .Ctx }}
  {{ range .Section.Sections }}
    <details{{ if hasPrefix $ctx.CurrentURL .URL }} open{{ end }}>
      <summary>{{ .Title }}</summary>
      {{ template "nav-items" (dict "Section" . "Ctx" $ctx) }}
    </details>
  {{ end }}
  {{ range .Section.Pages }}
    <a{{ if eq .URL $ctx.CurrentURL }} aria-current="page"{{ end }} href="{{ $ctx.BasePath }}{{ .URL }}">{{ .Title }}</a>
  {{ end }}
{{ end }}

{{ with .Index.GetSection "" }}{{ template "nav-items" (dict "Section" . "Ctx" $) }}{{ end }}
```

For documentation, where pages and subsections are interleaved, `.Index.Nav` returns one list per section ordered by `weight` and then title. Each entry has `.Title`, `.URL`, `.Weight`, `.IsSection`, `.Page` or `.Section`, and `.Children` for sections:

```html
{{ define "nav" }}
<ul>
  {{ range .Entries }}
  <li>
    <a{{ if eq .URL $.Ctx.CurrentURL }} aria-current="page"{{ end }} href="{{ $.Ctx.BasePath }}{{ .URL }}">{{ .Title }}</a>
    {{ with .Children }}{{ template "nav" (dict "Entries" . "Ctx" $.Ctx) }}{{ end }}
  </li>
  {{ end }}
</ul>
{{ end }}

{{ template "nav" (dict "Entries" (.Index.Nav "") "Ctx" .) }}
```

`.Index.Neighbors` reads the same tree top to bottom, with each section before its children, and returns the previous and next entries for a page:

```html
{{ with .Index.Neighbors "" .CurrentURL }}
  {{ with .Prev }}<a rel="prev" href="{{ $.BasePath }}{{ .URL }}">← {{ .Title }}</a>{{ end }}
  {{ with .Next }}<a rel="next" href="{{ $.BasePath }}{{ .URL }}">{{ .Title }} →</a>{{ end }}
{{ end }}
```

Pass a section path instead of `""` to limit either one to part of the site.

`.Index.Ancestors` returns the sections from the root down to a section path, for breadcrumbs:

```html
<nav aria-label="Breadcrumb">
  {{ range .Index.Ancestors .Page.SectionPath }}<a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a> / {{ end }}
  {{ .Page.Title }}
</nav>
```

With `edit_url` set, `.EditURL` links to the page's source file:

```html
{{ with .EditURL }}<a href="{{ . }}">Edit this page</a>{{ end }}
```

## Data files

Files in `data/` are available as `.Data`, keyed by file name and nested by directory. `data/authors.toml` is `.Data.authors` and `data/team/core.json` is `.Data.team.core`. TOML, JSON, and YAML are supported. Themes can ship a `data/` directory too; site files replace theme files with the same name.

```html
{{ range .Data.links }}<a href="{{ .url }}">{{ .name }}</a>{{ end }}
```

## Translations

`t` looks up a UI string from `i18n/<language>.toml` and returns the key when there is no translation. Placeholders in braces are filled from an optional `dict`:

```html
<a href="{{ $.BasePath }}{{ .URL }}">{{ t "read_more" }}</a>
<p>{{ t "posts_count" (dict "count" (len .Pages)) }}</p>
```

See [Translations](@/reference/config.md#translations) for file layout and fallback rules.

## Looking up content

`.Index` holds the whole site: `.AllPages` (published pages, newest first), `.SectionLookup` (sections by path), `.Taxonomies`, and the helper methods below and in the next sections.

`.Index` has two lookup methods. Both return nothing when the path doesn't exist, so wrap them in `with`:

```html
{{ with .Index.GetPage "pages/about.md" }}<a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a>{{ end }}
{{ with .Index.GetSection "posts" }}{{ len .Pages }} posts{{ end }}
```

`GetPage` takes a content path, with or without `.md` or a leading `@/`. `GetSection` takes a section path such as `posts`, or an empty string for the root.

## Related posts and series

`.Index.Related` returns pages that share taxonomy terms with a page, ranked by how many terms they share, then newest first:

```html
{{ with .Index.Related .Page 3 }}
<aside>
  <h2>Related</h2>
  <ul>{{ range . }}<li><a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a></li>{{ end }}</ul>
</aside>
{{ end }}
```

Give posts the same `series = "Building Nida"` value to group them. `.Index.Series` returns the series with `.Name`, `.Pages` (ordered by `weight`, then oldest first), the 1-based `.Position` of the current page, and `.Prev` and `.Next`:

```html
{{ with .Index.Series .Page }}
<nav class="series">
  Part {{ .Position }} of {{ len .Pages }} in <em>{{ .Name }}</em>.
  {{ with .Next }}Next: <a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a>{{ end }}
</nav>
{{ end }}
```

## Archives

`groupByYear` and `groupByMonth` group dated pages for archive pages. Month groups have `.Month` (the first day of the month) and `.Pages`:

```html
{{ range groupByMonth .Pages }}
  <h2>{{ formatDateWith .Month "%B %Y" }}</h2>
  <ul>{{ range .Pages }}<li><a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a></li>{{ end }}</ul>
{{ end }}
```

## SEO metadata

The built-in `nida-seo` partial writes the `<title>`, description, canonical link, robots meta, Open Graph and Twitter card tags, and a JSON-LD `BlogPosting` for pages. Include it in `<head>`:

```html
<head>
  <meta charset="utf-8">
  {{ template "nida-seo" . }}
</head>
```

It reads the page title and description, falling back to the site's; `date` and `updated`; `image` from the page's `extra`, falling back to `[extra] image` in the site config; and `[extra] twitter` for `twitter:site`. To change the output, define your own `{{ define "nida-seo" }}` in a template file; site and theme definitions replace the built-in one.

## Template functions

On top of Go's built-in template functions, Nida provides:

| Function | Example | Description |
| --- | --- | --- |
| `dict` | `dict "Page" . "Wide" true` | Build a map, usually to pass several values to a partial. |
| `merge` | `merge $a (dict "k" "v")` | A new map with the second map's keys laid over the first. |
| `slice` | `slice "a" "b"` | Build a list. |
| `seq` | `seq 3` | `1 2 3`. |
| `first`, `last` | `first 5 .Pages` | The first or last N items of a list. |
| `where` | `where .Pages "SectionPath" "posts"` | Keep items whose field equals a value. Paths can reach into maps (`"Extra.tags"`). A list field matches if it contains the value. Use `where .Pages "Extra.series" "!=" "x"` to negate. |
| `truncate` | `truncate 140 .Description` | Shorten text at a word boundary and add `…`. |
| `plainify` | `plainify .Summary` | Strip HTML tags. |
| `markdownify` | `markdownify .Description` | Render inline Markdown. A single paragraph is unwrapped. |
| `absURL`, `relURL` | `absURL .Page.URL` | Join onto `base_url`, or onto just its path. |
| `jsonify` | `jsonify (dict "@type" "BlogPosting")` | Encode as JSON, safe inside `<script type="application/ld+json">`. |
| `formatDate`, `formatDateWith` | `formatDateWith .Page.Date "%B %d, %Y"` | Format dates. `formatDateWith` accepts `strftime`-style codes. |
| `groupByYear`, `groupByMonth` | `range groupByYear .Pages` | Group dated pages into `{Year, Pages}` or `{Month, Pages}`. |
| `dig` | `dig .Config.Extra "author.name"` | Read a nested map value by dotted path. |
| `default` | `default .Title "Untitled"` | The first argument, or the second when the first is empty. Note the order: value first, fallback second. |
| `lower`, `upper`, `title`, `trimSpace`, `replace`, `split`, `contains`, `hasPrefix`, `hasSuffix`, `join` | | String helpers. |
| `slugify`, `urlize` | `slugify "Hello World"` | `hello-world`. |
| `t` | `t "read_more"` | Translate a UI string. See [Translations](#translations). |
| `documentDirection` | `documentDirection .Config.Language` | `rtl` or `ltr` for a language code. |
| `now` | `(now).Year` | The build time. |
| `sortDesc` | `range sortDesc $names` | Sort a list of strings in reverse. |
| `bundledScripts` | `bundledScripts .Page` | The bundled math stylesheet when `markdown.math` is on, on every page. Math and diagrams are rendered at build time, so it returns no scripts. See [Bundled files](@/guide/writing.md#bundled-files). |
| `syntaxCSS` | `<style>{{ syntaxCSS }}</style>` | Highlighting CSS for `syntax_theme`, or for a named [Chroma style](https://xyproto.github.io/splash/docs/) such as `syntaxCSS "github-dark"`. |
| `add`, `sub` | `add .Paginator.CurrentIndex 1` | Integer math. |
| `asset`, `assetURL` | `asset "style.css"` | URL for a static file, with the base path and fingerprint applied. |
| `imageSrcset`, `imagePresetSrcset`, `imagePresetSizes` | | Responsive image attributes, described above. |
| `imageVariant` | `imageVariant "images/hero.jpg" 768` | URL of one pipeline width variant, `images/hero.768w.jpg`. |
| `resizeImage` | `resizeImage "images/hero.jpg" 400 300 "png"` | Resize to exact dimensions and return the URL. The file is cached in `static/processed_images/`; the format is `jpg`, `png`, or `gif` (anything else becomes `jpg`), or empty to keep the source format. |
| `readFile` | `readFile "static/logo.svg"` | Read a file from the content, template, or static directories, or from a theme's `templates/` or `static/`. |
| `unsafeHTML`, `unsafeCSS` | `unsafeHTML .Page.BodyHTML` | Mark trusted strings as HTML or CSS. `safeHTML` and `safeCSS` are older names. |

Next: publish a site in [Deployment](@/guide/deployment.md).
