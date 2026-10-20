SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

GO ?= go
SITE ?= ./example-site
DOCS_SITE ?= ./docs
EXAMPLE_SITE ?= ./example-site
ARABIC_EXAMPLE_SITE ?= ./example-site-ar
BINARY ?= nida
COVERAGE_FILE ?= coverage.out
ARGS ?=
GOJA ?= master
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILT_BY ?= local
LDFLAGS := -X github.com/MohamedElashri/nida/internal/buildinfo.Version=$(VERSION) -X github.com/MohamedElashri/nida/internal/buildinfo.Commit=$(COMMIT) -X github.com/MohamedElashri/nida/internal/buildinfo.Date=$(DATE) -X github.com/MohamedElashri/nida/internal/buildinfo.BuiltBy=$(BUILT_BY)

export GOCACHE := $(CURDIR)/.gocache
export GOMODCACHE := $(CURDIR)/.gomodcache
export GOPROXY ?= https://proxy.golang.org,direct
export GOSUMDB ?= sum.golang.org

.PHONY: help build install run rebuild size \
	fmt fmt-check vet lint test test-race test-cover tidy check dev ci \
	serve site-build example-build example-serve arabic-example-build arabic-example-serve sites \
	docs-release-notes docs-build docs-serve docs-check \
	vendor vendor-js vendor-goja goja-update vendor-check outdated \
	release-check release-snapshot clean

##@ General

help: ## Show this help
	@awk 'BEGIN { FS = ":.*## " } \
		/^##@/ { printf "\n%s\n", substr($$0, 5); next } \
		/^[a-zA-Z0-9_-]+:.*## / { printf "  make %-22s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf "\nVariables: SITE=%s ARGS='...' GOJA=%s VERSION=%s\n" "$(SITE)" "$(GOJA)" "$(VERSION)"

##@ Build

build: ## Build the nida binary
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/nida

install: ## Install nida into GOBIN
	$(GO) install -ldflags '$(LDFLAGS)' ./cmd/nida

run: ## Run nida with ARGS, such as ARGS='check ./docs'
	$(GO) run ./cmd/nida $(ARGS)

rebuild: clean build ## Clean and rebuild the binary

size: ## Build a stripped release binary and print its size
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '-s -w $(LDFLAGS)' -o $(BINARY)-release ./cmd/nida
	@printf "%s: %s bytes\n" "$(BINARY)-release" "$$(wc -c < $(BINARY)-release | tr -d ' ')"
	@rm -f $(BINARY)-release

##@ Quality

fmt: ## Format Go code
	$(GO) fmt ./...

fmt-check: ## Fail if any Go file needs gofmt
	@files="$$(gofmt -l $$(git ls-files '*.go'; git ls-files -o --exclude-standard '*.go'))"; \
	if [[ -n "$$files" ]]; then echo "These files need gofmt:"; echo "$$files"; exit 1; fi

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

test: ## Run the full Go test suite
	$(GO) test ./...

test-race: ## Run the tests with the race detector
	$(GO) test -race ./...

test-cover: ## Run the tests with coverage output
	$(GO) test -coverprofile=$(COVERAGE_FILE) ./...

tidy: ## Sync go.mod and go.sum
	$(GO) mod tidy

check: fmt vet lint test sites ## Format, vet, lint, test, and build every site
dev: check ## Alias for check

ci: fmt-check vet vendor-check test sites ## Run what CI runs, without changing files

##@ Sites

serve: ## Serve SITE with live reload
	$(GO) run ./cmd/nida serve --site $(SITE)

site-build: ## Build SITE
	$(GO) run ./cmd/nida build --site $(SITE)

example-build: ## Build the example site
	$(GO) run ./cmd/nida build --site $(EXAMPLE_SITE)

example-serve: ## Serve the example site
	$(GO) run ./cmd/nida serve --site $(EXAMPLE_SITE)

arabic-example-build: ## Build the Arabic example site
	$(GO) run ./cmd/nida build --site $(ARABIC_EXAMPLE_SITE)

arabic-example-serve: ## Serve the Arabic example site
	$(GO) run ./cmd/nida serve --site $(ARABIC_EXAMPLE_SITE)

sites: example-build arabic-example-build docs-check ## Build both example sites and check the docs

##@ Docs

docs-release-notes: ## Generate the docs release notes from CHANGELOG.md
	$(GO) run ./scripts/generate_release_notes.go

docs-build: docs-release-notes ## Build the docs site
	$(GO) run ./cmd/nida build --site $(DOCS_SITE)

docs-serve: docs-release-notes ## Serve the docs site
	$(GO) run ./cmd/nida serve --site $(DOCS_SITE)

docs-check: docs-release-notes ## Check the docs site for broken links and templates
	$(GO) run ./cmd/nida check --site $(DOCS_SITE)

##@ Bundled dependencies (see third_party/README.md)

vendor: vendor-js vendor-goja ## Regenerate all embedded third-party code

vendor-js: ## Regenerate embedded Pele and Temml files from third_party/js
	$(GO) run ./scripts/vendorjs

vendor-goja: ## Rebuild third_party/goja from third_party/goja.version
	$(GO) run ./scripts/vendorgoja

goja-update: ## Move third_party/goja to GOJA (default master)
	$(GO) run ./scripts/vendorgoja -update $(GOJA)
	$(GO) test ./internal/jsengine ./internal/diagram ./internal/mathml ./internal/markdown

vendor-check: ## Verify embedded files and third_party/goja match their pins
	$(GO) run ./scripts/vendorjs -check
	$(GO) run ./scripts/vendorgoja -check

outdated: ## List Go modules and goja commits with newer versions
	@$(GO) list -m -u -f '{{if and .Update (not .Indirect)}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all
	@latest="$$($(GO) run ./scripts/vendorgoja -outdated)"; \
	if [[ -n "$$latest" ]]; then echo "goja (third_party) -> $$latest: make goja-update GOJA=$$latest"; fi
	@echo "Pele and Temml: Dependabot opens pull requests for third_party/js"

##@ Release

release-check: ## Validate .goreleaser.yaml
	goreleaser check

release-snapshot: ## Build release archives locally into dist/ without publishing
	goreleaser release --snapshot --clean

clean: ## Remove build output and caches
	rm -f $(BINARY) $(BINARY)-release $(COVERAGE_FILE)
	chmod -R u+w .gomodcache 2>/dev/null || true
	rm -rf .gocache .gomodcache dist
	find $(SITE)/public -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
	find $(EXAMPLE_SITE)/public -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
	find $(ARABIC_EXAMPLE_SITE)/public -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
