package migrate

import (
	"strings"
	"testing"
	"time"
)

func TestEleventyGolden(t *testing.T) {
	result := runGolden(t, "eleventy")
	if result.Report.URLsKept == 0 {
		t.Error("expected URLs to be checked")
	}
}

func TestEleventyDetect(t *testing.T) {
	site := t.TempDir()
	writeTree(t, site, map[string]string{"eleventy.config.js": "export default function () {}\n"})
	if s, err := pickSource(site, "auto"); err != nil || s.Name() != "eleventy" {
		t.Errorf("detect eleventy: %v %v", s, err)
	}
	pkgOnly := t.TempDir()
	writeTree(t, pkgOnly, map[string]string{"package.json": `{"devDependencies": {"@11ty/eleventy": "^2.0.0"}}`})
	if (eleventySource{}).Detect(pkgOnly) == 0 {
		t.Error("a package.json depending on @11ty/eleventy must be detected")
	}
	if (eleventySource{}).Detect(t.TempDir()) != 0 {
		t.Error("an empty directory is not an Eleventy site")
	}
}

func TestEleventyConfigScraping(t *testing.T) {
	src := stripJSComments(`
const { EleventyHtmlBasePlugin } = require("@11ty/eleventy");
const pluginRss = require("@11ty/eleventy-plugin-rss");
module.exports = function (eleventyConfig) {
  // eleventyConfig.addPassthroughCopy("commented-out");
  eleventyConfig.addPassthroughCopy("img");
  eleventyConfig.addPassthroughCopy({ "src/static/": "assets", "robots.txt": "robots.txt" });
  eleventyConfig.addPlugin(EleventyHtmlBasePlugin);
  eleventyConfig.addPairedNunjucksShortcode("callout", fn);
  eleventyConfig.addNunjucksAsyncShortcode("image", fn);
  eleventyConfig.addLiquidFilter("postDate", fn);
  eleventyConfig.setLibrary("md", lib);
  eleventyConfig.setInputDirectory("ignored-by-dir-block");
  return {
    templateFormats: "md,njk",
    markdownTemplateEngine: false,
    pathPrefix: "/sub/",
    dir: { input: "src", output: "dist", includes: "_includes", data: "_data" },
  };
};`)
	cfg := &eleventyConfig{input: ".", formats: eleventyDefaultFormats, mdEngine: "liquid"}
	cfg.scrape(src)
	if cfg.input != "src" && cfg.input != "ignored-by-dir-block" {
		t.Errorf("input = %q", cfg.input)
	}
	if cfg.output != "dist" || cfg.pathPrefix != "/sub/" || cfg.mdEngine != "" {
		t.Errorf("output=%q prefix=%q engine=%q", cfg.output, cfg.pathPrefix, cfg.mdEngine)
	}
	if strings.Join(cfg.formats, ",") != "md,njk" {
		t.Errorf("formats = %v", cfg.formats)
	}
	if len(cfg.passthrough) != 3 || cfg.passthrough[0].src != "img" || !cfg.passthrough[1].set {
		t.Errorf("passthrough = %+v", cfg.passthrough)
	}
	if strings.Join(cfg.paired, ",") != "callout" || strings.Join(cfg.shortcodes, ",") != "image" || strings.Join(cfg.filters, ",") != "postDate" {
		t.Errorf("paired=%v single=%v filters=%v", cfg.paired, cfg.shortcodes, cfg.filters)
	}
	if len(cfg.plugins) != 1 || cfg.plugins[0].pkg != "@11ty/eleventy" {
		t.Errorf("plugins = %+v", cfg.plugins)
	}
	if strings.Join(cfg.others, ",") != "setLibrary" {
		t.Errorf("others = %v", cfg.others)
	}
}

func TestStripJSCommentsKeepsStrings(t *testing.T) {
	got := stripJSComments(`a("https://x.example/") // gone` + "\n" + `b('/* kept */') /* gone */ c`)
	if got != `a("https://x.example/") `+"\n"+`b('/* kept */')  c` {
		t.Errorf("got %q", got)
	}
}

