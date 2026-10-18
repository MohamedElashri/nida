# Nida

Nida is a small Go static site generator for blogs, personal sites, and
documentation. It reads Markdown with TOML or YAML front matter, renders pages
with Go HTML templates, and writes a static site you can serve anywhere.

## Features

* posts, standalone pages, nested sections, and page bundles
* Markdown with summaries, tables of contents, heading anchors, callouts,
  code block titles and line highlighting, math, and Mermaid diagrams
* shortcodes from templates, with built-in `figure`, `youtube`, and `callout`
* Go template themes with inheritance, partials, shortcodes, and data files
* taxonomies, related posts, series, and archive helpers
* Atom and RSS feeds, including per-section and per-tag feeds
* SEO partial with Open Graph, Twitter cards, and JSON-LD
* sitemap, robots.txt, and a client-side search index
* image dimensions and responsive `srcset` for Markdown images
* `nida check` for broken links, template errors, and config typos
* `nida new` with archetypes
* local development server with watch mode and livereload
* translations for theme UI strings and RTL support for languages such as Arabic and Persian

## Install

```bash
curl -fsSL https://melashri.net/nida/install.sh | sh   # Linux and macOS
brew install MohamedElashri/nida/nida                   # Homebrew
go install github.com/MohamedElashri/nida/cmd/nida@latest
```

Prebuilt archives for Linux, macOS, and Windows are on
[GitHub Releases](https://github.com/MohamedElashri/nida/releases). See
[Getting Started](https://melashri.net/nida/guide/getting-started/) for all
install options.

## Quick start

```bash
nida init ./my-site                        # create an example site
nida serve --site ./my-site                # preview at http://127.0.0.1:1702
nida new --site ./my-site posts/hello.md   # add a post
nida check --site ./my-site                # find broken links and errors
nida build --site ./my-site                # write the site to my-site/public
```

## Documentation

* [Getting Started](https://melashri.net/nida/guide/getting-started/)
* [Writing Content](https://melashri.net/nida/guide/writing/)
* [Templates](https://melashri.net/nida/guide/templates/) and [Themes](https://melashri.net/nida/guide/themes/)
* [Migrating from Zola, Hugo, or Jekyll](https://melashri.net/nida/guide/migrating/)
* [CLI Reference](https://melashri.net/nida/reference/cli/) and [Config Reference](https://melashri.net/nida/reference/config/)
* [Changelog](./CHANGELOG.md)

## Examples

* [example-site](./example-site): English example blog
* [example-site-ar](./example-site-ar): Arabic RTL example blog
* [docs](./docs): the documentation site, built with its own theme in `docs/themes/nida-docs`

## License

MIT. See [LICENSE](./LICENSE).

## Name Meaning
Nida (نداء) is Arabic for "call" or "summons". It reflects the idea of a blog as a call to share thoughts and ideas with the world. The name also has a nice ring to it and is easy to remember.
