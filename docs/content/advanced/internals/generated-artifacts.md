+++
title = "Generated Artifacts"
description = "Feeds, sitemap, robots, and search index generation."
weight = 80
template = "page"
+++

Generated artifacts are files that are not direct renders of content templates. They are still part of the write plan, and their paths can conflict with rendered pages.

## Artifact List

The build includes artifact paths for enabled features before writing:

- the Atom feed (`[feed]`) and the optional RSS feed (`[rss]`)
- section feeds and taxonomy term feeds
- sitemap
- robots
- search index

That list is passed to `output.ValidateWritePlan` so conflicts fail before output is touched.

## Feeds

`internal/feeds.GenerateAll` writes the site Atom feed from `[feed]` and, when enabled, an RSS 2.0 feed from `[rss]`. Both are built from a `spec` (title, filename, alternate URL, and items) and rendered by `renderAtom` or `renderRSS`.

Site feeds start from `state.Index.AllPages`, which is already sorted and filtered for drafts and future dates. When `feed.sections` or `rss.sections` is set, the list is narrowed to pages whose root section matches before the limit is applied.

`scopedSpecs` adds more feeds:

- one for each section with `generate_feeds = true`, written at `<section>/feed.xml` (and `<section>/rss.xml`), including pages in nested sections
- one for each term of a taxonomy with `feed = true`, such as `tags/go/feed.xml`

Scoped feeds use the site feed's filename and limit. `feeds.Filenames` returns the same list of paths so the build can validate and clean them up.

Entries use the page `description` as the summary, falling back to `.Page.Summary`. Atom `updated` uses the page's `updated` date when it is newer than `date`. Links inside entry HTML are made absolute with `resolveURL`, which resolves them the way a browser would against the page URL.

When changing feed behavior, test date formatting, empty dates, author fallback, limit handling, scoped feed paths, and canonical URL construction.

## Sitemap

`internal/sitemap.Generate` receives config, site state, and rendered pages.

The sitemap uses rendered page canonical URLs, deduplicates them, sorts output by URL, and adds `lastmod` when a rendered route matches a content page with a date.

This means redirects and generated pages can appear if they have canonical URLs. If that behavior changes, update sitemap tests and think through alias pages and pagination redirects.

## Robots

`internal/robots.Generate` writes `robots.txt` when enabled.

If custom robots content is configured, that content is used. Otherwise Nida writes a default file. The output path is configurable and should remain part of write-plan validation.

## Search Index

`internal/searchindex.Generate` writes JavaScript that assigns JSON data to `window.searchIndex`.

The output format is described in the [Search reference](@/reference/search.md#index-format).

Pages and sections, including nested sections, are included. `search.sections` restricts the index to root sections and `search.body_limit` truncates bodies. HTML is stripped with a simple tag scanner, entities are unescaped, and whitespace is collapsed.

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
