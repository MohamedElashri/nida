# Changelog

All notable changes to Nida are documented here.

## [0.7.0] - Unreleased

### Added
* Optional RSS 2.0 feed through a new `[rss]` table, written next to the Atom feed. Off by default.
* Warnings for unrecognized `config.toml` keys, so typos and settings from other generators no longer get dropped without notice.
* `future` config option and `--future`/`-f` flag. Future-dated pages are now skipped by default.
* `robots_meta` config option, exposed to templates as `.Robots`.
* `[home]` table with `sections` and `limit` to control which pages the homepage lists.
* `markdown.hard_wraps` option.
* `.Theme.Extra`, the merged theme and site `[extra]` table.
* Full YAML front matter support, including block lists and nested maps.
* Zola-style `[taxonomies]` front matter tables are read as taxonomy terms.
* Summaries: `<!-- more -->` splits a page, and `.Page.Summary` falls back to the first paragraph. Feeds use it when a page has no `description`.
* `.Page.TOC` and `.Section.TOC`: nested heading data for tables of contents. Custom heading IDs with `## Title {#id}`.
* `markdown.heading_anchors` adds a `#` self-link to each heading.
* GitHub-style callouts (`> [!NOTE]`, `> [!WARNING] Custom title`).
* Code block options in the info string: `title`, `hl_lines`, `linenos`, and `linenostart`.
* `markdown.math` passes `$...$` and `$$...$$` TeX through for KaTeX or MathJax.
* ```` ```mermaid ```` blocks render as `<pre class="mermaid">` for Mermaid.
* `.Page.WordCount`.
* Template functions: `dict`, `slice`, `seq`, `first`, `last`, `where`, `truncate`, `plainify`, `markdownify`, `absURL`, `relURL`, `jsonify`, `upper`, `title`, `split`, and `urlize`.
* `.Index.GetPage` and `.Index.GetSection` look up content by path.
* A "Writing Content" docs page and a template function reference.
* Template shortcodes: `{{< name args >}}` and paired `{{< name >}}...{{< /name >}}`, loaded from `templates/shortcodes/` in the site and themes. Built-in `figure`, `youtube`, and `callout` shortcodes.
* Themes can omit `config.toml`, and a site using a theme no longer needs its own `base.html`.
* `nida new` creates content from archetypes (`archetypes/<section>.md` or `default.md`, in the site or a theme).
* `.Index.Related` ranks pages by shared taxonomy terms. `.Index.Series` returns a post's series with position, previous, and next.
* `groupByMonth` and `merge` template functions.
* `syntaxCSS` template function returns highlighting CSS for `syntax_theme` (or a named Chroma style), so the setting has a visible effect.
* Built-in `nida-seo` partial for title, description, canonical, Open Graph, Twitter cards, and JSON-LD.
* `updated_from_git` sets `updated` dates from git history.
* Section feeds (`generate_feeds = true`) and per-term taxonomy feeds (`feed = true`). Both settings were parsed before but had no effect.
* Markdown images get `width` and `height` from the image file, and a `srcset` when the image pipeline is enabled. Page bundle images are resized too, with variants written next to the copied original.
* `nida check` validates config, content links, templates, shortcodes, and routes without writing output; `--external` also checks outbound links.
* `--base-url` and `--output`/`-o` flags for `build` and `serve`, for preview deployments.
* Build output reports elapsed time.
* `.Index.Ancestors` for breadcrumbs, section `weight` for ordering child sections, and `edit_url` with `.EditURL`.
* Data files: TOML, JSON, and YAML from `data/` (and themes) as `.Data`.
* Translations: `i18n/<language>.toml` files and the `t` template function, with language fallback. Default callout titles can be translated.
* `search.sections` and `search.body_limit`, plus a Pagefind guide for larger sites.
* `.Index.Nav` returns pages and child sections interleaved by weight for documentation sidebars, and `.Index.Neighbors` returns previous and next pages across that tree.
* Heading IDs keep Unicode letters, so headings in Arabic and other non-Latin scripts get readable anchors instead of `heading`, `heading-1`, and so on. ASCII IDs are unchanged.
* New docs: [Themes](https://melashri.net/nida/guide/themes/) (theme authoring with a minimal example theme) and [Migrating](https://melashri.net/nida/guide/migrating/) (Zola, Hugo, and Jekyll). The rest of the guide, reference, and internals pages are updated for 0.7.0.
* The docs site is rebuilt as the `nida-docs` theme in `docs/themes/nida-docs`, with a new design, dark mode, a keyboard-driven search dialog, generated navigation, an on-page outline, "last updated" dates from git, and tabs, cards, steps, and version-badge shortcodes.
* CI runs `nida check` on the documentation site.

### Changed
* Markdown rendering, page templates, and page writes now run in parallel. A 2,000-post site builds about 1.7× faster.
* **Breaking:** single newlines in Markdown paragraphs are now soft breaks (a space), matching CommonMark. Set `markdown.hard_wraps = true` to keep the old `<br>` behavior.
* **Breaking:** pages with a `date` in the future are no longer published unless `future = true` or `--future` is used.
* **Breaking:** pages no longer get `noai, noimageai` in `.Robots` by default. Set `robots_meta = "noai, noimageai"` to keep it.
* The homepage list now respects `[home]` instead of always showing the five newest pages from every section.
* Atom entry `updated` times now use the page's `updated` front matter when it is newer than `date`.

### Fixed
* `@/section/_index.md` links resolved through page permalink patterns and could produce URLs like `/posts/{slug}/`. Section links and `.Section.URL` now use the route the section renders at.
* Nested sections were missing from the search index, and explicit sections had an empty `.URL`.
* `pipeline.minify_css` and `pipeline.minify_js` were documented but did nothing. CSS and JS assets are now minified (files ending in `.min.css` or `.min.js` are left alone), before fingerprinting so hashes match the served bytes.
* The asset pipeline only processed the site's `static/` directory. Theme static files are now fingerprinted and minified too, with site files still taking precedence.
* `minify_html` removed whitespace between inline elements, so `<a>one</a> <b>two</b>` rendered as "onetwo". HTML is now minified with the same library as CSS and JS.
* Taxonomy terms given as a single string, such as `tags = "go"`, were ignored.
* Atom and RSS entry categories now come from the configured taxonomies instead of always `tags` and `categories`.
* `.Page.PrevURL`, `.PrevTitle`, `.NextURL`, and `.NextTitle` were always empty on rendered pages.
* A site with pages in `content/` but no `content/_index.md` produced no home page.
* `nida serve` partial rebuilds broke `@/section/_index.md` links.
* With the image pipeline enabled, a static `.webp` file failed the build. WebP files are now copied without resizing.
* `resizeImage` ignored `static_dir`, `content_dir`, and `pipeline.images.quality`.
* `readFile` and `style.css.html` lookup ignored parent themes, and a theme's `style.css.html` overrode the site's.
* A shortcode inside a heading, such as `## Callouts {{< since "0.7.0" >}}`, leaked its internal placeholder into the heading ID and the TOC title.
* A site with sections but no `section` or `list` template failed with `missing template "list"`. The error now explains which template to add.
* Theme inheritance now applies to templates, partials, static files, and SCSS. Before, only the selected theme's own files were used and `extends` affected `[extra]` only.
* A parent theme's `[extra]` values no longer override its child theme's values.
* Site partials and `base.html` now override theme versions as documented. Before, the order depended on file paths.
* Shortcode syntax inside code blocks and inline code is no longer processed, including the `details` shortcode.
* `@/` internal links in Markdown left out the `base_url` path, so they broke on sites published under a subpath such as `/nida/`.
* Feeds now make links absolute the way a browser does. Root-relative links previously got the `base_url` path added a second time.
* Pagination links ignored a custom `paginate_path` and always pointed to `/page/N/`, so Next/Previous links led to 404s.
* `[sections]` `paginate_path`, `default_sort_by`, and `default_page_template` were documented but had no effect.
* The canonical URL for a section without pagination was missing its trailing slash.
* `nida init` wrote `[rss]` and `[atom]` tables that the 0.6 engine ignored. It now writes `[feed]`.
* The docs described `[rss]`/`[atom]` settings that did not exist. They now match the engine.
* Removed unsupported `taxonomies.path` and `taxonomies.page_template` keys from the example sites.
* The serve rebuild now cleans up a disabled search index file.

