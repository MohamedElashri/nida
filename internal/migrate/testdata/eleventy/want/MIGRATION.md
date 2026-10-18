# Migration report

Migrated from Eleventy by `nida migrate`.

- Source: `testdata/eleventy/site`
- Pages: 7, sections: 4, files copied: 15
- URLs: 4 unchanged, 2 kept with a path or alias
- Templates: Nida starter templates (restyle them, or port your old layouts from `_migration/original/`)
- `nida check`: passed

| Severity | Notes |
| --- | --- |
| Needs manual work | 5 |
| Not supported by Nida | 5 |
| Needs review | 20 |
| Changed automatically | 26 |

Work through the sections from the top. When `nida check` passes, compare the old and new `sitemap.xml` files to confirm every URL still exists.

## Needs manual work

These were left in place. `nida check` fails on most of them until they are fixed.

- Shortcode `aside` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/aside.html`; the original is in `_migration/original/`. (`content/blog/2024-01-05-first-post.md:14`)
- Shortcode `currentBuildDate` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/currentBuildDate.html`; the original is in `_migration/original/`. (`content/blog/2024-01-05-first-post.md:33`)
- Template tag `{% for item in collections.posts %}` was left as it was; Nida content has no template logic. Rewrite it as Markdown or a shortcode. (`content/blog/2024-01-05-first-post.md:35`)
- `{{ item.data.title }}` was left as it was; Nida content cannot run template expressions. Replace it with its text. (`content/blog/2024-01-05-first-post.md:35`)
- Template page (.html) at `/docs/templated/` was not migrated; recreate it as a Nida template or a Markdown page. The original is in `_migration/original/`. (`content/docs/templated.html`)

## Not supported by Nida

Nida has no equivalent for these, so they were dropped or kept only for reference.

- JavaScript data file computes its data (the export is not a plain object or array), so it was not migrated. Write its output to `data/build.json`. (`_data/build.js`)
- `eleventyComputed` values are computed when Eleventy builds and were not migrated; they stay in extra data unevaluated. (`content/notes/Hello World.md`)
- Custom collections (featured) have no Nida equivalent. Nida lists a section's pages with `.Pages` and filters them with `where .Index.AllPages`. (`eleventy.config.js`)
- Filters (readableDate) are JavaScript and were not migrated. Nida's template functions cover many of them; see the template function reference. (`eleventy.config.js`)
- The Markdown library setup (`setLibrary`/`amendLibrary`) was not migrated: Nida renders Markdown with goldmark and its `[markdown]` options, so markdown-it plugins have no effect. (`eleventy.config.js`)

## Needs review

These were converted, but the result may not match the old site exactly.

- 1 files were neither templates nor passthrough copies, so Eleventy did not publish them, and they were not migrated: `content/blog/notes.txt`.
- Content uses raw HTML, so `markdown.unsafe_html` is on. Nida turns it off by default because it lets Markdown authors inject scripts; keep it only if you trust everyone who writes content.
- Nida writes the 404 page from `templates/404.html`; move this page's text there. (`content/404.md`)
- Content asks for template `base`, which the starter templates do not have, so it renders with the default template until you port it. (`content/about.md`)
- `eleventyNavigation` is kept in extra data, and its `order` became `weight`. Nida's navigation comes from sections. (`content/about.md`)
- This template page listed content at `/blog/`; the Nida section page there replaces it, rendered by `templates/list.html`. (`content/blog.njk`)
- Tags used to build Eleventy collections (posts) moved to `collections` in extra data; the rest are `tags` terms. (3 places)
  - `content/blog/2023-03-01-second-post.md`
  - `content/blog/2024-01-05-first-post.md`
  - `content/blog/unlisted.md`
