package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestLoadLayersLanguagesAndRoots(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"themes/t/i18n/en.toml": "read_more = \"Read more\"\nposts = \"{count} posts\"\n[nav]\nhome = \"Home\"\n",
		"themes/t/i18n/ar.toml": "read_more = \"اقرأ المزيد\"\n",
		"i18n/ar.toml":          "[nav]\nhome = \"الرئيسية\"\n",
		"i18n/ar-eg.toml":       "read_more = \"كمّل قراءة\"\n",
	}
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
	cfg.Theme = "t"
	cfg.Language = "ar-EG"

	tr, err := Load(root, cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	cases := map[string]string{
		"read_more": "كمّل قراءة",
		"nav.home":  "الرئيسية",
		"missing":   "missing",
	}
	for key, want := range cases {
		if got := tr.T(key); got != want {
			t.Errorf("T(%q) = %q, want %q", key, got, want)
		}
	}
	if got := tr.T("posts", map[string]any{"count": 3}); got != "3 posts" {
		t.Errorf("placeholder replacement: got %q", got)
	}
}
