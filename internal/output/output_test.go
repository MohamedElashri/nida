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
		{URL: "/404.html", Content: "not found"},
	}

	if err := WriteSite(dir, cfg, pages); err != nil {
		t.Fatalf("WriteSite returned error: %v", err)
	}

	assertFile(t, filepath.Join(dir, "public", "index.html"), "home")
	assertFile(t, filepath.Join(dir, "public", "posts", "hello", "index.html"), "post")
	assertFile(t, filepath.Join(dir, "public", "404.html"), "not found")
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

func TestWriteFileRejectsEscapingArtifactPath(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	err := WriteFile(dir, cfg, "../outside.xml", []byte("feed"))
	if err == nil {
		t.Fatal("expected escaping artifact path to be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "outside.xml")); !os.IsNotExist(statErr) {
		t.Fatalf("expected outside file not to be written, stat err=%v", statErr)
	}
}

func TestWriteFileRejectsSymlinkedOutputParent(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "public", "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	err := WriteFile(dir, cfg, "linked/rss.xml", []byte("feed"))
	if err == nil {
		t.Fatal("expected symlinked output parent to be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "rss.xml")); !os.IsNotExist(statErr) {
		t.Fatalf("expected outside file not to be written, stat err=%v", statErr)
	}
}

func TestWriteFileRejectsSymlinkedOutputRoot(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	if err := os.Symlink(outside, filepath.Join(dir, "public")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	err := WriteFile(dir, cfg, "rss.xml", []byte("feed"))
	if err == nil {
		t.Fatal("expected symlinked output root to be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "rss.xml")); !os.IsNotExist(statErr) {
		t.Fatalf("expected outside file not to be written, stat err=%v", statErr)
	}
}

func TestWriteSiteRejectsSymlinkedOutputAncestor(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = filepath.Join("linked", "public")

	if err := os.Symlink(outside, filepath.Join(dir, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	victim := filepath.Join(outside, "public", "keep.txt")
	if err := os.MkdirAll(filepath.Dir(victim), 0o755); err != nil {
		t.Fatalf("mkdir victim dir: %v", err)
	}
	if err := os.WriteFile(victim, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write victim: %v", err)
	}

	err := WriteSite(dir, cfg, []render.Page{{URL: "/", Content: "home"}})
	if err == nil {
		t.Fatal("expected symlinked output ancestor to be rejected")
	}
	assertFile(t, victim, "keep")
}

func TestWriteSiteRejectsEscapingOutputDir(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = ".."

	err := WriteSite(dir, cfg, []render.Page{{URL: "/", Content: "home"}})
	if err == nil {
		t.Fatal("expected escaping output dir to be rejected")
	}
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

func TestValidateWritePlanRejectsEscapingArtifactPath(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	err := ValidateWritePlan(dir, cfg, nil, []Artifact{{Path: "../rss.xml"}})
	if err == nil {
		t.Fatal("expected escaping artifact path to be rejected")
	}
}

func TestRemovePagesCleansEmptyDirectories(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	pages := []render.Page{
		{URL: "/posts/hello/", Content: "post"},
		{URL: "/posts/world/", Content: "post"},
		{URL: "/posts/", Content: "listing"},
	}
	if err := WriteSite(dir, cfg, pages); err != nil {
		t.Fatalf("WriteSite returned error: %v", err)
	}

	if err := RemovePages(dir, cfg, []string{"/posts/hello/"}); err != nil {
		t.Fatalf("RemovePages returned error: %v", err)
	}

	helloDir := filepath.Join(dir, "public", "posts", "hello")
	if _, err := os.Stat(helloDir); !os.IsNotExist(err) {
		t.Fatalf("expected empty directory %q removed, stat err=%v", helloDir, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "public", "posts", "world", "index.html")); err != nil {
		t.Fatalf("expected sibling page preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "public", "posts", "index.html")); err != nil {
		t.Fatalf("expected listing page preserved: %v", err)
	}
}

func TestRemovePagesKeepsNonEmptyDirectories(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultSiteConfig()
	cfg.OutputDir = "public"

	pages := []render.Page{
		{URL: "/", Content: "home"},
		{URL: "/bundle/", Content: "post"},
	}
	if err := WriteSite(dir, cfg, pages); err != nil {
		t.Fatalf("WriteSite returned error: %v", err)
	}
	bundleDir := filepath.Join(dir, "public", "bundle")
	if err := os.WriteFile(filepath.Join(bundleDir, "note.txt"), []byte("resource"), 0o644); err != nil {
		t.Fatalf("write bundle resource: %v", err)
	}

	if err := RemovePages(dir, cfg, []string{"/bundle/"}); err != nil {
		t.Fatalf("RemovePages returned error: %v", err)
	}

	if _, err := os.Stat(bundleDir); err != nil {
		t.Fatalf("expected non-empty directory preserved, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "public", "index.html")); err != nil {
		t.Fatalf("expected output root preserved: %v", err)
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
