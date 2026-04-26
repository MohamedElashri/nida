package sitemap

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/site"
)

func TestGenerateGolden(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"

	state := site.State{
		Index: site.SiteIndex{
			AllPages: []content.Page{
				{URL: "/posts/hello-world/", Date: mustDate(t, "2026-04-12T10:00:00Z")},
				{URL: "/about/", Date: mustDate(t, "2026-04-12T10:30:00Z")},
			},
		},
	}

	pages := []render.Page{
		{URL: "/", CanonicalURL: "https://example.com/"},
		{URL: "/about/", CanonicalURL: "https://example.com/about/"},
		{URL: "/posts/hello-world/", CanonicalURL: "https://example.com/posts/hello-world/"},
		{URL: "/tags/intro/", CanonicalURL: "https://example.com/tags/intro/"},
	}

	out, err := Generate(cfg, state, pages)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "sitemap.golden.xml"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(out.Content) != string(want) {
		t.Fatalf("golden mismatch\nwant:\n%s\ngot:\n%s", string(want), string(out.Content))
	}
}

func TestGenerateDisabledSitemap(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.Sitemap.Enabled = false

	out, err := Generate(cfg, site.State{}, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil output when disabled, got %+v", out)
	}
}

func TestGenerateDeduplicatesCanonicalURLs(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"

	out, err := Generate(cfg, site.State{}, []render.Page{
		{URL: "/", CanonicalURL: "https://example.com/"},
		{URL: "/", CanonicalURL: "https://example.com/"},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	got := string(out.Content)
	if countOccurrences(got, "<url>") != 1 {
		t.Fatalf("expected exactly 1 url entry, got %s", got)
	}
}

func countOccurrences(s, sub string) int {
	count := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			count++
		}
	}
	return count
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return parsed
}
