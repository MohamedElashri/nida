package migrate

import (
	"strings"
	"testing"
)

func upper(prose string, _ int) string { return strings.ToUpper(prose) }

func TestRewriteProseSkipsCode(t *testing.T) {
	in := "a `b` c\n\n```go\nd\n```\ne ``f ` g`` h\n\n~~~~\n```\ni\n~~~~\nj\n   ```\nk\n   ```\nl\n"
	want := "A `b` C\n\n```go\nd\n```\nE ``f ` g`` H\n\n~~~~\n```\ni\n~~~~\nJ\n   ```\nk\n   ```\nL\n"
	if got := RewriteProse(in, upper); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRewriteProseUnclosedFenceRunsToEnd(t *testing.T) {
	in := "a\n```\nb\n"
	if got := RewriteProse(in, upper); got != "A\n```\nb\n" {
		t.Errorf("got %q", got)
	}
}

func TestRewriteProseUnmatchedBacktickIsText(t *testing.T) {
	if got := RewriteProse("it`s here", upper); got != "IT`S HERE" {
		t.Errorf("got %q", got)
	}
	if got := RewriteProse("a `b\n\nc` d", upper); got != "A `B\n\nC` D" {
		t.Errorf("a code span must not cross a blank line, got %q", got)
	}
}

func TestRewriteProseOffsets(t *testing.T) {
	body := "one\n```\ntwo\n```\nthree {{x}}\n"
	var lines []int
	RewriteProse(body, func(prose string, offset int) string {
		if i := strings.Index(prose, "{{x}}"); i >= 0 {
			lines = append(lines, LineOf(body, offset+i))
		}
		return prose
	})
	if len(lines) != 1 || lines[0] != 5 {
		t.Errorf("lines = %v, want [5]", lines)
	}
}

func TestHasRawHTML(t *testing.T) {
	for in, want := range map[string]bool{
		"plain text":                   false,
		"a <!-- more --> b":            false,
		"<div>x</div>":                 true,
		"line<br>break":                true,
		"a < b and c > d":              false,
		"```\n<div>in code</div>\n```": false,
		"`<span>`":                     false,
		"<!-- a comment -->":           true,
	} {
		if got := HasRawHTML(in); got != want {
			t.Errorf("HasRawHTML(%q) = %t, want %t", in, got, want)
		}
	}
}

func TestShortcodeFormatting(t *testing.T) {
	got := Shortcode("figure", []Arg{{Key: "src", Value: `a "b".png`}, {Value: `c\d`}})
	if want := `{{< figure src="a \"b\".png" "c\\d" >}}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
