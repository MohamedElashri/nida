package pipeline

import "testing"

func TestRewriteHTMLPreservesBasePath(t *testing.T) {
	manifest := Manifest{
		"style.css": "style.12345678.css",
	}

	got := RewriteHTML(`<link rel="stylesheet" href="/nida/style.css">`, manifest)
	want := `<link rel="stylesheet" href="/nida/style.12345678.css">`
	if got != want {
		t.Fatalf("RewriteHTML() = %q, want %q", got, want)
	}
}

func TestRewriteHTMLRewritesSrcsetCandidates(t *testing.T) {
	manifest := Manifest{
		"images/photo.480w.jpg": "images/photo.aaaaaaaa.jpg",
		"images/photo.768w.jpg": "images/photo.bbbbbbbb.jpg",
	}

	got := RewriteHTML(`<img src="/nida/images/photo.jpg" srcset="/nida/images/photo.480w.jpg 480w, /nida/images/photo.768w.jpg 768w">`, manifest)
	want := `<img src="/nida/images/photo.jpg" srcset="/nida/images/photo.aaaaaaaa.jpg 480w, /nida/images/photo.bbbbbbbb.jpg 768w">`
	if got != want {
		t.Fatalf("RewriteHTML() = %q, want %q", got, want)
	}
}

func TestRewriteHTMLRewritesPictureSourceSrcset(t *testing.T) {
	manifest := Manifest{
		"images/photo.480w.jpg": "images/photo.aaaaaaaa.jpg",
	}

	got := RewriteHTML(`<source srcset="/images/photo.480w.jpg 480w">`, manifest)
	want := `<source srcset="/images/photo.aaaaaaaa.jpg 480w">`
	if got != want {
		t.Fatalf("RewriteHTML() = %q, want %q", got, want)
	}
}

func TestRewriteHTMLRewritesFontPreload(t *testing.T) {
	manifest := Manifest{
		"fonts/site.woff2": "fonts/site.12345678.woff2",
	}

	got := RewriteHTML(`<link rel="preload" href="/fonts/site.woff2" as="font" type="font/woff2" crossorigin>`, manifest)
	want := `<link rel="preload" href="/fonts/site.12345678.woff2" as="font" type="font/woff2" crossorigin>`
	if got != want {
		t.Fatalf("RewriteHTML() = %q, want %q", got, want)
	}
}

func TestRewriteHTMLLeavesExternalAssets(t *testing.T) {
	manifest := Manifest{
		"style.css": "style.12345678.css",
	}

	got := RewriteHTML(`<link rel="stylesheet" href="https://cdn.example.com/style.css">`, manifest)
	want := `<link rel="stylesheet" href="https://cdn.example.com/style.css">`
	if got != want {
		t.Fatalf("RewriteHTML() = %q, want %q", got, want)
	}
}
