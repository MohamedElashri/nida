package templates

import (
	"os"
	"path/filepath"
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

func osMkdirAll(path string, mode uint32) error {
	return os.MkdirAll(path, os.FileMode(mode))
}
