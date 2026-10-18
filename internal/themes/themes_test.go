package themes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func writeTheme(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "themes", name)
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestChainIsParentFirstAndChildExtraWins(t *testing.T) {
	root := t.TempDir()
	writeTheme(t, root, "base", "[extra]\ncolor = \"blue\"\nfont = \"serif\"\n")
	writeTheme(t, root, "ink", "extends = \"base\"\n[extra]\ncolor = \"black\"\n")
	cfg := config.DefaultSiteConfig()
	cfg.Theme = "ink"

	chain, err := Chain(root, cfg)
	if err != nil {
		t.Fatalf("Chain returned error: %v", err)
	}
	if len(chain) != 2 || chain[0].Name != "base" || chain[1].Name != "ink" {
		t.Fatalf("unexpected chain %+v", chain)
	}

	merged := MergeExtra(chain, map[string]any{"font": "mono"})
	if merged["color"] != "black" || merged["font"] != "mono" {
		t.Fatalf("expected child and site values to win, got %v", merged)
	}

	dirs, err := Dirs(root, cfg, "templates")
	if err != nil {
		t.Fatalf("Dirs returned error: %v", err)
	}
	if len(dirs) != 2 || !strings.HasSuffix(dirs[0], filepath.Join("base", "templates")) {
		t.Fatalf("unexpected dirs %v", dirs)
	}
}

func TestChainRejectsCycles(t *testing.T) {
	root := t.TempDir()
	writeTheme(t, root, "a", "extends = \"b\"\n")
	writeTheme(t, root, "b", "extends = \"a\"\n")
	cfg := config.DefaultSiteConfig()
	cfg.Theme = "a"

	if _, err := Chain(root, cfg); err == nil || !strings.Contains(err.Error(), "circular") {
		t.Fatalf("expected circular inheritance error, got %v", err)
	}
}
