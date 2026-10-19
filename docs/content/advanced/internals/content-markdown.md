+++
title = "Content And Markdown"
description = "How Markdown files become pages, sections, bundles, and rendered HTML."
weight = 30
template = "page"
+++

Content loading is split between `internal/content`, `internal/frontmatter`, and `internal/markdown`. Shortcode templates live in `internal/shortcodes`, which `internal/markdown` reaches through an interface because the templates themselves need the template function map, and `internal/templates` already imports `internal/markdown` for `markdownify`.

## Discovery Rules

`content.Discover` walks the configured content directory and classifies Markdown files:

- `_index.md` creates a section
- `index.md` inside a subdirectory creates a page bundle
- any other `.md` file creates a regular page

Symlinked files are ignored during discovery. Page bundles collect non-Markdown, non-directory, non-symlink files beside `index.md` as bundle resources.

If a directory contains pages but no `_index.md`, Nida synthesizes an implicit section. That keeps rendering predictable because pages can always refer to a section object.

## Page Fields

A loaded `content.Page` includes source path, relative path, section path, body Markdown, front matter fields, slug, dates, draft state, weight, template override, aliases, and arbitrary `Extra`. Markdown rendering later fills `BodyHTML`, `Summary`, `TOC`, `WordCount`, and `ReadingTime`.

Bundle pages also include:

- `BundleDir`
- `IsBundle`
- `Resources`

The route URL is not assigned during discovery. Routing belongs to `internal/site`.

## Section Fields

A loaded `content.Section` includes source path, section path, body Markdown, title, weight, template overrides, pagination settings, sorting settings, transparency, feed flag, and `Extra`. Section defaults for `paginate_path` and `sort_by` come from the `[sections]` config table.

Implicit sections have no source path. They derive a human-readable title and slug from the directory name.

## Front Matter

`internal/frontmatter` parses TOML (`+++`) or YAML (`---`) front matter into `frontmatter.Metadata`. YAML uses `go.yaml.in/yaml/v3`, so block lists and nested maps work. Known fields become typed values. Unknown top-level values are preserved in `Extra`, and a Zola-style `[taxonomies]` table is flattened into `Extra` so its terms behave like top-level `tags = [...]`.

Taxonomies are intentionally read from `Extra`: if a configured taxonomy is named `tags`, page metadata with `tags = [...]` is available to taxonomy building through `page.Extra["tags"]`.

## Slugs

If a page slug is not provided, Nida derives it from the filename or bundle directory. `content.DeriveSlug` lowercases input, keeps ASCII letters and numbers, keeps Unicode letters and numbers, turns separators into hyphens, trims extra hyphens, and has a small transliteration table for known scientific characters.

Route validation later rejects unsafe generated routes, so slug changes should be tested through both content and site packages.

## Markdown Rendering

`internal/markdown` uses Goldmark with:

- GFM and footnotes
- auto heading IDs from `headingIDs`, which follows Goldmark's rules for ASCII but keeps Unicode letters, so Arabic headings get readable IDs
- heading attributes, for custom IDs such as `## Title {#id}`
- a callout transformer that turns `> [!NOTE]` blockquotes into callout nodes
- math block and inline parsers, always registered so TeX fails the build while `markdown.math` is off
- custom renderers for fenced code, links, images, headings, callouts, and math, which `internal/mathml` converts to MathML with Temml

Hard wraps and raw HTML are both off by default; `markdown.hard_wraps` and `markdown.unsafe_html` turn them on.

Rendering takes a `markdown.Env`:

| Field | Purpose |
| --- | --- |
| `Lookup` | Content paths to URLs, for `@/` links. |
| `Shortcodes` | Shortcode templates (`internal/shortcodes.Set`). |
| `Images` | Image dimensions and srcsets (`site.NewImages`). |
| `Translate` | UI strings, for default callout titles. |
| `Page` | The page being rendered, passed to shortcodes and the image resolver. |

`RenderDocument` returns HTML plus the nested `TOC`. `RenderItem` also splits the summary at `<!-- more -->`, falling back to the first paragraph. `RenderItems` renders pages concurrently with `internal/parallel`.

## Shortcodes

Before Goldmark sees the source, `processShortcodes` replaces each shortcode with a placeholder such as `@@NIDA_SHORTCODE_HTML_0@@`, and `restoreShortcodeHTML` swaps the rendered HTML back in afterwards. That way shortcode output is never escaped or wrapped by Markdown, even with raw HTML disabled.

- `codeRanges` marks fenced code blocks and inline code first, so shortcode syntax inside code is left alone.
- `{{< name >}}` tags are parsed with named and positional arguments. A tag with a matching `{{< /name >}}` is paired; nested tags of the same name are counted.
- A shortcode alone on its lines becomes a block placeholder; otherwise it stays inline.
- Paired bodies are rendered recursively for `.InnerHTML`.
- `{% details(...) %}` blocks and `{{< rawhtml >}}` markers keep their Zola-compatible behavior.
- Heading IDs and TOC titles ignore placeholders, so `## Title {{< since "1.0" >}}` still gets the ID `title`.

## Internal Links

Nida supports Zola-style internal links such as:

```markdown
[Read more](@/posts/hello.md)
```

`site.Load` builds a path lookup before Markdown rendering. Link and image renderers call `resolveWithBasePath`: if a path is known, it becomes the routed URL prefixed with the `base_url` path, so links work on sites published under a subpath. Unknown paths are left alone, and `nida check` reports them.

## URL Safety

Markdown links are sanitized before rendering:

- empty or control-character links become `#`
- protocol-relative links become `#`
- `http`, `https`, `mailto`, and `tel` are allowed for links
- images allow safe relative, root-relative, and HTTP(S) URLs
- unsupported image schemes become empty image sources

External link attributes are controlled by `[markdown]` settings.

## Code Blocks

Fenced code blocks are rendered by `internal/highlight` using the configured syntax theme. `parseCodeInfo` reads options from the info string, such as `go {title="main.go", hl_lines="2-3", linenos=true}`. `hl_lines` count from the first line of the block and are shifted by `linenostart` before they reach Chroma. A `mermaid` block is not highlighted: `internal/diagram` renders it to SVG with Pele, and the SVG is written inside `<div class="diagram">`. A diagram Pele cannot parse is a render error.

Pele and Temml are JavaScript libraries run by `internal/jsengine` in goja, a JavaScript engine written in Go. Each library has a pool of at most four runtimes, started only when a site first needs one, and results are cached by input, so `nida serve` rebuilds only re-render diagrams and formulas that changed. A call that runs longer than 10 seconds is interrupted.

The libraries and goja are pinned in `third_party/`. `third_party/js/package-lock.json` pins Pele and Temml, and Dependabot opens pull requests that bump it; `make vendor-js` then regenerates the embedded scripts, stylesheets, fonts, and version constants, and CI fails until it has been run. goja is a patched copy in `third_party/goja` that drops the collation tables behind `String.prototype.localeCompare`, which neither library calls; a weekly workflow opens an issue when upstream goja moves. See `third_party/README.md`.

A code block can start with:

```text
[!COLLAPSE] Show code | Hide code
```

That marker wraps the highlighted code in collapsible HTML and removes the marker from the code content.

## Adding Content Behavior

When changing content discovery, test regular pages, sections, bundle pages, draft filtering, and implicit sections. When changing Markdown rendering, add tests for unsafe inputs as well as normal authoring cases, and for the same syntax inside code blocks, where it must stay literal.
