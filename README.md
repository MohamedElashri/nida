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
* RTL document support for languages such as Arabic/Persian

## Install

### Homebrew

```bash
brew tap MohamedElashri/nida
brew install nida
```

### Prebuilt Binary

Download a release archive from
[GitHub Releases](https://github.com/MohamedElashri/nida/releases).

For shell installs, resolve the latest release tag first:

```bash
TAG=$(curl -fsSL https://api.github.com/repos/MohamedElashri/nida/releases/latest | sed -n 's/.*"tag_name": "\(v[^"]*\)".*/\1/p')
VERSION=${TAG#v}
```

or choose a specific release tag:

```bashTAG=v0.2.0
VERSION=${TAG#v}
```

Then download and install the appropriate archive for your platform. For example,

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
$TAG = (Invoke-RestMethod "https://api.github.com/repos/MohamedElashri/nida/releases/latest").tag_name
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
http://127.0.0.1:1307
```

Commands:

```bash
nida build [-s PATH] [--site PATH] [-c PATH] [--config PATH] [-d] [--drafts]
nida serve [-s PATH] [--site PATH] [-c PATH] [--config PATH] [-d] [--drafts] [-p PORT] [--port PORT]
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

## Name Meaning
Nida (نداء) is Arabic for "call" or "summons". It reflects the idea of a blog as a call to share thoughts and ideas with the world. The name also has a nice ring to it and is easy to remember.
