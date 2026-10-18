package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/site"
)

// hugoTemplatePlan plans a fixture whose content and config are already
// in Nida's format, so the test exercises only the layout translation.
func hugoTemplatePlan(t *testing.T, fixture string) *Plan {
	t.Helper()
	root, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	p := newPlan(root, &Report{From: "Hugo"})
	data, err := os.ReadFile(filepath.Join(root, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := toml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	for _, key := range sortedKeys(cfg) {
		switch v := cfg[key].(type) {
		case map[string]any:
			for _, k := range sortedKeys(v) {
				p.Config.Table(key).Set(k, v[k])
			}
		case []any:
			for _, item := range v {
				m := item.(map[string]any)
				tax := p.Config.Taxonomy(m["name"].(string))
				for _, k := range sortedKeys(m) {
					tax.Set(k, m[k])
				}
			}
		default:
			p.Config.Set(key, v)
		}
	}
	if err := p.Walk("content", func(rel string) error {
		data, err := p.ReadFile(rel)
		if err != nil {
			return err
		}
		doc, err := ParseDocument(data)
		if err != nil {
			return err
		}
		p.Page(rel, doc, rel, "")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return p
}

// buildHugoTemplates translates the fixture's layouts, lets the core pick
// a template set, and renders the result, keyed by URL.
func buildHugoTemplates(t *testing.T, fixture string) (*Plan, map[string]string, string) {
	t.Helper()
	p := hugoTemplatePlan(t, fixture)
	planHugoTemplates(p, []string{"layouts"})
	if os.Getenv("HUGO_DUMP") != "" {
		for _, name := range sortedKeys(p.candidates) {
			t.Logf("=== %s\n%s", name, p.candidates[name])
		}
	}
	if err := finishPlan(p); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := p.apply(dest); err != nil {
		t.Fatal(err)
	}
	if err := chooseTemplates(p, dest, hugoSource{}, checkSite); err != nil {
		t.Fatal(err)
	}
	for _, n := range p.Report.Notes {
		if strings.Contains(n.Message, "did not build") {
			t.Log(n.Message)
		}
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
		t.Fatalf("render: %v", err)
	}
	out := map[string]string{}
	for _, page := range pages {
		out[page.URL] = page.Content
	}
	return p, out, dest
}

func TestHugoTemplatesAreChosenAndRender(t *testing.T) {
	p, pages, dest := buildHugoTemplates(t, filepath.Join("testdata", "hugo-templates", "site"))
	if !strings.HasPrefix(p.Report.TemplateSet, "translated from Hugo") {
		for _, n := range p.Report.Notes {
			t.Log(n)
		}
		t.Fatalf("translated templates were not used: %s", p.Report.TemplateSet)
	}

	checks := map[string][]string{
		"/": {
			`<body class="home">`,
			`<title>Hugo Theme Fixture</title>`,
			`Hello from the <strong>home</strong> page.`,
			`<a href="https://example.org/blog/posts/post-3/">POST 3</a> <span>2024</span>`,
			`<h3 class="year">2024</h3>`,
			`<a class="brand" href="/blog/">Hugo Theme Fixture</a>`,
			`<a class="" href="/blog/posts/">Posts</a>`,
			`<p class="tagline">Notes <em>and</em> things</p>`,
			`&copy; `, `Ada. Some <em>rights</em>`,
			`<a href="https://github.com/ada">Github</a>`,
			`Mastodon Home`,
			`<p class="home-only">Welcome home.</p>`,
			`<meta name="generator" content="Nida">`,
			`href="/blog/scss/main.css"`,
			`href="https://example.org/blog/feed.xml"`,
		},
		"/posts/post-1/": {
			`<body class="inner">`,
			`<title>Post 1 | Hugo Theme Fixture</title>`,
			`<link rel="canonical" href="https://example.org/blog/posts/post-1/">`,
			`posts-layout`,
			`<h2 id="heading-1">Heading 1</h2>`,
			`<aside class="note note-warning">`,
			`<strong>Careful</strong>`,
			`Inner <strong>markdown</strong> 1.`,
			`on Post 1 at Hugo Theme Fixture`,
			`<p class="series">Part of Basics</p>`,
		},
		"/about/": {
			`<article class="post">`,
			`<h1>About</h1>`,
			`<div class="body"><p>About this site.</p>`,
		},
		"/posts/": {
			`<section class="list">`,
			`All the posts.`,
			`<article class="card">`,
			`<h2><a href="/blog/posts/post-3/">Post 3</a></h2>`,
			`<time datetime="2024-03-10">March 10, 2024</time>`,
			`<li><a href="/blog/tags/go/">go</a></li>`,
			`<ul class="pagination">`,
			`<a class="page-link" href="/blog/posts/page/2/">2</a>`,
		},
		"/tags/": {
			`<a href="/blog/tags/go/">go</a> (3)`,
		},
		"/tags/go/": {
			`<h1 class="term">go</h1>`,
			`<p class="term-page"><a href="/blog/posts/post-1/">Post 1</a></p>`,
		},
		"/404.html": {
			`<h1>Lost?</h1>`,
			`Back to Hugo Theme Fixture`,
		},
	}
	for url, wants := range checks {
		html, ok := pages[url]
		if !ok {
			t.Errorf("no page at %s", url)
			continue
		}
		for _, want := range wants {
			if !strings.Contains(html, want) {
				t.Errorf("%s: missing %q in:\n%s", url, want, html)
			}
		}
	}

	// The section-specific layout is attached to its section.
	section, err := os.ReadFile(filepath.Join(dest, "content", "posts", "_index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(section), `page_template = "posts-single"`) {
		t.Errorf("posts/_index.md does not use the posts layout:\n%s", section)
	}

	// Untranslatable constructs are reported with their lines.
	wantNotes := []struct {
		sev  Severity
		file string
		text string
	}{
		{Manual, "layouts/_default/single.html", "Scratch"},
		{Review, "layouts/_default/single.html", ".Related"},
		{Review, "layouts/partials/head.html", "opengraph"},
		{Review, "layouts/partials/head.html", "Hugo Pipes asset"},
		{Unsupported, "layouts/_default/_markup/render-link.html", "Render hooks"},
		{Unsupported, "layouts/index.json", "output formats"},
	}
	for _, want := range wantNotes {
		found := false
		for _, n := range p.Report.Notes {
			if n.Severity == want.sev && n.File == want.file && strings.Contains(n.Message, want.text) {
				found = true
				if want.sev != Unsupported && n.Line == 0 {
					t.Errorf("note %q has no line", n.Message)
				}
			}
		}
		if !found {
			t.Errorf("missing %s note about %q in %s; notes: %v", want.sev, want.text, want.file, p.Report.Notes)
		}
	}
}

func TestHugoUntranslatableLayoutsStillBuild(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"config.toml":       "base_url = \"https://x.org\"\ntitle = \"T\"\n",
		"content/_index.md": "+++\ntitle = \"Home\"\n+++\n",
		"content/a.md":      "+++\ntitle = \"A\"\n+++\nbody\n",
		"layouts/_default/baseof.html": `<html><body>{{ block "main" . }}{{ end }}` +
			`{{ $x := getJSON "https://example.com/x.json" }}{{ range $x }}{{ .name }}{{ end }}` +
			`{{ .Site.Data.foo.bar }}{{ with .Site.GetPage "/missing" }}{{ .Title }}{{ end }}</body></html>`,
		"layouts/_default/single.html": `{{ define "main" }}{{ .NoSuchMethod.Deeper }}{{ partial "nope.html" . }}` +
			`{{ $v := partial "fn.html" . }}{{ mul 2 3 }}{{ range where .Site.Pages "Type" "in" (slice "posts") }}{{ .Title }}{{ end }}` +
			`<p>{{ .Title }}</p>{{ end }}`,
		"layouts/partials/fn.html": `{{ return 3 }}`,
	})
	_, pages, _ := buildHugoTemplates(t, root)
	if html := pages["/a/"]; !strings.Contains(html, "<p>A</p>") {
		t.Errorf("page did not render with fallbacks: %q", html)
	}
}

func TestHugoBrokenTranslationFallsBackToStarter(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"config.toml":                  "base_url = \"https://x.org\"\ntitle = \"T\"\n",
		"content/_index.md":            "+++\ntitle = \"Home\"\n+++\n",
		"content/a.md":                 "+++\ntitle = \"A\"\n+++\nbody\n",
		"layouts/_default/baseof.html": `<html><body>{{ block "main" . }}{{ end }}</body></html>`,
		// Valid Go template syntax that fails when it runs: index into a
		// string with a non-integer.
		"layouts/_default/single.html": `{{ define "main" }}{{ index "abc" "x" }}{{ end }}`,
	})
	p, pages, dest := buildHugoTemplates(t, root)
	if strings.HasPrefix(p.Report.TemplateSet, "translated") {
		t.Fatalf("a failing translation must not be used")
	}
	if !strings.Contains(pages["/a/"], "nida-migrate.css") {
		t.Errorf("expected the starter templates")
	}
	if _, err := os.Stat(filepath.Join(dest, translatedDir, "post.html")); err != nil {
		t.Errorf("translations must be kept for reference: %v", err)
	}
}

func TestHugoLayoutThatDoesNotParse(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"config.toml":                  "base_url = \"https://x.org\"\ntitle = \"T\"\n",
		"content/_index.md":            "+++\ntitle = \"Home\"\n+++\n",
		"content/a.md":                 "+++\ntitle = \"A\"\n+++\nbody\n",
		"layouts/_default/single.html": `{{ if .Title }}unclosed`,
	})
	p, pages, _ := buildHugoTemplates(t, root)
	if !strings.Contains(pages["/a/"], "<h1>A</h1>") {
		t.Errorf("a generated single template should render the page: %q", pages["/a/"])
	}
	if p.Report.Count(Manual) == 0 {
		t.Error("the parse failure must be reported")
	}
}

