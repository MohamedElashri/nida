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

func TestRenderItemsStoresBodyHTML(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	items := []content.Item{{
		RelativePath: "posts/example.md",
		BodyMarkdown: "# Hello\n",
	}}

	rendered, err := RenderItems(items, cfg)
	if err != nil {
		t.Fatalf("RenderItems returned error: %v", err)
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
