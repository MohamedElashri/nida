SHELL := /bin/zsh

GO ?= go
SITE ?= ./example-site
EXAMPLE_SITE ?= ./example-site
ARABIC_EXAMPLE_SITE ?= ./example-site-ar
BINARY ?= nida
COVERAGE_FILE ?= coverage.out

export GOCACHE := $(CURDIR)/.gocache
export GOMODCACHE := $(CURDIR)/.gomodcache
export GOPROXY := file:///home/melashri/go/pkg/mod/cache/download
export GOSUMDB := off

.PHONY: help dev build rebuild test test-cover serve site-build example-build example-serve arabic-example-build arabic-example-serve clean fmt tidy check

help:
	@printf "Available targets:\n"
	@printf "  make build        Build the nida binary\n"
	@printf "  make rebuild      Clean and rebuild the binary\n"
	@printf "  make test         Run the full Go test suite\n"
	@printf "  make test-cover   Run tests with coverage output\n"
	@printf "  make serve        Run nida serve against SITE=%s\n" "$(SITE)"
	@printf "  make site-build   Build nida against SITE=%s\n" "$(SITE)"
	@printf "  make example-build Build the example website\n"
	@printf "  make example-serve Serve the example website locally\n"
	@printf "  make arabic-example-build Build the Arabic example website\n"
	@printf "  make arabic-example-serve Serve the Arabic example website locally\n"
	@printf "  make fmt          Format Go code\n"
	@printf "  make tidy         Sync go.mod/go.sum\n"
	@printf "  make check        Run fmt, test, and site build\n"
	@printf "  make dev          Alias for check\n"
	@printf "  make clean        Remove build and cache artifacts\n"

build:
	$(GO) build -o $(BINARY) ./cmd/nida

rebuild: clean build

test:
	$(GO) test ./...

test-cover:
	$(GO) test -coverprofile=$(COVERAGE_FILE) ./...

serve:
	$(GO) run ./cmd/nida serve --site $(SITE)

site-build:
	$(GO) run ./cmd/nida build --site $(SITE)

example-build:
	$(GO) run ./cmd/nida build --site $(EXAMPLE_SITE)

example-serve:
	$(GO) run ./cmd/nida serve --site $(EXAMPLE_SITE)

arabic-example-build:
	$(GO) run ./cmd/nida build --site $(ARABIC_EXAMPLE_SITE)

arabic-example-serve:
	$(GO) run ./cmd/nida serve --site $(ARABIC_EXAMPLE_SITE)

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

check: fmt test site-build

dev: check

clean:
	rm -f $(BINARY) $(COVERAGE_FILE)
	chmod -R u+w .gomodcache 2>/dev/null || true
	rm -rf .gocache .gomodcache
	find $(SITE)/public -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
	find $(EXAMPLE_SITE)/public -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
	find $(ARABIC_EXAMPLE_SITE)/public -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
