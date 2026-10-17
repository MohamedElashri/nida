+++
title = "Deployment"
description = "Build static output and publish it to GitHub Pages, Netlify, Cloudflare Pages, or any static host."
weight = 40
template = "page"
+++

Nida writes a static site to the configured `output_dir`, which defaults to `public`.

```bash
nida build --site ./my-site
```

The generated files can be hosted by any static file host. Before publishing, run a check in CI so broken links and template errors never reach production:

```bash
nida check --site ./my-site
```

## GitHub Pages

GitHub Pages works well with Nida because Nida produces plain static output. The recommended setup is to keep source files in the repository and let GitHub Actions publish the generated output as a Pages artifact.

Do not commit generated output unless you specifically want to maintain a generated branch.

## Project Pages base URL

For a repository project site, the published URL usually includes the repository name:

```text
https://OWNER.github.io/REPOSITORY/
```

For Nida's own docs, the docs config uses:

```toml
base_url = "https://melashri.net/nida/"
```

Nida derives `.BasePath` from that URL. In this case, `.BasePath` is `/nida`.

Use `asset` for static files and `.BasePath` for navigation:

```html
<link rel="stylesheet" href="{{ asset "style.css" }}">
<a href="{{ .BasePath }}/guide/">Guide</a>
```

This keeps links working both on GitHub Pages and in local `nida serve` previews.

## Repository settings

In the GitHub repository settings:

1. Open `Settings`.
2. Open `Pages`.
3. Set the source to `GitHub Actions`.

After that, a workflow can upload and deploy the generated site.

## Workflow shape

Nida's own documentation site follows this model in the [repository workflow](https://github.com/MohamedElashri/nida/blob/main/.github/workflows/pages.yml).

The flow is:

```text
main branch source files
        |
        v
GitHub Actions checks out the repository
        |
        v
GitHub Actions sets up Go
        |
        v
Nida builds docs/public
        |
        v
GitHub Pages deploys docs/public as an artifact
```

A minimal workflow looks like this:

```yaml
name: pages

on:
  push:
    branches:
      - main
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          # Full history, so updated_from_git can date every page.
          fetch-depth: 0

      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true

      - uses: actions/configure-pages@v5

      - name: Generate release notes page
        run: go run ./scripts/generate_release_notes.go

      - name: Check documentation site
        run: go run ./cmd/nida check --site ./docs

      - name: Build documentation site
        run: go run ./cmd/nida build --site ./docs

      - uses: actions/upload-pages-artifact@v4
        with:
          path: docs/public

  deploy:
    runs-on: ubuntu-latest
    needs: build
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

Adjust the build command and artifact path for your site layout. If the site root is the repository root, the build command might be `nida build --site .` and the artifact path might be `public`.

> [!NOTE]
> `actions/checkout` fetches a single commit by default. With `updated_from_git = true`, that makes every page look last updated by the latest commit. Set `fetch-depth: 0` as shown above.

## Other hosts

Any host that serves files from a directory works. Point it at the output directory and use the same build command.

| Host | Build command | Output directory |
| --- | --- | --- |
| Netlify | `nida build --site .` | `public` |
| Cloudflare Pages | `nida build --site .` | `public` |
| Vercel | `nida build --site .` | `public` |
| Any server | `nida build --site .`, then `rsync -a public/ server:/var/www/site/` | `public` |

Hosts without Nida preinstalled can download a release in the build step with the [install script](@/guide/getting-started.md#install), for example `curl -fsSL https://melashri.net/nida/install.sh | sh && ~/.local/bin/nida build --site .`.

All of these serve `404.html` for missing pages, either automatically or with a one-line setting.

## Preview deployments

Preview builds for pull requests usually live at a different URL than production. Override `base_url` for those builds so canonical URLs, feeds, and asset links point at the preview:

```bash
nida build --site . --base-url "$DEPLOY_PRIME_URL/"
```

Use `--output` to write a preview next to the production build without overwriting it:

```bash
nida build --site . --base-url https://example.com/preview/ --output public-preview
```

## Local preview

Use the same config locally:

```bash
nida serve --site ./docs
```

If `base_url` contains a path like `/nida/`, Nida serves that path locally too. For the Nida docs site, preview the GitHub Pages shape at:

```text
http://127.0.0.1:1702/nida/
```

This helps catch asset and navigation problems before merging.

## Common mistakes

If CSS loads as HTML, check that template asset links use `asset` when the site is hosted under a project path.

If the deployment succeeds but the site is blank or missing assets, check the artifact path in `actions/upload-pages-artifact`.

If Pages never deploys, check that the repository Pages source is set to `GitHub Actions`.

If canonical URLs are wrong, check `base_url` in `config.toml`.
