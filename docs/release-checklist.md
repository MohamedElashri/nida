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

## Versioning

Until `v1.0.0`, use `v0.x.y` tags to signal that user-facing behavior may still evolve.

Treat `v1.0.0` as the release where:

* command behavior is intentionally stable
* the supported config shape is intentionally stable
* the documented site layout and output guarantees are intentionally stable
