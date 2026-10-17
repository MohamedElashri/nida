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

Nida currently requires these template names:

- `index`
- `post`
- `page`

A docs site can make `post.html` reuse the same layout as `page.html` if it does not distinguish posts from pages yet.

## Template context

Templates receive useful values such as:

- `.Config` for site configuration
- `.Page` for the current page
- `.Section` for the current section
- `.Pages` for page lists
- `.Index` for the site index
- `.CurrentURL` for the current route
- `.CanonicalURL` for the absolute canonical URL
- `.BasePath` for the path prefix derived from `base_url`
- `.Paginator` on paginated section and taxonomy pages, with `.Previous`, `.Next`, and `.PageLinks`
- `.Robots` for the configured `robots_meta` value
- `.Theme.Extra` for the theme `[extra]` table merged with the site's `[extra]` (site values win)

```html
{{ with .Robots }}<meta name="robots" content="{{ . }}">{{ end }}
{{ with dig .Theme.Extra "analytics.src" }}<script defer src="{{ . }}"></script>{{ end }}
```

`.Theme` also has older shortcut fields: `MainMenu`, `Social`, `FooterText`, `DateFormat`, `AuthorName`, `Favicon`, and `Umami`. They still work, but new themes should read from `.Theme.Extra` so the engine doesn't need to know about each theme setting.

Rendered Markdown is available as `.Page.BodyHTML` or `.Section.BodyHTML`.

```html
<article>
  <h1>{{ .Page.Title }}</h1>
  <div>{{ safeHTML .Page.BodyHTML }}</div>
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
| `.PrevURL`, `.PrevTitle`, `.NextURL`, `.NextTitle` | Neighbors within the section. |
| `.Extra` | Custom front matter values. |

## Looking up content

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
| `add`, `sub` | `add .Paginator.CurrentIndex 1` | Integer math. |
| `asset`, `imageSrcset`, `imagePresetSrcset`, `imagePresetSizes`, `imageVariant`, `resizeImage` | | Asset and image helpers, described above. |
| `readFile` | `readFile "static/logo.svg"` | Read a file from the content, template, or static directories. |
| `unsafeHTML`, `unsafeCSS` | `unsafeHTML .Page.BodyHTML` | Mark trusted strings as HTML or CSS. `safeHTML` and `safeCSS` are older names. |

Next: publish a site in [Deployment](../deployment/).
