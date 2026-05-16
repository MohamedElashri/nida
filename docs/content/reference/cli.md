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

## Build

```bash
nida build --site ./my-site
```

Flags:

- `--site`, `-s`: site root directory
- `--config`, `-c`: config file path relative to the site root or absolute
- `--drafts`, `-d`: include draft content

## Serve

```bash
nida serve --site ./my-site --port 2906
```

Flags:

- `--site`, `-s`: site root directory
- `--config`, `-c`: config file path relative to the site root or absolute
- `--drafts`, `-d`: include draft content
- `--port`, `-p`: override the configured server port

## Version

```bash
nida version
```

Prints the Nida version summary.
