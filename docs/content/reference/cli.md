+++
title = "CLI Reference"
description = "Commands and flags exposed by the Nida command line."
weight = 10
template = "page"
+++

Nida intentionally keeps the command surface small.

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
nida serve --site ./my-site --port 1307
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
