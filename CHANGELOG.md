# Changelog

All notable changes to Nida are documented here.

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
* Change the default port for `nida serve` to `1307`.


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

[0.3.0]: https://github.com/MohamedElashri/nida/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/MohamedElashri/nida/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/MohamedElashri/nida/releases/tag/v0.1.0
