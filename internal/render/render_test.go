package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/site"
)

func TestRenderSiteGolden(t *testing.T) {
	siteRoot := filepath.Join("..", "..", "example-site")
	cfg, _, err := config.Load(config.Options{SiteRoot: siteRoot})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	state, err := site.Load(siteRoot, cfg)
	if err != nil {
		t.Fatalf("site.Load: %v", err)
	}

	pages, err := RenderSite(siteRoot, cfg, state)
	if err != nil {
		t.Fatalf("RenderSite returned error: %v", err)
	}

	assertGoldenPage(t, pages, "/", "homepage.golden.html")
	assertGoldenPage(t, pages, "/posts/launching-nida/", "post.golden.html")
	assertGoldenPage(t, pages, "/about/", "page.golden.html")
	assertGoldenPage(t, pages, "/tags/", "tags.golden.html")
	assertGoldenPage(t, pages, "/tags/architecture/", "tag-architecture.golden.html")
	assertGoldenPage(t, pages, "/categories/", "categories.golden.html")
	assertGoldenPage(t, pages, "/categories/engineering/", "category-engineering.golden.html")
	assertGoldenPage(t, pages, "/404.html", "404.golden.html")
}

func TestRenderSiteMissingTemplateFailsClearly(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.TemplateDir = "templates"

	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "base.html"), []byte(`{{ define "base" }}{{ template "content" . }}{{ end }}`), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}

	_, err := RenderSite(dir, cfg, site.State{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderSiteIncludesBuiltin404WithoutThemeTemplate(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Title = "Example"
	cfg.TemplateDir = "templates"

	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "base.html"), []byte(`{{ define "base" }}{{ template "content" . }}{{ end }}`), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(`{{ define "index" }}home{{ end }}`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "post.html"), []byte(`{{ define "post" }}post{{ end }}`), 0o644); err != nil {
		t.Fatalf("write post: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "page.html"), []byte(`{{ define "page" }}page{{ end }}`), 0o644); err != nil {
		t.Fatalf("write page: %v", err)
	}

	pages, err := RenderSite(dir, cfg, site.State{})
	if err != nil {
		t.Fatalf("RenderSite returned error: %v", err)
	}

	page := findPage(t, pages, "/404.html")
	if page.TemplateName != "builtin-404" {
		t.Fatalf("expected builtin 404 template, got %q", page.TemplateName)
	}
	if page.CanonicalURL != "https://example.com/404.html" {
		t.Fatalf("unexpected canonical URL %q", page.CanonicalURL)
	}
	if !containsAll(page.Content, "<title>Page not found | Example</title>", "<meta name=\"robots\" content=\"noindex\">", "<h1>Page not found</h1>") {
		t.Fatalf("unexpected builtin 404 content:\n%s", page.Content)
	}
}

func TestRenderArabicExampleSiteUsesRTL(t *testing.T) {
	siteRoot := filepath.Join("..", "..", "example-site-ar")
	cfg, _, err := config.Load(config.Options{SiteRoot: siteRoot})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	state, err := site.Load(siteRoot, cfg)
	if err != nil {
		t.Fatalf("site.Load: %v", err)
	}

	pages, err := RenderSite(siteRoot, cfg, state)
	if err != nil {
		t.Fatalf("RenderSite returned error: %v", err)
	}

	home := findPage(t, pages, "/")
	if !containsAll(home.Content, `<html lang="ar" dir="rtl">`, "أحدث المقالات", "الرئيسية") {
		t.Fatalf("unexpected arabic homepage content:\n%s", home.Content)
	}
}

func TestRenderSiteBuiltin404UsesConfiguredRTLDirection(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Title = "مثال"
	cfg.Language = "ar"
	cfg.TemplateDir = "templates"

	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "base.html"), []byte(`{{ define "base" }}{{ template "content" . }}{{ end }}`), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(`{{ define "index" }}home{{ end }}`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "post.html"), []byte(`{{ define "post" }}post{{ end }}`), 0o644); err != nil {
		t.Fatalf("write post: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "page.html"), []byte(`{{ define "page" }}page{{ end }}`), 0o644); err != nil {
		t.Fatalf("write page: %v", err)
	}

	pages, err := RenderSite(dir, cfg, site.State{})
	if err != nil {
		t.Fatalf("RenderSite returned error: %v", err)
	}

	page := findPage(t, pages, "/404.html")
	if !containsAll(page.Content, `<html lang="ar" dir="rtl">`, `<a href="/">Return to the homepage</a>`) {
		t.Fatalf("unexpected rtl 404 content:\n%s", page.Content)
	}
}

func assertGoldenPage(t *testing.T, pages []Page, url, goldenName string) {
	t.Helper()

	got := findPage(t, pages, url).Content

	want, err := os.ReadFile(filepath.Join("testdata", goldenName))
	if err != nil {
		t.Fatalf("read golden %q: %v", goldenName, err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", goldenName, string(want), got)
	}
}

func findPage(t *testing.T, pages []Page, url string) Page {
	t.Helper()

	for _, page := range pages {
		if page.URL == url {
			return page
		}
	}
	t.Fatalf("page %q not found", url)
	return Page{}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
