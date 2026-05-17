package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/markdown"
)

func TestCheckReportsBrokenInternalLink(t *testing.T) {
	dir := t.TempDir()
	page := writeDiagnosticPage(t, dir, "content/posts/hello.md", `[Missing](@/posts/missing.md)`)

	err := Check(dir, testConfig(), []content.Page{page}, nil, markdown.PathLookup{})
	if err == nil {
		t.Fatal("expected diagnostics error")
	}
	if !strings.Contains(err.Error(), `broken internal link "@/posts/missing.md"`) {
		t.Fatalf("expected broken internal link diagnostic, got %v", err)
	}
}

func TestCheckIgnoresInternalLinkInsideCodeBlock(t *testing.T) {
	dir := t.TempDir()
	page := writeDiagnosticPage(t, dir, "content/posts/hello.md", "```md\n[Missing](@/posts/missing.md)\n```\n")

	err := Check(dir, testConfig(), []content.Page{page}, nil, markdown.PathLookup{})
	if err != nil {
		t.Fatalf("expected no diagnostics error, got %v", err)
	}
}

func TestCheckReportsMissingRelativeImage(t *testing.T) {
	dir := t.TempDir()
	page := writeDiagnosticPage(t, dir, "content/posts/hello/index.md", `![Missing](missing.png)`)

	err := Check(dir, testConfig(), []content.Page{page}, nil, markdown.PathLookup{})
	if err == nil {
		t.Fatal("expected diagnostics error")
	}
	if !strings.Contains(err.Error(), `missing image asset "missing.png"`) {
		t.Fatalf("expected missing image diagnostic, got %v", err)
	}
}

func TestCheckAcceptsRelativeAndStaticImages(t *testing.T) {
	dir := t.TempDir()
	page := writeDiagnosticPage(t, dir, "content/posts/hello/index.md", `![Local](screenshot.png) ![Static](/images/logo.png)`)
	writeFile(t, dir, "content/posts/hello/screenshot.png", "image")
	writeFile(t, dir, "static/images/logo.png", "image")

	err := Check(dir, testConfig(), []content.Page{page}, nil, markdown.PathLookup{})
	if err != nil {
		t.Fatalf("expected no diagnostics error, got %v", err)
	}
}

func TestCheckAcceptsInternalLinkWithFragment(t *testing.T) {
	dir := t.TempDir()
	page := writeDiagnosticPage(t, dir, "content/posts/hello.md", `[About](@/pages/about.md#team)`)
	lookup := markdown.PathLookup{"pages/about.md": "/pages/about/"}

	err := Check(dir, testConfig(), []content.Page{page}, nil, lookup)
	if err != nil {
		t.Fatalf("expected no diagnostics error, got %v", err)
	}
}

func testConfig() config.SiteConfig {
	cfg := config.DefaultSiteConfig()
	cfg.ContentDir = "content"
	cfg.StaticDir = "static"
	return cfg
}

func writeDiagnosticPage(t *testing.T, root, rel, body string) content.Page {
	t.Helper()
	path := writeFile(t, root, rel, body)
	contentRel := strings.TrimPrefix(filepath.ToSlash(rel), "content/")
	return content.Page{
		SourcePath:   path,
		RelativePath: contentRel,
		BodyMarkdown: body,
	}
}

func writeFile(t *testing.T, root, rel, body string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
	return path
}
