package migrate

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJekyllGolden(t *testing.T) {
	result := runGolden(t, "jekyll")
	if result.Report.URLsKept == 0 {
		t.Error("expected posts to keep their URLs through the permalink pattern")
	}
}

func TestJekyllDetect(t *testing.T) {
	if s, err := pickSource(filepath.Join("testdata", "jekyll", "site"), "auto"); err != nil || s.Name() != "jekyll" {
		t.Errorf("detect jekyll: %v %v", s, err)
	}
	if (jekyllSource{}).Detect(filepath.Join("testdata", "zola", "site")) != 0 {
		t.Error("a Zola site must not look like Jekyll")
	}
}

func TestJekyllSlugify(t *testing.T) {
	cases := []struct {
		in, mode string
		cased    bool
		want     string
	}{
		{"Hello World!", "default", false, "hello-world"},
		{"Hello World!", "pretty", true, "Hello-World!"},
		{"The _config.yml file", "default", false, "the-config-yml-file"},
		{"The _config.yml file", "pretty", false, "the-_config.yml-file"},
		{"Café crème", "default", false, "café-crème"},
		{"Café crème", "ascii", false, "caf-cr-me"},
		{"  spaced  out ", "raw", false, "spaced-out"},
		{"--edge--", "default", false, "edge"},
	}
	for _, tc := range cases {
		if got := jekyllSlugify(tc.in, tc.mode, tc.cased); got != tc.want {
			t.Errorf("slugify(%q, %s, %t) = %q, want %q", tc.in, tc.mode, tc.cased, got, tc.want)
		}
	}
}

// Expected URLs follow Jekyll's documented permalink examples.
func TestJekyllPostURLs(t *testing.T) {
	date := time.Date(2009, 4, 29, 0, 0, 0, 0, time.UTC)
	doc := &jekyllDoc{kind: "posts", basename: "2009-04-29-slap-chop", slugData: "slap-chop", hasDate: true, date: date, categories: []string{"Food", "Kitchen"}}
	for permalink, want := range map[string]string{
		"date":                          "/food/kitchen/2009/04/29/slap-chop.html",
		"":                              "/food/kitchen/2009/04/29/slap-chop.html",
		"pretty":                        "/food/kitchen/2009/04/29/slap-chop/",
		"ordinal":                       "/food/kitchen/2009/119/slap-chop.html",
		"weekdate":                      "/food/kitchen/2009/W18/Wed/slap-chop.html",
		"none":                          "/food/kitchen/slap-chop.html",
		"/:year/:i_month/:i_day/:title": "/2009/4/29/slap-chop.html",
		"/blog/:short_year/:title/":     "/blog/09/slap-chop/",
		"/:categories/:title":           "/food/kitchen/slap-chop.html",
	} {
		s := &jekyllSite{permalink: permalink, p: &Plan{Report: &Report{}}}
		if got := s.oldURL(doc); got != want {
			t.Errorf("permalink %q: got %s, want %s", permalink, got, want)
		}
	}
}

func TestJekyllPageURLs(t *testing.T) {
	for _, tc := range []struct {
		origin, permalink, style, want string
	}{
		{"about.md", "", "date", "/about.html"},
		{"about.md", "", "pretty", "/about/"},
		{"docs/install.md", "", "date", "/docs/install.html"},
		{"docs/install.md", "", "/:year/:title/", "/docs/install/"},
		{"docs/install.md", "/setup/", "date", "/setup/"},
		{"docs/install.md", "/:path/:basename:output_ext", "pretty", "/docs/install.html"},
		{"team.md", "/people", "date", "/people.html"},
	} {
		base := strings.TrimSuffix(filepath.Base(tc.origin), ".md")
		s := &jekyllSite{permalink: tc.style, p: &Plan{Report: &Report{}}}
		d := &jekyllDoc{kind: "pages", origin: tc.origin, basename: base, permalink: tc.permalink}
		if got := s.oldURL(d); got != tc.want {
			t.Errorf("%s permalink=%q style=%q: got %s, want %s", tc.origin, tc.permalink, tc.style, got, tc.want)
		}
	}
}

func TestNidaPattern(t *testing.T) {
	for _, tc := range []struct {
		template, categories, want, token string
		ok                                bool
	}{
		{jekyllStyles["date"], "", "/{year}/{month}/{day}/{slug}.html", "title", true},
		{jekyllStyles["pretty"], "{categories}", "/{categories}/{year}/{month}/{day}/{slug}/", "title", true},
		{jekyllStyles["ordinal"], "", "", "", false},
		{"/:collection/:name/", "", "/recipes/{slug}/", "name", true},
		{"/:title/:slug/", "", "", "", false},
	} {
		got, token, ok := nidaPattern(tc.template, "recipes", tc.categories)
		if got != tc.want || token != tc.token || ok != tc.ok {
			t.Errorf("nidaPattern(%q) = %q %q %t, want %q %q %t", tc.template, got, token, ok, tc.want, tc.token, tc.ok)
		}
	}
}

func TestJekyllHighlightFence(t *testing.T) {
	got := jekyllHighlightFence("ruby", ` linenos mark_lines="1 3"`, "\nputs 1\n```\nputs 2\n")
	want := "````ruby {linenos=true, hl_lines=\"1 3\"}\nputs 1\n```\nputs 2\n````"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestJekyllKramdown(t *testing.T) {
	s := &jekyllSite{p: newPlan(t.TempDir(), &Report{})}
	d := &jekyllDoc{origin: "a.md", doc: NewDocument(YAML, "")}
	in := "* TOC\n{:toc}\n\n## One {: #one .x}\n\nTitle\n-----\n{: #two}\n\nPara\n{: .note}\n\n[a](b){:target=\"_blank\"}\n\n```\n{: .kept}\n```\n"
	got := s.convertKramdown(d, in, func(string) int { return 1 })
	want := "\n## One {#one .x}\n\n## Title {#two}\n\nPara\n\n[a](b)\n\n```\n{: .kept}\n```\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestJekyllLiquid(t *testing.T) {
	p := newPlan(t.TempDir(), &Report{})
	s := &jekyllSite{p: p, cfg: map[string]any{"title": "Site"}, byOrigin: map[string]*jekyllDoc{}, staticPaths: map[string]bool{"files/a.pdf": true}}
	post := &jekyllDoc{kind: "posts", origin: "_posts/2020-01-01-x.md", basename: "2020-01-01-x", dest: "content/posts/x.md"}
	page := &jekyllDoc{kind: "pages", origin: "docs/b.md", dest: "content/docs/b.md", vars: map[string]any{"title": "B"}}
	s.docs = []*jekyllDoc{post, page}
	s.byOrigin[post.origin], s.byOrigin[page.origin] = post, page
	in := `{{ page.title }} / {{ site.title }} / {% post_url 2020-01-01-x %} / {% link files/a.pdf %} / [x](b.md#top) / {{ site.baseurl }}/img.png / {{ '/c' | absolute_url }} / {% include figure.html src="a.png" %} / {% include media/clip.html id=page.x %}`
	got := s.convertLiquid(page, in, func(string) int { return 1 })
	want := `B / Site / @/posts/x.md / /files/a.pdf / [x](@/docs/b.md#top) / /img.png / /c / {{< figure src="a.png" >}} / {% include media/clip.html id=page.x %}`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
