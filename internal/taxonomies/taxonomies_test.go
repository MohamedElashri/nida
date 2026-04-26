package taxonomies

import (
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

func TestBuildAllCreatesCollections(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Taxonomies = []config.TaxonomyConfig{
		{Name: "tags", Render: true, PaginateBy: 10},
	}

	pages := []content.Page{
		{
			Slug:         "first",
			RelativePath: "posts/first.md",
			Date:         mustDate(t, "2026-04-13T10:00:00Z"),
			Extra:        map[string]any{"tags": []string{"Go Lang"}},
		},
		{
			Slug:         "second",
			RelativePath: "posts/second.md",
			Date:         mustDate(t, "2026-04-12T10:00:00Z"),
			Extra:        map[string]any{"tags": []string{"Go Lang"}},
		},
	}

	collections, taxMap, err := BuildAll(cfg, pages)
	if err != nil {
		t.Fatalf("BuildAll returned error: %v", err)
	}

	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}
	collection := collections[0]
	if collection.URL != "/tags/" {
		t.Fatalf("unexpected collection URL %q", collection.URL)
	}
	if len(collection.Terms) != 1 {
		t.Fatalf("expected 1 term, got %d", len(collection.Terms))
	}
	if collection.Terms[0].Slug != "go-lang" || collection.Terms[0].URL != "/tags/go-lang/" {
		t.Fatalf("unexpected term: %+v", collection.Terms[0])
	}
	if collection.Terms[0].Items[0].Slug != "first" {
		t.Fatalf("expected sorted term items by date desc, got %+v", collection.Terms[0].Items)
	}

	if _, ok := taxMap["tags"]["Go Lang"]; !ok {
		t.Fatalf("expected tags in taxonomy map")
	}
}

func TestBuildAllSkipsDisabledTaxonomies(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.Taxonomies = []config.TaxonomyConfig{
		{Name: "tags", Render: false},
	}

	pages := []content.Page{
		{
			Slug:         "first",
			RelativePath: "posts/first.md",
			Extra:        map[string]any{"tags": []string{"news"}},
		},
	}

	collections, _, err := BuildAll(cfg, pages)
	if err != nil {
		t.Fatalf("BuildAll returned error: %v", err)
	}
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection in slice, got %d", len(collections))
	}
	if collections[0].Name != "tags" {
		t.Fatalf("expected tags collection, got %q", collections[0].Name)
	}
	if collections[0].Render {
		t.Fatalf("expected tags collection to have Render=false")
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