// translateSnippet translates one template in a single-page scope.
func translateSnippet(t *testing.T, flavor, src string) (string, *Report) {
	t.Helper()
	p := newPlan(t.TempDir(), &Report{})
	tr := &hugoTr{p: p, files: map[string]hugoFile{"partials/x.html": {origin: "x", text: "x"}}, done: map[string]bool{}, partials: map[string]*strings.Builder{}, assets: map[string]bool{}}
	main, _, err := parseHugo("t", src)
	if err != nil {
		t.Fatal(err)
	}
	sc := &scope{dot: ctxVal(flavor), dollar: hval{expr: "$", k: kCtx, flavor: flavor}, vars: map[string]hval{}, root: "$", rootKind: "entry", file: "t", tree: main}
	return tr.list(sc, main.Root), p.Report
}

func TestHugoExpressionTranslation(t *testing.T) {
	cases := []struct{ flavor, in, want string }{
		{"single", `{{ .Title }}`, `{{ .Title }}`},
		{"single", `{{ .Content }}`, `{{ unsafeHTML .Page.BodyHTML }}`},
		{"list", `{{ .Content }}`, `{{ unsafeHTML .Section.BodyHTML }}`},
		{"single", `{{ .Params.author }}`, `{{ .Page.Extra.author }}`},
		{"single", `{{ .Params.author.name }}`, `{{ dig .Page.Extra "author.name" }}`},
		{"single", `{{ .Site.Title }}`, `{{ $.Config.Title }}`},
		{"single", `{{ .Site.Params.logo }}`, `{{ $.Config.Extra.logo }}`},
		{"single", `{{ .Permalink }}`, `{{ .CanonicalURL }}`},
		{"single", `{{ .RelPermalink }}`, `{{ relURL .CurrentURL }}`},
		{"single", `{{ .Date.Format "2006" }}`, `{{ .Page.Date.Format "2006" }}`},
		{"single", `{{ dateFormat "Jan 2" .Date }}`, `{{ .Page.Date.Format "Jan 2" }}`},
		{"single", `{{ .Date | time.Format ":date_medium" }}`, `{{ .Page.Date.Format "Jan 2, 2006" }}`},
		{"single", `{{ .Title | default "x" }}`, `{{ or .Title "x" }}`},
		{"single", `{{ default "x" .Params.sub }}`, `{{ or .Page.Extra.sub "x" }}`},
		{"single", `{{ .Summary | plainify | truncate 50 }}`, `{{ truncate 50 (print (plainify (print .Page.Summary))) }}`},
		{"single", `{{ T "readMore" }}`, `{{ t "readMore" }}`},
		{"single", `{{ i18n "x" . }}`, `{{ t "x" }}`},
		{"single", `{{ .Title | safeHTML }}`, `{{ unsafeHTML (print .Title) }}`},
		{"single", `{{ delimit .Params.tags ", " }}`, `{{ join .Page.Extra.tags ", " }}`},
		{"single", `{{ range .Site.RegularPages }}{{ .Title }}{{ end }}`, `{{ range $.Index.AllPages }}{{ .Title }}{{ end }}`},
		{"single", `{{ range .Site.RegularPages }}{{ .Content }}{{ end }}`, `{{ range $.Index.AllPages }}{{ unsafeHTML .BodyHTML }}{{ end }}`},
		{"single", `{{ range .Site.RegularPages }}{{ .Permalink }}{{ end }}`, `{{ range $.Index.AllPages }}{{ absURL .URL }}{{ end }}`},
		{"single", `{{ range $p := .Pages }}{{ $p.Title }}{{ end }}`, `{{ range $p := .Pages }}{{ $p.Title }}{{ end }}`},
		{"single", `{{ range where .Site.RegularPages "Params.featured" true }}{{ .Title }}{{ end }}`, `{{ range where $.Index.AllPages "Extra.featured" true }}{{ .Title }}{{ end }}`},
		{"single", `{{ range where .Site.RegularPages "Section" "!=" "drafts" }}x{{ end }}`, `{{ range where $.Index.AllPages "SectionPath" "!=" "drafts" }}x{{ end }}`},
		{"single", `{{ with .Site.Params.social }}{{ .github }}{{ end }}`, `{{ with $.Config.Extra.social }}{{ .github }}{{ end }}`},
		{"single", `{{ $x := .Title }}{{ $x }}`, `{{ $x := .Title }}{{ $x }}`},
		{"single", `{{ $c := .Content }}{{ $c }}`, `{{ $c := .Page.BodyHTML }}{{ unsafeHTML $c }}`},
		{"single", `{{ if .IsHome }}h{{ end }}`, `{{ if eq .CurrentURL "/" }}h{{ end }}`},
		{"list", `{{ .IsPage }}`, `{{ false }}`},
		{"single", `{{ .Site.Copyright }}`, `{{ dig $.Config.Extra "copyright" }}`},
		{"single", `{{ len .Pages }}`, `{{ len .Pages }}`},
		{"single", `{{ .Section }}`, `{{ index (split .Page.SectionPath "/") 0 }}`},
		{"single", `{{ now.Year }}`, `{{ (now).Year }}`},
		{"single", `{{ humanize "my-post" }}`, `{{ title (replace (print "my-post") "-" " ") }}`},
		{"single", `{{ .Site.Language.Lang }}`, `{{ $.Config.Language }}`},
		{"single", `{{ in .Params.tags "go" }}`, `{{ contains (join .Page.Extra.tags "|") (print "go") }}`},
		{"single", `{{ cond .Draft "d" "p" }}`, `{{ or (and .Page.Draft "d") "p" }}`},
		{"single", `{{ .Paginator.TotalPages }}`, `{{ or (and .Paginator .Paginator.NumberPagers) 1 }}`},
		{"single", `{{ partial "x.html" . }}`, `{{ template "partials/x#ctx-single#entry" (dict "ctx" $ "dot" .) }}`},
		{"single", `{{ range .Pages }}{{ partial "x.html" . }}{{ end }}`, `{{ range .Pages }}{{ template "partials/x#page#entry" (dict "ctx" $ "dot" .) }}{{ end }}`},
		{"single", `{{ .TableOfContents }}`, `{{ template "nida-migrate-toc" .Page.TOC }}`},
		{"single", `{{ block "main" . }}x{{ end }}`, `{{ template "main" . }}`},
		{"single", `{{ template "content" . }}`, `{{ template "hugo-content" . }}`},
		{"single", `{{ .Site.Menus.main }}`, `{{ dig $.Config.Extra "menu.main" }}`},
		{"single", `{{ range $k, $v := .Site.Taxonomies.tags }}{{ $k }}{{ $v.Count }}{{ end }}`, `{{ range $k, $v := index $.Index.TaxonomyMap "tags" }}{{ $k }}{{ len $v }}{{ end }}`},
		{"terms", `{{ range $n, $t := .Data.Terms }}{{ $n }}{{ end }}`, `{{ range $t := .Terms }}{{ $n := $t.Name }}{{ $n }}{{ end }}`},
		{"single", `{{ with .PrevInSection }}{{ .Title }}{{ end }}`, `{{ with and .Page.PrevURL (dict "Title" .Page.PrevTitle "LinkTitle" .Page.PrevTitle "RelPermalink" (relURL .Page.PrevURL) "Permalink" (absURL .Page.PrevURL) "URL" .Page.PrevURL) }}{{ .Title }}{{ end }}`},
		{"single", `{{ .Unknown }}`, ``},
		{"single", `{{ $s := resources.Get "a.css" }}{{ with $s }}<link href="{{ .RelPermalink }}">{{ end }}`, `{{ with true }}<link href="{{ asset "a.css" }}">{{ end }}`},
		{"single", `{{ with .Site }}{{ .Title }}{{ end }}`, `{{ with true }}{{ $.Config.Title }}{{ end }}`},
		{"single", `{{ .Site }}`, ``},
		{"single", `{{ printf "%s" .Site.Language }}`, `{{ printf "%s" "" }}`},
		{"single", `{{ range .Unknown }}x{{ end }}`, `{{ range slice }}x{{ end }}`},
	}
	for _, tc := range cases {
		got, _ := translateSnippet(t, tc.flavor, tc.in)
		if got != tc.want {
			t.Errorf("%s %s\n got: %s\nwant: %s", tc.flavor, tc.in, got, tc.want)
		}
	}
}

