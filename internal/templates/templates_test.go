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

func TestDocumentDirectionTemplateHelper(t *testing.T) {
	got, err := executeTemplateText(`{{ documentDirection . }}`, "ar")
	if err != nil {
		t.Fatalf("execute template helper: %v", err)
	}
	if got != "rtl" {
		t.Fatalf("expected rtl, got %q", got)
	}
}

func osMkdirAll(path string, mode uint32) error {
	return os.MkdirAll(path, os.FileMode(mode))
}

func executeTemplateText(text string, data any) (string, error) {
	tmpl, err := template.New("test").Funcs(funcMap()).Parse(text)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}