### Migration Notes
* To keep hard line breaks: `[markdown] hard_wraps = true`.
* To keep publishing future-dated posts: `future = true`.
* To keep the previous robots meta value: `robots_meta = "noai, noimageai"`.

## [0.6.0] - 2026-08-02

### Added
* N/A

### Changed
* Changed the default feed protocol from RSS to Atom. Updated the default feeds generator as well
* Removed the `config_version` field from configuration files and the core engine. Nida no longer tracks configuration versions.

### Fixed
* N/A

## [0.5.10] - 2026-08-02

### Fixed
* Bug: Draft flag not respected

## Developer Notes
* Update all the github actions versions



## [0.5.9] - 2026-07-25

### Added
* Enabled pagination for the home page.

### Changed
* Refactored section tree construction to use pointer-based, bottom-up building to ensure parent/child states are correctly synchronized.
* Fixed implicit section generation to use correct relative paths, restoring the ability to use `@/` internal links in markdown files.
* Fixed taxonomy term slugs to correctly use original config patterns during interpolation instead of stripped URLs.
* Updated asset compilation so site-level static files correctly override theme static files without destroying generated pipeline outputs.
* Resolved various codebase warnings found by `golangci-lint` (errcheck, staticcheck, ineffassign).

## Developer Notes
* Added the `lint` target in `Makefile`.

