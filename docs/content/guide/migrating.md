+++
title = "Migrating to Nida"
description = "Move a Hugo, Jekyll, Zola, Eleventy, or Pelican site to Nida with nida migrate: what it converts, how it keeps your URLs, and what is left to do by hand."
weight = 45
template = "page"
+++

`nida migrate` converts a site built with another generator into a new Nida site. It handles the config, content, front matter, shortcodes, static files, and data, and keeps every page at its old URL. It also writes a report of everything it changed and everything it could not convert.

```bash
nida migrate ./old-site ./new-site
```

It supports Hugo, Jekyll, Zola, Eleventy, and Pelican. Generators differ in features, so most migrations leave some work by hand, usually templates and custom shortcodes. The report lists that work by file and line. Nothing is silently dropped.

## Migrate a site

{{< steps >}}

### Preview the migration

```bash
nida migrate --dry-run ./old-site ./new-site
```

A dry run migrates into a temporary directory, prints the report, and deletes everything. Use it to see how much work is left before committing to anything. The generator is detected from files such as `hugo.toml`, `_config.yml`, `pelicanconf.py`, or `.eleventy.js`. If detection guesses wrong, name the generator with `--from hugo`, `jekyll`, `zola`, `eleventy`, or `pelican`.

### Run it

```bash
nida migrate ./old-site ./new-site
```

```text
nida migrate: complete from=jekyll pages=142 sections=4 copied=310 urls_kept=139 urls_moved=3 manual=6 unsupported=8 review=18 check=passed time=210ms
Report: new-site/MIGRATION.md
Next: nida serve new-site
```

The old site is only read, never changed. The new directory must not exist or must be empty. To migrate again after changing the old site, delete the new directory first.

### Work through MIGRATION.md

The report groups its notes by how much attention they need, most urgent first:

| Section | Meaning |
| --- | --- |
| Needs manual work | Left in place for you to fix, such as a Liquid `{% include %}` or a shortcode with no Nida template. |
| Not supported by Nida | No Nida equivalent exists, such as a second language or a plugin. The item was dropped or kept for reference. |
| Needs review | Converted, but the result may differ from the old site, such as a template name the starter templates lack. |
| Changed automatically | What the migrator rewrote, listed so the changes are not a surprise. |

Each note names its file and line in the old site.

### Preview and check

```bash
nida serve ./new-site
nida check ./new-site --external
```

Shortcodes that still need a template render as a dashed **Migration TODO** box, so they are easy to spot in the preview.

### Compare URLs

Build both sites and diff their sitemaps, or the lists of files they write. `nida migrate` predicts each page's old URL from the old generator's rules, so a URL scheme it gets wrong would go unnoticed without this check.

```bash
nida build ./new-site
diff <(grep -o '<loc>[^<]*' old-site/_site/sitemap.xml | sort) \
     <(grep -o '<loc>[^<]*' new-site/public/sitemap.xml | sort)
```

{{< /steps >}}

## What gets migrated

