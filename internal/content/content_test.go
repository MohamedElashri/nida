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
	if got := DeriveSlug("  Hello_world!.md "); got != "hello-world-md" {
		t.Fatalf("unexpected slug %q", got)
	}
	if got := DeriveSlug("M. Elashri"); got != "m-elashri" {
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

func TestDiscoverBundlePage(t *testing.T) {
	dir := t.TempDir()
	writeSiteConfig(t, dir)

	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "index.md"), `+++
title = "Bundle Page"
date = 2026-05-01T10:00:00Z
+++

Bundle body content.
`)
	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "image.png"), "fake-png-data")
	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "chart.svg"), "<svg></svg>")

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

	bundle := pages[0]
	if !bundle.IsBundle {
		t.Fatal("expected page to be a bundle")
	}
	if bundle.Slug != "my-bundle" {
		t.Fatalf("expected slug my-bundle, got %q", bundle.Slug)
	}
	if bundle.SectionPath != "posts" {
		t.Fatalf("expected SectionPath posts, got %q", bundle.SectionPath)
	}
	if bundle.BundleDir != "posts/my-bundle" {
		t.Fatalf("expected BundleDir posts/my-bundle, got %q", bundle.BundleDir)
	}

	if len(bundle.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d: %v", len(bundle.Resources), bundle.Resources)
	}

	if bundle.Resources[0] != "chart.svg" || bundle.Resources[1] != "image.png" {
		t.Fatalf("expected sorted resources [chart.svg image.png], got %v", bundle.Resources)
	}
}

func TestBundlePageWithCustomSlug(t *testing.T) {
	dir := t.TempDir()
	writeSiteConfig(t, dir)

	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "index.md"), `+++
title = "Custom Slug Bundle"
slug = "custom-bundle-slug"
date = 2026-05-01T10:00:00Z
+++

Body.
`)

	cfg, _, err := config.Load(config.Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	pages, _, err := Discover(dir, cfg)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if pages[0].Slug != "custom-bundle-slug" {
		t.Fatalf("expected slug custom-bundle-slug, got %q", pages[0].Slug)
	}
}

func TestBundlePageNoResources(t *testing.T) {
	dir := t.TempDir()
	writeSiteConfig(t, dir)

	writeContentFile(t, filepath.Join(dir, "content", "posts", "empty-bundle", "index.md"), `+++
title = "Empty Bundle"
date = 2026-05-01T10:00:00Z
+++

No resources here.
`)

	cfg, _, err := config.Load(config.Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	pages, _, err := Discover(dir, cfg)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if !pages[0].IsBundle {
		t.Fatal("expected page to be a bundle even without resources")
	}
	if len(pages[0].Resources) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(pages[0].Resources))
	}
}

func TestBundleAndFlatPagesCoexist(t *testing.T) {
	dir := t.TempDir()
	writeSiteConfig(t, dir)

	writeContentFile(t, filepath.Join(dir, "content", "posts", "_index.md"), `+++
title = "Posts"
+++
`)
	writeContentFile(t, filepath.Join(dir, "content", "posts", "regular-post.md"), `+++
title = "Regular Post"
date = 2026-05-01T10:00:00Z
+++

Regular body.
`)
	writeContentFile(t, filepath.Join(dir, "content", "posts", "bundled-post", "index.md"), `+++
title = "Bundled Post"
date = 2026-05-02T10:00:00Z
+++

Bundle body.
`)
	writeContentFile(t, filepath.Join(dir, "content", "posts", "bundled-post", "img.jpg"), "jpg-data")

	cfg, _, err := config.Load(config.Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	pages, sections, err := Discover(dir, cfg)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if len(sections) < 1 {
		t.Fatalf("expected at least 1 section, got %d", len(sections))
	}

	var regularPage, bundlePage Page
	for _, p := range pages {
		if p.IsBundle {
			bundlePage = p
		} else {
			regularPage = p
		}
	}

	if regularPage.Slug == "" || regularPage.IsBundle {
		t.Fatal("regular page should not be a bundle")
	}
	if !bundlePage.IsBundle || bundlePage.Slug != "bundled-post" {
		t.Fatal("bundle page not identified correctly")
	}
	if len(bundlePage.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(bundlePage.Resources))
	}
}

func TestBundleResourcesExcludeMarkdown(t *testing.T) {
	dir := t.TempDir()
	writeSiteConfig(t, dir)

	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "index.md"), `+++
title = "Bundle With Extra MD"
date = 2026-05-01T10:00:00Z
+++

Body.
`)
	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "notes.md"), "extra notes")
	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "image.png"), "png-data")

	cfg, _, err := config.Load(config.Options{SiteRoot: dir})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	pages, _, err := Discover(dir, cfg)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if len(pages[0].Resources) != 1 {
		t.Fatalf("expected 1 resource (notes.md excluded), got %d: %v", len(pages[0].Resources), pages[0].Resources)
	}
	if pages[0].Resources[0] != "image.png" {
		t.Fatalf("expected resource image.png, got %q", pages[0].Resources[0])
	}
}

func TestBundlePageAtRoot(t *testing.T) {
	dir := t.TempDir()
	writeSiteConfig(t, dir)

	writeContentFile(t, filepath.Join(dir, "content", "index.md"), `+++
title = "Root Page"
date = 2026-05-01T10:00:00Z
+++

Root page body.
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

	page := pages[0]
	if page.IsBundle {
		t.Fatal("index.md at content root should not be a bundle")
	}
	if page.Slug != "index" {
		t.Fatalf("expected slug 'index', got %q", page.Slug)
	}
	if page.SectionPath != "" {
		t.Fatalf("expected root SectionPath, got %q", page.SectionPath)
	}
}