func TestParseJSDataFile(t *testing.T) {
	v, err := parseJSDataFile(`// site data
module.exports = {
  title: 'It\'s mine',
  "url": "https://x.example",
  nested: { list: [1, 2.5, true, null,], },
};`)
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if m["title"] != "It's mine" || m["url"] != "https://x.example" {
		t.Errorf("got %v", m)
	}
	list := m["nested"].(map[string]any)["list"].([]any)
	if len(list) != 4 || list[0] != 1 || list[1] != 2.5 || list[2] != true || list[3] != nil {
		t.Errorf("list = %#v", list)
	}
	for _, src := range []string{
		"export default function () { return {} }",
		"export default { now: Date.now() }",
		"export default { t: `${x}` }",
		"module.exports = { a: 1 }; doSomething();",
		"const x = 1;",
	} {
		if _, err := parseJSDataFile(src); err == nil {
			t.Errorf("%q: computed data must be rejected", src)
		}
	}
}

func TestEleventyFileSlugAndDefaultURL(t *testing.T) {
	cases := []struct{ in, slug, stem, url string }{
		{"posts/2018-01-01-myfile.md", "myfile", "/posts/2018-01-01-myfile", "/posts/2018-01-01-myfile/"},
		{"posts/myfile.md", "myfile", "/posts/myfile", "/posts/myfile/"},
		{"posts/index.md", "posts", "/posts/index", "/posts/"},
		{"index.md", "", "/index", "/"},
		{"2018-01-01.md", "2018-01-01", "/2018-01-01", "/2018-01-01/"},
		{"about/My Page.njk", "My Page", "/about/My Page", "/about/My Page/"},
		{"feed.11ty.js", "feed", "/feed", "/feed/"},
		{"posts/2018-01-01-trip/index.md", "trip", "/posts/2018-01-01-trip/index", "/posts/2018-01-01-trip/"},
	}
	for _, tc := range cases {
		pg := eleventyPage{inputRel: tc.in}
		if got := pg.fileSlug(); got != tc.slug {
			t.Errorf("fileSlug(%q) = %q, want %q", tc.in, got, tc.slug)
		}
		if got := pg.filePathStem(); got != tc.stem {
			t.Errorf("filePathStem(%q) = %q, want %q", tc.in, got, tc.stem)
		}
		if got := pg.defaultURL(); got != tc.url {
			t.Errorf("defaultURL(%q) = %q, want %q", tc.in, got, tc.url)
		}
	}
}

func TestEvalPermalink(t *testing.T) {
	pg := eleventyPage{
		inputRel: "posts/2024-03-09-hello.md",
		date:     time.Date(2024, 3, 9, 0, 0, 0, 0, time.UTC),
		hasDate:  true,
		data:     map[string]any{"title": "Hello Wörld, fooBar!", "meta": map[string]any{"section": "notes"}},
	}
	for tmpl, want := range map[string]string{
		"/about-me/":                               "/about-me/",
		"/{{ page.fileSlug }}/":                    "/hello/",
		"{{ page.filePathStem }}.html":             "/posts/2024-03-09-hello.html",
		"/{{ title | slugify }}/":                  "/hello-woerld-foo-bar/",
		`/{{ page.date | date: "%Y/%m/%d" }}/x/`:   "/2024/03/09/x/",
		"/{{ page.date | dateFilter('yyyy') }}/":   "/2024/",
		"/{{ page.date | htmlDateString }}/":       "/2024-03-09/",
		"/{{ meta.section }}/{{ page.fileSlug }}/": "/notes/hello/",
		"/{{- page.fileSlug | lower -}}/":          "/hello/",
		"/{{ 'literal' }}/":                        "/literal/",
	} {
		got, err := evalPermalink(tmpl, pg)
		if err != nil || got != want {
			t.Errorf("evalPermalink(%q) = %q, %v; want %q", tmpl, got, err, want)
		}
	}
	for _, tmpl := range []string{
		"/{{ page.date }}/",
		"/{{ unknown }}/",
		"/{% if draft %}drafts/{% endif %}{{ page.fileSlug }}/",
		`/{{ page.date | date: "%B" }}/`,
		"/{{ title | customFilter }}/",
	} {
		if got, err := evalPermalink(tmpl, pg); err == nil {
			t.Errorf("evalPermalink(%q) = %q, want an error", tmpl, got)
		}
	}
	noDate := eleventyPage{inputRel: "x.md"}
	if _, err := evalPermalink("/{{ page.date | date: '%Y' }}/", noDate); err == nil {
		t.Error("a date permalink without a date must not evaluate")
	}
}

