package templates

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

func execute(t *testing.T, cfg config.SiteConfig, src string, data any) string {
	t.Helper()
	tmpl, err := template.New("t").Funcs(funcMap(t.TempDir(), cfg)).Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return b.String()
}

func TestTemplateCollectionFuncs(t *testing.T) {
	pages := []content.Page{
		{Title: "A", SectionPath: "posts", Extra: map[string]any{"tags": []any{"go", "web"}}},
		{Title: "B", SectionPath: "notes", Extra: map[string]any{"tags": []any{"go"}}},
		{Title: "C", SectionPath: "posts", Extra: map[string]any{"series": "intro"}},
	}
	cfg := config.DefaultSiteConfig()
	cases := map[string]string{
		`{{ range where . "SectionPath" "posts" }}{{ .Title }}{{ end }}`:           "AC",
		`{{ range where . "Extra.tags" "go" }}{{ .Title }}{{ end }}`:               "AB",
		`{{ range where . "Extra.series" "!=" "intro" }}{{ .Title }}{{ end }}`:     "AB",
		`{{ range first 2 . }}{{ .Title }}{{ end }}`:                               "AB",
		`{{ range last 1 . }}{{ .Title }}{{ end }}`:                                "C",
		`{{ range first 9 (where . "SectionPath" "posts") }}{{ .Title }}{{ end }}`: "AC",
		`{{ with dict "name" "x" "n" 2 }}{{ .name }}{{ .n }}{{ end }}`:             "x2",
		`{{ range slice "a" "b" }}{{ . }}{{ end }}`:                                "ab",
		`{{ range seq 3 }}{{ . }}{{ end }}`:                                        "123",
	}
	for src, want := range cases {
		if got := execute(t, cfg, src, pages); got != want {
			t.Errorf("%s: got %q, want %q", src, got, want)
		}
	}
}

func TestTemplateStringFuncs(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/blog/"
	cases := map[string]string{
		`{{ truncate 12 "The quick brown fox" }}`:     "The quick…",
		`{{ truncate 50 "short" }}`:                   "short",
		`{{ plainify "<p>Hi &amp; <b>bye</b></p>" }}`: "Hi &amp; bye",
		`{{ markdownify "Use *this*" }}`:              "Use <em>this</em>",
		`{{ absURL "posts/a/" }}`:                     "https://example.com/blog/posts/a/",
		`{{ relURL "/posts/a/" }}`:                    "/blog/posts/a/",
		`{{ relURL "https://other.org/x" }}`:          "https://other.org/x",
		`{{ title "hello wide world" }}`:              "Hello Wide World",
	}
	for src, want := range cases {
		if got := execute(t, cfg, src, nil); got != want {
			t.Errorf("%s: got %q, want %q", src, got, want)
		}
	}
}

func TestJsonifyInScript(t *testing.T) {
	got := execute(t, config.DefaultSiteConfig(), `<script type="application/ld+json">{{ jsonify (dict "headline" "</script>") }}</script>`, nil)
	if !strings.Contains(got, `{"headline":"\u003c/script\u003e"}`) {
		t.Fatalf("expected escaped JSON, got %s", got)
	}
}

func TestDictRejectsOddArguments(t *testing.T) {
	if _, err := dict("only-key"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSEOPartial(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"base.html": `{{ define "head" }}{{ template "nida-seo" . }}{{ end }}`,
		"page.html": `{{ define "page" }}{{ template "head" . }}{{ end }}`,
	} {
		if err := os.WriteFile(filepath.Join(root, "templates", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/blog/"
	cfg.Title = "Site"
	cfg.Author = "Ada"
	set, err := Load(root, cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	type ctx struct {
		Title, Description, CurrentURL, CanonicalURL, Robots string
		Config                                               config.SiteConfig
		Page                                                 content.Page
	}
	page := content.Page{
		RelativePath: "posts/a/index.md",
		Title:        "Hello & welcome",
		URL:          "/posts/a/",
		Date:         time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC),
		Extra:        map[string]any{"image": "cover.png"},
	}
	got, err := set.Execute("page", ctx{
		Title: page.Title, Description: "About things", CurrentURL: page.URL,
		CanonicalURL: "https://example.com/blog/posts/a/", Config: cfg, Page: page,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	for _, want := range []string{
		`<title>Hello &amp; welcome | Site</title>`,
		`<meta property="og:type" content="article">`,
		`<meta property="og:image" content="https://example.com/blog/posts/a/cover.png">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<meta property="article:published_time" content="2026-04-01T09:00:00Z">`,
		`"@type":"BlogPosting"`,
		`"author":{"@type":"Person","name":"Ada"}`,
		`"headline":"Hello \u0026 welcome"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in\n%s", want, got)
		}
	}
}

func TestTranslateFunc(t *testing.T) {
	root := t.TempDir()
	for rel, body := range map[string]string{
		"i18n/en.toml":        "read_more = \"Read more\"\nposts = \"{count} posts\"\n",
		"templates/base.html": "",
		"templates/page.html": `{{ define "page" }}{{ t "read_more" }}|{{ t "posts" (dict "count" 2) }}|{{ t "nope" }}{{ end }}`,
	} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	set, err := Load(root, config.DefaultSiteConfig())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	got, err := set.Execute("page", nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got != "Read more|2 posts|nope" {
		t.Fatalf("unexpected output %q", got)
	}
}
