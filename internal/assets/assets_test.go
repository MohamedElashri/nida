package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestCopyPreservesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.StaticDir = "static"
	cfg.OutputDir = "public"

	src := filepath.Join(dir, "static", "images", "logo.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(src, []byte("asset"), 0o644); err != nil {
		t.Fatalf("write static file: %v", err)
	}

	if err := Copy(dir, cfg); err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "public", "images", "logo.txt"))
	if err != nil {
		t.Fatalf("read copied asset: %v", err)
	}
	if string(got) != "asset" {
		t.Fatalf("unexpected copied content %q", string(got))
	}
}

func TestCopyRejectsGeneratedOutputConflicts(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.StaticDir = "static"
	cfg.OutputDir = "public"

	src := filepath.Join(dir, "static", "rss.xml")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(src, []byte("asset"), 0o644); err != nil {
		t.Fatalf("write static file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public", "rss.xml"), []byte("generated"), 0o644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}

	err := Copy(dir, cfg)
	if err == nil {
		t.Fatal("expected conflict error")
	}
}
