package shortcodes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/markdown"
)

func setup(t *testing.T, files map[string]string) (config.SiteConfig, markdown.Env) {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/blog/"
	set, err := Load(root, cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	return cfg, markdown.Env{Shortcodes: set, Lookup: markdown.PathLookup{"posts/a.md": "/posts/a/"}}
}

func render(t *testing.T, cfg config.SiteConfig, env markdown.Env, src string) string {
	t.Helper()
	result, err := markdown.RenderDocument(src, cfg, env)
	if err != nil {
		t.Fatalf("RenderDocument returned error: %v", err)
	}
	return result.HTML
}

func TestBuiltinShortcodes(t *testing.T) {
	cfg, env := setup(t, nil)
	got := render(t, cfg, env, `Intro.

{{< figure src="shot.png" caption="The *new* editor" link="@/posts/a.md" >}}

{{< youtube dQw4w9WgXcQ >}}

{{< callout type="warning" title="Careful" >}}
Back up **first**.
{{< /callout >}}
`)
	for _, want := range []string{
		`<figure><a href="/blog/posts/a/"><img src="shot.png" alt="The *new* editor"`,
		`<figcaption>The <em>new</em> editor</figcaption></figure>`,
		`src="https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ"`,
		`<div class="callout callout-warning" data-callout="warning">`,
		`<p class="callout-title">Careful</p>`,
		`<p>Back up <strong>first</strong>.</p>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "<p><figure") || strings.Contains(got, "<p><div") {
		t.Fatalf("block shortcodes must not be wrapped in paragraphs:\n%s", got)
	}
}

func TestSiteShortcodesOverrideBuiltinsAndRenderInline(t *testing.T) {
	cfg, env := setup(t, map[string]string{
		"templates/shortcodes/youtube.html": `<a class="yt" href="https://youtu.be/{{ .Get 0 }}">video</a>`,
		"templates/shortcodes/kbd.html":     `<kbd>{{ .Get 0 }}</kbd>`,
		"templates/shortcodes/note.html":    `<aside data-page="{{ with .Page }}{{ .Title }}{{ end }}">{{ .InnerHTML }}</aside>`,
	})
	got := render(t, cfg, env, "Press {{< kbd \"Ctrl+K\" >}} now.\n\n{{< youtube abc />}}\n\n{{< note >}}outer {{< note >}}inner{{< /note >}}{{< /note >}}\n")
	for _, want := range []string{
		`<p>Press <kbd>Ctrl&#43;K</kbd> now.</p>`,
		`<a class="yt" href="https://youtu.be/abc">video</a>`,
		`<aside data-page=""><p>outer <aside data-page=""><p>inner</p>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in\n%s", want, got)
		}
	}
}

func TestShortcodesInsideCodeAreLiteral(t *testing.T) {
	cfg, env := setup(t, nil)
	got := render(t, cfg, env, "Use `{{< figure src=\"x\" >}}` inline.\n\n```markdown\n{{< youtube id >}}\n{% details(summary=\"x\") %}\nbody\n{% end %}\n```\n")
	if strings.Contains(got, "<figure") || strings.Contains(got, "iframe") || strings.Contains(got, "<details") {
		t.Fatalf("shortcodes in code must not render:\n%s", got)
	}
	if !strings.Contains(got, "<code>{{&lt; figure src=&quot;x&quot; &gt;}}</code>") {
		t.Fatalf("expected literal inline code, got\n%s", got)
	}
}

func TestUnknownShortcodeFails(t *testing.T) {
	cfg, env := setup(t, nil)
	_, err := markdown.RenderDocument("{{< nope >}}\n", cfg, env)
	if err == nil || !strings.Contains(err.Error(), `unknown shortcode "nope"`) {
		t.Fatalf("expected unknown shortcode error, got %v", err)
	}
}
