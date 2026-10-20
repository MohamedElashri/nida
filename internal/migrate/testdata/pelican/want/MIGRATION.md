# Migration report

Migrated from Pelican by `nida migrate`.

- Source: `testdata/pelican/site`
- Pages: 9, sections: 3, files copied: 10
- URLs: 7 unchanged, 0 kept with a path or alias
- Templates: Nida starter templates (restyle them, or port your old layouts from `_migration/original/`)
- `nida check`: passed

| Severity | Notes |
| --- | --- |
| Needs manual work | 1 |
| Not supported by Nida | 8 |
| Needs review | 19 |
| Changed automatically | 4 |

Work through the sections from the top. When `nida check` passes, compare the old and new `sitemap.xml` files to confirm every URL still exists.

## Needs manual work

These were left in place. `nida check` fails on most of them until they are fixed.

- Directive `.. graphviz::` is not converted; it is kept as reStructuredText. (`content/notes/math.rst:66`)

## Not supported by Nida

Nida has no equivalent for these, so they were dropped or kept only for reference.

- Category pages were `/category/{slug}.html`; Nida writes `/category/{slug}/`. Add a redirect on your host, such as Netlify's `/category/:slug.html /category/:slug/ 301`. (`pelicanconf.py`)
- Author pages (`author/{slug}.html`) have no equivalent in Nida. (`pelicanconf.py`)
- Pelican's index pages `archives.html`, `authors.html`, `categories.html`, `tags.html` have no direct equivalent. Nida lists taxonomies at `/tag/` and `/category/`; build an archive page with `groupByMonth` if you need one. (`pelicanconf.py`)
- Pelican plugins do not run in Nida: sitemap, neighbors, render_math. Equivalents: sitemap: Nida writes sitemap.xml itself; neighbors: use `.Page.PrevURL` and `.Page.NextURL`; render_math: set `markdown.math`. (`pelicanconf.py`)
- The `MARKDOWN` settings do not carry over (`markdown.extensions.codehilite`, `markdown.extensions.extra`, `markdown.extensions.toc`). Nida renders CommonMark with GFM tables, footnotes, heading ids, and highlighting built in; tables of contents come from `.Page.TOC`. (`pelicanconf.py`)
- `GITHUB_URL` has no Nida meaning; it is kept as `extra.github_url` for templates. (`pelicanconf.py`)
- `DISQUS_SITENAME` is not a literal value (it calls os.environ.get()), so it was not read. Set its equivalent by hand if it matters. (`pelicanconf.py`)
- Jinja templates in `themes/quiet/` are not translated; the site uses Nida's starter templates. The originals are in `_migration/original/themes/quiet/`. (`themes/quiet`)

## Needs review

These were converted, but the result may not match the old site exactly.

- Content uses math, so `markdown.math` is on. TeX renders to MathML at build time; the built-in `nida-scripts` partial adds its stylesheet.
- Content uses raw HTML, so `markdown.unsafe_html` is on. Nida turns it off by default because it lets Markdown authors inject scripts; keep it only if you trust everyone who writes content.
- The `[TOC]` marker was removed; build a table of contents from `.Page.TOC` in the template. (`content/engines/difference-engine.md:11`)
- Attribute list `{: .note }` is not supported and was removed. (`content/engines/difference-engine.md:20`)
- Abbreviation definitions (`*[ABBR]: ...`) are not supported and were removed. (`content/engines/difference-engine.md:28`)
- Converted from reStructuredText. Compare the result with the original in `_migration/original/`. (`content/notes/math.rst`)
- `.. contents::` was removed; build a table of contents from `.Page.TOC` in the template. (`content/notes/math.rst:11`)
- A definition list became bold terms followed by their definitions. (`content/notes/math.rst:70`)
- Pelican skips files without a title, so this one became a draft titled "Untitled". Publish it or delete it. (`content/notes/untitled.md`)
- `{category}` links now point at Nida's categories pages. (`content/pages/about.md:4`)
- `{tag}` links now point at Nida's tags pages. (`content/pages/about.md:4`)
- `Status: hidden` has no equivalent; the page is published and listed in its section. (`content/pages/colophon.md`)
- Pelican does not publish this file (it is outside the article, page, and static paths), so it was not copied. (`content/random.txt`)
- Pagination is 10 per page, but later pages move from Pelican's `index2.html` to Nida's `/page/2/`. (`pelicanconf.py`)
- Pelican feeds list every article; Nida's list the newest 20. Raise `limit` in `[feed]` if you want more. (`pelicanconf.py`)
- Category feeds move from `/feeds/{slug}.atom.xml` to `/category/<slug>/feeds/all.atom.xml`. (`pelicanconf.py`)
- `MENUITEMS` is kept as `extra.menu` (a list of name and url); the starter templates do not show it. (`pelicanconf.py`)
- `LINKS` is kept as `extra.links` (a list of name and url); the starter templates do not show it. (`pelicanconf.py`)
- `SOCIAL` is kept as `extra.social` (a list of name and url); the starter templates do not show it. (`pelicanconf.py`)

## Changed automatically

The migrator converted these. They are listed so you know what changed.

- Converted codehilite `:::lang` code blocks to fenced code. (`content/engines/difference-engine.md`)
- The site has its own robots.txt, so Nida's generated one is off. (`content/extra/robots.txt`)
- Rewrote Python-Markdown code fence attributes (`{.lang}`) as plain info strings. (`content/notes/2023-04-01-cafe.md`)
- This page was saved as the home page, so it became the home section `content/_index.md`. (`content/pages/home.md`)
