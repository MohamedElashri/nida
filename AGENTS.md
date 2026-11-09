# Working on Nida

Nida is a static site generator written in Go. Keep the single-binary workflow:
normal builds use the embedded dependencies and do not need Node.

## Find the source of truth

- [Development workflow](docs/content/advanced/development.md): setup and repository layout.
- [Internals](docs/content/advanced/internals/_index.md): package boundaries and build behavior.
- [Makefile](Makefile) and [.github/workflows/ci.yml](.github/workflows/ci.yml): commands and required checks. Use `make help` to discover targets.
- [third_party/README.md](third_party/README.md): dependency pins, patches, and regeneration.

Use the Go version in `go.mod` and read nearby tests before changing behavior.

## Preserve these contracts

- Use `internal/safepath` for paths under site/output roots, including symlink checks. Builds clear the output directory: preserve source/output separation and write-plan validation before modifying output.
- Keep `nida check` free of output writes.
- Preserve deterministic ordering and output when work runs in parallel.
- Resolve theme resources through `internal/themes` so site overrides and theme inheritance work consistently.
- Consider both full builds and serve-mode rebuilds. Changes to routes, assets, or generated artifacts must also update or remove stale output after edits, deletions, and config changes.
- Document public behavior changes in `docs/content/`. For config additions, cover defaults, normalization, validation, and unknown-key warnings.

## Validate the change

- Start with focused package tests and add regression coverage for behavior changes. Use CLI tests for command behavior.
- Format changed Go files and run `make test` for Go changes. Run race tests on affected packages for concurrency or watcher/rebuild changes (`make test-race` covers the full suite).
- Run `make sites` for rendering, routing, theme, or asset changes; it builds both example sites and checks the docs. Preserve Arabic/RTL support.
- Run `make docs-check` for docs-site changes. Prose-only changes outside that site need link and diff checks. Use `make ci` for the combined checks in the Makefile.
- Review golden/fixture diffs for intended behavior; do not update expectations just to silence failures.

## Edit sources, not generated copies

Follow `third_party/README.md` to update bundled dependencies; change their pins
or patches and regenerate rather than editing `third_party/goja/` or embedded
library assets directly. Keep `go install ...@latest` working: do not add a
`replace` directive to `go.mod`.

`CHANGELOG.md` is the source for `docs/content/release-notes.md`; regenerate with
`make docs-release-notes`. Build output and caches are not source, but checked-in
`testdata` trees are fixtures and may intentionally contain generated-looking files.

Keep this guide to durable constraints and links. Maintain details in the sources
above rather than duplicating them here.
