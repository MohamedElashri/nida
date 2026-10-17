package site

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

func writePNG(t *testing.T, path string, width, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
}

func TestImageResolver(t *testing.T) {
	root := t.TempDir()
	writePNG(t, filepath.Join(root, "content", "posts", "trip", "map.png"), 300, 200)
	writePNG(t, filepath.Join(root, "static", "img", "hero.png"), 1000, 500)

	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/blog/"
	r := newImageResolver(root, cfg)

	info, ok := r.Image("map.png", &content.Page{IsBundle: true, BundleDir: "posts/trip"})
	if !ok || info.Width != 300 || info.Height != 200 || info.Srcset != "" {
		t.Fatalf("unexpected bundle image info %+v ok=%v", info, ok)
	}

	info, ok = r.Image("/blog/img/hero.png", nil)
	if !ok || info.Width != 1000 || info.Srcset != "" {
		t.Fatalf("expected static size without srcset while the pipeline is off, got %+v", info)
	}

	cfg.Pipeline.Images.Enabled = true
	r = newImageResolver(root, cfg)
	info, _ = r.Image("/blog/img/hero.png", nil)
	want := "/blog/img/hero.480w.png 480w, /blog/img/hero.768w.png 768w, /blog/img/hero.png 1000w"
	if info.Srcset != want || info.Sizes == "" {
		t.Fatalf("srcset = %q sizes = %q, want %q", info.Srcset, info.Sizes, want)
	}

	if _, ok := r.Image("https://cdn.example.com/x.png", nil); ok {
		t.Fatal("external images must be skipped")
	}
	if _, ok := r.Image("missing.png", &content.Page{IsBundle: true, BundleDir: "posts/trip"}); ok {
		t.Fatal("missing images must be skipped")
	}
}

func TestImageResolverBundleSrcset(t *testing.T) {
	root := t.TempDir()
	writePNG(t, filepath.Join(root, "content", "posts", "trip", "map.png"), 900, 300)
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com/"
	cfg.Pipeline.Images.Enabled = true

	info, ok := newImageResolver(root, cfg).Image("map.png", &content.Page{IsBundle: true, BundleDir: "posts/trip"})
	want := "map.480w.png 480w, map.768w.png 768w, map.png 900w"
	if !ok || info.Srcset != want {
		t.Fatalf("srcset = %q, want %q", info.Srcset, want)
	}
}