## [0.5.8] - 2026-06-30

### Changed
* nida default port is now 1702
* depndencies updates



## [0.5.7] - 2026-05-17

### Added

* Generate the docs release notes page from `CHANGELOG.md` during docs builds.
* Add `nida init [PATH]` to create a buildable example site with sample posts, tags, feeds, search, and page-bundle content in a new or empty directory.
* Add documented search support with a generated index format reference, a header search UI in the Nida docs, and a scaffolded client-side search example for `nida init`.
* Add asset URL template helpers, responsive image helpers, and default `thumb`, `content`, and `hero` image presets for the asset pipeline.
* Add opt-in content diagnostics that fail builds on broken `@/` internal links and missing Markdown image assets.

### Changed

* Change the default `nida serve` port and bundled site configs to `2906`.
* Include page and section descriptions in generated search index documents and unescape HTML entities in indexed body text.
* Improve asset manifest rewriting so fingerprinted CSS, JS, font, image, and `srcset` URLs preserve configured base paths.

## Fixed 
- `readFile` now allows paths under content_dir as well as templates/static. Otherwise, it was impossible to read files from page bundles or sections, which limited the usefulness of the function for content-driven generation.

## [0.5.6] - 2026-05-17

### Added

* Add configurable RSS and Atom section filtering with `sections = ["post"]`, preserving all-page feeds when omitted or empty and applying feed limits after filtering.

## [0.5.5] - 2026-05-16

### Added

* Add a Nida-built documentation site under `docs/` and a GitHub Pages deployment workflow.
* Derive and expose a template `.BasePath` from `base_url` for project-hosted sites such as GitHub Pages.
* Teach `nida serve` to serve local previews from the configured base path.

## [0.5.2] - 2026-05-16

### Security

* Escape fallback built-in `404.html` values in their correct HTML contexts.
* Validate custom slugs, aliases, generated routes, and taxonomy permalink routes to reject unsafe path segments, control characters, query strings, fragments, and backslashes.
* Reject symlinked SCSS input and output paths before invoking the external `sass` compiler.
* Restrict template `readFile` to template/static roots and refuse hidden path segments.

### Fixed

* Use JavaScript-string-safe quoting for redirect page targets.

### Changed

* Add clearer `unsafeHTML` and `unsafeCSS` template helper aliases while keeping `safeHTML` and `safeCSS` for compatibility.
* Document `markdown.unsafe_html` as an explicit trusted-author opt-in feature.

## [0.5.1] - 2026-05-16

### Added

* Add colorized `nida build` and `nida serve` terminal output for easier-to-scan logs, while keeping redirected output plain and respecting `NO_COLOR` and `TERM=dumb`.

### Fixed

* Fix live static asset synchronization so newly created files under `static/` are copied during `nida serve` rebuilds.
* Treat non-Markdown changes under `content/` as full rebuild triggers so page bundle resource additions and deletions refresh correctly.
* Allow theme SCSS to compile even when a site does not have a `static/` directory.
* Return the real fallback `static/site.css` read error when inline CSS loading fails.
* Make `make build` use the public Go proxy by default instead of a machine-specific local module cache.

### Changed

* Consolidate reading-time calculation on the shared content helper.
* Remove stray local debug command code and unused internal helpers.

## [0.5.0] - 2026-05-10

### Security

* Harden output path handling so configured directories and generated artifacts cannot escape the site/output root
* Disable unsafe Markdown HTML by default; set `markdown.unsafe_html = true` to preserve trusted raw HTML behavior
* Sanitize Markdown link and image URL schemes to block dangerous protocols
* Restrict template `readFile` and CSS include helpers to paths under the site tree
* Refuse symlinked site input/output paths in static assets, bundles, templates, generated output, and the dev server
* Add image processing size/dimension limits and local dev server header timeouts

