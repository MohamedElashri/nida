package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultSiteConfig(t *testing.T) {
	cfg := DefaultSiteConfig()

	if cfg.Language != "en" {
		t.Fatalf("expected default language en, got %q", cfg.Language)
	}
	if cfg.Server.Port != 1307 {
		t.Fatalf("expected default server port 1307, got %d", cfg.Server.Port)
	}
	if !cfg.RSS.Enabled {
		t.Fatal("expected RSS to be enabled by default")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	cfg, path, err := Load(Options{
		SiteRoot: filepath.Join("..", "..", "example-site"),
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if !strings.HasSuffix(path, filepath.Join("example-site", "config.toml")) {
		t.Fatalf("unexpected config path %q", path)
	}
	if cfg.ContentDir != "content" {
		t.Fatalf("expected default content_dir, got %q", cfg.ContentDir)
	}
	if cfg.Server.Port != 1307 {
		t.Fatalf("expected default port 1307, got %d", cfg.Server.Port)
	}
}

func TestLoadNormalizesValues(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
base_url = "https://example.com/"
title = " My Site "
content_dir = "./content"
output_dir = "public/"

[permalinks]
posts = "posts/{slug}"
pages = "{slug}"
tags = "tags/{slug}"
categories = "categories/{slug}"
`)

	cfg, _, err := Load(Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Title != "My Site" {
		t.Fatalf("expected trimmed title, got %q", cfg.Title)
	}
	if cfg.ContentDir != "content" {
		t.Fatalf("expected cleaned content dir, got %q", cfg.ContentDir)
	}
	if cfg.Permalinks.Posts != "/posts/{slug}/" {
		t.Fatalf("expected normalized posts permalink, got %q", cfg.Permalinks.Posts)
	}
}

func TestLoadMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
title = "Missing URL"
`)

	_, _, err := Load(Options{SiteRoot: dir})
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "base_url is required") {
		t.Fatalf("expected base_url validation error, got %v", err)
	}
}

func TestLoadReportsParseErrors(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
base_url = "https://example.com"
title = "Broken
`)

	_, _, err := Load(Options{SiteRoot: dir})
	if err == nil {
		t.Fatal("expected parse error")
	}

	if !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("expected parse config error, got %v", err)
	}
}

func TestLoadReportsMissingConfig(t *testing.T) {
	_, _, err := Load(Options{SiteRoot: t.TempDir()})
	if err == nil {
		t.Fatal("expected missing config error")
	}
	if !strings.Contains(err.Error(), "file does not exist") {
		t.Fatalf("expected missing config error, got %v", err)
	}
}

func TestDocumentDirection(t *testing.T) {
	tests := []struct {
		language string
		want     string
	}{
		{language: "", want: "ltr"},
		{language: "en", want: "ltr"},
		{language: "ar", want: "rtl"},
		{language: "ar-SA", want: "rtl"},
		{language: "fa_IR", want: "rtl"},
	}

	for _, test := range tests {
		if got := DocumentDirection(test.language); got != test.want {
			t.Fatalf("DocumentDirection(%q) = %q, want %q", test.language, got, test.want)
		}
	}
}
