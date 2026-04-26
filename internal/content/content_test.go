package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestDiscoverFixtureSite(t *testing.T) {
	siteRoot := filepath.Join("..", "..", "example-site")
	cfg, _, err := config.Load(config.Options{SiteRoot: siteRoot})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	pages, sections, err := Discover(siteRoot, cfg)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if len(pages) != 5 {
		t.Fatalf("expected 5 pages, got %d", len(pages))
	}
	if len(sections) < 2 {
		t.Fatalf("expected at least 2 sections (root + posts), got %d", len(sections))
	}

	for _, p := range pages {
		if p.RelativePath == "" || p.Slug == "" {
			t.Fatalf("page missing required fields: %+v", p)
		}
	}
}

func TestDiscoverDerivesSlugFromFilename(t *testing.T) {
	dir := t.TempDir()
	writeSiteConfig(t, dir)
	writeContentFile(t, filepath.Join(dir, "content", "posts", "Hello There.md"), `+++
title = "Hello There"
date = 2026-04-12T10:00:00Z
+++

Body
`)

	cfg, _, err := config.Load(config.Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	pages, _, err := Discover(dir, cfg)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if pages[0].Slug != "hello-there" {
		t.Fatalf("expected slug hello-there, got %q", pages[0].Slug)
	}
}

func TestDiscoverReportsParseErrorsWithPath(t *testing.T) {
	dir := t.TempDir()
	writeSiteConfig(t, dir)
	writeContentFile(t, filepath.Join(dir, "content", "posts", "broken.md"), `+++
title = "Broken"
`)

	cfg, _, err := config.Load(config.Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	_, _, err = Discover(dir, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "broken.md") {
		t.Fatalf("expected source path in error, got %v", err)
	}
}

func TestDeriveSlug(t *testing.T) {
	if got := DeriveSlug("  Hello_world!.md "); got != "hello-world" {
		t.Fatalf("unexpected slug %q", got)
	}
}

func TestDeriveSlugNonASCII(t *testing.T) {
	if got := DeriveSlug("البنية"); got == "" {
		t.Fatalf("expected non-empty slug for Arabic input")
	}
}

func TestEstimateReadingTime(t *testing.T) {
	if got := EstimateReadingTime(""); got != 0 {
		t.Fatalf("expected empty content reading time 0, got %d", got)
	}
	if got := EstimateReadingTime(strings.Repeat("word ", 201)); got != 2 {
		t.Fatalf("expected 201 words to round up to 2 minutes, got %d", got)
	}
}

func writeSiteConfig(t *testing.T, dir string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
base_url = "https://example.com"
title = "Test Site"
config_version = "0.4"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func writeContentFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir content dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write content file: %v", err)
	}
}
