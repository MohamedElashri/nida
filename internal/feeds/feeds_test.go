package feeds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
	"github.com/MohamedElashri/nida/internal/taxonomies"
)

func TestGenerateGolden(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Title = "Fixture Site"
	cfg.Description = "Fixture feed"
	cfg.Author = "Fixture Author"

	index := site.SiteIndex{
		AllPages: []content.Page{
			{
				Title:       "Hello World",
				URL:         "/posts/hello-world/",
				Description: "A fixture post for early tests.",
				Date:        mustDate(t, "2026-04-12T10:00:00Z"),
				BodyHTML:    "<p>Hello feed.</p>\n",
			},
		},
	}

	out, err := Generate(cfg, index)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "feed.golden.xml"))
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
	cfg.Feed.Limit = 1

	index := site.SiteIndex{
		AllPages: []content.Page{
			{Title: "Newest", URL: "/posts/newest/", Date: mustDate(t, "2026-04-13T10:00:00Z")},
			{Title: "Older", URL: "/posts/older/", Date: mustDate(t, "2026-04-12T10:00:00Z")},
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

func TestGenerateIncludesAllPagesWithoutSections(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"

	index := site.SiteIndex{
		AllPages: []content.Page{
			{Title: "Post", URL: "/post/post/", SectionPath: "post", Date: mustDate(t, "2026-04-13T10:00:00Z")},
			{Title: "Micro", URL: "/micro/micro/", SectionPath: "micro", Date: mustDate(t, "2026-04-12T10:00:00Z")},
		},
	}

	out, err := Generate(cfg, index)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	body := string(out.Content)
	if !contains(body, "Post") || !contains(body, "Micro") {
		t.Fatalf("expected unfiltered feed to include all pages, got %s", body)
	}
}

func TestGenerateFiltersFeedBySections(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Feed.Sections = []string{"post"}

	index := site.SiteIndex{
		AllPages: []content.Page{
			{Title: "Post", URL: "/post/post/", SectionPath: "post", Date: mustDate(t, "2026-04-13T10:00:00Z")},
			{Title: "Micro", URL: "/micro/micro/", SectionPath: "micro", Date: mustDate(t, "2026-04-12T10:00:00Z")},
		},
	}

	out, err := Generate(cfg, index)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	body := string(out.Content)
	if !contains(body, "Post") || contains(body, "Micro") {
		t.Fatalf("expected Feed feed to include only post section pages, got %s", body)
	}
}

func TestGenerateAppliesLimitAfterSectionFiltering(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Feed.Limit = 2
	cfg.Feed.Sections = []string{"post"}

	index := site.SiteIndex{
		AllPages: []content.Page{
			{Title: "Newest Micro", URL: "/micro/newest/", SectionPath: "micro", Date: mustDate(t, "2026-04-14T10:00:00Z")},
			{Title: "Newest Post", URL: "/post/newest/", SectionPath: "post", Date: mustDate(t, "2026-04-13T10:00:00Z")},
			{Title: "Older Post", URL: "/post/older/", SectionPath: "post", Date: mustDate(t, "2026-04-12T10:00:00Z")},
			{Title: "Oldest Post", URL: "/post/oldest/", SectionPath: "post", Date: mustDate(t, "2026-04-11T10:00:00Z")},
		},
	}

	out, err := Generate(cfg, index)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	body := string(out.Content)
	if !contains(body, "Newest Post") || !contains(body, "Older Post") || contains(body, "Newest Micro") || contains(body, "Oldest Post") {
		t.Fatalf("expected Feed limit to apply after section filtering, got %s", body)
	}
}

func TestGenerateFiltersNestedSectionsByRoot(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Feed.Sections = []string{"post"}

	index := site.SiteIndex{
		AllPages: []content.Page{
			{Title: "Nested Post", URL: "/post/notes/nested/", SectionPath: "post/notes", Date: mustDate(t, "2026-04-13T10:00:00Z")},
			{Title: "Micro", URL: "/micro/micro/", SectionPath: "micro", Date: mustDate(t, "2026-04-12T10:00:00Z")},
		},
	}

	out, err := Generate(cfg, index)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	body := string(out.Content)
	if !contains(body, "Nested Post") || contains(body, "Micro") {
		t.Fatalf("expected nested post section to match root section filter, got %s", body)
	}
}

func TestGenerateAllReturnsEnabledFeeds(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.RSS.Enabled = true

	index := site.SiteIndex{
		AllPages: []content.Page{{Title: "Post", URL: "/posts/post/"}},
	}

	outputs, err := GenerateAll(cfg, index)
	if err != nil {
		t.Fatalf("GenerateAll returned error: %v", err)
	}
	if len(outputs) != 2 || outputs[0].Filename != "feed.xml" || outputs[1].Filename != "rss.xml" {
		t.Fatalf("expected feed.xml and rss.xml, got %+v", outputs)
	}
}

func TestGenerateDisabledFeed(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.Feed.Enabled = false

	out, err := Generate(cfg, site.SiteIndex{})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil output when disabled, got %+v", out)
	}
}

func TestGenerateRSSGolden(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Title = "Fixture Site"
	cfg.Description = "Fixture feed"
	cfg.Author = "Fixture Author"

	index := site.SiteIndex{
		AllPages: []content.Page{
			{
				Title:       "Hello World",
				URL:         "/posts/hello-world/",
				Description: "A fixture post for early tests.",
				Date:        mustDate(t, "2026-04-12T10:00:00Z"),
				Updated:     mustDate(t, "2026-04-14T10:00:00Z"),
				BodyHTML:    "<p>Hello <a href=\"/about/\">feed</a>.</p>\n",
				Extra:       map[string]any{"tags": []any{"go"}},
			},
		},
	}

	cfg.RSS.Enabled = true
	out, err := GenerateRSS(cfg, index)
	if err != nil {
		t.Fatalf("GenerateRSS returned error: %v", err)
	}
	assertGolden(t, "rss.golden.xml", out.Content)
}

func TestAtomUsesUpdatedDate(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	index := site.SiteIndex{AllPages: []content.Page{{
		Title:   "Edited",
		URL:     "/posts/edited/",
		Date:    mustDate(t, "2026-04-12T10:00:00Z"),
		Updated: mustDate(t, "2026-05-01T10:00:00Z"),
	}}}

	out, err := Generate(cfg, index)
	if err != nil {
		t.Fatalf("GenerateAtom returned error: %v", err)
	}
	body := string(out.Content)
	if !strings.Contains(body, "<published>2026-04-12T10:00:00Z</published>") || !strings.Contains(body, "<updated>2026-05-01T10:00:00Z</updated>") {
		t.Fatalf("expected published and updated dates to differ, got %s", body)
	}
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("golden mismatch\nwant:\n%s\ngot:\n%s", want, got)
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

func TestResolveURLUnderSubpath(t *testing.T) {
	base := "https://example.com/docs/"
	cases := map[string]string{
		"/docs/guide/":  "https://example.com/docs/guide/",
		"shot.png":      "https://example.com/docs/posts/a/shot.png",
		"../b/":         "https://example.com/docs/posts/b/",
		"#top":          "#top",
		"https://x.org": "https://x.org",
	}
	for raw, want := range cases {
		if got := resolveURL(base, "/posts/a/", raw); got != want {
			t.Errorf("resolveURL(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestGenerateAllWritesSectionAndTermFeeds(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/"
	cfg.Title = "Site"
	cfg.RSS.Enabled = true
	cfg.Feed.Limit = 1

	index := site.SiteIndex{
		AllPages: []content.Page{
			{Title: "Nested", URL: "/posts/2026/b/", SectionPath: "posts/2026", Date: mustDate(t, "2026-04-13T10:00:00Z")},
			{Title: "Post", URL: "/posts/a/", SectionPath: "posts", Date: mustDate(t, "2026-04-12T10:00:00Z")},
			{Title: "Note", URL: "/notes/c/", SectionPath: "notes", Date: mustDate(t, "2026-04-14T10:00:00Z")},
		},
		SectionLookup: map[string]content.Section{
			"posts": {Title: "Posts", SectionPath: "posts", GenerateFeeds: true},
			"notes": {Title: "Notes", SectionPath: "notes"},
		},
		Taxonomies: []taxonomies.Collection{{
			Name: "tags",
			Feed: true,
			Terms: []taxonomies.Term{{
				Name: "go", URL: "/tags/go/", CanonicalURL: "https://example.com/tags/go/",
				Items: []content.Page{{Title: "Post", URL: "/posts/a/"}},
			}},
		}},
	}

	outputs, err := GenerateAll(cfg, index)
	if err != nil {
		t.Fatalf("GenerateAll returned error: %v", err)
	}
	byName := map[string]string{}
	for _, out := range outputs {
		byName[out.Filename] = string(out.Content)
	}
	for _, name := range []string{"feed.xml", "rss.xml", "posts/feed.xml", "posts/rss.xml", "tags/go/feed.xml", "tags/go/rss.xml"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("missing %s in %v", name, Filenames(cfg, index))
		}
	}
	posts := byName["posts/feed.xml"]
	if !strings.Contains(posts, "<title>Site - Posts</title>") || !strings.Contains(posts, "Nested") || strings.Contains(posts, "Note") {
		t.Fatalf("unexpected section feed:\n%s", posts)
	}
	if strings.Contains(posts, "<title>Post</title>") {
		t.Fatalf("section feed should respect feed.limit:\n%s", posts)
	}
	if !strings.Contains(byName["posts/feed.xml"], `href="https://example.com/posts/feed.xml" rel="self"`) {
		t.Fatalf("expected self link for section feed:\n%s", posts)
	}
	if len(Filenames(cfg, index)) != len(outputs) {
		t.Fatalf("Filenames and GenerateAll disagree: %v vs %d outputs", Filenames(cfg, index), len(outputs))
	}
}

func TestFeedsKeepAngleBracketsInTitles(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.RSS.Enabled = true
	index := site.SiteIndex{AllPages: []content.Page{{
		Title: "Why 1 < 2 and  3 > 2",
		URL:   "/posts/compare/",
		Date:  mustDate(t, "2026-04-12T10:00:00Z"),
	}}}

	outputs, err := GenerateAll(cfg, index)
	if err != nil {
		t.Fatalf("GenerateAll returned error: %v", err)
	}
	for _, out := range outputs {
		if !strings.Contains(string(out.Content), "<title>Why 1 &lt; 2 and 3 &gt; 2</title>") {
			t.Fatalf("%s lost the title text:\n%s", out.Filename, out.Content)
		}
	}
}
