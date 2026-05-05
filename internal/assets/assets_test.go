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

	if err := Copy(dir, cfg); err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "public", "rss.xml"))
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(got) != "generated" {
		t.Fatalf("expected generated content to be preserved, got %q", string(got))
	}
}

func TestCopyPageBundles(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.ContentDir = "content"
	cfg.OutputDir = "public"

	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "index.md"), "markdown")
	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "image.png"), "fake-png")
	writeContentFile(t, filepath.Join(dir, "content", "posts", "my-bundle", "chart.svg"), "<svg></svg>")

	err := CopyPageBundles(dir, cfg, []BundlePage{
		{
			URL:       "/posts/my-bundle/",
			BundleDir: "posts/my-bundle",
			Resources: []string{"chart.svg", "image.png"},
		},
	})
	if err != nil {
		t.Fatalf("CopyPageBundles returned error: %v", err)
	}

	imageData, err := os.ReadFile(filepath.Join(dir, "public", "posts", "my-bundle", "image.png"))
	if err != nil {
		t.Fatalf("read copied image: %v", err)
	}
	if string(imageData) != "fake-png" {
		t.Fatalf("unexpected image content %q", string(imageData))
	}

	svgData, err := os.ReadFile(filepath.Join(dir, "public", "posts", "my-bundle", "chart.svg"))
	if err != nil {
		t.Fatalf("read copied svg: %v", err)
	}
	if string(svgData) != "<svg></svg>" {
		t.Fatalf("unexpected svg content %q", string(svgData))
	}
}

func TestCopyPageBundlesNoResources(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.ContentDir = "content"
	cfg.OutputDir = "public"

	err := CopyPageBundles(dir, cfg, []BundlePage{
		{
			URL:       "/posts/empty/",
			BundleDir: "posts/empty",
			Resources: nil,
		},
	})
	if err != nil {
		t.Fatalf("CopyPageBundles returned error: %v", err)
	}
}

func TestCopyPageBundlesMissingSource(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.ContentDir = "content"
	cfg.OutputDir = "public"

	err := CopyPageBundles(dir, cfg, []BundlePage{
		{
			URL:       "/posts/ghost/",
			BundleDir: "posts/ghost",
			Resources: []string{"does-not-exist.png"},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing resource file")
	}
}

func writeContentFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir content dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write content file: %v", err)
	}
}
