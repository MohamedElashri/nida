# Nida

Nida is a small Go static site generator for blogs and personal sites. It reads
Markdown with TOML front matter, renders pages with Go HTML templates, and writes
a static site you can serve anywhere.

## Features

* posts and standalone pages
* Go template themes using `.html` files
* tags and categories
* RSS and sitemap generation
* static asset copying
* local development server with watch mode and livereload
* optional custom `404.html`
* RTL document support for languages such as Arabic

## Install

### Homebrew

```bash
brew tap MohamedElashri/nida
brew install nida
```

### Prebuilt Binary

Download a release archive from
[GitHub Releases](https://github.com/MohamedElashri/nida/releases).

For shell installs, set the tag and package version separately:

```bash
TAG=v0.2.0 # replace with the latest release tag
VERSION=${TAG#v}
```

Linux x86_64:

```bash
curl -L -o nida.tar.gz "https://github.com/MohamedElashri/nida/releases/download/${TAG}/nida_${VERSION}_linux_x86_64.tar.gz"
tar -xzf nida.tar.gz
chmod +x nida
sudo mv nida /usr/local/bin/nida
nida version
```

macOS Apple Silicon:

```bash
curl -L -o nida.tar.gz "https://github.com/MohamedElashri/nida/releases/download/${TAG}/nida_${VERSION}_darwin_arm64.tar.gz"
tar -xzf nida.tar.gz
chmod +x nida
sudo mv nida /usr/local/bin/nida
nida version
```

Other archives are published for Linux arm64, macOS Intel, and Windows x86_64.

Windows PowerShell:

```powershell
$TAG = "v0.2.0"
$VERSION = $TAG.TrimStart("v")
Invoke-WebRequest -Uri "https://github.com/MohamedElashri/nida/releases/download/${TAG}/nida_${VERSION}_windows_x86_64.zip" -OutFile "nida.zip"
Expand-Archive -Path "nida.zip" -DestinationPath ".\nida"
.\nida\nida.exe version
```

### Go

```bash
go install github.com/MohamedElashri/nida/cmd/nida@latest
```

Or build from source:

```bash
git clone https://github.com/MohamedElashri/nida.git
cd nida
go build ./cmd/nida
./nida version
```

## Usage

Build a site:

```bash
nida build --site ./example-site
```

Serve it locally:

```bash
nida serve --site ./example-site
```

The default local address is:

```text
http://127.0.0.1:2906
```

Commands:

```bash
nida build [--site PATH] [--config PATH] [--drafts]
nida serve [--site PATH] [--config PATH] [--drafts] [--port PORT]
nida version
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

Templates live in `templates/` and use `.html` filenames:

```text
templates/base.html
templates/index.html
templates/post.html
templates/page.html
templates/404.html
```

The filename stem is the template name. For example, `post.html` should define
`{{ define "post" }}`.

## Examples

The repository includes two example sites:

* [example-site](./example-site): English example blog
* [example-site-ar](./example-site-ar): Arabic RTL example blog

Useful starting points:

* [example-site/config.toml](./example-site/config.toml)
* [example-site/content/posts/launching-nida.md](./example-site/content/posts/launching-nida.md)
* [example-site/templates/base.html](./example-site/templates/base.html)
* [example-site/static/site.css](./example-site/static/site.css)

## Documentation

* [Development](./docs/dev.md)
* [Release process](./docs/release.md)
* [Changelog](./CHANGELOG.md)

## License

MIT. See [LICENSE](./LICENSE).
