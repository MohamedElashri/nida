+++
title = "Nida Roadmap"
description = "A practical view of what is shipped, what is next, and what is being considered."
weight = 90
template = "page"
+++

Nida's roadmap is intentionally focused. The project is not trying to become a large website framework; it is trying to make static publishing feel complete, predictable, and pleasant.

This roadmap is directional, not a release contract. Items move when real usage, maintenance cost, or compatibility work changes the order.

## Roadmap at a glance

| Area | Status | Outcome |
| --- | --- | --- |
| Correctness and config warnings | Shipped in 0.7.0 | Docs, scaffold, and engine agree; unknown keys warn. |
| Authoring | Shipped in 0.7.0 | Summaries, TOC, callouts, code options, math, Mermaid. |
| Shortcodes and theme inheritance | Shipped in 0.7.0 | Template shortcodes and a consistent override order. |
| Blog features | Shipped in 0.7.0 | Related posts, series, SEO partial, scoped feeds, `nida new`. |
| Docs features | Shipped in 0.7.0 | Generated navigation, breadcrumbs, edit links, data files, translations, `nida check`. |
| Theme authoring guide | Next | A walkthrough for building and sharing a small theme. |
| Migration guides | Next | Step-by-step notes for Zola, Hugo, and Jekyll sites. |
| Versioned documentation | Later | Docs for multiple release lines. |
| Deploy presets | Later | Less setup for common static hosts. |

## Current focus

### Theme authoring

Theme inheritance now applies to templates, partials, shortcodes, static files, SCSS, data, and translations. The next step is a guide and a small starter theme that show the pieces together.

### Migration support

Nida now reads full YAML front matter, Zola `[taxonomies]` tables, and Zola's `details` and `rawhtml` shortcodes, and it warns about config keys it does not understand. Guides for moving a Zola, Hugo, or Jekyll site should build on those warnings.

## Not planned

Some features fit general-purpose site frameworks better than a focused publishing tool: plugins or a scripting runtime, JavaScript bundling, arbitrary content collections, full multilingual content trees, and alternative template languages.

## Product principles

Nida features should be useful for real static sites, easy to explain, friendly to static output, and small enough to keep the tool understandable.

When a feature makes Nida more powerful but harder to reason about, it should earn its place through clear examples, tests, and documentation.
