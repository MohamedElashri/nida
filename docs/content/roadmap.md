+++
title = "Nida Roadmap"
description = "Where Nida is likely to grow next."
weight = 90
template = "page"
+++

Nida's roadmap is intentionally small. The goal is not to become a giant website framework; it is to make static publishing feel complete, predictable, and pleasant.

Near-term work is grouped into a few areas.

## Asset pipeline

Nida already copies static files, fingerprints assets, compiles SCSS, minifies CSS/JS, and can process images. The next step is making those pieces easier to use together.

Planned:

- clearer asset URL helpers for templates
- better manifest docs
- practical image presets
- stronger CSS, JS, font, and image examples

## Theme authoring

Themes work today, including inheritance and site overrides. They need a smoother authoring path.

Planned:

- documented minimal theme structure
- clearer template override rules
- examples for menus, metadata, and `extra`
- theme-friendly asset conventions
- small starter themes

## Search

Nida can generate a search index. The missing piece is a simple documented search UI.

Planned:

- documented search index format
- default client-side search example
- template pattern for loading the index
- guidance for keeping search small

## Migration support

Nida borrows familiar static-site ideas: front matter, sections, permalinks, taxonomies, and `@/` internal links. Migration docs and diagnostics would make trial runs easier.

Planned:

- clearer migration guides from Zola-style sites
- checks for unsupported front matter or shortcode patterns
- compatibility notes for Markdown, permalinks, taxonomies, and aliases

## Later, maybe

- versioned content or documentation support
- richer taxonomy pages
- configurable generated archives
- deploy presets for common hosts

## Principles

Features should be useful for real static sites, easy to explain, friendly to static output, and small enough to keep the tool understandable.

If a feature makes Nida more powerful but harder to reason about, it should earn its place slowly.
