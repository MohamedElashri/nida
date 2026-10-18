# Migration report

Migrated from Hugo by `nida migrate`.

- Source: `testdata/hugo/site`
- Pages: 9, sections: 5, files copied: 15
- URLs: 7 unchanged, 2 kept with a path or alias
- Templates: translated from Hugo
- `nida check`: passed

| Severity | Notes |
| --- | --- |
| Needs manual work | 4 |
| Not supported by Nida | 8 |
| Needs review | 20 |
| Changed automatically | 26 |

Work through the sections from the top. When `nida check` passes, compare the old and new `sitemap.xml` files to confirm every URL still exists.

## Needs manual work

These were left in place. `nida check` fails on most of them until they are fixed.

- Archetype uses Hugo template features Nida's archetypes lack (they get `.Title`, `.Date`, `.Slug`, `.Section`, and `.Path`); it is kept in `_migration/original/` to rewrite. (`archetypes/posts.md`)
- Content in `.org` format is not supported (Nida reads Markdown only); convert it to Markdown. The file is kept in `_migration/original/`. (`content/docs/guide.org`)
- Shortcode `notice` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/notice.html`; the original is in `_migration/original/`. (`content/posts/first-post.md:16`)
- Shortcode `gist` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/gist.html`; the original is in `_migration/original/`. (`content/posts/second.md:11`)

## Not supported by Nida

Nida has no equivalent for these, so they were dropped or kept only for reference.

- `markup.highlight.linenos` has no Nida equivalent; set code block options per block instead. (`config.toml`)
- `summaryLength = 40` has no equivalent; Nida summaries are the text before `<!-- more -->` or the first paragraph. (`config.toml`)
- Languages fr are not migrated: Nida builds one language per site. Only `en` content was migrated; translated files are kept in `_migration/original/`. (`config.toml`)
- `outputs` has no equivalent; Nida writes HTML pages, feeds, the sitemap, and the search index only. (`config.toml`)
- Config key `someunknownkey` has no Nida equivalent and was dropped. (`config.toml`)
- Translation for language `fr` was not migrated (Nida builds one language per site); it is kept in `_migration/original/`. (`content/about.fr.md`)
- `expirydate` has no equivalent; the page stays published. (`content/posts/second.md`)
- Data files in this format are not read by Nida (TOML, JSON, and YAML are); it is kept in `_migration/original/`. (`data/table.csv`)

## Needs review

These were converted, but the result may not match the old site exactly.

- Hugo layouts are kept in `_migration/original/` for reference.
- Content uses raw HTML, so `markdown.unsafe_html` is on. Nida turns it off by default because it lets Markdown authors inject scripts; keep it only if you trust everyone who writes content.
- Hugo Pipes assets in `assets/` were copied to `static/` at the same paths. Nida does not fingerprint, minify, or bundle them the way Pipes did; see `[pipeline]` for the options it has. (`assets`)
- SCSS compiles through Nida's pipeline from `static/scss/`, which needs the `sass` command installed (Hugo uses its own Dart Sass or LibSass). (`assets`)
- Hugo turns straight quotes and dashes into typographic ones by default; Nida leaves them as written. (`config.toml`)
- Nida has no menus; Hugo's menus were copied to `[extra.menu]`, one list of `name`, `url`, and `weight` entries per menu, for templates to read with `dig .Config.Extra "menu.main"`. Menu entries in page front matter stay in each page's `extra`. (`config.toml`)
- The Google Analytics ID is in `[extra] google_analytics`; add the tracking snippet to your base template. (`config.toml`)
- Permalink `docs = "/docs/:filename/:yearday/"` uses tokens Nida cannot express, so its pages keep their URLs through `path`. (`config.toml`)
- Environment config in `config/production/` was not merged; only `config/_default/` and the root config were read. Copy any settings you need into config.toml. (`config/production`)
- Hugo served this section at `/notes/`, but Nida serves it at `/Notes/`. Rename the directory to keep the old URL. (`content/Notes`)
- The file had no front matter, which Nida requires; its title was set from the file name. (`content/Notes/plain.markdown`)
- Headless bundle (or `build.render: never`) became a draft, since Nida renders every published page. Templates that read its content need rewriting. (`content/headless-bundle/index.md`)
- `figure` title was merged into the caption; Nida's figure has no separate title. (`content/posts/first-post.md:12`)
- `figure` option `target` has no Nida equivalent and was dropped. (`content/posts/first-post.md:12`)
- `youtube` option `autoplay` has no Nida equivalent and was dropped. (`content/posts/first-post.md:14`)
- `highlight` option `style` has no Nida equivalent and was dropped. (`content/posts/first-post.md:20`)
- `details` option `open` was dropped. (`content/posts/first-post.md:26`)
- Hugo ran the `youtube` shortcode inside a code block; Nida shows it as text. (`content/posts/first-post.md:36`)
- Both `date` and `publishdate` were set; Nida has one publication date, so `date` was kept. Pages are hidden while their `date` is in the future. (`content/posts/second.md`)
- Content asks for layout `wide` through `layout` or `type`; it renders with the default template unless a template with that name exists. (`content/posts/second.md`)

