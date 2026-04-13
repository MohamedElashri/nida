package highlight

import (
	"strings"
	"testing"
)

func TestRenderHighlightedCode(t *testing.T) {
	html, err := Render("package main\n", "go", "github")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(html, "<pre") {
		t.Fatalf("expected pre tag, got %q", html)
	}
	if !strings.Contains(html, "package") {
		t.Fatalf("expected highlighted code output, got %q", html)
	}
}

func TestRenderUnknownLanguageFallsBack(t *testing.T) {
	html, err := Render("hello\n", "unknownlang", "github")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(html, `class="language-unknownlang"`) {
		t.Fatalf("expected language class fallback, got %q", html)
	}
	if !strings.Contains(html, "hello") {
		t.Fatalf("expected escaped code output, got %q", html)
	}
}
