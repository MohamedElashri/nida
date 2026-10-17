package datafiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestLoadMergesFormatsDirectoriesAndThemes(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"themes/t/data/social.toml": "github = \"theme\"\nmastodon = \"theme\"\n",
		"themes/t/data/nav.yaml":    "- name: Theme\n",
		"data/social.toml":          "github = \"site\"\n",
		"data/team/core.json":       `[{"name": "Ada"}]`,
		"data/releases.yml":         "latest: 0.7.0\n",
		"data/.hidden/secret.toml":  "x = 1\n",
		"data/notes.txt":            "ignored",
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

	data, err := Load(root, cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	social := data["social"].(map[string]any)
	if social["github"] != "site" || social["mastodon"] != nil {
		t.Fatalf("site file should replace the theme file, got %v", social)
	}
	if nav, ok := data["nav"].([]any); !ok || len(nav) != 1 {
		t.Fatalf("expected theme nav data, got %#v", data["nav"])
	}
	core := data["team"].(map[string]any)["core"].([]any)
	if core[0].(map[string]any)["name"] != "Ada" {
		t.Fatalf("unexpected nested data %#v", core)
	}
	if data["releases"].(map[string]any)["latest"] != "0.7.0" {
		t.Fatalf("unexpected yaml data %#v", data["releases"])
	}
	if _, ok := data[".hidden"]; ok {
		t.Fatal("hidden directories must be skipped")
	}
}
