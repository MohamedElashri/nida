+++
title = "Release Workflow"
description = "How maintainers cut tagged Nida releases."
weight = 30
template = "page"
+++

Nida releases are tag-driven. Maintainers push a version tag, GitHub Actions runs GoReleaser, and the release workflow publishes GitHub release assets plus an updated Homebrew formula.

Manual `workflow_dispatch` runs are snapshots. They build artifacts without publishing a GitHub release or updating Homebrew.

## Version Tags

Release tags must match:

```text
vMAJOR.MINOR.PATCH
vMAJOR.MINOR.PATCH-prerelease
```

Examples:

```text
v0.2.0
v0.2.0-rc.1
v1.0.0
```

Until `v1.0.0`, use `v0.x.y` tags to signal that user-facing behavior may still change.

## Preflight

Before tagging, make sure documented behavior matches the shipped behavior for builds, serve mode, config loading, content discovery, rendering, taxonomies, feeds, sitemap, output writing, and static assets.

Run from the repository root:

```bash
go test ./...
go build ./...
go run ./cmd/nida build -s ./example-site
go run ./cmd/nida build -s ./example-site-ar
make docs-build
go run ./cmd/nida version
goreleaser check
```

If the normal Go build cache is not writable:

```bash
GOCACHE="$PWD/.gocache" go test ./...
```

For a full local release dry run:

```bash
GOCACHE="$PWD/.gocache" goreleaser release --snapshot --clean
```

This writes artifacts under `dist/` and does not publish anything.

Also confirm:

- `go run ./cmd/nida serve -s ./example-site` starts on the default port or an explicit `-p` override
- `go run ./cmd/nida version` reports expected metadata for the release candidate build
- `MohamedElashri/homebrew-nida` exists and contains a `Formula/` directory
- this repository has a `HOMEBREW_TAP_TOKEN` secret with contents write access to the tap

## Cut A Release

1. Make sure `main` contains the release commit.
2. Update `CHANGELOG.md` with the version, date, and migration notes.
3. Run the preflight checks.
4. Create and push the tag.

```bash
git tag v0.2.0
git push origin v0.2.0
```

After the workflow starts, confirm the GitHub Release has archives and `checksums.txt`, then confirm the Homebrew tap received an updated `Formula/nida.rb`.

Install from the tap and verify:

```bash
brew update
brew install MohamedElashri/nida/nida
nida version
```

## Automation

`.github/workflows/release.yml` runs on `v*` tags and manual dispatch.

For tag runs, it validates the tag shape, installs GoReleaser, runs `goreleaser release --clean`, publishes release assets with `GITHUB_TOKEN`, checks out `MohamedElashri/homebrew-nida`, renders `packaging/homebrew/Formula/nida.rb.tpl`, and pushes the formula when it changed.

For manual runs, it runs `goreleaser release --snapshot --clean` and stops before publishing or touching the tap.

## GoReleaser

`.goreleaser.yaml` builds `./cmd/nida` with `CGO_ENABLED=0` for:

- Linux amd64 and arm64
- macOS amd64 and arm64
- Windows amd64

Archives include the `nida` binary, `LICENSE`, `README.md`, and `CHANGELOG.md`. Example sites are release verification fixtures, but they are not included in archives.

Build metadata is injected into `internal/buildinfo`: version, commit SHA, build date, and `builtBy=goreleaser`. Release binaries print the public version, while development builds include more metadata.

## Homebrew Tap

After GoReleaser publishes assets, the workflow reads `dist/checksums.txt`, extracts checksums for macOS and Linux archives, renders `Formula/nida.rb`, and commits it to `MohamedElashri/homebrew-nida`.

The formula uses the full Git tag for release URLs, such as `v0.2.0`, and the tag without `v` for Homebrew's package version and archive names, such as `0.2.0`.

If the GitHub Release succeeds but the tap update fails, fix the tap problem and rerun the failed workflow job when the fix is outside this repository. If the fix changes this repository's workflow, create a new patch release unless the failed tag was never consumed.

## Failure Handling

If validation fails before publishing, fix the issue, delete the local tag if needed, create a corrected tag, and push again.

If publishing partially succeeds, inspect the release assets before rerunning. Delete an incomplete GitHub Release or tag only when the release was never announced or consumed.

If a bad release was already consumed, keep the tag immutable and publish a new patch release.

## Hardening Ideas

- add GoReleaser snapshot checks to CI for release-file changes
- sign checksums or artifacts when users need stronger provenance
- add a smoke test that downloads a just-published archive and runs `nida version`
