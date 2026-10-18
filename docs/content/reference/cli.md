+++
title = "CLI Reference"
description = "Commands and flags exposed by the Nida command line."
weight = 10
template = "page"
+++

Nida intentionally keeps the command surface small.

`build`, `serve`, and `check` work on the current directory unless you name a
site. Pass the site root as an argument or with `--site`; `nida serve ./my-site`
and `nida serve --site ./my-site` are the same. Giving both is an error. Flags
can come before or after the path.

## Init

```bash
nida init ./my-site
```

Creates a buildable example site in a new or empty directory. The example
includes sample posts, tags, feeds, search settings, a page bundle, and editable
templates. If no path is provided, Nida initializes the current directory.

`nida init` refuses to write into a non-empty directory.

## New

```bash
nida new posts/my-first-post.md
nida new posts/trip-report
```

Creates a content file from an archetype. A path ending in `.md` creates that file. A path without `.md` creates a page bundle, `posts/trip-report/index.md`. The path is relative to the content directory, and Nida refuses to overwrite existing files.

Nida looks for `archetypes/<section>.md`, then `archetypes/default.md`, first in the site and then in each theme. Without either, it uses:

```toml
+++
title = "My First Post"
date = 2026-10-17T09:30:00Z
draft = true
+++
```

Archetypes are Go `text/template` files and receive `.Title` (from the file name), `.Date` (RFC 3339), `.Slug`, `.Section`, and `.Path`.

Flags:

- `--site`, `-s`: site root directory
- `--config`, `-c`: config file path

## Build

```bash
nida build ./my-site
nida build --site ./my-site
```

Flags:

- `--site`, `-s`: site root directory, also accepted as the first argument
- `--config`, `-c`: config file path relative to the site root or absolute
- `--drafts`, `-d`: include draft content
- `--future`, `-f`: include pages dated in the future
- `--base-url`: override `base_url`, for example for preview deployments
- `--output`, `-o`: override `output_dir` (relative to the site root)

## Serve

```bash
nida serve ./my-site --port 1702
nida serve --site ./my-site --port 1702
```

Flags:

- `--site`, `-s`: site root directory, also accepted as the first argument
- `--config`, `-c`: config file path relative to the site root or absolute
- `--drafts`, `-d`: include draft content
- `--future`, `-f`: include pages dated in the future
- `--base-url`: override `base_url`, for example for preview deployments
- `--output`, `-o`: override `output_dir` (relative to the site root)
- `--port`, `-p`: override the configured server port

## Check

```bash
nida check ./my-site
nida check --site ./my-site --external
```

Runs a full build without writing any files. It reports unknown config keys, broken `@/` links, missing Markdown images, template errors, unknown shortcodes, and route conflicts, and exits with status 1 if any of these fail. Content [diagnostics](@/reference/config.md#diagnostics) always run here, whatever `diagnostics.enabled` says. It is a good fit for CI and pre-commit hooks.

`--external` also requests every external link in page content (HEAD, falling back to GET) and fails on network errors or HTTP 4xx/5xx. It needs network access, so it is opt-in.

Flags: `--site`/`-s`, `--config`/`-c`, `--drafts`/`-d`, `--future`/`-f`, and `--external`.

## Version

```bash
nida version
```

Prints the Nida version summary. `nida --version` does the same, and `nida help`, `-h`, or `--help` prints the command list.
