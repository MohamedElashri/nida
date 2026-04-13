# Nida

`nida` is a small Go static site generator for blogs and personal sites.

It keeps the workflow simple:

* `nida build`
* `nida serve`

## What It Does

* loads site settings from `config.toml`
* reads Markdown content with TOML front matter
* renders posts and pages with Go templates
* supports tags and categories
* generates a default `404.html` page, with optional theme override
* generates `rss.xml`
* generates `sitemap.xml`
* serves a local development site with rebuilds

## Quick Start

Build the binary:

```bash
go build ./cmd/nida
```

Build the bundled example site:

```bash
./nida build --site ./example-site
```

Serve it locally:

```bash
./nida serve --site ./example-site
```

Default local address:

```text
http://127.0.0.1:2906
```

## Commands

```bash
nida build [--site PATH] [--config PATH] [--drafts]
nida serve [--site PATH] [--config PATH] [--drafts] [--port PORT]
```

## Site Layout

```text
site/
├── config.toml
├── content/
│   ├── posts/
│   └── pages/
├── templates/
├── static/
└── public/
```

## Example Site

A complete example project lives in [example-site](/home/melashri/projects/nida/example-site).

Useful files:

* config: [example-site/config.toml](/home/melashri/projects/nida/example-site/config.toml:1)
* content: [example-site/content/posts/launching-nida.md](/home/melashri/projects/nida/example-site/content/posts/launching-nida.md:1)
* templates: [example-site/templates/base.tmpl](/home/melashri/projects/nida/example-site/templates/base.tmpl:1)
* custom 404 template: [example-site/templates/404.tmpl](/home/melashri/projects/nida/example-site/templates/404.tmpl:1)
* styles: [example-site/static/site.css](/home/melashri/projects/nida/example-site/static/site.css:1)

Optional theme template:

* add `templates/404.tmpl` to customize the generated `/404.html`
* if no `404.tmpl` exists, `nida` emits a built-in fallback page automatically

## Development

The repository includes a `Makefile`:

* `make build`
* `make test`
* `make site-build`
* `make serve`
* `make example-build`
* `make example-serve`
* `make check`
* `make clean`

## Notes

Current behavior:

* watch mode uses polling
* rebuilds are full rebuilds
* `livereload` exists in config but is not implemented yet

## License

MIT. See [LICENSE](/home/melashri/projects/nida/LICENSE).
