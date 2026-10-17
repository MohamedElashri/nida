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
| Theme authoring guide | Shipped in 0.7.0 | [Themes](@/guide/themes.md), with a minimal theme and the `nida-docs` theme as examples. |
| Migration guides | Shipped in 0.7.0 | [Migrating](@/guide/migrating.md) from Zola, Hugo, and Jekyll. |
| Starter themes | Next | A small blog theme that `nida init` can use. |
| Migration diagnostics | Next | Warnings for unsupported front matter and shortcode patterns from other generators. |
| Versioned documentation | Later | Docs for multiple release lines. |
| Deploy presets | Later | Less setup for common static hosts. |

## Current focus

### Starter themes

`nida init` writes plain templates into the site. A small, maintained blog theme would give new sites a better default while still showing the override rules from the [Themes](@/guide/themes.md) guide.

### Migration diagnostics

`nida check` already warns about unknown config keys. The next step is warning about content patterns from other generators that Nida does not support, such as Zola function-call shortcodes or Hugo `{{%` shortcodes, so a migration shows a clear to-do list.

## Not planned

Some features fit general-purpose site frameworks better than a focused publishing tool: plugins or a scripting runtime, JavaScript bundling, arbitrary content collections, full multilingual content trees, and alternative template languages.

## Product principles

Nida features should be useful for real static sites, easy to explain, friendly to static output, and small enough to keep the tool understandable.

When a feature makes Nida more powerful but harder to reason about, it should earn its place through clear examples, tests, and documentation.
