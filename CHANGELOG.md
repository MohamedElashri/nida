# Changelog

All notable changes to Nida are documented here.


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