- `{% image %}` became Nida's `figure` shortcode. Turn on `[pipeline.images]` for responsive sizes like eleventy-img's. (`content/blog/2024-01-05-first-post.md`)
- `{{ page.url }}` is inside code. Eleventy ran it as a template without `{% raw %}`; Nida shows it as written. (`content/blog/2024-01-05-first-post.md:30`)
- `permalink: false` meant Eleventy did not write this page, so it is a draft, which Nida does not build either. (`content/blog/unlisted.md`)
- HTML page became Markdown with raw HTML; blank lines outside `<pre>` were removed so it stays one HTML block. Check how it renders. (`content/docs/page.html`)
- This template page listed content at `/`; the Nida section page there replaces it, rendered by `templates/index.html`. (`content/index.njk`)
- `notes/index.md` became the section page `content/notes/_index.md`, since Nida would otherwise treat it as a page bundle and hide the other pages in its directory. (`content/notes/index.md`)
- Tag pages are replaced by Nida's taxonomy pages at `/tags/<slug>/`. (`content/tags.njk`)
- The config file is JavaScript and was read without running it. These settings were picked up: addPassthroughCopy, dir.data, dir.includes, dir.input, dir.output, excerpt options, markdownTemplateEngine, pathPrefix, templateFormats. Settings computed in code were not seen. (`eleventy.config.js`)
- Shortcodes defined in the config (currentBuildDate, aside) are JavaScript. Uses in content became Nida shortcodes with placeholder templates. (`eleventy.config.js`)
- Plugin `@11ty/eleventy-navigation`: Nida builds navigation from sections with `.Index.Nav`; `eleventyNavigation` values are kept in each page's extra data. (`eleventy.config.js`)
- markdown-it-anchor became `markdown.heading_anchors`, which adds a `#` link after each heading. (`eleventy.config.js`)

## Changed automatically

The migrator converted these. They are listed so you know what changed.

- The `blog` permalink `/blog/{{ page.date | date: "%Y/%m" }}/{{ page.fileSlug }}/` became the Nida permalink pattern `/blog/{year}/{month}/{slug}/`.
- JavaScript data file exports a plain value, so it became `data/metadata.json`. (`_data/metadata.js`)
- `base_url` includes the path prefix `/blog/`, so Nida serves the site under it. (`_data/metadata.js`)
- Site title, description, language, author, and URL came from this data file. (`_data/metadata.js`)
- Removed `permalink`, which Nida does not read; the URL check below keeps the page at its old URL. (2 places)
  - `content/about.md`
  - `content/blog/2023-03-01-second-post.md`
- Removed template comments, which Eleventy did not output. (`content/about.md`)
- Replaced template variables such as `{{ title }}` with their values from the front matter and global data. (2 places)
  - `content/about.md`
  - `content/blog/2024-01-05-first-post.md`
- Added a `title`, taken from the first heading or the file name. (2 places)
  - `content/about.md`
  - `content/notes/no-front-matter.md`
- Set `path` so the page keeps its old URL. (2 places)
  - `content/about.md`
  - `content/blog/2023-03-01-second-post.md`
- `---json` front matter was rewritten in a form Nida reads. (`content/blog/2023-03-01-second-post.md:1`)
- Set `date` from the file name, which Eleventy reads and Nida does not. (`content/blog/2024-01-05-first-post.md`)
- Unwrapped `{% raw %}` blocks; Nida does not run Liquid, so their contents are shown as written. (`content/blog/2024-01-05-first-post.md`)
- Converted Eleventy shortcodes to Nida's `{{< name >}}` syntax. (`content/blog/2024-01-05-first-post.md`)
- `{% highlight %}` blocks became fenced code blocks; Nida highlights code natively. (`content/blog/2024-01-05-first-post.md`)
- The excerpt separator became `<!-- more -->`. (`content/blog/2024-01-05-first-post.md`)
- Data file values were copied into the front matter of the pages they apply to, since Nida has no data cascade. (`content/blog/blog.11tydata.json`)
- Rewrote `date` dates in RFC 3339 form, which Nida reads. (`content/notes/Hello World.md`)
- Set `slug` to the file name, which Eleventy uses in the URL as it is. (`content/notes/Hello World.md`)
- Nida writes `sitemap.xml` itself (`[sitemap]`). (`content/sitemap.xml.njk`)
- Plugin `@11ty/eleventy-plugin-syntaxhighlight`: Nida highlights fenced code blocks natively; set `syntax_theme` to choose a style. (`eleventy.config.js`)
- Plugin `@11ty/eleventy-plugin-rss`: Nida writes feeds itself through `[feed]` and `[rss]`. (`eleventy.config.js`)
- The feed at `/feed/feed.xml` is written by Nida's `[feed]`, at the same path. (`eleventy.config.js`)