func TestHugoUnknownFieldIsReportedWithLine(t *testing.T) {
	_, r := translateSnippet(t, "single", "line one\n{{ .Unknown }}\n")
	if len(r.Notes) != 1 || r.Notes[0].Line != 2 || r.Notes[0].Severity != Review {
		t.Errorf("notes = %v", r.Notes)
	}
}

func TestHugoLayoutShapes(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"config.toml":                "base_url = \"https://x.org\"\ntitle = \"Shapes\"\npaginate = 10\n",
		"content/_index.md":          "+++\ntitle = \"Home\"\n+++\n",
		"content/notes/_index.md":    "+++\ntitle = \"Notes\"\n+++\n",
		"content/notes/one/index.md": "+++\ntitle = \"One\"\n+++\n{{< img src=\"pic.png\" >}}\n\nSee {{< gist >}}.\n",
		"content/notes/one/pic.png":  "png",
		// No baseof: Hugo 0.146 flat layout, old-style header/footer
		// partials in _partials/, and a theme the site overrides.
		"layouts/single.html": `{{ partial "header.html" . }}<h1 class="site-single">{{ .Title }}</h1>{{ .Content }}{{ partial "footer.html" . }}`,
		"layouts/_partials/header.html": `<!doctype html><html><head><title>{{ .Title }}</title></head><body>` +
			`{{ with .Site.GetPage "/notes" }}<a class="up" href="{{ .RelPermalink }}">{{ .Title }}</a>{{ end }}`,
		"layouts/_partials/footer.html":        `{{ partial "copy.html" . }}</body></html>`,
		"themes/t/layouts/_partials/copy.html": `<small>{{ .Site.Title }} {{ partial "header.html" . | safeHTML }}</small>`,
		"themes/t/layouts/list.html": `{{ partial "header.html" . }}<ul class="theme-list">` +
			`{{ range .Paginator.Pages }}<li>{{ .Title }}</li>{{ end }}</ul>` +
			`{{ if .Paginator.HasNext }}<a href="{{ .Paginator.Next.URL }}">next</a>{{ end }}<p>page {{ .Paginator.PageNumber }}/{{ .Paginator.TotalPages }}</p>{{ partial "footer.html" . }}`,
		"themes/t/layouts/single.html": `theme single must lose to the site`,
		"layouts/_shortcodes/img.html": `{{ $img := .Page.Resources.GetMatch (.Get "src") }}<img src="{{ $img.RelPermalink }}" width="{{ $img.Width }}" alt="{{ $.Page.Site.Title }}">`,
	})
	p := hugoTemplatePlan(t, root)
	p.Placeholder("gist", "content/notes/one/index.md", 3)
	planHugoTemplates(p, []string{"layouts", "themes/t/layouts"})
	if err := finishPlan(p); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := p.apply(dest); err != nil {
		t.Fatal(err)
	}
	if err := chooseTemplates(p, dest, hugoSource{}, checkSite); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.Report.TemplateSet, "translated") {
		t.Fatalf("not translated: %v", p.Report.Notes)
	}
	cfg, _, _ := config.Load(config.Options{SiteRoot: dest})
	state, err := site.Load(dest, cfg)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := render.RenderSite(dest, cfg, state)
	if err != nil {
		t.Fatal(err)
	}
	byURL := map[string]string{}
	for _, page := range pages {
		byURL[page.URL] = page.Content
	}
	for url, wants := range map[string][]string{
		"/notes/one/": {`<h1 class="site-single">One</h1>`, `<img src="pic.png" width="0" alt="Shapes">`, `<a class="up" href="/notes/">Notes</a>`, `<small>Shapes </small>`, "Migration TODO"},
		"/notes/":     {`<ul class="theme-list"><li>One</li></ul>`, `<p>page 1/1</p>`},
	} {
		for _, want := range wants {
			if !strings.Contains(byURL[url], want) {
				t.Errorf("%s: missing %q in:\n%s", url, want, byURL[url])
			}
		}
	}
	if strings.Contains(byURL["/notes/one/"], "theme single") {
		t.Error("the site's layout must override the theme's")
	}
	manual := false
	for _, n := range p.Report.Notes {
		if n.Severity == Manual && strings.Contains(n.Message, "used as a value") && n.File == "themes/t/layouts/_partials/copy.html" {
			manual = true
		}
	}
	if !manual {
		t.Errorf("a partial used as a value must be reported: %v", p.Report.Notes)
	}
}
