+++
title = "Config And Path Safety"
description = "How configuration is loaded, normalized, validated, and kept inside the site root."
weight = 20
template = "page"
+++

`internal/config` owns the public configuration contract. Most other packages receive a `config.SiteConfig` and should not reinterpret raw TOML.

## Load Sequence

`config.Load` follows this sequence:

1. resolve the site root
2. choose `config.toml` or the explicit config path
3. read TOML
4. start from `DefaultSiteConfig`
5. decode TOML into the default config
6. normalize values
7. validate values

Starting from defaults means new optional fields should usually be added to `DefaultSiteConfig`, not patched after loading.


## Normalization

`normalize` trims text values and cleans path-like values:

- content, template, static, output, and themes directories
- feed, sitemap, robots, and search filenames
- SCSS entry directory
- theme name and syntax theme
- server host

When adding a config field, decide whether it is display text, a URL, a relative path, or an open-ended value. Only path-like fields should go through path cleaning.

## Validation

`Validate` enforces:

- required `base_url`
- absolute `http` or `https` base URL
- required `title`
- positive pagination
- positive enabled feed limits
- valid server port
- unique non-empty taxonomy names
- safe relative paths for configured directories and filenames
- safe theme name
- safe SCSS entry path when SCSS is enabled

Validation errors are collected and returned together, which is nicer for users editing a config file.

## Safepath Contract

Filesystem-facing packages should use `internal/safepath` when joining configured paths with the site root or output root.

The usual pattern is:

```go
absSiteRoot, err := filepath.Abs(siteRoot)
target, err := safepath.Join(absSiteRoot, cfg.ContentDir)
err = safepath.EnsureNoSymlinkPath(absSiteRoot, target)
```

Use the relevant root for the operation. Content, templates, static files, and themes are checked under the site root. Rendered files and generated artifacts are checked under the output root.

## Adding Config

When adding a config key:

1. add the field to `SiteConfig` or a nested config type
2. add a default when the feature is optional
3. normalize if the value is path-like or whitespace-sensitive
4. validate if bad values can cause confusing output or unsafe IO
5. update `docs/content/reference/config.md`
6. add config tests

If the config changes public behavior, update example sites when useful.
