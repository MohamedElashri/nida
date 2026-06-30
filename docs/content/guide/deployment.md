+++
title = "Deployment"
description = "Build static output with Nida and publish it with GitHub Pages."
weight = 40
template = "page"
+++

Nida writes a static site to the configured `output_dir`, which defaults to `public`.

```bash
nida build --site ./my-site
```

The generated files can be hosted by any static file host.

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

      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true

      - uses: actions/configure-pages@v5

      - name: Generate release notes page
        run: go run ./scripts/generate_release_notes.go

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
