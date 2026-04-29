package site

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

func TestBuildIndexSortsPagesByDate(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	pages := []content.Page{
		{RelativePath: "posts/b.md", Slug: "b", Date: mustDate(t, "2026-04-12T10:00:00Z"), SectionPath: "posts"},
		{RelativePath: "posts/a.md", Slug: "a", Date: mustDate(t, "2026-04-13T10:00:00Z"), SectionPath: "posts"},
		{RelativePath: "posts/c.md", Slug: "c", Date: mustDate(t, "2026-04-11T10:00:00Z"), SectionPath: "posts"},
	}

	index, _, err := BuildIndex(pages, nil, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	if len(index.AllPages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(index.AllPages))
	}
	if index.AllPages[0].Slug != "a" || index.AllPages[1].Slug != "b" || index.AllPages[2].Slug != "c" {
		t.Fatalf("unexpected page order: %+v", index.AllPages)
	}
}

func TestBuildIndexDetectsRouteConflicts(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	pages := []content.Page{
		{RelativePath: "posts/one.md", Slug: "same", SectionPath: "posts"},
		{RelativePath: "posts/two.md", Slug: "same", SectionPath: "posts"},
	}

	_, _, err := BuildIndex(pages, nil, cfg)
	if err == nil {
		t.Fatal("expected route conflict error")
	}
	if !strings.Contains(err.Error(), "route conflict") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildIndexPopulatesRouteRegistry(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	pages := []content.Page{
		{RelativePath: "pages/about.md", Slug: "about", SectionPath: "pages"},
	}

	index, _, err := BuildIndex(pages, nil, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	if index.RouteRegistry["/pages/about/"] != "pages/about.md" {
		t.Fatalf("unexpected route registry: %+v", index.RouteRegistry)
	}
}

func TestLoadOrchestratesDiscoveryRenderAndIndex(t *testing.T) {
	siteRoot := filepath.Join("..", "..", "example-site")
	cfg, _, err := config.Load(config.Options{SiteRoot: siteRoot})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	state, err := Load(siteRoot, cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(state.Index.AllPages) != 5 {
		t.Fatalf("expected 5 pages, got %d", len(state.Index.AllPages))
	}
	for _, p := range state.Index.AllPages {
		if p.BodyHTML == "" {
			t.Fatalf("expected rendered body in page %q", p.RelativePath)
		}
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return parsed
}

func TestRoutePageSupportsTaxonomyPlaceholders(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Taxonomies = []config.TaxonomyConfig{
		{Name: "categories"},
		{Name: "tags"},
	}
	cfg.Permalinks = config.PermalinkConfig{
		"posts": "/{categories}/{slug}/",
	}

	page := content.Page{
		RelativePath: "posts/hello.md",
		Slug:         "hello",
		SectionPath:  "posts",
		Date:         mustDate(t, "2026-04-13T10:00:00Z"),
		Extra: map[string]any{
			"categories": []string{"Tech", "Programming"},
		},
	}

	route, err := routePage(page, cfg)
	if err != nil {
		t.Fatalf("routePage returned error: %v", err)
	}

	if route != "/tech/hello/" {
		t.Fatalf("expected /tech/hello/, got %q", route)
	}
}

func TestRoutePageSupportsDatePlaceholders(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Permalinks = config.PermalinkConfig{
		"posts": "/{year}/{month}/{day}/{slug}/",
	}

	page := content.Page{
		RelativePath: "posts/hello.md",
		Slug:         "hello",
		SectionPath:  "posts",
		Date:         mustDate(t, "2026-04-13T10:00:00Z"),
	}

	route, err := routePage(page, cfg)
	if err != nil {
		t.Fatalf("routePage returned error: %v", err)
	}

	if route != "/2026/04/13/hello/" {
		t.Fatalf("expected /2026/04/13/hello/, got %q", route)
	}
}

func TestRoutePageEmptyTaxonomyFallsBack(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Taxonomies = []config.TaxonomyConfig{
		{Name: "categories"},
	}
	cfg.Permalinks = config.PermalinkConfig{
		"posts": "/{categories}/{slug}/",
	}

	page := content.Page{
		RelativePath: "posts/hello.md",
		Slug:         "hello",
		SectionPath:  "posts",
		Date:         mustDate(t, "2026-04-13T10:00:00Z"),
		Extra:        map[string]any{},
	}

	route, err := routePage(page, cfg)
	if err != nil {
		t.Fatalf("routePage returned error: %v", err)
	}

	if route != "//hello/" {
		t.Fatalf("expected //hello/, got %q", route)
	}
}

func TestExtractTaxonomyValue(t *testing.T) {
	tests := []struct {
		name string
		extra map[string]any
		tax  string
		want string
	}{
		{
			name:  "string slice",
			extra: map[string]any{"categories": []string{"Tech", "Science"}},
			tax:   "categories",
			want:  "tech",
		},
		{
			name:  "any slice",
			extra: map[string]any{"tags": []any{"Go", "Rust"}},
			tax:   "tags",
			want:  "go",
		},
		{
			name:  "single string",
			extra: map[string]any{"category": "Tech"},
			tax:   "category",
			want:  "tech",
		},
		{
			name:  "missing key",
			extra: map[string]any{},
			tax:   "categories",
			want:  "",
		},
		{
			name:  "nil extra",
			extra: nil,
			tax:   "categories",
			want:  "",
		},
	}

	for _, tt := range tests {
		got := extractTaxonomyValue(tt.extra, tt.tax)
		if got != tt.want {
			t.Errorf("%s: extractTaxonomyValue() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
