package render

import "testing"

func TestMinifyHTMLCollapsesWhitespaceBetweenTags(t *testing.T) {
	got := minifyHTML("<main>\n  <h1>Hello</h1>\n  <p>World</p>\n</main>\n")
	want := "<main><h1>Hello</h1><p>World</p></main>\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
