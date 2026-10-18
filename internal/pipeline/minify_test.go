package pipeline

import "testing"

func TestMinifyHTMLKeepsMeaningfulWhitespace(t *testing.T) {
	got := MinifyHTML("<main>\n  <h1>Hello</h1>\n  <p><a href=\"/a\">one</a> <b>two</b></p>\n  <pre>  keep\n   this</pre>\n</main>\n")
	want := "<main><h1>Hello</h1><p><a href=\"/a\">one</a> <b>two</b></p><pre>  keep\n   this</pre></main>\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
