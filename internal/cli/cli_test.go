package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
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
	if cfg.Server.Port != 2906 {
		t.Fatalf("expected default port 2906, got %d", cfg.Server.Port)
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