| Part of the site | What happens |
| --- | --- |
| Config | Mapped onto `config.toml`: title, base URL, language, taxonomies, permalinks, feeds, pagination, Markdown options. Custom values go into `[extra]`. Keys with no Nida equivalent are reported. |
| Content | Copied into `content/`. Non-`.md` Markdown extensions such as `.markdown` are renamed to `.md`. Sections get an `_index.md`, and page bundles stay bundles. |
| Front matter | Edited in place. A custom `path` field moves to `extra.path`, since `path` sets the URL in Nida. YAML stays YAML and TOML stays TOML, with comments and key order kept. Keys are renamed to Nida's (`lastmod` becomes `updated`, `layout` becomes `template`, `redirect_from` becomes `aliases`). Dates are rewritten into a form Nida reads, and files that had no front matter get some. |
| Shortcodes and template tags in content | Converted to Nida's `{{< name >}}` syntax. Code blocks and inline code are left alone. Shortcodes Nida has no template for render a visible placeholder until you write one. |
| URLs | Kept. See [Keeping URLs](#keeping-urls). |
| Static files | Copied into `static/` at the paths they were served from. |
| Data and translations | Copied into `data/` and `i18n/` when Nida reads the format: TOML, YAML, and JSON. |
| Templates | Nida's starter templates, or translated templates for Hugo. See [Templates](#templates). Old layouts are kept in `_migration/original/`. |
| Raw HTML in content | If any page uses it, `markdown.unsafe_html` is turned on so pages render as before, and the report says so. |

Files that have no Nida equivalent, such as Liquid layouts, Tera templates, Jinja themes, reStructuredText originals, and Hugo layouts, are copied to `_migration/original/` with their old paths. Delete the directory once you are done with it; Nida never reads it.

## Keeping URLs

Before writing the report, `nida migrate` routes the new site with Nida's own router and compares each page's URL with the one it had before:

- It first sets `[permalinks]` patterns that reproduce the old scheme, so most pages need no change. Jekyll's default `/:categories/:year/:month/:day/:title.html`, for example, becomes `/{year}/{month}/{day}/{slug}.html`. Nida writes patterns that end in a file name as that file.
- A page that would still move gets a `path` with its old URL in its front matter. Pass `--urls redirect` to let pages move to Nida's URLs instead; the old URL then becomes an alias, which writes a redirect page.
- A page whose old URL now belongs to another page keeps its new URL, and the clash is reported.

Some URLs cannot be reproduced. Examples are Pelican's `.html` tag pages and Jekyll's paginated `/page2/`. The report lists these, with redirect rules to add on your web host.

## Templates

Templates are where generators differ most, so they are rarely converted exactly.

- **Hugo** layouts are translated into Nida templates: `baseof` blocks, partials, menus, params, pagination, taxonomies, and shortcode templates. The translated set is used only if the site builds with it. Constructs that cannot be translated are reported with their line, and render nothing so the rest still builds. Compare pages with the old site, since the translation is best effort.
- **Jekyll** (Liquid), **Zola** (Tera), **Eleventy** (Nunjucks, Liquid, and others), and **Pelican** (Jinja) templates are not translated. The site uses Nida's starter templates, which build at once and list every section, page, and term. Restyle them, or port your layout from `_migration/original/`. [Templates](@/guide/templates.md) and [Themes](@/guide/themes.md) explain Nida's template model.

If translated Hugo templates do not build, the site falls back to the starter templates. The translations are then kept in `_migration/translated-templates/`, and the report says why the build failed.

## From Hugo

Converted automatically:

- Config from `hugo.*` or `config.*` in TOML, YAML, or JSON, merged with `config/_default/`. Keys are matched case-insensitively, as Hugo does.
- `params` become `[extra]`, with keys lowercased the way Hugo reads them. Menus become `[extra.menu]`, and the copyright and analytics IDs go into `[extra]` as well.
- Permalink tokens `:year`, `:month`, `:day`, `:section`, `:sections`, `:slug`, `:title`, `:filename`, and `:contentbasename`, plus `uglyURLs`.
- RSS feeds keep `index.xml`, including section and term feeds.
- Front matter: `lastmod` becomes `updated` and `publishDate` becomes `date`. `url` becomes `path`, and relative aliases are made absolute. `summary` becomes `description`, the first of `images` becomes `image`, and `params` are flattened to the top level. Headless bundles become drafts. JSON front matter becomes YAML.
- Shortcodes:
  - `{{% x %}}` becomes `{{< x >}}`.
  - `ref` and `relref` become `@/` links.
  - `highlight` becomes a fenced code block with `hl_lines` and `linenos`.
  - `figure`, `youtube`, and `details` map onto Nida's built-ins, and `param` is replaced by its value.
  - Escaped shortcodes stay literal.
- Pipes assets are copied to `static/` at the same paths, and SCSS entry files compile through `[pipeline.scss]`.
- Data, i18n files (flattened to `key = "value"`), archetypes using common expressions, and static files. Theme directories are merged under the site's.

Not converted:

- **Languages:** only the default language is migrated. Other languages' content and config are kept in `_migration/original/`.
- **Pipes processing:** fingerprinting, `js.Build`, PostCSS, and image processing are not done; assets are copied as they are.
- **Config:** `outputs`, `outputFormats`, `mediaTypes`, `related`, `summaryLength`, `cascade`, `expiryDate`, page `resources` metadata, render hooks (`_markup/`), modules, and `markup` options beyond those listed above.
- **Shortcodes:** `vimeo`, `gist`, `tweet`/`x`, `instagram`, `qr`, nested shortcodes, and custom shortcodes in content get placeholders. Custom shortcode templates are translated along with the layouts when they can be.
- **Templates:** `.Scratch` and `.Store`, partials used as return values, `getJSON`/`getCSV`/`resources.GetRemote`, math functions, `replaceRE`/`findRE`, and sorting methods such as `.ByDate`. Translations report these and render nothing in their place.
- **Formats:** content in `.org`, `.adoc`, `.rst`, `.pandoc`, or `.ipynb` is kept for reference, and CSV and XML data files are not read.

## From Jekyll

Converted automatically:

- `url` and `baseurl` become `base_url`. Settings such as title, description, author, language, `future`, and `show_drafts` are mapped, and other site variables go into `[extra]`.
- Posts from `_posts/`, including nested ones such as `blog/_posts/`, which add the `blog` category. Dates come from the file name unless the front matter has one. Dates without an offset use the site's `timezone`.
- Drafts from `_drafts/` and `published: false`.
- Pages, kept in their directories. `index.md` and `index.html` become section index pages, and HTML pages without Liquid become Markdown.
- Collections with `output: true` become sections.
- Front matter `defaults` are written into every matching file. `layout` becomes `template`, `redirect_from` becomes `aliases`, `excerpt` becomes `description`, and `last_modified_at` becomes `updated`.
- Every permalink style (`date`, `pretty`, `ordinal`, `weekdate`, `none`) and placeholder, including `.html` URLs, for posts, pages, and collections.
- Liquid in content:
  - `{% highlight %}` becomes a fenced code block, and `{% raw %}` is kept verbatim.
  - `{% post_url %}` and `{% link %}` become `@/` links, and `relative_url` and `{{ site.baseurl }}` become plain paths.
  - Simple `page.x` and `site.x` values are filled in.
  - `{% include %}` becomes a placeholder shortcode named after the include.
- kramdown:
  - `{:toc}` is removed (build the table of contents in the template instead), and heading IDs become `## Title {#id}`.
  - `{::comment}` blocks are handled, a custom `excerpt_separator` becomes `<!-- more -->`, and `$$` math turns on `markdown.math`.
- jekyll-feed, jekyll-sitemap, jekyll-seo-tag, jekyll-paginate, and jekyll-redirect-from map onto Nida's feed, sitemap, SEO partial, pagination, and aliases. The plugins GitHub Pages enables by default are honored.
- SCSS entry files compile through `[pipeline.scss]` without changing their URLs. `_data/` YAML, JSON, and TOML go to `data/`.

Not converted:

- Liquid layouts and includes, and Liquid control flow (`if`, `for`, `assign`) in content. Each place is reported.
- jekyll-archives, jemoji, jekyll-mentions, Ruby plugins, and gem-based themes.
- Pagination URLs move from `/page2/` to `/posts/page/2/`. `redirect_to` and `sitemap: false` have no equivalent.
- kramdown block and span attributes such as `{: .class}`, definition lists, abbreviations, and smart quotes. Heading IDs Nida generates may differ from kramdown's for headings with numbers, punctuation, or non-ASCII text.
- CSV and TSV data files. Drafts get no date, since Jekyll used the file's modification time.

## From Zola

Zola is closest to Nida. TOML front matter, `_index.md` sections, page bundles, `[taxonomies]`, `[extra]`, `@/` links, `<!-- more -->`, and `{% details() %}` blocks all work unchanged.

Converted automatically:

- `default_language` becomes `language`. Feeds keep their file names: `atom.xml` maps to `[feed]` and `rss.xml` to `[rss]`. `build_search_index`, `[[taxonomies]]`, external link options, and `highlight_theme` are mapped when Chroma has a style of that name.
- Shortcodes: `{{ name(a="x") }}` becomes `{{< name a="x" >}}`, and block shortcodes ending in `{% end %}` become paired shortcodes. Escaped shortcodes stay literal.
- Dates and slugs in file names such as `2024-05-01-hello.md`, which Zola reads and Nida does not, are written into front matter.
- `sort_by = "update_date"` and `"title_bytes"` map onto Nida's sorts, and `insert_anchor_links` becomes `markdown.heading_anchors`.
- Assets next to section files move to `static/` at the same URL. `sass/` moves to `static/css/` with `[pipeline.scss]`; its stylesheets move from `/style.css` to `/css/style.css`, and the report says so.

Not converted: Tera templates and themes, multilingual sites, `smart_punctuation`, `render_emoji`, `[link_checker]`, `[slugify]`, section `redirect_to` and `aliases`, `paginate_reversed`, and `in_search_index`. Zola's default slug strategy transliterates non-ASCII titles, which Nida does not; affected pages keep their old URL through `path`.

## From Eleventy

Eleventy's config is JavaScript, which `nida migrate` reads without running it. It picks up `dir`, passthrough copies, `pathPrefix`, `templateFormats`, `markdownTemplateEngine`, and feed plugin settings. Anything computed at runtime is reported.

Converted automatically:

- Site metadata from `_data/metadata.json` or `site.json`: title, URL, description, and author. Global JSON and YAML data, and JS data files that export a plain object.
- The data cascade: directory and template data files (`posts.json`, `posts.11tydata.json`) are merged into each page's front matter, the way Eleventy merges them.
- Front matter in YAML, JSON, and TOML. `layout` becomes `template`, and `eleventyNavigation.order` becomes `weight`. Dates such as `Last Modified` or `git Created` come from git or the file time, and `permalink: false` becomes a draft.
- Tags: collection markers such as `post` or `posts` are moved aside, and the rest become `tags` terms.
- Permalinks, including templates using `page.fileSlug`, `page.filePathStem`, `slugify`, and `date` filters. A permalink shared through directory data becomes a `[permalinks]` pattern.
- Shortcodes in Markdown: `{% raw %}`, `{% highlight %}` becomes a fenced block, `{% image %}` becomes `figure`, and `{% youtube %}` maps to the built-in; custom shortcodes get placeholders. `{{ variable }}` values that come from front matter or data are filled in.
- `.njk` and other template pages at a section's URL become that section's index. Feed, sitemap, and tag pages map onto Nida's own.

Not converted: config computed in JavaScript, custom filters, collections and transforms, `eleventyComputed`, pagination templates, JS front matter, and data files with logic. Pages written in a template language other than Markdown are kept for reference. Eleventy formats dates in the build machine's local timezone and the migrator uses UTC, so a date near midnight can move by a day in a URL. eleventy-img's responsive images are not reproduced.

## From Pelican

Pelican's settings are Python, which `nida migrate` reads as literal values without running them. It understands `from pelicanconf import *` and lets `publishconf.py` win, since that file usually holds the real `SITEURL`.

Converted automatically:

- Site settings, feeds (`FEED_ALL_ATOM` and `FEED_ALL_RSS`), tag and category taxonomies, pagination, and `MENUITEMS`/`LINKS`/`SOCIAL` (into `[extra]`).
- Article and page URLs from the `*_SAVE_AS` settings, including the default `{slug}.html`, as `[permalinks]` patterns.
- Markdown with Pelican's `Key: value` metadata headers, which become YAML front matter, and Markdown with YAML front matter. Dates use `TIMEZONE`, `DEFAULT_DATE`, and metadata taken from paths and file names.
- Slugs from titles, the way Pelican makes them, including `SLUG_REGEX_SUBSTITUTIONS`. Categories come from folder names.
- `Status: draft` becomes a draft. A page saved as `index.html` becomes the home page.
- `{filename}` links become `@/` links, `{static}` and `{attach}` become file URLs, and `{tag}` and `{category}` become term URLs.
- Python-Markdown code block markers (`:::python`, `#!python`) and `[TOC]` markers.
- reStructuredText for the common subset: titles, docinfo fields, lists, literal and code blocks, images and figures, admonitions (as callouts), footnotes, simple tables, and math. Anything outside it stays in a fenced `rst` block and is reported, and the original file is kept.
- HTML articles: metadata from `<title>` and `<meta>`, and the body as raw HTML.

Not converted:

- Plugins, Typogrify, `SUMMARY_MAX_LENGTH`, translations (`Lang`/`Translation`), `Status: hidden`, author pages, archives, and `.ipynb`, `.org`, or AsciiDoc content.
- Tag and category pages at `.html` URLs move to `/tag/x/`.
- reStructuredText grid tables, substitutions, most roles, and `include`.
- Slug transliteration covers Latin, Greek, and Cyrillic titles. Titles in other scripts are reported, since their old URLs cannot be predicted exactly.

## Porting templates by hand

When you port a layout to Nida, these are the common changes from Hugo's Go templates. Liquid, Tera, and Jinja templates follow the same ideas with different syntax.

| Hugo template | Nida template |
| --- | --- |
| `.Content` | `unsafeHTML .Page.BodyHTML` |
| `.Params.foo` | `.Page.Extra.foo` |
| `.Site.Title`, `.Site.Params.foo` | `.Config.Title`, `.Config.Extra.foo` |
| `.Permalink`, `.RelPermalink` | `.CanonicalURL`, `{{ .BasePath }}{{ .Page.URL }}` |
| `.Summary` | `.Page.Summary` |
| `.TableOfContents` | Build from `.Page.TOC`; see [Writing Content](@/guide/writing.md#table-of-contents). |
| `.Site.RegularPages`, `where` | `.Index.AllPages`, `where` |
| `.Site.GetPage "/about"` | `.Index.GetPage "pages/about.md"` |
| `partial "x" .` | `template "x" .` with `{{ define "x" }}` in a partial file |
| `T "key"`, `i18n "key"` | `t "key"` |
| `relURL`, `absURL`, `dict`, `slice`, `first`, `jsonify`, `markdownify`, `truncate`, `plainify` | Available with the same names. |
| `default "x" .Foo` | `default .Foo "x"`: Nida takes the value first. |

For other generators, start from the starter templates the migration wrote. They already show how to list a section's pages, link taxonomy terms, and include the `nida-seo` and `nida-scripts` partials.
