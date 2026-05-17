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
| Asset pipeline ergonomics | Shipped in 0.5.7 | Easier asset URLs and responsive image presets. |
| Search | Shipped in 0.5.7 | Generated client-side search index. |
| Content diagnostics | Shipped in 0.5.7 | Build checks for broken internal links and missing Markdown image assets. |
| Theme authoring | Next | A clearer path for building, overriding, and sharing small themes. |
| Migration support | Next | Better guidance and diagnostics for trying Nida with existing static sites. |
| Page table of contents | Future | Structured heading data and anchor links for long-form content. |
| Local data files | Future | Reusable TOML, YAML, or JSON data for templates. |
| Publishing workflows | Later | More examples for common hosts and project-hosted sites. |

## Current focus

### Theme authoring

Themes work today, including inheritance and site-level overrides. The next step is making theme creation easier to understand without reading internals.

Planned outcomes:

- Minimal theme structure.
- Clear template override rules.
- Theme patterns for menus, metadata, and assets.
- Small starter themes.

### Migration support

Nida already borrows familiar static-site ideas: front matter, sections, permalinks, taxonomies, aliases, and `@/` internal links. Migration support should make trial runs easier and less mysterious.

Planned outcomes:

- Migration guidance for Zola-style sites.
- Compatibility notes for Markdown, shortcodes, permalinks, and taxonomies.
- Diagnostics for unsupported front matter and shortcode patterns.

## Recently shipped

- Asset pipeline ergonomics: clearer template helpers, responsive image presets, and base-path-aware manifest rewriting.
- Search: generated client-side search indexes for small static sites.
- Content diagnostics: build checks for broken `@/` links and missing Markdown image assets.

## Later candidates

These are useful ideas, but they should wait until the current surface area is well documented and stable.

| Candidate | Why it might matter |
| --- | --- |
| Page table of contents | Improve docs, manuals, and long posts without custom parsing in templates. |
| Local data files | Make reusable structured data available to templates without turning Nida into a CMS. |
| Versioned documentation | Support docs for multiple release lines. |
| Richer taxonomy pages | Better navigation for larger blogs and knowledge bases. |
| Generated archive pages | A simple built-in archive pattern. |
| Deploy presets | Less setup for common static hosts. |

## Product principles

Nida features should be useful for real static sites, easy to explain, friendly to static output, and small enough to keep the tool understandable.

When a feature makes Nida more powerful but harder to reason about, it should earn its place through clear examples, tests, and documentation.
