+++
title = "Build Flow"
description = "The command path from CLI flags to generated files."
weight = 10
template = "page"
+++

The build flow starts in `internal/cli`. Both `nida build` and the first pass of `nida serve` call the same core path, so build changes usually affect serve startup too.

## Entry Points

`cmd/nida/main.go` delegates to `internal/cli.Run`. `Run` dispatches one of three public commands:

- `build`
- `serve`
- `version`

The command parser supports long and short flags for site root, config path, drafts, and serve port. Parsed values are stored in `commandOptions`.

## Full Build

`runBuild` calls `buildSite`, which performs the full build:

1. `loadCommandConfig` loads config and applies command-line overrides.
2. `site.Load` discovers content, renders Markdown, and builds the site index.
3. `render.RenderSite` renders pages from templates.
4. `output.ValidateWritePlan` checks for page and artifact path conflicts.
5. `output.WriteSite` clears the output directory and writes rendered pages.
6. `pipeline.Process` runs when fingerprinting, image processing, or SCSS is enabled.
7. `feeds.GenerateAll`, `sitemap.Generate`, `robots.Generate`, and `searchindex.Generate` write generated files.
8. `assets.Copy` copies theme and site static files.
9. `assets.CopyPageBundles` copies page-bundle resources beside rendered bundle pages.

The returned `buildResult` stores the config, config path, `site.State`, and rendered pages. Serve mode keeps this result as the baseline for later rebuilds.

## Artifact Planning

Generated artifacts are included in the write plan before writing:

- RSS
- Atom
- sitemap
- robots
- search index

If a page route would write the same file as an artifact, `output.ValidateWritePlan` fails before the output directory is modified.

## Command Overrides

Command-line options are applied after config loading:

- `--drafts` or `-d` sets `cfg.Drafts = true`
- `--port` or `-p` overrides `cfg.Server.Port`

The final serve port is validated after the override. This matters because config validation already ran before command overrides.

## Where To Change Things

Add or change public CLI flags in `internal/cli/cli.go`, then update CLI tests and reference docs.

Add a new generated artifact by extending the artifact list, writing it after rendered pages, and including it in incremental rebuild artifact handling.

Add a build phase only after deciding whether it should run before or after rendered HTML exists. For example, output rewriting needs generated HTML, while content discovery must happen before routing.

## Tests

Use `internal/cli/cli_test.go` for command behavior and end-to-end build expectations. Use package tests for subsystem behavior, especially when a change can be checked without invoking the CLI.
