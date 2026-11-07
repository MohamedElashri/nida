package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestRewriteCSSPointsReferencesAtFingerprints(t *testing.T) {
	manifest := Manifest{
		"fonts/a.woff2": "fonts/a.11111111.woff2",
		"img/bg.png":    "img/bg.22222222.png",
	}
	css := `@font-face{src:url("../fonts/a.woff2") format("woff2")}` +
		`.a{background:url('../img/bg.png?v=1#x')}` +
		`.b{background:url(/blog/fonts/a.woff2)}` +
		`.c{background:url(data:image/png;base64,AAAA)}` +
		`.d{background:url(https://cdn.example.com/a.woff2)}` +
		`.e{background:url(missing.png)}`

	got := rewriteCSS(css, "css/site.css", manifest, "/blog")
	want := `@font-face{src:url("../fonts/a.11111111.woff2") format("woff2")}` +
		`.a{background:url('../img/bg.22222222.png?v=1#x')}` +
		`.b{background:url(/blog/fonts/a.11111111.woff2)}` +
		`.c{background:url(data:image/png;base64,AAAA)}` +
		`.d{background:url(https://cdn.example.com/a.woff2)}` +
		`.e{background:url(missing.png)}`
	if got != want {
		t.Fatalf("rewriteCSS() =\n%s\nwant\n%s", got, want)
	}
}

func TestRewriteCSSWithoutBasePath(t *testing.T) {
	manifest := Manifest{"fonts/a.woff2": "fonts/a.11111111.woff2"}
	got := rewriteCSS(`src:url(/fonts/a.woff2) url(fonts/a.woff2)`, "site.css", manifest, "")
	want := `src:url(/fonts/a.11111111.woff2) url(fonts/a.11111111.woff2)`
	if got != want {
		t.Fatalf("rewriteCSS() = %q, want %q", got, want)
	}
}

func TestFinishStylesheetsWritesImportsFirst(t *testing.T) {
	out := t.TempDir()
	manifest := Manifest{"fonts/a.woff2": "fonts/a.11111111.woff2"}
	pending := []pendingStylesheet{
		{rel: "main.css", data: []byte(`@import "theme.css";body{font:1em x}`)},
		{rel: "theme.css", data: []byte(`@font-face{src:url(fonts/a.woff2)}`)},
	}
	if err := finishStylesheets(out, pending, manifest, ""); err != nil {
		t.Fatalf("finishStylesheets: %v", err)
	}

	theme := readOutput(t, out, manifest["theme.css"])
	if !strings.Contains(theme, "fonts/a.11111111.woff2") {
		t.Fatalf("theme.css not rewritten: %q", theme)
	}
	main := readOutput(t, out, manifest["main.css"])
	if !strings.Contains(main, `@import "`+filepath.Base(manifest["theme.css"])+`"`) {
		t.Fatalf("main.css import not rewritten to %s: %q", manifest["theme.css"], main)
	}
}

func TestFinishStylesheetsSurvivesImportCycle(t *testing.T) {
	out := t.TempDir()
	manifest := Manifest{}
	pending := []pendingStylesheet{
		{rel: "a.css", data: []byte(`@import "b.css";`)},
		{rel: "b.css", data: []byte(`@import "a.css";`)},
	}
	if err := finishStylesheets(out, pending, manifest, ""); err != nil {
		t.Fatalf("finishStylesheets: %v", err)
	}
	if manifest["a.css"] == "" || manifest["b.css"] == "" {
		t.Fatalf("expected both stylesheets fingerprinted, manifest %v", manifest)
	}
}

func TestProcessFingerprintsAssetsReferencedFromCSS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "static", "site.css"), `@font-face{src:url("fonts/a.woff2") format("woff2")}`)
	writeFile(t, filepath.Join(dir, "static", "fonts", "a.woff2"), "font bytes")

	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/blog/"
	cfg.StaticDir = "static"
	cfg.OutputDir = "public"
	cfg.Pipeline.Fingerprint = true
	cfg.Pipeline.MinifyCSS = true

	manifest, err := Process(dir, cfg)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	out := filepath.Join(dir, "public")
	font := manifest["fonts/a.woff2"]
	css := readOutput(t, out, manifest["site.css"])
	if !strings.Contains(css, filepath.Base(font)) {
		t.Fatalf("site.css %q does not reference %s", css, font)
	}

	sum := sha256.Sum256([]byte(css))
	if want := "site." + hex.EncodeToString(sum[:])[:8] + ".css"; manifest["site.css"] != want {
		t.Fatalf("site.css fingerprint %s does not hash the rewritten bytes (want %s)", manifest["site.css"], want)
	}
	if _, err := os.Stat(filepath.Join(out, "site.css")); !os.IsNotExist(err) {
		t.Fatalf("unfingerprinted site.css should not be written, stat err %v", err)
	}

	var written Manifest
	if err := json.Unmarshal([]byte(readOutput(t, out, "manifest.json")), &written); err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}
	if written["site.css"] != manifest["site.css"] {
		t.Fatalf("manifest.json has %q for site.css, want %q", written["site.css"], manifest["site.css"])
	}
}

func readOutput(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
