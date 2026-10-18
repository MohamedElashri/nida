package migrate

import (
	"regexp"
	"strings"
)

// fenceOpen matches a fenced code block opening line: up to three spaces of
// indentation, then three or more backticks or tildes.
var fenceOpen = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})")

// RewriteProse calls fn on each run of Markdown prose in body and splices
// in the results. Fenced code blocks and inline code spans are passed
// through untouched, so a shortcode shown as an example in a code block is
// not converted. fn receives the byte offset of the run in body, which
// LineOf turns into a line number.
func RewriteProse(body string, fn func(prose string, offset int) string) string {
	var out strings.Builder
	proseStart := 0
	flush := func(end int) {
		if end > proseStart {
			out.WriteString(rewriteInline(body[proseStart:end], proseStart, fn))
		}
	}

	offset := 0
	for offset < len(body) {
		lineEnd := strings.IndexByte(body[offset:], '\n')
		if lineEnd < 0 {
			lineEnd = len(body)
		} else {
			lineEnd += offset + 1
		}
		m := fenceOpen.FindStringSubmatch(body[offset:lineEnd])
		if m == nil {
			offset = lineEnd
			continue
		}
		fence := m[2]
		// Find the closing fence: same character, at least as long.
		closeEnd := len(body)
		for pos := lineEnd; pos < len(body); {
			next := strings.IndexByte(body[pos:], '\n')
			end := len(body)
			if next >= 0 {
				end = pos + next + 1
			}
			// Match Nida's own scanner: a run of the fence character at
			// least as long as the opening, with only spaces after it.
			trimmed := strings.TrimLeft(strings.TrimRight(body[pos:end], "\r\n"), " ")
			run := len(trimmed) - len(strings.TrimLeft(trimmed, fence[:1]))
			if run >= len(fence) && strings.TrimSpace(trimmed[run:]) == "" {
				closeEnd = end
				break
			}
			pos = end
		}
		flush(offset)
		out.WriteString(body[offset:closeEnd])
		proseStart = closeEnd
		offset = closeEnd
	}
	flush(len(body))
	return out.String()
}

// rewriteInline applies fn to text outside inline code spans.
func rewriteInline(text string, base int, fn func(string, int) string) string {
	var out strings.Builder
	start := 0
	for i := 0; i < len(text); {
		if text[i] != '`' {
			i++
			continue
		}
		run := i
		for i < len(text) && text[i] == '`' {
			i++
		}
		ticks := text[run:i]
		closeAt := findClosingTicks(text, i, len(ticks))
		if closeAt < 0 {
			continue // an unmatched backtick run is literal text
		}
		out.WriteString(fn(text[start:run], base+start))
		out.WriteString(text[run : closeAt+len(ticks)])
		i = closeAt + len(ticks)
		start = i
	}
	out.WriteString(fn(text[start:], base+start))
	return out.String()
}

func findClosingTicks(text string, from, n int) int {
	for i := from; i < len(text); {
		if text[i] != '`' {
			// Code spans do not cross blank lines.
			if strings.HasPrefix(text[i:], "\n\n") {
				return -1
			}
			i++
			continue
		}
		j := i
		for j < len(text) && text[j] == '`' {
			j++
		}
		if j-i == n {
			return i
		}
		i = j
	}
	return -1
}

// LineOf returns the 1-based line of a byte offset in text.
func LineOf(text string, offset int) int {
	if offset > len(text) {
		offset = len(text)
	}
	return strings.Count(text[:offset], "\n") + 1
}

// rawHTMLTag matches the start of an HTML tag or comment in prose. Nida
// disables raw HTML in Markdown by default, so migrated content that uses
// it needs markdown.unsafe_html.
var rawHTMLTag = regexp.MustCompile(`<(?:[A-Za-z][A-Za-z0-9-]*[\s/>]|!--)`)

// HasRawHTML reports whether the prose of body contains raw HTML, other
// than the <!-- more --> summary marker.
func HasRawHTML(body string) bool {
	found := false
	RewriteProse(body, func(prose string, _ int) string {
		if !found {
			cleaned := strings.ReplaceAll(prose, "<!-- more -->", "")
			cleaned = strings.ReplaceAll(cleaned, "<!--more-->", "")
			found = rawHTMLTag.MatchString(cleaned)
		}
		return prose
	})
	return found
}
