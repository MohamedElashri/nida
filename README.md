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
* serves a local development site with watch mode and livereload
* supports RTL document rendering for languages like Arabic

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

## Install

### Install A Prebuilt Binary From Releases

Set `VERSION` to the release tag you want to install, for example:

macOS or Linux:

```bash
VERSION=<latest-tag>
```

Windows PowerShell:

```powershell
$VERSION = "<latest-tag>"
```

<details>
<summary>Linux x86_64</summary>

```bash
curl -L -o nida.tar.gz "https://github.com/MohamedElashri/nida/releases/download/${VERSION}/nida_${VERSION}_linux_x86_64.tar.gz"
```

</details>

<details>
<summary>Linux arm64</summary>

```bash
curl -L -o nida.tar.gz "https://github.com/MohamedElashri/nida/releases/download/${VERSION}/nida_${VERSION}_linux_arm64.tar.gz"
```

</details>

<details>
<summary>macOS Intel</summary>

```bash
curl -L -o nida.tar.gz "https://github.com/MohamedElashri/nida/releases/download/${VERSION}/nida_${VERSION}_darwin_x86_64.tar.gz"
```

</details>

<details>
<summary>macOS Apple Silicon</summary>

```bash
curl -L -o nida.tar.gz "https://github.com/MohamedElashri/nida/releases/download/${VERSION}/nida_${VERSION}_darwin_arm64.tar.gz"
```

</details>

<details>
<summary>Windows x86_64</summary>

```powershell
Invoke-WebRequest -Uri "https://github.com/MohamedElashri/nida/releases/download/$VERSION/nida_${VERSION}_windows_x86_64.zip" -OutFile "nida.zip"
```

</details>

Extract and install:

macOS or Linux:

```bash
tar -xzf nida.tar.gz
chmod +x nida
sudo mv nida /usr/local/bin/nida
nida version
```

Windows PowerShell:

```powershell
Expand-Archive -Path "nida.zip" -DestinationPath ".\\nida"
.\\nida\\nida.exe version
```

You can then move `nida.exe` into a directory that is already on your `PATH`.

### Build It Yourself

Build from source with Go:

```bash
git clone https://github.com/MohamedElashri/nida.git
cd nida
go build ./cmd/nida
./nida version
```

Or install it into your Go bin directory:

```bash
go install github.com/MohamedElashri/nida/cmd/nida@latest
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

A complete example project lives in [example-site](./example-site).
An Arabic RTL example also lives in [example-site-ar](./example-site-ar).

Useful files:

* config: [example-site/config.toml](./example-site/config.toml:1)
* content: [example-site/content/posts/launching-nida.md](./example-site/content/posts/launching-nida.md:1)
* templates: [example-site/templates/base.tmpl](./example-site/templates/base.tmpl:1)
* custom 404 template: [example-site/templates/404.tmpl](./example-site/templates/404.tmpl:1)
* styles: [example-site/static/site.css](./example-site/static/site.css:1)
* Arabic example config: [example-site-ar/config.toml](./example-site-ar/config.toml:1)

Optional theme template:

* add `templates/404.tmpl` to customize the generated `/404.html`
* if no `404.tmpl` exists, `nida` emits a built-in fallback page automatically

RTL support:

* set `language = "ar"` in `config.toml` for an Arabic site
* theme templates can use `{{ documentDirection .Config.Language }}` to switch between `ltr` and `rtl`

## Development

The repository includes a `Makefile`:

* `make build`
* `make test`
* `make site-build`
* `make serve`
* `make example-build`
* `make example-serve`
* `make arabic-example-build`
* `make arabic-example-serve`
* `make check`
* `make clean`

## Releases

Tagged releases are built from Git tags matching `v*` with GitHub Actions and GoReleaser.

Release artifacts include:

* the `nida` binary for each supported platform
* `README.md`
* `LICENSE`
* `checksums.txt`

The bundled example sites are used for release verification, but they are not packaged into binary archives.

## Notes

Current behavior:

* watch mode uses native filesystem events on Linux and macOS, with polling fallback where event watching is unavailable
* serve mode updates output incrementally: asset-only changes sync assets, while content/template/config changes rewrite only the outputs that changed
* `server.livereload` is implemented for local development and auto-refreshes the browser after successful rebuilds

## License

MIT. See [LICENSE](./LICENSE).
