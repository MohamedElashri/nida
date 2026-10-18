package migrate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/site"
)

// copyDir copies a fixture directory tree for an end-to-end test.
func copyDir(t *testing.T, from, to string) {
	t.Helper()
	for name, body := range readTree(t, from) {
		writeTree(t, to, map[string]string{name: body})
	}
}

// TestHugoEndToEndWithTranslatedTheme runs a real Hugo config (menus,
// params, copyright) through the Hugo source and the layout translator,
// and checks the rendered pages, which the separate source and
// translator tests cannot see together.
func TestHugoEndToEndWithTranslatedTheme(t *testing.T) {
	src := t.TempDir()
	fixture := filepath.Join("testdata", "hugo-templates", "site")
	copyDir(t, filepath.Join(fixture, "layouts"), filepath.Join(src, "layouts"))
	copyDir(t, filepath.Join(fixture, "content"), filepath.Join(src, "content"))
	writeTree(t, src, map[string]string{"hugo.toml": `baseURL = "https://example.org/blog/"
title = "Hugo Theme Fixture"
copyright = "Some <em>rights</em>"

[pagination]
pagerSize = 2

[params]
description = "A site description."
tagline = "Notes *and* things"
[params.author]
name = "Ada"
[params.social]
github = "https://github.com/ada"

[[menus.main]]
name = "Posts"
pageRef = "/posts"
weight = 1

[[menus.main]]
name = "About"
url = "/about/"
weight = 2
`})

	dest := filepath.Join(t.TempDir(), "out")
	result, err := Run(Options{Source: src, Dest: dest, From: "hugo", Check: checkSite})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.Report.TemplateSet, "translated") {
		t.Fatalf("expected the translated templates, got %q; notes: %v", result.Report.TemplateSet, result.Report.Notes)
	}

	cfg, _, err := config.Load(config.Options{SiteRoot: dest})
	if err != nil {
		t.Fatal(err)
	}
	state, err := site.Load(dest, cfg)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := render.RenderSite(dest, cfg, state)
	if err != nil {
		t.Fatal(err)
	}
	html := map[string]string{}
	for _, page := range pages {
		html[page.URL] = page.Content
	}
	home := html["/"]
	for _, want := range []string{`/blog/posts/`, `>Posts<`, `/blog/about/`, `>About<`, "Notes <em>and</em> things", "Ada", "Some <em>rights</em>", "https://github.com/ada"} {
		if !strings.Contains(home, want) {
			t.Errorf("home page is missing %q:\n%s", want, home)
		}
	}
	post := ""
	for url, body := range html {
		if strings.HasPrefix(url, "/posts/") && url != "/posts/" && !strings.Contains(url, "/page/") {
			post = body
			break
		}
	}
	if post == "" || !strings.Contains(post, "<article") {
		t.Errorf("expected a rendered post, got %q", post)
	}
}
