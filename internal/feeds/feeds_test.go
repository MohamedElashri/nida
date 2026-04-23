package feeds

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
)

func TestGenerateGolden(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Title = "Fixture Site"
	cfg.Description = "Fixture feed"

	index := site.SiteIndex{
		Posts: []content.Item{
			{
				Title:       "Hello World",
				URL:         "/posts/hello-world/",
				Description: "A fixture post for early tests.",
				Date:        mustDate(t, "2026-04-12T10:00:00Z"),
			},
		},
		CanonicalLookup: map[string]string{
			"/posts/hello-world/": "https://example.com/posts/hello-world/",
		},
	}

	out, err := Generate(cfg, index)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "rss.golden.xml"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(out.Content) != string(want) {
		t.Fatalf("golden mismatch\nwant:\n%s\ngot:\n%s", string(want), string(out.Content))
	}
}

func TestGenerateRespectsLimit(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.RSS.Limit = 1

	index := site.SiteIndex{
		Posts: []content.Item{
			{Title: "Newest", URL: "/posts/newest/", Date: mustDate(t, "2026-04-13T10:00:00Z")},
			{Title: "Older", URL: "/posts/older/", Date: mustDate(t, "2026-04-12T10:00:00Z")},
		},
		CanonicalLookup: map[string]string{
			"/posts/newest/": "https://example.com/posts/newest/",
			"/posts/older/":  "https://example.com/posts/older/",
		},
	}

	out, err := Generate(cfg, index)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if !contains(string(out.Content), "Newest") || contains(string(out.Content), "Older") {
		t.Fatalf("expected only newest item in feed, got %s", string(out.Content))
	}
}

func TestGenerateAtomGolden(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Title = "Fixture Site"
	cfg.Description = "Fixture feed"
	cfg.Author = "Fixture Author"
	cfg.RSS.Enabled = false
	cfg.Atom.Enabled = true

	index := site.SiteIndex{
		Posts: []content.Item{
			{
				Title:       "Hello World",
				URL:         "/posts/hello-world/",
				Description: "A fixture post for early tests.",
				Date:        mustDate(t, "2026-04-12T10:00:00Z"),
				BodyHTML:    "<p>Hello feed.</p>\n",
			},
		},
		CanonicalLookup: map[string]string{
			"/posts/hello-world/": "https://example.com/posts/hello-world/",
		},
	}

	out, err := GenerateAtom(cfg, index)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "atom.golden.xml"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(out.Content) != string(want) {
		t.Fatalf("golden mismatch\nwant:\n%s\ngot:\n%s", string(want), string(out.Content))
	}
}

func TestGenerateAllReturnsEnabledFeeds(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.RSS.Enabled = true
	cfg.Atom.Enabled = true

	index := site.SiteIndex{
		Posts: []content.Item{{Title: "Post", URL: "/posts/post/"}},
		CanonicalLookup: map[string]string{
			"/posts/post/": "https://example.com/posts/post/",
		},
	}

	outputs, err := GenerateAll(cfg, index)
	if err != nil {
		t.Fatalf("GenerateAll returned error: %v", err)
	}
	if len(outputs) != 2 {
		t.Fatalf("expected 2 feeds, got %+v", outputs)
	}
}

func TestGenerateDisabledFeed(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.RSS.Enabled = false

	out, err := Generate(cfg, site.SiteIndex{})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil output when disabled, got %+v", out)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (filepath.Base(needle) == needle && stringIndex(haystack, needle) >= 0)
}

func stringIndex(s, sep string) int {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return parsed
}
