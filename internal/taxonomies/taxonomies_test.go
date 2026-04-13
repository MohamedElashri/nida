package taxonomies

import (
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

func TestBuildCollection(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"

	collection, err := Build("tags", true, cfg.Permalinks.Tags, "/tags/", map[string][]content.Item{
		"Go Lang": {
			{Slug: "second", RelativePath: "posts/second.md", Date: mustDate(t, "2026-04-12T10:00:00Z")},
			{Slug: "first", RelativePath: "posts/first.md", Date: mustDate(t, "2026-04-13T10:00:00Z")},
		},
	}, cfg)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

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
		t.Fatalf("expected sorted term items, got %+v", collection.Terms[0].Items)
	}
}

func TestBuildDisabledCollection(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	collection, err := Build("tags", false, cfg.Permalinks.Tags, "/tags/", map[string][]content.Item{}, cfg)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if collection.Name != "" || len(collection.Terms) != 0 {
		t.Fatalf("expected zero collection when disabled, got %+v", collection)
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
