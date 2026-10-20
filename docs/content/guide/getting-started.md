+++
title = "Getting Started"
description = "Install Nida, create a site, and learn the write, preview, and publish loop."
weight = 10
template = "page"
+++

Nida keeps the publishing loop small: write Markdown, preview it with live reload, and build static files you can host anywhere. This page takes you from installing Nida to a published site in a few minutes.

## Install

{{< tabs >}}
{{< tab "Install script" >}}
On Linux or macOS, install the latest release into `$HOME/.local/bin`:

```bash
curl -fsSL https://melashri.net/nida/install.sh | sh
```

The script detects your OS and CPU, downloads the matching archive from [GitHub Releases](https://github.com/MohamedElashri/nida/releases), and verifies its checksum. Set `NIDA_INSTALL_DIR` to pick another directory, `NIDA_INSTALL_SYSTEM=1` for a system-wide install, or `NIDA_VERSION=vX.Y.Z` for a specific release:

```bash
curl -fsSL https://melashri.net/nida/install.sh | NIDA_INSTALL_DIR="$HOME/bin" sh
```
{{< /tab >}}
{{< tab "Homebrew" >}}
On macOS or Linux with Homebrew:

```bash
brew tap MohamedElashri/nida
brew install nida
```
{{< /tab >}}
{{< tab "Go" >}}
With Go installed, build and install the latest version. Nida needs a 64-bit platform, such as amd64 or arm64:

```bash
go install github.com/MohamedElashri/nida/cmd/nida@latest
```

Or build from a checkout:

```bash
git clone https://github.com/MohamedElashri/nida.git
cd nida
go build ./cmd/nida
```
{{< /tab >}}
{{< tab "Manual download" >}}
Download an archive for your platform from [GitHub Releases](https://github.com/MohamedElashri/nida/releases). Archives exist for Linux (x86_64, arm64), macOS (Intel, Apple Silicon), and Windows (x86_64):

```bash
TAG=$(curl -fsSL https://api.github.com/repos/MohamedElashri/nida/releases/latest | sed -n 's/.*"tag_name": "\(v[^"]*\)".*/\1/p')
curl -L -o nida.tar.gz "https://github.com/MohamedElashri/nida/releases/download/${TAG}/nida_${TAG#v}_linux_x86_64.tar.gz"
tar -xzf nida.tar.gz
sudo mv nida /usr/local/bin/nida
```
{{< /tab >}}
{{< /tabs >}}

Check that it worked:

```bash
nida version
```

## Create your first site

{{< steps >}}

### Create a site

```bash
nida init ./my-site
```

`nida init` writes a working example into a new or empty directory: a config file, an about page, a few posts with tags, a page bundle, a search page, templates, and a stylesheet.

### Start the dev server

```bash
nida serve --site ./my-site
```

Open `http://127.0.0.1:1702`. The server rebuilds when you save a file and reloads the browser.

### Write a post

In another terminal, create a post from an [archetype](@/reference/cli.md#new):

```bash
nida new --site ./my-site posts/hello-world.md
```

The new file has a title from the file name, today's date, and `draft = true`. Drafts are hidden unless you pass `--drafts`, so restart the server with `nida serve --site ./my-site --drafts` to see it, and remove the `draft` line when you are ready to publish.

### Check and build

```bash
nida check --site ./my-site
nida build --site ./my-site
```

`nida check` runs the whole build without writing files and reports broken links, missing images, template errors, and config typos. `nida build` writes the finished site to `my-site/public/`.

{{< /steps >}}

## What's in a site

```text
my-site/
├── config.toml        site settings
├── content/           Markdown pages, posts, and sections
│   ├── _index.md      the home page
│   ├── posts/
│   └── pages/
├── templates/         Go HTML templates
├── static/            files copied as-is: CSS, images, fonts
├── themes/            optional themes
├── data/              optional TOML, JSON, or YAML for templates
├── i18n/              optional translated UI strings
├── archetypes/        optional templates for nida new
└── public/            generated output
```

Only `config.toml`, `content/`, and templates (from `templates/` or a theme) are required.


## Next steps

{{< cards >}}
{{< card title="Configuration" href="@/guide/configuration.md" >}}Set your URL, feeds, permalinks, and Markdown options.{{< /card >}}
{{< card title="Writing content" href="@/guide/writing.md" >}}Summaries, callouts, code blocks, math, diagrams, and shortcodes.{{< /card >}}
{{< card title="Templates" href="@/guide/templates.md" >}}Build page layouts with Go templates and Nida's helpers.{{< /card >}}
{{< card title="Deployment" href="@/guide/deployment.md" >}}Publish to GitHub Pages or any static host.{{< /card >}}
{{< /cards >}}
