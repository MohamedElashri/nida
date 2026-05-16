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

Pages and sections are included. Draft pages are excluded unless the build uses
`drafts = true` or `--drafts`.

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
- use a custom search template if you want fewer fields, section-only search, or
  a hosted search service
