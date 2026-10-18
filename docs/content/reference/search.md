+++
title = "Search"
description = "Search index format and a small client-side search pattern."
weight = 40
template = "page"
+++

Nida can generate a JavaScript search index for small client-side search UIs.

Enable it in `config.toml`:

```toml
[search]
enabled = true
filename = "search_index.en.js"
```

The file is written to the output directory next to generated pages. It assigns
the index data to `window.searchIndex`, so templates can load it with a regular
script tag.

## Index Format

The generated file has this shape:

```js
window.searchIndex = {
  "documentStore": {
    "docs": {
      "/page/url/": {
        "title": "Page title",
        "description": "Optional page description",
        "body": "Plain text body"
      }
    }
  }
};
```

The keys in `documentStore.docs` are generated site-relative URLs. Each document
contains:

| Field | Description |
| --- | --- |
| `title` | Page or section title. |
| `description` | Page or section description, omitted when empty. |
| `body` | Rendered HTML converted to plain text with whitespace collapsed. |

Pages and sections, including nested sections, are included. Draft pages are excluded unless the build uses
`drafts = true` or `--drafts`. Set `search.sections` to index only some root
sections and `search.body_limit` to cap the body text per document.

## Template Pattern

Add the form to a shared template, such as `base.html`, then load the generated
index before the script that reads it:

```html
<form role="search" data-search-form data-base-path="{{ .BasePath }}">
  <label for="search-query">Search</label>
  <input id="search-query" type="search" name="q" data-search-input>
</form>
<ol data-search-results></ol>

<script src="{{ .BasePath }}/{{ .Config.Search.Filename }}"></script>
<script>
  const basePath = document
    .querySelector("[data-search-form]")
    .getAttribute("data-base-path") || "";
  const docs = window.searchIndex.documentStore.docs;
  const entries = Object.keys(docs).map((url) => ({ url, ...docs[url] }));

  function runSearch(query) {
    const terms = query.toLowerCase().trim().split(/\s+/).filter(Boolean);
    return entries.filter((entry) => {
      const text = `${entry.title} ${entry.description || ""} ${entry.body}`.toLowerCase();
      return terms.every((term) => text.includes(term));
    }).map((entry) => ({
      title: entry.title,
      url: basePath + entry.url,
      summary: entry.description || entry.body.slice(0, 160),
    }));
  }
</script>
```

The important detail is URL handling: index keys are site-relative, so templates
should prefix result links with `.BasePath` when the site is published under a
subpath. A header search can render the returned entries into a positioned
dropdown, while a larger site can render the same data into a dedicated results
view.

## Keeping Search Small

The default index is deliberately plain JSON inside JavaScript. That keeps the
feature dependency-free, but it also means the whole index is downloaded by the
browser.

Useful habits:

- write concise descriptions so results can show summaries without relying on
  long body snippets
- keep generated search enabled for small and medium sites
- disable search for sites where a full-text body index would be too large
- avoid putting large generated text dumps into Markdown pages
- set `search.sections` and `search.body_limit` to index less
- use a hosted search service or Pagefind for large sites

## Larger sites: Pagefind

The built-in index downloads in full on first search. For a large blog or documentation set, [Pagefind](https://pagefind.app) is a better fit: it indexes the generated HTML after the build and loads only the index fragments a query needs. It works with any Nida site without configuration:

```bash
nida build --site ./my-site
npx pagefind --site ./my-site/public
```

Then add its UI to a template:

```html
<link href="{{ .BasePath }}/pagefind/pagefind-ui.css" rel="stylesheet">
<script src="{{ .BasePath }}/pagefind/pagefind-ui.js"></script>
<div id="search"></div>
<script>new PagefindUI({ element: "#search", baseUrl: "{{ .BasePath }}/" });</script>
```

Add `data-pagefind-body` to the element that wraps page content so navigation and footers stay out of the index. Disable Nida's `[search]` when you switch.

