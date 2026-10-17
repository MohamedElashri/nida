+++
title = "CLI Reference"
description = "Commands and flags exposed by the Nida command line."
weight = 10
template = "page"
+++

Nida intentionally keeps the command surface small.

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
nida build --site ./my-site
```

Flags:

- `--site`, `-s`: site root directory
- `--config`, `-c`: config file path relative to the site root or absolute
- `--drafts`, `-d`: include draft content
- `--future`, `-f`: include pages dated in the future

## Serve

```bash
nida serve --site ./my-site --port 1702
```

Flags:

- `--site`, `-s`: site root directory
- `--config`, `-c`: config file path relative to the site root or absolute
- `--drafts`, `-d`: include draft content
- `--future`, `-f`: include pages dated in the future
- `--port`, `-p`: override the configured server port

## Version

```bash
nida version
```

Prints the Nida version summary.
