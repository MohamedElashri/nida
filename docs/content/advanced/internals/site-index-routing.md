+++
title = "Site Index And Routing"
description = "How pages, sections, routes, taxonomies, and navigation data are assembled."
weight = 40
template = "page"
+++

`internal/site` builds the canonical in-memory model used by rendering, feeds, sitemap generation, search, and serve-mode diffs.

## State

`site.State` contains:

- `Pages`: sorted rendered content pages
- `Sections`: rendered content sections
- `Index`: lookup and navigation data

`site.Load` creates this state by discovering content, rendering Markdown, then calling `BuildIndex`.

## SiteIndex

`SiteIndex` stores:

- ordered sections
- section lookup by section path
- root section pointer
- all routable pages
- taxonomy collections
- taxonomy map
- route registry
- canonical lookup

The route registry is used to detect duplicate page routes. It maps generated route to source relative path.

## Page Sorting, Drafts, And Future Pages

Pages are sorted by date descending and then slug. Draft pages are removed unless `cfg.Drafts` is true, and pages dated after the build time are removed unless `cfg.Future` is true.

This happens before sections and taxonomies are built, so downstream packages normally see only pages that should be published.

## Page Routes

`routePage` chooses a permalink pattern by section path:

- matching entry in `[permalinks]`
- otherwise `/{section}/{slug}/`

Supported placeholders include:

- `{section}`
- `{slug}`
- `{year}`
- `{month}`
- `{day}`
- configured taxonomy names, such as `{tags}`

After placeholder expansion, routes must start and end with `/`. Unsupported placeholders fail the build.

## Section Routes

Sections always render at `/{section}/`, and the root section at `/`. `[permalinks]` patterns apply to the pages in a section, not the section itself; `ResolveSectionURL` returns the same route the renderer writes, which `@/section/_index.md` links and `.Section.URL` both use.

## Section Tree

`BuildIndex` assigns section URLs, builds section relationships, and attaches pages to their sections. Child sections are ordered by `weight`, then title. It also computes previous and next page links within each section after section pages are ordered.

The root section is the section with an empty section path. If the root has no `_index.md`, behavior depends on what content discovery synthesized or found.

## Taxonomies

`internal/taxonomies` builds collections from configured `[[taxonomies]]` entries.

For each page, taxonomy values are read from `page.Extra`. Values must be string lists. Each configured taxonomy receives a collection with terms, term URLs, canonical URLs, pagination settings, feed flag, render flag, and sorted items.

Taxonomy term slugs use `content.DeriveSlug`. Empty term slugs fail the build.

## Index Helpers

`SiteIndex` has methods that templates call directly:

| Method | Returns |
| --- | --- |
| `GetPage(path)`, `GetSection(path)` | A page or section by content path, or nil. |
| `Ancestors(sectionPath)` | Sections from the root down to a section, for breadcrumbs. |
| `Nav(sectionPath)` | Pages and child sections interleaved by weight, recursively (`nav.go`). |
| `Neighbors(sectionPath, url)` | Previous and next entries when reading `Nav` top to bottom. |
| `Related(page, limit)` | Pages ranked by shared taxonomy terms (`related.go`). |
| `Series(page)` | The page's series, ordered by weight then date, with position, previous, and next. |

They work on value receivers and copy nothing large, so they are cheap to call from every page.

## Images

`site.NewImages` returns the `markdown.Images` implementation (`images.go`). It finds the file behind a Markdown image (static roots in override order, or the page bundle), reads its dimensions once, and, when the image pipeline is enabled, builds a `srcset` from the variant widths that the pipeline actually writes.

## Route Conflicts

Page route conflicts fail during index building. Page and artifact output conflicts fail later in `output.ValidateWritePlan`.

Keep both checks. Route conflicts explain source content collisions. Output conflicts catch cases where different URLs still map to the same output file.

## Adding Routing Features

When adding a route placeholder or route rule:

1. update route expansion in `internal/site`
2. normalize and validate the resulting route
3. add conflict tests
4. update permalink docs
5. check feeds, sitemap, aliases, and search output for assumptions about canonical URLs
