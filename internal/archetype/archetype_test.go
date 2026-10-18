package archetype

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestCreateUsesDefaultAndSectionArchetypes(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultSiteConfig()
	now := time.Date(2026, 10, 17, 9, 30, 0, 0, time.UTC)

	res, err := Create(root, cfg, "posts/my-first_post.md", now)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	data, _ := os.ReadFile(res.Path)
	want := "+++\ntitle = \"My First Post\"\ndate = 2026-10-17T09:30:00Z\ndraft = true\n+++\n\n"
	if string(data) != want || res.Archetype != "built-in default" {
		t.Fatalf("unexpected file %q from %s", data, res.Archetype)
	}

	if err := os.MkdirAll(filepath.Join(root, "archetypes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "archetypes", "notes.md"), []byte("---\ntitle: {{ .Title }}\nsection: {{ .Section }}\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = Create(root, cfg, "content/notes/quick-tip", now)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if !strings.HasSuffix(res.Path, filepath.Join("content", "notes", "quick-tip", "index.md")) {
		t.Fatalf("expected a page bundle, got %s", res.Path)
	}
	data, _ = os.ReadFile(res.Path)
	if string(data) != "---\ntitle: Quick Tip\nsection: notes\n---\n" {
		t.Fatalf("unexpected section archetype output %q", data)
	}
}

func TestCreateRefusesExistingAndEscapingPaths(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultSiteConfig()
	now := time.Now()
	if _, err := Create(root, cfg, "posts/a.md", now); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(root, cfg, "posts/a.md", now); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected already exists error, got %v", err)
	}
	for _, bad := range []string{"../escape.md", "/abs.md", "posts/_index.md"} {
		if _, err := Create(root, cfg, bad, now); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}