## [0.4.6] - 2026-05-09

### Added

* `aliases` front matter support: pages can now declare `aliases = ["/old-url/"]` and Nida generates HTML redirect pages with JavaScript and `<noscript>` meta-refresh fallback
* Taxonomy pagination: taxonomy term pages now respect `paginate_by` and `paginate_path` config fields
* Automatic `page/1/` redirect generation for paginated sections and taxonomy terms
* Template functions: `groupByYear`, `now`, `readFile`, `resizeImage`, `dig`, `hasPrefix`, `contains`, `lower`, `trimSpace`, `sortDesc`, `hasSuffix`
* Search index generation (`internal/searchindex/searchindex.go`) for local JavaScript search
* Image resize support with WebP output (`internal/templates/images.go`)
* `CurrentURL` available in template context
* `PrevURL`, `NextURL`, `PrevTitle`, `NextTitle` on `content.Page` for prev/next navigation

### Fixed

* Fixed root-level pages producing double-slash URLs (e.g. `//search/`) when `{section}` placeholder resolved to empty string
* Fixed template names with `.html` suffix not resolving correctly in front matter
* Fixed `DeriveSlug` incorrectly stripping content after dots in names like `"M. Elashri"`
* Fixed non-ASCII character transliteration in slugs (ϕ, φ, ℓ now map to `ph`, `l`)
* Fixed taxonomy term pages rendering all items on a single page regardless of `paginate_by`
* Fixed release build failure by removing CGO-dependent `chai2010/webp` dependency; `resizeImage` now outputs JPEG instead of WebP

### Changed

* Enabled `robots.txt` generation by default

## [0.4.5] - 2026-05-06

### Added
* Add page bundle feature with resource management and tests

## [0.4.4] - 2026-05-06

### Fixed 
* Filter out draft pages when building the site index because it was excluded only from taxonomies and categories.

### Changed
* Bump github.com/alecthomas/chroma/v2 from 2.23.1 to 2.24.1
* Bump github.com/pelletier/go-toml/v2 from 2.3.0 to 2.3.1


## [0.4.3] - 2026-05-05

### Added
* Enable class-based formatting in Chroma and add ChromaCSS function for theme styling
* Add support for denced code to detect collapse markers and pass them to chroma
* Add support for denced code to detect collapse markers and pass them to chroma
* Add support for Zola-style permalink patterns to nida
* Add automatic path resolution for internal links in nida

## [0.4.1] - 2026-04-28

### Fixed
* Fix a bug in markdown render not rendering footnotes correctly.

## [0.4.0] - 2026-04-28

### Added
* Arbitrary content sections: any directory with `_index.md` is now a section; any other `.md` file is a page
* New `Page` and `Section` content types replace the old `Item` type with `TypePost`/`TypePage`/`TypeSection` constants
* Sections can be nested with parent/child relationships
* `transparent = true` section option promotes pages to parent section
* Per-section `page_template` and `sort_by` front matter settings
* Generalized taxonomy system: users can define arbitrary taxonomies beyond just tags and categories
* `nida migrate` command for upgrading v0.3.x config files to v0.4 format


### Changed
* **Breaking**: `posts_dir` and `pages_dir` config fields removed; directory structure now determines section layout
* **Breaking**: Taxonomies changed from `[taxonomies]` struct with `tags = true, categories = true` to `[[taxonomies]]` array with `name`, `path`, `render`, `paginate_by` fields
* **Breaking**: `SiteIndex.Posts`, `.Pages`, `.RecentPosts`, `.TagMap`, `.CategoryMap` removed and replaced with `TaxonomyMap` and section-based page organization
* Homepage is now the root section rendered with the `index` template (no special-case `renderHomePage`)
* Config version tracking via `config_version = "0.4"` field
* RSS/Atom feed generation now uses canonical URLs directly instead of `CanonicalLookup`

### Migration from v0.3.x

If you have an existing v0.3.x site, run `nida migrate` in your site root to upgrade `config.toml` to v0.4 format. The command:
- Creates a backup at `config.toml.bk`
- Converts `posts_dir`/`pages_dir` to section-based structure
- Migrates taxonomy config to the new `[[taxonomies]]` format
- Updates permalink patterns to the new structure

