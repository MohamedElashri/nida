+++
title = "Generated Artifacts"
description = "Feeds, sitemap, robots, and search index generation."
weight = 80
template = "page"
+++

Generated artifacts are files that are not direct renders of content templates. They are still part of the write plan, and their paths can conflict with rendered pages.

## Artifact List

The build includes artifact paths for enabled features before writing:

- RSS
- Atom
- sitemap
- robots
- search index

That list is passed to `output.ValidateWritePlan` so conflicts fail before output is touched.

## Feeds

`internal/feeds.GenerateAll` can generate RSS and Atom.

Both feed types start from `state.Index.AllPages`, which is already sorted and draft-filtered by the site index. When `rss.sections` or `atom.sections` is set, feed generation narrows that list to pages whose root section matches one of the configured names before applying the feed limit. When the section list is omitted or empty, the feed keeps the all-page behavior.

RSS uses page title, canonical link, GUID, publish date, and description. Atom includes feed metadata, author metadata where available, page summary, and rendered HTML content.

When changing feed behavior, test date formatting, empty dates, author fallback, limit handling, and canonical URL construction.

## Sitemap

`internal/sitemap.Generate` receives config, site state, and rendered pages.

The sitemap uses rendered page canonical URLs, deduplicates them, sorts output by URL, and adds `lastmod` when a rendered route matches a content page with a date.

This means redirects and generated pages can appear if they have canonical URLs. If that behavior changes, update sitemap tests and think through alias pages and pagination redirects.

## Robots

`internal/robots.Generate` writes `robots.txt` when enabled.

If custom robots content is configured, that content is used. Otherwise Nida writes a default file. The output path is configurable and should remain part of write-plan validation.

## Search Index

`internal/searchindex.Generate` writes JavaScript that assigns JSON data to `window.searchIndex`.

The current shape is:

```json
{
  "documentStore": {
    "docs": {
      "/page/url/": {
        "title": "Title",
        "description": "Optional description",
        "body": "Plain text"
      }
    }
  }
}
```

Pages and sections are included. HTML is stripped with a simple tag scanner, entities are unescaped, and whitespace is collapsed.

## Adding Artifacts

To add a generated artifact:

1. add config for enabling and output filename
2. add defaults and validation
3. include the path in full build write-plan validation
4. write the artifact after rendered pages
5. include it in incremental rebuild artifact lists
6. remove stale output when config disables it
7. document the config key and output behavior

Generated artifacts often depend on the whole site, so avoid updating only the changed page in serve mode unless the artifact is explicitly local to that page.
