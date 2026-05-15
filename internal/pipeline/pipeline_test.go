package pipeline

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestProcessCompilesThemeSCSSWithoutSiteStaticDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake sass shell script is Unix-only")
	}

	dir := t.TempDir()
	themeSCSSDir := filepath.Join(dir, "themes", "calm", "scss")
	if err := os.MkdirAll(themeSCSSDir, 0o755); err != nil {
		t.Fatalf("mkdir theme scss dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeSCSSDir, "site.scss"), []byte("$color: #123456;\nbody { color: $color; }\n"), 0o644); err != nil {
		t.Fatalf("write theme scss: %v", err)
	}

	binDir := t.TempDir()
	sassPath := filepath.Join(binDir, "sass")
	sassScript := "#!/bin/sh\nout=\"$4\"\nmkdir -p \"${out%/*}\"\nprintf 'compiled css\\n' > \"$out\"\n"
	if err := os.WriteFile(sassPath, []byte(sassScript), 0o755); err != nil {
		t.Fatalf("write fake sass: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.DefaultSiteConfig()
	cfg.StaticDir = "static"
	cfg.OutputDir = "public"
	cfg.Theme = "calm"
	cfg.Pipeline.SCSS.Enabled = true
	cfg.Pipeline.SCSS.EntryDir = "css"

	if _, err := Process(dir, cfg); err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "public", "css", "site.css"))
	if err != nil {
		t.Fatalf("read compiled css: %v", err)
	}
	if string(got) != "compiled css\n" {
		t.Fatalf("unexpected compiled css %q", string(got))
	}
}
