package site

import (
	"os"
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

func TestLoadRunsContentDiagnostics(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "content", "posts"), 0o755); err != nil {
		t.Fatalf("mkdir content: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "_index.md"), []byte("+++\ntitle = \"Home\"\n+++\n"), 0o644); err != nil {
		t.Fatalf("write root section: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "posts", "hello.md"), []byte("+++\ntitle = \"Hello\"\n+++\n\n[Missing](@/posts/missing.md)\n"), 0o644); err != nil {
		t.Fatalf("write page: %v", err)
	}

	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Title = "Diagnostics"
	cfg.Diagnostics.Enabled = true

	_, err := Load(dir, cfg)
	if err == nil {
		t.Fatal("expected diagnostics error")
	}
	if !strings.Contains(err.Error(), "broken internal link") {
		t.Fatalf("expected broken internal link diagnostic, got %v", err)
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

	if route != "/hello/" {
		t.Fatalf("expected /hello/, got %q", route)
	}
}

func TestExtractTaxonomyValue(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]any
		tax   string
		want  string
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

func TestBuildIndexExcludesFutureDatedPages(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	pages := []content.Page{
		{RelativePath: "posts/past.md", Slug: "past", Date: mustDate(t, "2020-01-01T00:00:00Z"), SectionPath: "posts"},
		{RelativePath: "posts/later.md", Slug: "later", Date: time.Now().Add(48 * time.Hour), SectionPath: "posts"},
	}

	index, _, err := BuildIndex(pages, nil, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}
	if len(index.AllPages) != 1 || index.AllPages[0].Slug != "past" {
		t.Fatalf("expected only the past page, got %+v", index.AllPages)
	}

	cfg.Future = true
	index, _, err = BuildIndex(pages, nil, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}
	if len(index.AllPages) != 2 {
		t.Fatalf("expected future page with future enabled, got %d pages", len(index.AllPages))
	}
}

func TestBuildPathLookupSkipsUnpublishedPages(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	pages := []content.Page{
		{RelativePath: "posts/past.md", Slug: "past", Date: mustDate(t, "2020-01-01T00:00:00Z"), SectionPath: "posts"},
		{RelativePath: "posts/draft.md", Slug: "draft", Draft: true, SectionPath: "posts"},
		{RelativePath: "posts/later.md", Slug: "later", Date: time.Now().Add(48 * time.Hour), SectionPath: "posts"},
	}
	lookup := BuildPathLookup(pages, nil, cfg)
	if lookup["posts/past.md"] == "" || lookup["posts/draft.md"] != "" || lookup["posts/later.md"] != "" {
		t.Fatalf("expected only the published page, got %v", lookup)
	}
	cfg.Drafts, cfg.Future = true, true
	if lookup := BuildPathLookup(pages, nil, cfg); len(lookup) != 6 {
		t.Fatalf("expected every page with drafts and future on, got %v", lookup)
	}
}

func TestIndexGetPageAndSection(t *testing.T) {
	index := SiteIndex{
		AllPages: []content.Page{
			{RelativePath: "posts/hello.md", Title: "Hello"},
			{RelativePath: "posts/bundle/index.md", Title: "Bundle"},
		},
		SectionLookup: map[string]content.Section{
			"":      {Title: "Home"},
			"posts": {Title: "Posts"},
		},
	}
	for _, path := range []string{"posts/hello.md", "@/posts/hello.md", "posts/hello"} {
		if page := index.GetPage(path); page == nil || page.Title != "Hello" {
			t.Fatalf("GetPage(%q) = %+v", path, page)
		}
	}
	if page := index.GetPage("posts/bundle"); page == nil || page.Title != "Bundle" {
		t.Fatalf("expected bundle lookup, got %+v", page)
	}
	if index.GetPage("missing.md") != nil {
		t.Fatal("expected nil for a missing page")
	}
	if section := index.GetSection("posts/_index.md"); section == nil || section.Title != "Posts" {
		t.Fatalf("unexpected section %+v", section)
	}
	if section := index.GetSection(""); section == nil || section.Title != "Home" {
		t.Fatalf("unexpected root section %+v", section)
	}
}

func TestNavInterleavesPagesAndSectionsByWeight(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Permalinks = config.PermalinkConfig{}
	pages := []content.Page{
		{RelativePath: "guide/intro.md", Slug: "intro", Title: "Intro", SectionPath: "guide", Weight: 1},
		{RelativePath: "guide/later.md", Slug: "later", Title: "Later", SectionPath: "guide", Weight: 30},
		{RelativePath: "guide/deep/a.md", Slug: "a", Title: "Deep A", SectionPath: "guide/deep", Weight: 1},
	}
	sections := []content.Section{
		{SectionPath: "", Title: "Home", SortBy: "weight"},
		{SectionPath: "guide", Title: "Guide", SortBy: "weight", Weight: 1},
		{SectionPath: "guide/deep", Title: "Deep", SortBy: "weight", Weight: 20},
	}
	index, _, err := BuildIndex(pages, sections, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	guide := index.Nav("guide")
	var titles []string
	for _, entry := range guide {
		titles = append(titles, entry.Title)
	}
	if strings.Join(titles, ",") != "Intro,Deep,Later" {
		t.Fatalf("unexpected order %v", titles)
	}
	if !guide[1].IsSection() || guide[1].URL != "/guide/deep/" || len(guide[1].Children) != 1 {
		t.Fatalf("unexpected section entry %+v", guide[1])
	}

	n := index.Neighbors("", "/guide/deep/")
	if n.Prev == nil || n.Prev.Title != "Intro" || n.Next == nil || n.Next.Title != "Deep A" {
		t.Fatalf("unexpected neighbors %+v %+v", n.Prev, n.Next)
	}
	if n := index.Neighbors("", "/guide/"); n.Prev != nil || n.Next == nil || n.Next.Title != "Intro" {
		t.Fatalf("unexpected neighbors at the start: %+v", n)
	}
	if n := index.Neighbors("", "/guide/later/"); n.Next != nil {
		t.Fatalf("expected no next entry at the end, got %+v", n.Next)
	}
}

func TestBuildIndexSetsPrevNextOnRenderedPages(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	pages := []content.Page{
		{RelativePath: "posts/a.md", Slug: "a", Title: "A", SectionPath: "posts", Date: mustDate(t, "2026-01-01T00:00:00Z")},
		{RelativePath: "posts/b.md", Slug: "b", Title: "B", SectionPath: "posts", Date: mustDate(t, "2026-01-02T00:00:00Z")},
	}
	sections := []content.Section{{SectionPath: "posts", Title: "Posts", SortBy: "date"}}
	index, sorted, err := BuildIndex(pages, sections, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}
	byPath := map[string]content.Page{}
	for _, p := range sorted {
		byPath[p.RelativePath] = p
	}
	if byPath["posts/a.md"].NextTitle != "B" || byPath["posts/b.md"].PrevTitle != "A" {
		t.Fatalf("prev/next missing on rendered pages: %+v", byPath)
	}
	if index.SectionLookup["posts"].Pages[0].PrevTitle != "A" {
		t.Fatalf("prev/next missing on section pages: %+v", index.SectionLookup["posts"].Pages)
	}
}

func TestBuildIndexRegistersSynthesizedRootSection(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	pages := []content.Page{{RelativePath: "about.md", Slug: "about", Title: "About"}}
	index, _, err := BuildIndex(pages, nil, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}
	if _, ok := index.SectionLookup[""]; !ok {
		t.Fatal("expected a root section so the home page renders")
	}
}