func TestPermalinkPattern(t *testing.T) {
	for tmpl, want := range map[string]string{
		`/blog/{{ page.date | date: "%Y/%m" }}/{{ page.fileSlug }}/`: "/blog/{year}/{month}/{slug}/",
		"/notes/{{ title | slugify }}.html":                          "/notes/{slug}.html",
		"/{{ page.date | htmlDateString }}/":                         "/{year}-{month}-{day}/",
	} {
		got, _, ok := permalinkPattern(tmpl)
		if !ok || got != want {
			t.Errorf("permalinkPattern(%q) = %q, %t; want %q", tmpl, got, ok, want)
		}
	}
	for _, tmpl := range []string{
		"/{{ page.fileSlug }}/{{ title | slugify }}/", // two slug sources
		"{{ page.filePathStem }}/",
		`/{{ page.date | date: "%y" }}/{{ page.fileSlug }}/`,
		"/{{ category }}/{{ page.fileSlug }}/",
	} {
		if got, _, ok := permalinkPattern(tmpl); ok {
			t.Errorf("permalinkPattern(%q) = %q, want no pattern", tmpl, got)
		}
	}
	_, slug, _ := permalinkPattern("/x/{{ page.fileSlug }}/")
	if s, ok := slug(eleventyPage{inputRel: "x/2020-01-01-trip.md"}); !ok || s != "trip" {
		t.Errorf("slug = %q, %t", s, ok)
	}
}

func TestGlobMatch(t *testing.T) {
	for _, tc := range []struct {
		pattern, name string
		want          bool
	}{
		{"content/**/*.{svg,png}", "content/blog/a.png", true},
		{"content/**/*.{svg,png}", "content/a.svg", true},
		{"content/**/*.{svg,png}", "content/blog/a.jpg", false},
		{"**/README.md", "README.md", true},
		{"**/README.md", "docs/README.md", true},
		{"drafts", "drafts", true},
		{"img/*.png", "img/sub/a.png", false},
	} {
		if got := globMatch(tc.pattern, tc.name); got != tc.want {
			t.Errorf("globMatch(%q, %q) = %t", tc.pattern, tc.name, got)
		}
	}
}

func TestEleventyBodyConversion(t *testing.T) {
	cfg := &eleventyConfig{mdEngine: "liquid", paired: []string{"callout"}, shortcodes: []string{"badge"}}
	p := newPlan(t.TempDir(), &Report{})
	doc := NewDocument(YAML, `{% raw %}{{ kept }}{% endraw %} and {% comment %}gone{% endcomment %}
{% callout "warn" %}inside{% endcallout %}
{% badge "new" level=2 %}
{% youtube "abc123" %}
{{ "/about/" | url }} {{ title | upcase }}
{% if x %}left{% endif %}
`)
	b := &eleventyBody{p: p, cfg: cfg, file: "x.md", doc: doc, page: eleventyPage{inputRel: "x.md", data: map[string]any{"title": "Hi"}}, asset: func(s string) (string, bool) { return s, true }}
	b.convert()
	want := "{{ kept }} and " + `
{{< callout "warn" >}}inside{{< /callout >}}
{{< badge "new" level="2" >}}
{{< youtube "abc123" >}}
/about/ HI
{% if x %}left{% endif %}
`
	if doc.Body != want {
		t.Errorf("got:\n%s\nwant:\n%s", doc.Body, want)
	}
	if p.Report.Count(Manual) < 2 {
		t.Errorf("template logic must be reported: %v", p.Report.Notes)
	}
}

func TestHighlightFence(t *testing.T) {
	if got := highlightFence(`"js 1,3-4"`, "\nlet a\n"); got != "```js {hl_lines=\"1 3-4\"}\nlet a\n```" {
		t.Errorf("got %q", got)
	}
	if got := highlightFence(`"md"`, "```\nx\n```"); !strings.HasPrefix(got, "````md\n") {
		t.Errorf("fence must be longer than the code's own fences: %q", got)
	}
}

func TestHTMLAsMarkdown(t *testing.T) {
	got := htmlAsMarkdown("    <div>\n\n      <p>a</p>\n\n    <pre>\nx\n\ny\n</pre>\n    </div>\n")
	want := "<div>\n  <p>a</p>\n<pre>\nx\n\ny\n</pre>\n</div>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
