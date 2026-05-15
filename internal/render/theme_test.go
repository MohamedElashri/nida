package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestLoadInlineCSSReturnsFallbackReadError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatalf("mkdir templates dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "static", "site.css"), 0o755); err != nil {
		t.Fatalf("mkdir site.css directory: %v", err)
	}

	cfg := config.DefaultSiteConfig()
	cfg.TemplateDir = "templates"
	cfg.StaticDir = "static"

	_, err := loadInlineCSS(dir, cfg)
	if err == nil {
		t.Fatal("expected fallback read error")
	}
	if strings.Contains(err.Error(), "style.css.html") {
		t.Fatalf("expected fallback read error, got stale template error %q", err)
	}
}
