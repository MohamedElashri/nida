package markdown

import (
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

func TestRenderDocumentBuildsNestedTOC(t *testing.T) {
	result, err := RenderDocument("# Title\n\n## Install\n\n### From source\n\n## Usage {#how-to}\n\n#### Deep\n", config.DefaultSiteConfig(), Env{})
	if err != nil {
		t.Fatalf("RenderDocument returned error: %v", err)
	}
	toc := result.TOC
	if len(toc) != 1 || toc[0].Title != "Title" || len(toc[0].Children) != 2 {
		t.Fatalf("unexpected toc %+v", toc)
	}
	install, usage := toc[0].Children[0], toc[0].Children[1]
	if install.ID != "install" || len(install.Children) != 1 || install.Children[0].Title != "From source" {
		t.Fatalf("unexpected install entry %+v", install)
	}
	if usage.ID != "how-to" || len(usage.Children) != 1 || usage.Children[0].Level != 4 {
		t.Fatalf("expected custom id and skipped-level child, got %+v", usage)
	}
	if !strings.Contains(result.HTML, `<h2 id="how-to">Usage</h2>`) {
		t.Fatalf("expected custom heading id in HTML, got %s", result.HTML)
	}
}

func TestRenderHeadingAnchors(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.Markdown.HeadingAnchors = true

	got, err := Render("## Getting started\n", cfg, nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	want := `<h2 id="getting-started">Getting started<a class="heading-anchor" href="#getting-started" aria-hidden="true">#</a></h2>`
	if !strings.Contains(got, want) {
		t.Fatalf("expected heading anchor\nwant: %s\ngot:  %s", want, got)
	}
}

func TestRenderCallouts(t *testing.T) {
	got, err := Render("> [!WARNING]\n> Back up *first*.\n\n> [!tip] Faster builds\n> Use `--future`.\n\n> [!NOTE]\n\n> Plain quote.\n", config.DefaultSiteConfig(), nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	for _, want := range []string{
		`<div class="callout callout-warning" data-callout="warning">`,
		`<p class="callout-title">Warning</p>`,
		`<p>Back up <em>first</em>.</p>`,
		`<p class="callout-title">Faster builds</p>`,
		`<p>Use <code>--future</code>.</p>`,
		`<p class="callout-title">Note</p>`,
		"<blockquote>\n<p>Plain quote.</p>\n</blockquote>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "[!") {
		t.Fatalf("callout marker leaked into output:\n%s", got)
	}
}

func TestRenderCodeBlockOptions(t *testing.T) {
	src := "```go {title=\"main.go\", hl_lines=\"2\", linenos=true, linenostart=10}\npackage main\nfunc main() {}\n```\n"
	got, err := Render(src, config.DefaultSiteConfig(), nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(got, `<figure class="code-block"><figcaption class="code-title">main.go</figcaption>`) {
		t.Fatalf("expected code title figure, got %s", got)
	}
	if !strings.Contains(got, `z-hl`) || !strings.Contains(got, ">11<") {
		t.Fatalf("expected highlighted line and numbering from 10, got %s", got)
	}
}

func TestParseCodeInfo(t *testing.T) {
	info := parseCodeInfo(`rust {hl_lines=[1, "3-4"], title="lib.rs" linenos}`)
	if info.Language != "rust" || info.Title != "lib.rs" || !info.LineNumbers {
		t.Fatalf("unexpected info %+v", info)
	}
	if len(info.HighlightLn) != 2 || info.HighlightLn[1] != [2]int{3, 4} {
		t.Fatalf("unexpected ranges %+v", info.HighlightLn)
	}
}

func TestRenderMermaid(t *testing.T) {
	got, err := Render("```mermaid\ngraph TD; A-->B\n```\n", config.DefaultSiteConfig(), nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(got, "<pre class=\"mermaid\">graph TD; A--&gt;B\n</pre>") {
		t.Fatalf("expected mermaid pre, got %s", got)
	}
}

func TestRenderMath(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.Markdown.Math = true

	src := "Euler: $e^{i\\pi} + 1 = 0$ and $a_1 * b_2$, costs $5 and $10.\n\n$$\n\\sum_{i=1}^n x_i < y\n$$\n\nInline display $$x^2$$ here.\n"
	got, err := Render(src, cfg, nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	for _, want := range []string{
		`<span class="math math-inline">\(e^{i\pi} + 1 = 0\)</span>`,
		`<span class="math math-inline">\(a_1 * b_2\)</span>`,
		"costs $5 and $10.",
		`<div class="math math-display">\[\sum_{i=1}^n x_i &lt; y\]</div>`,
		`<span class="math math-display">\[x^2\]</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in\n%s", want, got)
		}
	}

	plain, err := Render("$a_1$ and $b_1$\n", config.DefaultSiteConfig(), nil)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if strings.Contains(plain, "math-inline") {
		t.Fatalf("math must be opt-in, got %s", plain)
	}
}

func TestRenderItemSummaries(t *testing.T) {
	cfg := config.DefaultSiteConfig()

	marked, err := RenderItem(content.Page{BodyMarkdown: "Intro **one**.\n\nIntro two.\n\n<!-- more -->\n\nRest of post.\n\n```html\n<!-- more -->\n```\n"}, cfg, Env{})
	if err != nil {
		t.Fatalf("RenderItem returned error: %v", err)
	}
	if marked.Summary != "<p>Intro <strong>one</strong>.</p>\n<p>Intro two.</p>\n" {
		t.Fatalf("unexpected manual summary %q", marked.Summary)
	}
	if !strings.Contains(marked.BodyHTML, "Rest of post.") || !strings.Contains(marked.BodyHTML, "&lt;!--") {
		t.Fatalf("body should keep everything except the marker, got %s", marked.BodyHTML)
	}
	if marked.WordCount == 0 {
		t.Fatal("expected a word count")
	}

	auto, err := RenderItem(content.Page{BodyMarkdown: "## Heading\n\nFirst paragraph.\n\nSecond paragraph.\n"}, cfg, Env{})
	if err != nil {
		t.Fatalf("RenderItem returned error: %v", err)
	}
	if auto.Summary != "<p>First paragraph.</p>" {
		t.Fatalf("unexpected automatic summary %q", auto.Summary)
	}
}

func TestRenderInternalLinksIncludeBasePath(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/docs/"
	lookup := PathLookup{"guide/intro.md": "/guide/intro/", "img/a.png": "/img/a.png"}

	got, err := Render("[Intro](@/guide/intro.md#start) ![a](@/img/a.png) [ext](https://x.org/) [root](/about/)\n", cfg, lookup)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	for _, want := range []string{`href="/docs/guide/intro/#start"`, `src="/docs/img/a.png"`, `href="https://x.org/"`, `href="/about/"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %s in %s", want, got)
		}
	}
}

type fakeImages struct{}

func (fakeImages) Image(src string, _ *content.Page) (ImageInfo, bool) {
	if src != "/img/a.png" {
		return ImageInfo{}, false
	}
	return ImageInfo{Width: 800, Height: 400, Srcset: "/img/a.480w.png 480w, /img/a.png 800w", Sizes: "100vw"}, true
}

func TestRenderImageAttributesFromResolver(t *testing.T) {
	result, err := RenderDocument("![A](/img/a.png) ![B](/img/b.png)\n", config.DefaultSiteConfig(), Env{Images: fakeImages{}})
	if err != nil {
		t.Fatalf("RenderDocument returned error: %v", err)
	}
	want := `<img src="/img/a.png" alt="A" srcset="/img/a.480w.png 480w, /img/a.png 800w" sizes="100vw" width="800" height="400" loading="lazy" decoding="async">`
	if !strings.Contains(result.HTML, want) {
		t.Fatalf("expected %s in %s", want, result.HTML)
	}
	if !strings.Contains(result.HTML, `<img src="/img/b.png" alt="B" loading="lazy" decoding="async">`) {
		t.Fatalf("unknown images should keep plain attributes, got %s", result.HTML)
	}
}
