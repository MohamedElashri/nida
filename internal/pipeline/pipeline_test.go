package pipeline

import (
	"image"
	"image/png"
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

func TestImageTargetWidthsIncludesPresetWidths(t *testing.T) {
	cfg := config.ImageConfig{
		Widths: []int{768, 480, 768},
		Presets: map[string]config.ImagePresetConfig{
			"hero":  {Widths: []int{1200, 768}},
			"thumb": {Widths: []int{320}},
		},
	}

	got := imageTargetWidths(cfg)
	want := []int{320, 480, 768, 1200}
	if len(got) != len(want) {
		t.Fatalf("got widths %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got widths %v, want %v", got, want)
		}
	}
}

func TestResizeVariantsWritesSmallerWidthsOnly(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "map.png")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 900, 300))); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	cfg := config.DefaultSiteConfig()
	cfg.Pipeline.Images.Enabled = true
	out := filepath.Join(dir, "out")
	if err := ResizeVariants(src, out, cfg); err != nil {
		t.Fatalf("ResizeVariants returned error: %v", err)
	}
	for _, name := range []string{"map.320w.png", "map.480w.png", "map.640w.png", "map.768w.png"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "map.1200w.png")); !os.IsNotExist(err) {
		t.Fatal("must not upscale beyond the original width")
	}

	notImage := filepath.Join(dir, "src", "fake.png")
	if err := os.WriteFile(notImage, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ResizeVariants(notImage, out, cfg); err != nil {
		t.Fatalf("undecodable files should be skipped, got %v", err)
	}
}
