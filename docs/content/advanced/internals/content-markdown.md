+++
title = "Content And Markdown"
description = "How Markdown files become pages, sections, bundles, and rendered HTML."
weight = 30
template = "page"
+++

Content loading is split between `internal/content`, `internal/frontmatter`, and `internal/markdown`.

## Discovery Rules

`content.Discover` walks the configured content directory and classifies Markdown files:

- `_index.md` creates a section
- `index.md` inside a subdirectory creates a page bundle
- any other `.md` file creates a regular page

Symlinked files are ignored during discovery. Page bundles collect non-Markdown, non-directory, non-symlink files beside `index.md` as bundle resources.

If a directory contains pages but no `_index.md`, Nida synthesizes an implicit section. That keeps rendering predictable because pages can always refer to a section object.

## Page Fields

A loaded `content.Page` includes source path, relative path, section path, body Markdown, front matter fields, slug, dates, draft state, template override, aliases, and arbitrary `Extra`.

Bundle pages also include:

- `BundleDir`
- `IsBundle`
- `Resources`

The route URL is not assigned during discovery. Routing belongs to `internal/site`.

## Section Fields

A loaded `content.Section` includes source path, section path, body Markdown, title, template overrides, pagination settings, sorting settings, transparency, feed flag, and `Extra`.

Implicit sections have no source path. They derive a human-readable title and slug from the directory name.

## Front Matter

`internal/frontmatter` parses TOML front matter into `frontmatter.Metadata`. Known fields become typed values. Unknown front matter values are preserved in `Extra`.

Taxonomies are intentionally read from `Extra`: if a configured taxonomy is named `tags`, page metadata with `tags = [...]` is available to taxonomy building through `page.Extra["tags"]`.

## Slugs

If a page slug is not provided, Nida derives it from the filename or bundle directory. `content.DeriveSlug` lowercases input, keeps ASCII letters and numbers, keeps Unicode letters and numbers, turns separators into hyphens, trims extra hyphens, and has a small transliteration table for known scientific characters.

Route validation later rejects unsafe generated routes, so slug changes should be tested through both content and site packages.

## Markdown Rendering

`internal/markdown` uses Goldmark with:

- GFM
- footnotes
- auto heading IDs
- hard wraps
- custom fenced code rendering
- custom link rendering
- custom image rendering

Raw HTML is disabled by default. It is enabled only when `markdown.unsafe_html = true`.

## Internal Links

Nida supports Zola-style internal links such as:

```markdown
[Read more](@/posts/hello.md)
```

`site.Load` builds a path lookup before Markdown rendering. Link and image renderers call `markdown.ResolveInternalPath`; if a path is known, it becomes the routed URL. If not, the original value is left alone.

## URL Safety

Markdown links are sanitized before rendering:

- empty or control-character links become `#`
- protocol-relative links become `#`
- `http`, `https`, `mailto`, and `tel` are allowed for links
- images allow safe relative, root-relative, and HTTP(S) URLs
- unsupported image schemes become empty image sources

External link attributes are controlled by `[markdown]` settings.

## Code Blocks

Fenced code blocks are rendered by `internal/highlight` using the configured syntax theme.

A code block can start with:

```text
[!COLLAPSE] Show code | Hide code
```

That marker wraps the highlighted code in collapsible HTML and removes the marker from the code content.

## Adding Content Behavior

When changing content discovery, test regular pages, sections, bundle pages, draft filtering, and implicit sections. When changing Markdown rendering, add tests for unsafe inputs as well as normal authoring cases.
