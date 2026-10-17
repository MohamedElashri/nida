package templates

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestLoadFixtureTemplates(t *testing.T) {
	siteRoot := filepath.Join("..", "..", "example-site")
	cfg, _, err := config.Load(config.Options{SiteRoot: siteRoot})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	set, err := Load(siteRoot, cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if !set.Has("index") || !set.Has("post") || !set.Has("page") {
		t.Fatalf("expected core templates to load, got %v", AvailableNames(set))
	}
}

func TestLoadMissingBaseTemplate(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.TemplateDir = "templates"

	if err := osMkdirAll(filepath.Join(dir, cfg.TemplateDir), 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	_, err := Load(dir, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadUsesHTMLTemplateFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.TemplateDir = "templates"
	templateDir := filepath.Join(dir, cfg.TemplateDir)

	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "base.html"), []byte(`{{ define "base" }}{{ template "content" . }}{{ end }}`), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "index.html"), []byte(`{{ define "index" }}home{{ end }}`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "ignored.txt"), []byte(`{{ define "ignored" }}old{{ end }}`), 0o644); err != nil {
		t.Fatalf("write ignored: %v", err)
	}

	set, err := Load(dir, cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !set.Has("index") {
		t.Fatalf("expected index.html to load, got %v", AvailableNames(set))
	}
	if set.Has("ignored") {
		t.Fatalf("expected non-HTML files to be ignored, got %v", AvailableNames(set))
	}
}

func TestDocumentDirectionTemplateHelper(t *testing.T) {
	got, err := executeTemplateText(`{{ documentDirection . }}`, "ar")
	if err != nil {
		t.Fatalf("execute template helper: %v", err)
	}
	if got != "rtl" {
		t.Fatalf("expected rtl, got %q", got)
	}
}

func TestAssetTemplateHelperUsesBasePath(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/docs/"

	got, err := executeTemplateTextWithConfig(`{{ asset "css/site.css" }}`, nil, cfg)
	if err != nil {
		t.Fatalf("execute asset helper: %v", err)
	}
	if got != "/docs/css/site.css" {
		t.Fatalf("expected base-path asset URL, got %q", got)
	}
}

func TestAssetTemplateHelperRejectsTraversal(t *testing.T) {
	_, err := executeTemplateText(`{{ asset "../secret.css" }}`, nil)
	if err == nil {
		t.Fatal("expected traversal asset path to fail")
	}
}

func TestImagePresetTemplateHelpers(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/blog/"

	got, err := executeTemplateTextWithConfig(`{{ imagePresetSrcset "thumb" "images/photo.jpg" }}|{{ imagePresetSizes "thumb" }}`, nil, cfg)
	if err != nil {
		t.Fatalf("execute image preset helpers: %v", err)
	}

	want := "/blog/images/photo.320w.jpg 320w, /blog/images/photo.640w.jpg 640w|(max-width: 700px) 50vw, 320px"
	if got != want {
		t.Fatalf("unexpected preset output\nwant: %q\n got: %q", want, got)
	}
}

func TestReadFileRejectsAbsolutePath(t *testing.T) {
	got, err := executeTemplateText(`{{ readFile "/etc/passwd" }}`, nil)
	if err == nil {
		t.Fatalf("expected absolute readFile path to fail, got %q", got)
	}
}

func TestReadFileRejectsTraversal(t *testing.T) {
	got, err := executeTemplateText(`{{ readFile "../secret.txt" }}`, nil)
	if err == nil {
		t.Fatalf("expected traversal readFile path to fail, got %q", got)
	}
}

func TestReadFileAllowsContentBundleResource(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	path := filepath.Join(dir, cfg.ContentDir, "publications", "example-paper")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir content bundle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "cite.bib"), []byte("@article{example}"), 0o644); err != nil {
		t.Fatalf("write bundle resource: %v", err)
	}

	got, err := executeTemplateTextWithSiteRoot(`{{ readFile "content/publications/example-paper/cite.bib" }}`, nil, dir, cfg)
	if err != nil {
		t.Fatalf("expected content bundle resource to be readable: %v", err)
	}
	if got != "@article{example}" {
		t.Fatalf("unexpected content bundle resource\nwant: %q\n got: %q", "@article{example}", got)
	}
}

func osMkdirAll(path string, mode uint32) error {
	return os.MkdirAll(path, os.FileMode(mode))
}

func executeTemplateText(text string, data any) (string, error) {
	return executeTemplateTextWithConfig(text, data, config.DefaultSiteConfig())
}

func executeTemplateTextWithConfig(text string, data any, cfg config.SiteConfig) (string, error) {
	return executeTemplateTextWithSiteRoot(text, data, ".", cfg)
}

func executeTemplateTextWithSiteRoot(text string, data any, siteRoot string, cfg config.SiteConfig) (string, error) {
	tmpl, err := template.New("test").Funcs(funcMap(siteRoot, cfg)).Parse(text)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func TestLoadAppliesThemeChainThenSiteOverrides(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("themes/base/config.toml", "")
	write("themes/base/templates/base.html", `{{ define "footer" }}base-footer{{ end }}{{ define "brand" }}base-brand{{ end }}`)
	write("themes/base/templates/page.html", `{{ define "page" }}base-page {{ template "brand" }} {{ template "footer" }}{{ end }}`)
	write("themes/base/templates/post.html", `{{ define "post" }}base-post{{ end }}`)
	write("themes/ink/config.toml", `extends = "base"`)
	write("themes/ink/templates/partials/brand.html", `{{ define "brand" }}ink-brand{{ end }}`)
	write("themes/ink/templates/post.html", `{{ define "post" }}ink-post {{ template "footer" }}{{ end }}`)
	write("templates/partials/footer.html", `{{ define "footer" }}site-footer{{ end }}`)

	cfg := config.DefaultSiteConfig()
	cfg.Theme = "ink"
	set, err := Load(root, cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	page, err := set.Execute("page", nil)
	if err != nil {
		t.Fatalf("execute page: %v", err)
	}
	if page != "base-page ink-brand site-footer" {
		t.Fatalf("unexpected page output %q", page)
	}
	post, err := set.Execute("post", nil)
	if err != nil {
		t.Fatalf("execute post: %v", err)
	}
	if post != "ink-post site-footer" {
		t.Fatalf("unexpected post output %q", post)
	}
}
