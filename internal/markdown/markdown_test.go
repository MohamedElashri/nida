package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

func TestRenderGolden(t *testing.T) {
	cfg := config.DefaultSiteConfig()

	got, err := Render(`# Heading

Here is a [link](https://example.com).

<div>raw html</div>
`, cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	assertGolden(t, "basic.golden.html", got)
}

func TestRenderHighlightedCodeGolden(t *testing.T) {
	cfg := config.DefaultSiteConfig()

	got, err := Render("```go\npackage main\n```\n", cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	assertGolden(t, "highlight.golden.html", got)
}

func TestRenderPreservesRawImageHTML(t *testing.T) {
	cfg := config.DefaultSiteConfig()

	got, err := Render(`<div><img src="/images/example.png" alt="Example"></div>`, cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(got, `<img src="/images/example.png" alt="Example">`) {
		t.Fatalf("expected raw img HTML to be preserved, got %q", got)
	}
}

func TestRenderRawHTMLShortcode(t *testing.T) {
	cfg := config.DefaultSiteConfig()

	got, err := Render(`Before

{{< rawhtml >}}
<video controls></video>
{{< /rawhtml >}}
`, cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(got, `<video controls></video>`) || strings.Contains(got, "rawhtml") {
		t.Fatalf("expected rawhtml shortcode markers to be stripped, got %q", got)
	}
}

func TestRenderDetailsShortcode(t *testing.T) {
	cfg := config.DefaultSiteConfig()

	got, err := Render(`{% details(summary="Original post") %}

**Hello** from inside.

{% end %}
`, cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for _, want := range []string{
		`<details class="collapsible-details">`,
		`<span class="collapsible-details-label">Original post</span>`,
		`<strong>Hello</strong> from inside.`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in rendered details shortcode, got %q", want, got)
		}
	}
}

func TestRenderUnknownLanguageFallsBackGracefully(t *testing.T) {
	cfg := config.DefaultSiteConfig()

	got, err := Render("```unknownlang\nhello\n```\n", cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(got, `class="language-unknownlang"`) {
		t.Fatalf("expected plain fallback output, got %q", got)
	}
}

func TestRenderExternalLinkAttributes(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.Markdown.ExternalLinksTargetBlank = true
	cfg.Markdown.ExternalLinksNoFollow = true
	cfg.Markdown.ExternalLinksNoReferrer = true

	got, err := Render(`[external](https://example.com) and [local](/about/)`, cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(got, `<a href="https://example.com" target="_blank" rel="nofollow noreferrer noopener">external</a>`) {
		t.Fatalf("expected external link attributes, got %q", got)
	}
	if !strings.Contains(got, `<a href="/about/">local</a>`) {
		t.Fatalf("expected local link without external attributes, got %q", got)
	}
}

func TestRenderPagesStoresBodyHTML(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	pages := []content.Page{{
		RelativePath: "posts/example.md",
		BodyMarkdown: "# Hello\n",
	}}

	rendered, err := RenderPages(pages, cfg)
	if err != nil {
		t.Fatalf("RenderPages returned error: %v", err)
	}

	if len(rendered) != 1 || !strings.Contains(rendered[0].BodyHTML, "<h1") {
		t.Fatalf("expected rendered HTML body, got %+v", rendered)
	}
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file %q: %v", path, err)
	}

	if got != string(want) {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\ngot:\n%s", name, string(want), got)
	}
}
