package site

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

func TestResolvePermalink(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"

	got, err := ResolvePermalink(content.Item{
		Type:         content.TypePost,
		Slug:         "hello-world",
		RelativePath: "posts/hello-world.md",
	}, cfg)
	if err != nil {
		t.Fatalf("ResolvePermalink returned error: %v", err)
	}

	if got != "/posts/hello-world/" {
		t.Fatalf("expected /posts/hello-world/, got %q", got)
	}
}

func TestBuildIndexFiltersDraftsAndSortsPosts(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	items := []content.Item{
		{Type: content.TypePost, RelativePath: "posts/b.md", Slug: "b", Date: mustDate(t, "2026-04-12T10:00:00Z")},
		{Type: content.TypePost, RelativePath: "posts/a.md", Slug: "a", Date: mustDate(t, "2026-04-13T10:00:00Z")},
		{Type: content.TypePost, RelativePath: "posts/draft.md", Slug: "draft", Date: mustDate(t, "2026-04-14T10:00:00Z"), Draft: true},
		{Type: content.TypePage, RelativePath: "pages/about.md", Slug: "about"},
	}

	index, err := BuildIndex(items, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	if len(index.Posts) != 2 {
		t.Fatalf("expected 2 public posts, got %d", len(index.Posts))
	}
	if index.Posts[0].Slug != "a" || index.Posts[1].Slug != "b" {
		t.Fatalf("unexpected post order: %+v", index.Posts)
	}
	if len(index.Pages) != 1 || index.Pages[0].URL != "/about/" {
		t.Fatalf("unexpected pages: %+v", index.Pages)
	}
}

func TestBuildIndexDetectsRouteConflicts(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	items := []content.Item{
		{Type: content.TypePost, RelativePath: "posts/one.md", Slug: "same"},
		{Type: content.TypePost, RelativePath: "posts/two.md", Slug: "same"},
	}

	_, err := BuildIndex(items, cfg)
	if err == nil {
		t.Fatal("expected route conflict error")
	}
	if !strings.Contains(err.Error(), "route conflict") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildIndexCanonicalLookup(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	items := []content.Item{
		{Type: content.TypePage, RelativePath: "pages/about.md", Slug: "about"},
	}

	index, err := BuildIndex(items, cfg)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	got := index.CanonicalLookup["/about/"]
	if got != "https://example.com/about/" {
		t.Fatalf("unexpected canonical URL %q", got)
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

	if len(state.Index.Posts) != 3 || state.Index.Posts[0].BodyHTML == "" {
		t.Fatalf("expected rendered post body in state index, got %+v", state.Index.Posts)
	}
	if state.Index.CanonicalLookup["/posts/launching-nida/"] != "https://nida.blog/posts/launching-nida/" {
		t.Fatalf("unexpected canonical lookup: %+v", state.Index.CanonicalLookup)
	}
	if len(state.Index.Tags.Terms) == 0 || state.Index.Tags.Terms[0].URL == "" {
		t.Fatalf("unexpected tags collection: %+v", state.Index.Tags)
	}
}
