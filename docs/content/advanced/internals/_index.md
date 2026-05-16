+++
title = "Internals"
description = "How Nida is put together, for developers changing the codebase."
sort_by = "weight"
weight = 10
+++

Nida is built as a small set of packages around a predictable site-generation pipeline. These pages document the boundaries, data flow, and invariants a contributor needs before changing behavior.

Use this section as a map when you are debugging a build, adding a feature, or reviewing a change that crosses packages.

## System Shape

The CLI is intentionally thin. `internal/cli` parses command flags and orchestrates work; the rest of the behavior lives in packages that can be tested directly.

The full build flow is:

1. load and validate config
2. discover content and render Markdown
3. build routes, sections, taxonomies, and site index data
4. load templates and render HTML pages
5. validate and write output files
6. write generated artifacts
7. process and copy assets

Serve mode starts with the same build, then watches the site tree and chooses between asset sync, incremental content rebuilds, and full rebuilds.

## Working Rules

When changing internals, keep these rules in mind:

- prefer package-level tests over only CLI tests
- keep filesystem access behind path validation
- reject symlink surprises before reading or writing
- validate routes and output conflicts before writes
- keep public config and front matter changes documented
- preserve deterministic ordering for pages, sections, taxonomies, and generated files

The pages below describe where those rules are enforced.
