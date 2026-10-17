package templates

import (
	"html/template"
	"strings"
	"testing"

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
