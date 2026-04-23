# Changelog

All notable changes to Nida are documented here.

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
