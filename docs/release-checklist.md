# Release Checklist

This checklist is for the first tagged Nida release and later cut releases.

## Scope

Release only when the documented behavior matches the shipped behavior for:

* `nida build`
* `nida serve`
* config loading and validation
* content discovery and rendering
* taxonomy pages
* RSS generation
* sitemap generation
* output writing and static asset copying

## Checklist

* `go test ./...` passes
* `go build ./...` passes
* `go run ./cmd/nida build --site ./example-site` succeeds
* `go run ./cmd/nida serve --site ./example-site` starts successfully on the documented default port or an explicitly chosen override
* `go run ./cmd/nida version` reports the expected version metadata for the release candidate build

## Publishing Workflow

Tagged releases use GitHub Actions and GoReleaser.

Expected release behavior:

* push a tag matching `v*`
* CI runs tests and example-site verification before packaging
* GitHub Releases receives platform archives plus `checksums.txt`
* archives contain the `nida` binary, `README.md`, and `LICENSE`
* example sites are not included in release archives
* the Homebrew tap repo receives an updated `Formula/nida.rb` generated from release checksums

## Homebrew Prerequisites

Before publishing a tagged release that should update Homebrew:

* the tap repo `MohamedElashri/homebrew-nida` exists
* the tap repo contains a `Formula/` directory
* this repo has a `HOMEBREW_TAP_TOKEN` Actions secret
* `HOMEBREW_TAP_TOKEN` has permission to write contents to `MohamedElashri/homebrew-nida`
* the release workflow on the default branch includes the Homebrew formula update steps

## Versioning

Until `v1.0.0`, use `v0.x.y` tags to signal that user-facing behavior may still evolve.

Treat `v1.0.0` as the release where:

* command behavior is intentionally stable
* the supported config shape is intentionally stable
* the documented site layout and output guarantees are intentionally stable
