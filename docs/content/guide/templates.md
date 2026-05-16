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

Next: publish a site in [Deployment](../deployment/).