## Changed automatically

The migrator converted these. They are listed so you know what changed.

- Translated 4 Hugo layouts into Nida templates. The translation is best effort: compare the pages with the old site.
- Archetype expressions now use Nida's `.Title`, `.Date`, and `.Slug`. (`archetypes/default.md`)
- `enableGitInfo` became `updated_from_git`, which sets each page's `updated` date from git. (`config.toml`)
- Hugo's RSS feeds map to `[rss]` with the file name `index.xml`, so `/index.xml`, section feeds, and term feeds keep their URLs. Nida's Atom feed is off, as Hugo has none. (`config.toml`)
- `params` became `[extra]`. Hugo lowercases param keys, and so does the migration: `.Site.Params.fooBar` is `.Config.Extra.foobar`. (`config.toml`)
- Hugo permalinks became `[permalinks]` patterns for each section directory. (`config.toml`)
- Renamed to `content/Notes/plain.md`: Nida reads Markdown from `.md` files and has one language. (`content/Notes/plain.markdown`)
- Set `path` so the page keeps its old URL. (2 places)
  - `content/Notes/plain.markdown`
  - `content/docs/intro.md`
- Renamed to `content/about.md`: Nida reads Markdown from `.md` files and has one language. (`content/about.en.md`)
- Rewrote `date` dates in RFC 3339 form, which Nida reads. (`content/posts/2024/nested.md`)
- Set `slug` to the one Hugo used in the URL, which Nida would derive differently. (2 places)
  - `content/posts/My_Draft.md`
  - `content/posts/second.md`
- Set `generate_feeds` so the section keeps its Hugo RSS feed at `index.xml`. (`content/posts/_index.md`)
- `url` became `path`, which sets the page URL exactly. (`content/posts/bundle/index.md`)
- Front matter keys are lowercase, as Nida reads them. (2 places)
  - `content/posts/first-post.md`
  - `content/posts/second.md`
- Renamed `lastmod` to `updated`. (`content/posts/first-post.md`)
- Relative `aliases` were made absolute, as Hugo resolves them against the page's section. (`content/posts/first-post.md`)
- `summary` became `description`, Nida's hand-written summary. (`content/posts/first-post.md`)
- The first of `images` became `image`, the social preview image of the `nida-seo` partial. (`content/posts/first-post.md`)
- `{{% %}}` shortcodes became `{{< >}}`; Nida renders their inner text as Markdown. (`content/posts/first-post.md`)
- `highlight` shortcodes became fenced code blocks. (`content/posts/first-post.md`)
- `param` shortcodes were replaced with their values. (`content/posts/first-post.md`)
- Moved `params` fields to the top level, where Nida reads custom fields into `.Page.Extra`. (`content/posts/second.md`)
- Translation files became Nida's `key = "value"` TOML; plural forms keep their `other` text (Nida has no plurals). Templates call `t "key"` instead of `i18n "key"`. (`i18n`)
