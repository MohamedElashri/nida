package markdown

import (
	"regexp"
	"strings"
)

// summaryMarker splits a page into summary and remainder, as in Hugo and
// Zola. It must sit on its own line.
var summaryMarker = regexp.MustCompile(`^\s*<!--\s*more\s*-->\s*$`)

// splitSummary removes the first marker outside code and returns the body,
// the Markdown before the marker, and whether one was found.
func splitSummary(source string) (body, summary string, found bool) {
	lines := strings.SplitAfter(source, "\n")
	// A fence closes with a run of the same character at least as long,
	// as in codeRanges, so a ``` line inside a ```` block does not.
	fenceChar, fenceLen := byte(0), 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		n := fenceRun(trimmed)
		if fenceChar == 0 && n >= 3 {
			fenceChar, fenceLen = trimmed[0], n
			continue
		}
		if fenceChar != 0 {
			if n >= fenceLen && trimmed[0] == fenceChar && strings.TrimSpace(trimmed[n:]) == "" {
				fenceChar = 0
			}
			continue
		}
		if summaryMarker.MatchString(line) {
			before := strings.Join(lines[:i], "")
			after := strings.Join(lines[i+1:], "")
			return before + "\n" + after, before, true
		}
	}
	return source, "", false
}

// firstParagraph returns the first top-level <p> element of rendered HTML,
// used as the summary for pages without a marker.
func firstParagraph(html string) string {
	start := strings.Index(html, "<p>")
	if start < 0 {
		return ""
	}
	end := strings.Index(html[start:], "</p>")
	if end < 0 {
		return ""
	}
	return html[start : start+end+len("</p>")]
}
