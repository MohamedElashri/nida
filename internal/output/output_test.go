package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/render"
)

func TestWriteSiteWritesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	pages := []render.Page{
		{URL: "/", Content: "home"},
		{URL: "/posts/hello/", Content: "post"},
	}

	if err := WriteSite(dir, cfg, pages); err != nil {
		t.Fatalf("WriteSite returned error: %v", err)
	}

	assertFile(t, filepath.Join(dir, "public", "index.html"), "home")
	assertFile(t, filepath.Join(dir, "public", "posts", "hello", "index.html"), "post")
}

func TestWriteSiteCleansStaleFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	stale := filepath.Join(dir, "public", "stale.txt")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatalf("mkdir stale dir: %v", err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	if err := WriteSite(dir, cfg, []render.Page{{URL: "/", Content: "home"}}); err != nil {
		t.Fatalf("WriteSite returned error: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("expected stale file removed, stat err=%v", err)
	}
}

func TestWriteFileWritesArtifact(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	if err := WriteFile(dir, cfg, "rss.xml", []byte("feed")); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	assertFile(t, filepath.Join(dir, "public", "rss.xml"), "feed")
}

func TestValidateWritePlanRejectsPageArtifactConflict(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	err := ValidateWritePlan(dir, cfg, []render.Page{
		{URL: "/rss.xml", Content: "page"},
	}, []Artifact{
		{Path: "rss.xml"},
	})
	if err == nil {
		t.Fatal("expected output conflict error")
	}
}

func TestValidateWritePlanRejectsDuplicatePageTargets(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	err := ValidateWritePlan(dir, cfg, []render.Page{
		{URL: "/posts/"},
		{URL: "/posts/"},
	}, nil)
	if err == nil {
		t.Fatal("expected duplicate page output conflict")
	}
}

func assertFile(t *testing.T, path string, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("unexpected content for %q: want %q got %q", path, want, string(got))
	}
}
