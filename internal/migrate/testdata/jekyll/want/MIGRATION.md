# Migration report

Migrated from Jekyll by `nida migrate`.

- Source: `testdata/jekyll/site`
- Pages: 15, sections: 4, files copied: 17
- URLs: 8 unchanged, 5 kept with a path or alias
- Templates: Nida starter templates (restyle them, or port your old layouts from `_migration/original/`)
- `nida check`: passed

| Severity | Notes |
| --- | --- |
| Needs manual work | 6 |
| Not supported by Nida | 8 |
| Needs review | 18 |
| Changed automatically | 47 |

Work through the sections from the top. When `nida check` passes, compare the old and new `sitemap.xml` files to confirm every URL still exists.

## Needs manual work

These were left in place. `nida check` fails on most of them until they are fixed.

- Shortcode `note` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/note.html`; the original is in `_migration/original/`. (`_posts/2024-01-15-hello-world.md:35`)
- Shortcode `gist` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/gist.html`; the original is in `_migration/original/`. (`_posts/2024-01-15-hello-world.md:37`)
- Liquid `{% if site.show_extra %}` is left as text; Nida does not run Liquid. Replace it with content or a shortcode. (`_posts/2024-01-15-hello-world.md:47`)
- Liquid `{% endif %}` is left as text; Nida does not run Liquid. Replace it with content or a shortcode. (`_posts/2024-01-15-hello-world.md:47`)
- Not migrated: Jekyll skips posts whose file name has no `YYYY-MM-DD-` date, so this was never published. Rename it to migrate it. (`_posts/notes.md`)
- Not migrated: Jekyll runs Liquid over this file, but Nida copies static files unchanged. Write its output to `static/search.json`, or generate it from a template. (`search.json`)

## Not supported by Nida

Nida has no equivalent for these, so they were dropped or kept only for reference.

- kramdown `smart_quotes` has no Nida equivalent. (`_config.yml`)
- Plugin `jekyll-gist` cannot run in Nida. (`_config.yml`)
- Plugin `jemoji` cannot run in Nida. (`_config.yml`)
- Nida data files are TOML, JSON, or YAML; this one was kept in `_migration/original/`. Convert it to use it from templates. (`_data/people.csv`)
- Liquid templates in `_includes/` are not translated; the site uses Nida's starter templates. The originals are in `_migration/original/_includes/`. (`_includes`)
- Liquid templates in `_layouts/` are not translated; the site uses Nida's starter templates. The originals are in `_migration/original/_layouts/`. (`_layouts`)
- Ruby plugins in `_plugins/` cannot run in Nida. Reimplement what they do with shortcodes or templates. (`_plugins`)
- Collection `team` has `output: false`, so its documents are only used by templates. They were kept in `_migration/original/`; move what templates need into `data/`. (`_team`)

## Needs review

These were converted, but the result may not match the old site exactly.

- Content links to `#anchors`. kramdown and Nida build heading IDs differently for headings with numbers, punctuation, or non-ASCII letters, so check in-page links.
- Content uses raw HTML, so `markdown.unsafe_html` is on. Nida turns it off by default because it lets Markdown authors inject scripts; keep it only if you trust everyone who writes content.
- Nida writes `/404.html` from the `404` template, so this page was not migrated. Move its content into `templates/404.html`. (`404.html`)
- `author` is a table; its name became `author`, and the whole table is `.Config.Extra.author` for templates. (`_config.yml`)
- Pagination pages move from Jekyll's `/page2/` to Nida's `/posts/page/2/`, and the home page lists the latest 5 posts. (`_config.yml`)
- Removed the kramdown `{:toc}` list. Templates can render a table of contents from `.Page.TOC`. (`_posts/2024-01-15-hello-world.md`)
- Code blocks contain Liquid, which Jekyll rendered and Nida shows as written. Replace it with its output, or keep it if it was meant as an example. (`_posts/2024-01-15-hello-world.md`)
- Removed kramdown block attributes `{: .notice}`; Nida cannot attach classes or attributes to paragraphs, lists, or tables. (`_posts/2024-01-15-hello-world.md:25`)
- Removed kramdown span attributes after a link, image, or emphasis; Nida cannot add attributes there. (`_posts/2024-01-15-hello-world.md:27`)
- HTML page moved into a Markdown file as raw HTML. Check that it renders as before. (2 places)
  - `_posts/2024-05-05-html-post.html`
  - `archive.html`
- SCSS compiles through Nida's pipeline, which needs the `sass` command. The `_sass/` partials were copied into `static/assets/css/` so imports resolve; they are also copied to the output as static files. (`_sass`)
- Dropped `layout: doc` from an index page; Nida renders sections with the `index` or `list` template. (`docs/index.md`)
- This may use kramdown definition lists (`term` then `: definition`), which Nida renders as plain paragraphs. (`docs/install.md`)
- kramdown abbreviations (`*[ABBR]: ...`) are not supported and show as text. (`docs/install.md`)
- Content uses layout `doc`, which the starter templates do not have, so it renders with the default template until you port it. (`docs/install.md`)
- Dropped `layout: home` from an index page; Nida renders sections with the `index` or `list` template. (`index.html`)
- This index page is built with Liquid, so its body was dropped; the section lists its pages with Nida's `index` or `list` template instead. The original is in `_migration/original/`. (`index.html`)

