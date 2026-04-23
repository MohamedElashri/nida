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

	items, err := Discover(siteRoot, cfg)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if len(items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(items))
	}

	if items[0].RelativePath != "pages/about.md" || items[0].Type != TypePage {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].RelativePath != "pages/colophon.md" || items[1].Type != TypePage {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
	if items[2].RelativePath != "posts/designing-the-cli.md" || items[2].Type != TypePost {
		t.Fatalf("unexpected third item: %+v", items[2])
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

	items, err := Discover(dir, cfg)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Slug != "hello-there" {
		t.Fatalf("expected slug hello-there, got %q", items[0].Slug)
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

	_, err = Discover(dir, cfg)
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
