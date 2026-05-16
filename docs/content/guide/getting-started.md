+++
title = "Getting Started"
description = "Install Nida, build the example site, and learn the basic workflow."
weight = 10
template = "page"
+++

Nida keeps the publishing loop intentionally small: write Markdown, render templates, and serve the generated static files.

## Install with curl

Prebuilt archives are published on [GitHub Releases](https://github.com/MohamedElashri/nida/releases).


On Linux or macOS, install the latest prebuilt release:

```bash
curl -fsSL https://melashri.net/nida/install.sh | sh
nida version
```

The installer detects the operating system and CPU architecture, downloads the matching release archive into a temporary directory, verifies the checksum, and installs `nida` into `/usr/local/bin`.

To install somewhere else:

```bash
curl -fsSL https://melashri.net/nida/install.sh | NIDA_INSTALL_DIR="$HOME/.local/bin" sh
```

To install a specific version:

```bash
curl -fsSL https://melashri.net/nida/install.sh | NIDA_VERSION=v0.5.1 sh
```

{% details(summary="Manual prebuilt archive install") %}


Resolve the latest release tag:

```bash
TAG=$(curl -fsSL https://api.github.com/repos/MohamedElashri/nida/releases/latest | sed -n 's/.*"tag_name": "\(v[^"]*\)".*/\1/p')
VERSION=${TAG#v}
```

Download and install the Linux x86_64 archive:

```bash
curl -L -o nida.tar.gz "https://github.com/MohamedElashri/nida/releases/download/${TAG}/nida_${VERSION}_linux_x86_64.tar.gz"
tar -xzf nida.tar.gz
chmod +x nida
sudo mv nida /usr/local/bin/nida
nida version
```

For macOS Apple Silicon, use the `darwin_arm64` archive name instead. Windows users can download the `windows_x86_64.zip` archive from the same release page.

{% end %}

## Install with Homebrew

On macOS or Linux with Homebrew:

```bash
brew tap MohamedElashri/nida
brew install nida
nida version
```


## Install from source

```bash
go install github.com/MohamedElashri/nida/cmd/nida@latest
```

Or build the binary from a local checkout:

```bash
go build ./cmd/nida
```

## Build a site

Run Nida against a site directory:

```bash
nida build --site ./example-site
```

During development, use the local server:

```bash
nida serve --site ./example-site
```

The server builds the site, watches for changes, and serves the configured output directory.

## Site shape

A minimal Nida site usually looks like this:

```text
my-site/
  config.toml
  content/
  templates/
  static/
  public/
```

The source files live in `content/`, `templates/`, and `static/`. The generated site goes to `public/` by default.

Next: learn how to configure a site in [Configuration](../configuration/).