## Changed automatically

The migrator converted these. They are listed so you know what changed.

- jekyll-feed became Nida's Atom feed at `/feed.xml`. (`_config.yml`)
- jekyll-redirect-from `redirect_from` lists became `aliases`. (`_config.yml`)
- jekyll-seo-tag is replaced by Nida's `nida-seo` partial, which the starter templates use. (`_config.yml`)
- Custom site settings (twitter_username) moved to `[extra]`; templates read `site.x` as `.Config.Extra.x`. (`_config.yml`)
- Wrote the `defaults` from the Jekyll config into the front matter, since Nida has no front matter defaults. (11 places)
  - `_drafts/a-draft.md`
  - `_posts/2023-12-31-update.md`
  - `_posts/2024-01-15-hello-world.md`
  - `_posts/2024-02-01-Second-Post.md`
  - `_posts/2024-03-01-update.md`
  - `_posts/2024-05-05-html-post.html`
  - `_posts/2024-06-01-crlf.md`
  - `_posts/2024-06-07-unpublished.md`
  - `docs/index.md`
  - `docs/install.md`
  - `journal/_posts/2024-04-01-nested.md`
- Posts in `_drafts/` became posts with `draft = true`. Jekyll dated them by file time; give them a `date` when you publish. (`_drafts/a-draft.md`)
- Set `date` from the file name, which Jekyll reads and Nida does not. (6 places)
  - `_posts/2023-12-31-update.md`
  - `_posts/2024-01-15-hello-world.md`
  - `_posts/2024-03-01-update.md`
  - `_posts/2024-05-05-html-post.html`
  - `_posts/2024-06-01-crlf.md`
  - `_posts/2024-06-07-unpublished.md`
- Removed `{% raw %}` tags; Nida does not run Liquid, so their content shows as written. (`_posts/2024-01-15-hello-world.md`)
- Removed `{% comment %}` blocks. (`_posts/2024-01-15-hello-world.md`)
- Converted `{% highlight %}` blocks to fenced code blocks. (`_posts/2024-01-15-hello-world.md`)
- Converted kramdown heading attributes to Nida's `{#id .class}` form. (`_posts/2024-01-15-hello-world.md`)
- Replaced the excerpt separator `<!--excerpt-->` with `<!-- more -->`. (`_posts/2024-01-15-hello-world.md`)
- Replaced simple `{{ page.x }}` and `{{ site.x }}` variables with their values. (`_posts/2024-01-15-hello-world.md`)
- Replaced `relative_url` and `absolute_url` filters with site-relative paths; Nida adds the base path itself. (`_posts/2024-01-15-hello-world.md`)
- Removed `{{ site.baseurl }}` and `{{ site.url }}` before links; Nida adds the base path to site-relative links itself. (`_posts/2024-01-15-hello-world.md`)
- Converted `{% post_url %}` and `{% link %}` tags to `@/` links. (`_posts/2024-01-15-hello-world.md`)
- Converted Liquid tags to Nida shortcodes. (`_posts/2024-01-15-hello-world.md`)
- Converted `{% include %}` tags to shortcodes named after the include file. Port each include from `_migration/original/_includes/` to `templates/shortcodes/`; its parameters arrive as `.Get "name"` instead of `include.name`. (`_posts/2024-01-15-hello-world.md`)
- Converted links to `.md` files into `@/` links. (2 places)
  - `_posts/2024-01-15-hello-world.md`
  - `about.md`
- Set `path` so the page keeps its old URL. (5 places)
  - `_posts/2024-01-15-hello-world.md`
  - `_posts/2024-06-01-crlf.md`
  - `archive.html`
  - `contact.md`
  - `journal/_posts/2024-04-01-nested.md`
- Set `title` from the file name, as Jekyll does. (`_posts/2024-02-01-Second-Post.md`)
- Gave dates without an offset the site time zone, America/New_York, as Jekyll does. (`_posts/2024-02-01-Second-Post.md`)
- Converted `redirect_from` to `aliases`. (`_posts/2024-02-01-Second-Post.md`)
- Moved `excerpt` to `description`. (`_posts/2024-03-01-update.md`)
- Nida writes the Atom feed at `/feed.xml` itself; this Liquid template was kept for reference only. (`feed.xml`)
- Rewrote `date` dates in RFC 3339 form, which Nida reads. (`journal/_posts/2024-04-01-nested.md`)
- The site has its own `robots.txt`, so Nida's generated one is off. (`robots.txt`)
