package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/buildinfo"
	"github.com/MohamedElashri/nida/internal/config"
)

func TestBuildLoadsConfigFromSiteRoot(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{
		"build",
		"--site", filepath.Join("..", "..", "example-site"),
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "config=") {
		t.Fatalf("expected config path in output, got %q", stdout.String())
	}
}

func TestBuildLoadsArabicExampleSite(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{
		"build",
		"--site", filepath.Join("..", "..", "example-site-ar"),
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "rendered=") {
		t.Fatalf("expected build summary in output, got %q", stdout.String())
	}
}

func TestInitCreatesBuildableExampleSite(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	target := filepath.Join(t.TempDir(), "my-site")

	code := run(stdout, stderr, []string{"init", target})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nida init: created") {
		t.Fatalf("expected init summary in output, got %q", stdout.String())
	}

	for _, path := range []string{
		"config.toml",
		"content/_index.md",
		"content/pages/about.md",
		"content/pages/search.md",
		"content/posts/_index.md",
		"content/posts/welcome.md",
		"content/posts/markdown-tour.md",
		"content/posts/content-model/index.md",
		"content/posts/content-model/bundle-note.txt",
		"templates/base.html",
		"templates/index.html",
		"templates/post.html",
		"templates/page.html",
		"templates/search.html",
		"templates/list.html",
		"templates/taxonomy.html",
		"static/site.css",
	} {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(path))); err != nil {
			t.Fatalf("expected scaffolded file %s: %v", path, err)
		}
	}

	buildOut := &bytes.Buffer{}
	buildErr := &bytes.Buffer{}
	code = run(buildOut, buildErr, []string{"build", "--site", target})
	if code != 0 {
		t.Fatalf("expected scaffolded site to build, got %d stderr=%s", code, buildErr.String())
	}
	if !strings.Contains(buildOut.String(), "rendered=") {
		t.Fatalf("expected build summary in output, got %q", buildOut.String())
	}
	if _, err := os.Stat(filepath.Join(target, "public", "index.html")); err != nil {
		t.Fatalf("expected generated homepage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "posts", "markdown-tour", "index.html")); err != nil {
		t.Fatalf("expected generated markdown tour: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "tags", "workflow", "index.html")); err != nil {
		t.Fatalf("expected generated taxonomy page: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "search", "index.html")); err != nil {
		t.Fatalf("expected generated search page: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "search_index.en.js")); err != nil {
		t.Fatalf("expected generated search index: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "posts", "content-model", "bundle-note.txt")); err != nil {
		t.Fatalf("expected copied bundle resource: %v", err)
	}
}

func TestInitRefusesNonEmptyDirectory(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "keep.txt"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	code := run(stdout, stderr, []string{"init", target})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "is not empty") {
		t.Fatalf("expected non-empty error, got %q", stderr.String())
	}
}

func TestLoadCommandConfigAppliesServeOverrides(t *testing.T) {
	cfg, _, err := loadCommandConfig(commandOptions{
		siteRoot: filepath.Join("..", "..", "example-site"),
		drafts:   true,
		port:     1313,
	})
	if err != nil {
		t.Fatalf("loadCommandConfig returned error: %v", err)
	}
	if !cfg.Drafts {
		t.Fatal("expected drafts override")
	}
	if cfg.Server.Port != 1313 {
		t.Fatalf("expected port override 1313, got %d", cfg.Server.Port)
	}
}

func TestLoadCommandConfigUsesUpdatedDefaultServePort(t *testing.T) {
	cfg, _, err := loadCommandConfig(commandOptions{
		siteRoot: filepath.Join("..", "..", "example-site"),
	})
	if err != nil {
		t.Fatalf("loadCommandConfig returned error: %v", err)
	}
	if cfg.Server.Port != 1702 {
		t.Fatalf("expected default port 1702, got %d", cfg.Server.Port)
	}
}

func TestBuildReportsConfigErrors(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{
		"build",
		"--site", t.TempDir(),
	})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "error: load config") {
		t.Fatalf("expected config error, got %q", stderr.String())
	}
}

func TestRebuildModeTreatsContentResourcesAsFull(t *testing.T) {
	cfg := testRebuildConfig()

	got := rebuildMode(cfg, []string{"content/posts/bundle/image.png"})
	if got != "full" {
		t.Fatalf("expected content resource change to trigger full rebuild, got %q", got)
	}
}

func TestVersionReportsReleaseVersionOnly(t *testing.T) {
	originalVersion := buildinfo.Version
	originalCommit := buildinfo.Commit
	originalDate := buildinfo.Date
	originalBuiltBy := buildinfo.BuiltBy
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
		buildinfo.Commit = originalCommit
		buildinfo.Date = originalDate
		buildinfo.BuiltBy = originalBuiltBy
	})

	buildinfo.Version = "0.2.0"
	buildinfo.Commit = "abc1234"
	buildinfo.Date = "2026-04-23T10:00:00Z"
	buildinfo.BuiltBy = "goreleaser"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{"version"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	output := stdout.String()
	if output != "nida version 0.2.0\n" {
		t.Fatalf("expected release version only, got %q", output)
	}
}

func testRebuildConfig() config.SiteConfig {
	cfg := config.DefaultSiteConfig()
	cfg.ContentDir = "content"
	cfg.StaticDir = "static"
	cfg.TemplateDir = "templates"
	return cfg
}

func TestVersionReportsDevBuildMetadata(t *testing.T) {
	originalVersion := buildinfo.Version
	originalCommit := buildinfo.Commit
	originalDate := buildinfo.Date
	originalBuiltBy := buildinfo.BuiltBy
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
		buildinfo.Commit = originalCommit
		buildinfo.Date = originalDate
		buildinfo.BuiltBy = originalBuiltBy
	})

	buildinfo.Version = "dev"
	buildinfo.Commit = "abc1234"
	buildinfo.Date = "2026-04-23T10:00:00Z"
	buildinfo.BuiltBy = "local"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{"version"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "nida version dev") {
		t.Fatalf("expected version in output, got %q", output)
	}
	if !strings.Contains(output, "commit=abc1234") {
		t.Fatalf("expected commit in output, got %q", output)
	}
	if !strings.Contains(output, "builtBy=local") {
		t.Fatalf("expected builtBy in output, got %q", output)
	}
}

func TestHelpIncludesVersionCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{"help"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nida version") {
		t.Fatalf("expected version command in help, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "nida init [PATH]") {
		t.Fatalf("expected init command in help, got %q", stdout.String())
	}
}

func TestColorsDoNotDecorateNonTerminalWriters(t *testing.T) {
	colors := colorsFor(&bytes.Buffer{})

	got := colors.command("nida build:")
	if got != "nida build:" {
		t.Fatalf("expected plain command for non-terminal writer, got %q", got)
	}
}

func TestColorsRespectNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	colors := terminalColors{enabled: writerSupportsColor(os.Stdout)}
	got := colors.error("error:")
	if got != "error:" {
		t.Fatalf("expected plain error with NO_COLOR, got %q", got)
	}
}
