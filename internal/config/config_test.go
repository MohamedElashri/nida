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
	if cfg.Server.Port != 1702 {
		t.Fatalf("expected default server port 1702, got %d", cfg.Server.Port)
	}
	if !cfg.Feed.Enabled {
		t.Fatal("expected Feed to be enabled by default")
	}
	if _, ok := cfg.Pipeline.Images.Presets["content"]; !ok {
		t.Fatal("expected default content image preset")
	}
	if cfg.Diagnostics.Enabled {
		t.Fatal("expected diagnostics to be disabled by default")
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
	if cfg.Server.Port != 1702 {
		t.Fatalf("expected default port 1702, got %d", cfg.Server.Port)
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
	if cfg.Permalinks["posts"] != "posts/{slug}" {
		t.Fatalf("expected posts permalink, got %q", cfg.Permalinks["posts"])
	}
}

func TestLoadFeedSections(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
base_url = "https://example.com"
title = "Feed Sections"

[feed]
sections = ["post"]
`)

	cfg, _, err := Load(Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Feed.Sections) != 1 || cfg.Feed.Sections[0] != "post" {
		t.Fatalf("expected Feed sections [post], got %#v", cfg.Feed.Sections)
	}
}

func TestLoadImagePresets(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
base_url = "https://example.com"
title = "Image Presets"

[pipeline.images]
enabled = true
widths = [600]

[pipeline.images.presets.card]
widths = [360, 720]
sizes = "(max-width: 720px) 100vw, 360px"
`)

	cfg, _, err := Load(Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	preset, ok := cfg.Pipeline.Images.Presets["card"]
	if !ok {
		t.Fatalf("expected card image preset, got %#v", cfg.Pipeline.Images.Presets)
	}
	if len(preset.Widths) != 2 || preset.Widths[0] != 360 || preset.Widths[1] != 720 {
		t.Fatalf("unexpected preset widths %#v", preset.Widths)
	}
	if preset.Sizes != "(max-width: 720px) 100vw, 360px" {
		t.Fatalf("unexpected preset sizes %q", preset.Sizes)
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

func TestLoadRejectsEscapingOutputDir(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
base_url = "https://example.com"
title = "Bad Paths"
output_dir = ".."
`)

	_, _, err := Load(Options{SiteRoot: dir})
	if err == nil {
		t.Fatal("expected unsafe output_dir validation error")
	}
	if !strings.Contains(err.Error(), "output_dir must not escape its root") {
		t.Fatalf("expected output_dir validation error, got %v", err)
	}
}

func TestLoadRejectsNonHTTPBaseURL(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
base_url = "javascript://example.com"
title = "Bad URL"
`)

	_, _, err := Load(Options{SiteRoot: dir})
	if err == nil {
		t.Fatal("expected unsafe base_url validation error")
	}
	if !strings.Contains(err.Error(), "base_url scheme must be http or https") {
		t.Fatalf("expected base_url scheme validation error, got %v", err)
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
