# Release Process

This document describes how Nida releases are cut, what automation runs, and
what to check when a release fails. It is written for maintainers.

## Summary

Nida releases are tag-driven:

1. A maintainer pushes a version tag such as `v0.1.0`.
2. GitHub Actions runs `.github/workflows/release.yml`.
3. The workflow runs GoReleaser with `.goreleaser.yaml`.
4. GoReleaser tests, builds, archives, checksums, and publishes GitHub release assets.
5. The workflow updates `MohamedElashri/homebrew-nida` from the generated checksums.

Manual `workflow_dispatch` runs are snapshot checks. They build release artifacts
without publishing a GitHub release or updating Homebrew.

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

Until `v1.0.0`, use `v0.x.y` tags to signal that user-facing behavior may still
change.

## Preflight

Run these checks from the repository root before tagging:

```bash
go test ./...
go build ./...
go run ./cmd/nida build --site ./example-site
go run ./cmd/nida build --site ./example-site-ar
go run ./cmd/nida version
goreleaser check
```

If the local Go build cache is not writable, set it inside the repo:

```bash
GOCACHE="$PWD/.gocache" go test ./...
```

For a full local release dry run:

```bash
GOCACHE="$PWD/.gocache" goreleaser release --snapshot --clean
```

This writes artifacts under `dist/` and does not publish anything.

## Cutting A Release

1. Make sure `main` contains the release commit.
2. Run the preflight checks above.
3. Create and push the tag:

```bash
git tag v0.2.0
git push origin v0.2.0
```

4. Watch the `release` workflow in GitHub Actions.
5. Confirm the GitHub Release has archives and `checksums.txt`.
6. Confirm the Homebrew tap received an updated `Formula/nida.rb`.
7. Install from the tap and verify:

```bash
brew update
brew install MohamedElashri/nida/nida
nida version
```

## What GoReleaser Builds

GoReleaser builds the `nida` binary from `./cmd/nida` for:

* Linux amd64 and arm64
* macOS amd64 and arm64
* Windows amd64

Archives include:

* `nida` binary
* `LICENSE`
* `README.md`

Example sites are used as release verification fixtures, but they are not
included in release archives.

Build metadata is injected into `internal/buildinfo`:

* version from the release tag, without the leading `v`
* commit SHA
* build date
* `builtBy=goreleaser`

The `nida version` command should show those values for release binaries.

## Homebrew Tap Update

After GoReleaser publishes assets, the release workflow:

1. Checks out `MohamedElashri/homebrew-nida`.
2. Reads SHA-256 values from `dist/checksums.txt`.
3. Renders `packaging/homebrew/Formula/nida.rb.tpl`.
4. Commits and pushes `Formula/nida.rb` to the tap.

The formula uses the full Git tag for the GitHub release URL, such as
`v0.2.0`, and the tag without `v` for Homebrew's package version and archive
filename, such as `0.2.0`.

Prerequisites:

* `MohamedElashri/homebrew-nida` exists.
* The tap has a `Formula/` directory.
* This repo has a `HOMEBREW_TAP_TOKEN` Actions secret.
* `HOMEBREW_TAP_TOKEN` can write contents to the tap repository.

If the GitHub Release succeeds but the tap update fails, fix the tap problem and
rerun the failed workflow job for the same tag.

## Failure Handling

If validation fails before publishing, fix the issue, delete the local tag if
needed, create a corrected tag, and push again.

If GitHub release publishing partially succeeds, inspect the release assets
before rerunning. Prefer deleting the incomplete GitHub Release and tag only when
the release was never announced or consumed.

If a bad release was already consumed, keep the tag immutable and publish a new
patch release instead.

## Robustness Notes

The release path currently has these safeguards:

* tag validation rejects malformed `v*` tags before publishing
* manual workflow runs are snapshots and do not publish
* GoReleaser runs tests, a package build, and both bundled example-site builds
* Homebrew checksum extraction fails if any expected platform archive is missing

Future hardening worth considering:

* add a CI workflow for every pull request, separate from release publishing
* add `goreleaser release --snapshot --clean` to CI for changes touching release files
* sign checksums or artifacts once release consumers need stronger provenance
* add a short smoke test that downloads a just-published archive and runs `nida version`