The `nida migrate` command is temporary and will be removed in a future release after the migration window closes.

## [0.3.3] - 2026-04-26

### Added
* Add Asset pipeline	Image resizing
* Add SCSS compilation and fingerprinting
* Add lazy-loading support for images

### Changed
* Change the default port for `nida serve` to `2906`.


## [0.3.2] - 2026-04-24

### Fixed
* Improve path skipping logic to exclude VCS directories like `.git` and `.svn` from content discovery and incremental rebuilds, preventing unnecessary processing and potential errors when such directories are present in the content tree.
* Refactor minifyHTML function to extract <pre> blocks and preserve whitespace, improving HTML minification logic

## [0.3.1] - 2026-04-23

### Added

* Added reading time estimation mechanism and related tests.
* Improved template function map with arithmetic operations.
* Added external live reload script handling and corresponding tests.

## [0.3.0] - 2026-04-23

### Added

* Atom feed generation with a new `[atom]` config section.
* Multi-feed output support so sites can publish RSS and Atom feeds together.
* `rawhtml` shortcode support for preserving raw HTML blocks imported from Zola-style content.
* `details` shortcode support for collapsible Markdown-backed detail blocks.
* Markdown external link options compatible with Zola-style settings:
  * `external_links_target_blank`
  * `external_links_no_follow`
  * `external_links_no_referrer`
* Optional generated `robots.txt` output with a new `[robots]` config section.
* Optional HTML minification with `minify_html = true`.
* Basic YAML front matter support for simple `key: value` metadata.
* Template helper support for joining string slices and list-like `extra` values.
* `/page/1/` section aliases for paginated sections to better match Zola route output.

### Changed

* Atom entries now include rendered HTML content, per-entry author metadata, and published timestamps.
* Incremental rebuilds now update all enabled feed artifacts and generated `robots.txt`.
* Markdown rendering now applies configured external-link attributes only to absolute HTTP(S) links.

### Fixed

* Nida can now import content files with a leading blank line before front matter.
* Nida can now build content that uses the Zola-style `rawhtml` and `details` shortcodes without leaking shortcode markers into output.
* Generated route output now matches Zola-style paginated section first-page aliases.

## [0.2.0] - 2026-04-23

### Changed

* Template files now use the standard `.html` extension instead of `.tmpl`.
* The bundled example sites were renamed to use `templates/*.html`.
* Documentation references for example templates and custom `404` templates now use `.html`.

### Added

* Homebrew tap release automation for `MohamedElashri/homebrew-nida`.
* A maintainer release guide in `docs/release.md`.
* Release preflight coverage for the Arabic example site.

### Fixed

* Homebrew formula rendering now separates the GitHub release tag, such as `v0.2.0`, from the archive/package version, such as `0.2.0`.
* Manual release workflow runs now build snapshots instead of attempting to publish a non-tagged release.
* Release tags are validated before publishing.

### Migration Notes

Rename custom template files from `.tmpl` to `.html`, for example:

```text
templates/base.tmpl -> templates/base.html
templates/post.tmpl -> templates/post.html
templates/page.tmpl -> templates/page.html
templates/404.tmpl -> templates/404.html
```

Template names inside files do not change. For example, `post.html` should still define `{{ define "post" }}`.

## [0.1.0] - 2026-04-13

### Added

* Initial release of Nida.
* `nida build`, `nida serve`, and `nida version`.
* Config loading and validation.
* Markdown content discovery and rendering.
* Posts, pages, sections, tags, and categories.
* RSS feed and sitemap generation.
* Static asset copying and output writing.
* GitHub Releases packaging with GoReleaser.

[0.5.1]: https://github.com/MohamedElashri/nida/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/MohamedElashri/nida/compare/v0.4.6...v0.5.0
[0.4.6]: https://github.com/MohamedElashri/nida/compare/v0.4.5...v0.4.6
[0.4.5]: https://github.com/MohamedElashri/nida/compare/v0.4.4...v0.4.5
[0.4.4]: https://github.com/MohamedElashri/nida/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/MohamedElashri/nida/compare/v0.4.1...v0.4.3
[0.4.1]: https://github.com/MohamedElashri/nida/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/MohamedElashri/nida/compare/v0.3.3...v0.4.0
[0.3.3]: https://github.com/MohamedElashri/nida/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/MohamedElashri/nida/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/MohamedElashri/nida/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/MohamedElashri/nida/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/MohamedElashri/nida/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/MohamedElashri/nida/releases/tag/v0.1.0
