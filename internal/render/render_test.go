package render

import (
	"os"
	"path/filepath"
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
}

func TestRenderSiteMissingTemplateFailsClearly(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.TemplateDir = "templates"

	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "base.tmpl"), []byte(`{{ define "base" }}{{ template "content" . }}{{ end }}`), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}

	_, err := RenderSite(dir, cfg, site.State{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func assertGoldenPage(t *testing.T, pages []Page, url, goldenName string) {
	t.Helper()

	var got string
	for _, page := range pages {
		if page.URL == url {
			got = page.Content
			break
		}
	}
	if got == "" {
		t.Fatalf("page %q not found", url)
	}

	want, err := os.ReadFile(filepath.Join("testdata", goldenName))
	if err != nil {
		t.Fatalf("read golden %q: %v", goldenName, err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", goldenName, string(want), got)
	}
}
