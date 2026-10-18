# Migration report

Migrated from Zola by `nida migrate`.

- Source: `testdata/zola/site`
- Pages: 5, sections: 4, files copied: 7
- URLs: 5 unchanged, 0 kept with a path or alias
- Templates: Nida starter templates (restyle them, or port your old layouts from `_migration/original/`)
- `nida check`: passed

| Severity | Notes |
| --- | --- |
| Needs manual work | 2 |
| Not supported by Nida | 5 |
| Needs review | 9 |
| Changed automatically | 4 |

Work through the sections from the top. When `nida check` passes, compare the old and new `sitemap.xml` files to confirm every URL still exists.

## Needs manual work

These were left in place. `nida check` fails on most of them until they are fixed.

- Shortcode `gallery` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/gallery.html`; the original is in `_migration/original/`. (`content/blog/shortcodes.md:10`)
- Shortcode `note` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/note.html`; the original is in `_migration/original/`. (`content/blog/shortcodes.md:12`)

## Not supported by Nida

Nida has no equivalent for these, so they were dropped or kept only for reference.

- Zola theme `even` is made of Tera templates, which Nida cannot use; the site uses Nida's starter templates instead. (`config.toml`)
- `markdown.smart_punctuation` has no Nida equivalent and was dropped. (`config.toml`)
- Zola `[slugify]` settings have no Nida equivalent and were dropped. (`config.toml`)
- Config key `smart_punctuation` has no Nida equivalent and was dropped. (`config.toml`)
- Tera templates in `templates/` are not translated; the site uses Nida's starter templates. The originals are in `_migration/original/templates/`. See the template guide to port them. (`templates`)

## Needs review

These were converted, but the result may not match the old site exactly.

- Content uses raw HTML, so `markdown.unsafe_html` is on. Nida turns it off by default because it lets Markdown authors inject scripts; keep it only if you trust everyone who writes content.
- The search index is `search_index.en.js` with a `documentStore.docs` map like Zola's, but its keys are site-relative URLs. Check your search script. (`config.toml`)
- Content asks for template `about.html`, which the starter templates do not have, so it renders with the default template until you port it. (`content/about.md`)
- `sort_by = "update_date"` became `"date"`; Nida sorts by publication date. (`content/blog/_index.md`)
- `insert_anchor_links` became the site-wide `markdown.heading_anchors`, which adds a link after each heading. (`content/blog/_index.md`)
- Content asks for template `blog-page.html`, which the starter templates do not have, so it renders with the default template until you port it. (`content/blog/_index.md`)
- Shortcode argument `images` is a list; Nida passes it as the text `a.png,b.png`. (`content/blog/shortcodes.md:10`)
- SCSS compiles through Nida's pipeline, which needs the `sass` command installed (Zola has a built-in compiler). (`sass`)
- The stylesheet moved from `/style.css` to `/css/style.css`; update links to it when you port your templates. (`sass/style.scss`)

## Changed automatically

The migrator converted these. They are listed so you know what changed.

- Feeds map to `[feed]` (Atom) and `[rss]`, keeping their file names. (`config.toml`)
- Set `date` from the file name, which Zola reads and Nida does not. (`content/blog/2024-03-05-dated-post.md`)
- Set `slug` without the date prefix, as Zola does. (`content/blog/2024-03-05-dated-post.md`)
- Converted Zola shortcodes to Nida's `{{< name >}}` syntax. (`content/blog/shortcodes.md`)
