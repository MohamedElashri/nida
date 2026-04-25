package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/buildinfo"
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
	if cfg.Server.Port != 1307 {
		t.Fatalf("expected default port 1307, got %d", cfg.Server.Port)
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
}
