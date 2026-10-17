package pipeline

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestProcessMinifiesAndFingerprintsThemeAndSiteAssets(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"static/site.css":               "body {\n  color : red ;\n}\n/* comment */\n",
		"themes/t/static/theme.js":      "// comment\nfunction greet ( name ) {\n  return 'hi ' + name ;\n}\n",
		"themes/t/static/site.css":      "body { color: blue; }",
		"themes/t/static/vendor.min.js": "keep  this",
	}
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultSiteConfig()
	cfg.Theme = "t"
	cfg.Pipeline.MinifyCSS = true
	cfg.Pipeline.MinifyJS = true

	if _, err := Process(root, cfg); err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	read := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, "public", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	if got := read("site.css"); got != "body{color:red}" {
		t.Fatalf("site CSS should win and be minified, got %q", got)
	}
	if got := read("theme.js"); strings.Contains(got, "comment") || strings.Contains(got, "\n  ") {
		t.Fatalf("theme JS should be minified, got %q", got)
	}
	if got := read("vendor.min.js"); got != "keep  this" {
		t.Fatalf(".min.js files must be left alone, got %q", got)
	}

	cfg.Pipeline.Fingerprint = true
	manifest, err := Process(root, cfg)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	fp, ok := manifest["theme.js"]
	if !ok || fp == "theme.js" {
		t.Fatalf("expected fingerprinted theme asset, manifest=%v", manifest)
	}
	if got := read(fp); strings.Contains(got, "comment") {
		t.Fatalf("fingerprinted file should hold minified bytes, got %q", got)
	}
}

func TestProcessCopiesWebPWithoutResizing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "static", "photo.webp"), []byte("RIFF....WEBP"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultSiteConfig()
	cfg.Pipeline.Images.Enabled = true
	if _, err := Process(root, cfg); err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "public", "photo.webp")); err != nil {
		t.Fatalf("expected the WebP file to be copied: %v", err)
	}
}
