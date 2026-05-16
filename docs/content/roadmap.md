+++
title = "Roadmap"
description = "Where Nida is likely to grow next."
weight = 90
template = "page"
+++

Nida's roadmap is intentionally small. The goal is not to become a giant website framework; it is to make static publishing feel complete, predictable, and pleasant.

These are the product areas that would make Nida more useful while keeping it understandable.

## 1. Stronger asset pipeline

Nida already supports static asset copying, optional fingerprinting, SCSS, CSS/JS minification, and image processing. The next step is making the pipeline more complete and easier to reason about.

Planned direction:

- clearer asset URL helpers for templates
- better fingerprint manifest documentation
- more predictable handling for processed images
- practical presets for common image sizes
- stronger examples for CSS, JS, fonts, and image assets

Why it matters: static sites often become asset-management projects. Nida should make cache-friendly, deployable assets feel boring in the best way.

## 2. Better theme authoring

Nida supports themes and theme inheritance, but creating a theme should feel more guided.

Planned direction:

- documented minimal theme structure
- starter themes for common site types
- clearer template override rules
- better examples for menus, metadata, and `extra` values
- theme-friendly asset conventions

Why it matters: themes should help people start quickly without hiding how the site is built.

## 3. Built-in search experience

Nida already has optional search index generation. The next step is making search practical out of the box.

Planned direction:

- documented search index format
- small default search UI example
- template pattern for loading the generated index
- configuration examples for enabling and disabling search
- guidance for keeping search lightweight

Why it matters: static sites should be searchable without requiring a heavy application framework.

## 4. Migration and compatibility tools

Nida already includes ideas inspired by Zola, such as front matter, sections, permalinks, and internal links. Migration support could make it easier to try Nida on existing sites.

Planned direction:

- clearer migration guides from Zola-style sites
- content checks for unsupported front matter or shortcode patterns
- import diagnostics that explain what needs manual attention
- compatibility notes for Markdown, permalinks, taxonomies, and aliases

Why it matters: people are more likely to try Nida if they can understand the migration cost before committing.

## Later, maybe

These ideas are useful but not urgent:

- versioned content or documentation support
- richer taxonomy pages
- configurable generated archives
- deploy presets for common hosts

## How priorities are chosen

Nida should prefer features that are:

- useful for real static sites
- easy to explain
- static-output friendly
- template-friendly
- small enough to keep the tool understandable

If a feature makes Nida more powerful but much harder to reason about, it should earn its place slowly.
