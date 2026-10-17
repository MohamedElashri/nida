package markdown

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

var (
	detailsStartRe = regexp.MustCompile(`^\{%\s*details\s*\(([^}]*)\)\s*%\}`)
	detailsEndRe   = regexp.MustCompile(`\{%\s*end\s*%\}`)
	summaryAttrRe  = regexp.MustCompile(`summary\s*=\s*"([^"]*)"`)
)

// Shortcodes renders template shortcodes such as {{< figure src="a.png" >}}.
// internal/shortcodes provides the implementation; it lives behind an
// interface because the templates it executes depend on this package.
type Shortcodes interface {
	Has(name string) bool
	Render(call *ShortcodeCall) (string, error)
}

// Env carries what rendering needs beyond the Markdown source.
type Env struct {
	Lookup     PathLookup
	Shortcodes Shortcodes
	Images     Images
	// Page is the page being rendered, or nil for sections and snippets.
	Page *content.Page
}

// Images reports the size of a Markdown image and, optionally, responsive
// variants. src is the resolved image URL as it will appear in the HTML.
type Images interface {
	Image(src string, page *content.Page) (ImageInfo, bool)
}

type ImageInfo struct {
	Width, Height int
	Srcset, Sizes string
}

// ShortcodeCall is the data a shortcode template receives.
type ShortcodeCall struct {
	Name string
	// Args holds named arguments; Pos holds positional ones in order.
	Args map[string]string
	Pos  []string
	// Inner is the raw text between opening and closing tags. InnerHTML is
	// that text rendered as Markdown.
	Inner     string
	InnerHTML template.HTML
	Page      *content.Page
	Config    config.SiteConfig
	BasePath  string
	lookup    PathLookup
}

// Get returns a named argument for a string key or a positional argument
// for an integer key, or "" when it is missing.
func (c *ShortcodeCall) Get(key any) string {
	switch k := key.(type) {
	case string:
		return c.Args[k]
	case int:
		if k >= 0 && k < len(c.Pos) {
			return c.Pos[k]
		}
	}
	return ""
}

// ResolveURL resolves @/ content paths and prefixes the base_url path the
// same way Markdown links do. Other values are returned unchanged.
func (c *ShortcodeCall) ResolveURL(value string) string {
	return resolveWithBasePath(value, c.lookup, c.BasePath)
}

type shortcodeResult struct {
	Source       string
	Replacements map[string]string
}

func processShortcodes(source string, cfg config.SiteConfig, env Env) (shortcodeResult, error) {
	code := codeRanges(source)
	replacements := map[string]string{}
	var out strings.Builder

	addPlaceholder := func(start, end int, rendered string) {
		placeholder := "@@NIDA_SHORTCODE_HTML_" + strconv.Itoa(len(replacements)) + "@@"
		replacements[placeholder] = rendered
		if standsAlone(source, start, end) {
			out.WriteString("\n\n" + placeholder + "\n\n")
		} else {
			out.WriteString(placeholder)
		}
	}

	last := 0
	for i := 0; i < len(source); {
		if code.contains(i) {
			i = code.skip(i)
			continue
		}
		switch {
		case strings.HasPrefix(source[i:], "{{<"):
			tag, tagEnd, ok := parseShortcodeTag(source, i)
			if !ok {
				i += 3
				continue
			}
			out.WriteString(source[last:i])
			if tag.name == "rawhtml" {
				// Zola-style raw HTML markers are dropped; the content between
				// them is ordinary Markdown subject to markdown.unsafe_html.
				i, last = tagEnd, tagEnd
				continue
			}
			if tag.closing {
				return shortcodeResult{}, fmt.Errorf("render markdown: closing shortcode %q has no opening tag", tag.name)
			}

			end, inner := tagEnd, ""
			if !tag.selfClosing {
				if closeStart, closeEnd, found := findShortcodeClose(source, tagEnd, tag.name, code); found {
					inner = source[tagEnd:closeStart]
					end = closeEnd
				}
			}

			rendered, err := renderShortcode(tag, inner, cfg, env)
			if err != nil {
				return shortcodeResult{}, err
			}
			addPlaceholder(i, end, rendered)
			i, last = end, end

		case strings.HasPrefix(source[i:], "{%"):
			match := detailsStartRe.FindStringSubmatchIndex(source[i:])
			if match == nil {
				i += 2
				continue
			}
			out.WriteString(source[last:i])
			args := source[i+match[2] : i+match[3]]
			bodyStart := i + match[1]
			endStart, endEnd := findOutsideCode(source, bodyStart, detailsEndRe, code)
			if endStart < 0 {
				return shortcodeResult{}, fmt.Errorf("render markdown: unclosed details shortcode")
			}
			rendered, err := renderDetailsShortcode(summaryValue(args), source[bodyStart:endStart], cfg, env)
			if err != nil {
				return shortcodeResult{}, err
			}
			// details always renders as a block.
			placeholder := "@@NIDA_SHORTCODE_HTML_" + strconv.Itoa(len(replacements)) + "@@"
			replacements[placeholder] = rendered
			out.WriteString("\n\n" + placeholder + "\n\n")
			i, last = endEnd, endEnd

		default:
			i++
		}
	}
	out.WriteString(source[last:])
	return shortcodeResult{Source: out.String(), Replacements: replacements}, nil
}

func renderShortcode(tag shortcodeTag, inner string, cfg config.SiteConfig, env Env) (string, error) {
	if env.Shortcodes == nil || !env.Shortcodes.Has(tag.name) {
		return "", fmt.Errorf("render markdown: unknown shortcode %q (add templates/shortcodes/%s.html)", tag.name, tag.name)
	}
	call := &ShortcodeCall{
		Name:     tag.name,
		Args:     tag.args,
		Pos:      tag.pos,
		Inner:    inner,
		Page:     env.Page,
		Config:   cfg,
		BasePath: config.BasePath(cfg.BaseURL),
		lookup:   env.Lookup,
	}
	if strings.TrimSpace(inner) != "" {
		innerHTML, err := renderShortcodeBody(inner, cfg, env)
		if err != nil {
			return "", fmt.Errorf("render shortcode %q body: %w", tag.name, err)
		}
		call.InnerHTML = template.HTML(innerHTML)
	}
	rendered, err := env.Shortcodes.Render(call)
	if err != nil {
		return "", fmt.Errorf("render shortcode %q: %w", tag.name, err)
	}
	return rendered, nil
}

type shortcodeTag struct {
	name        string
	args        map[string]string
	pos         []string
	closing     bool
	selfClosing bool
}

// parseShortcodeTag parses the tag starting at source[start:], which begins
// with "{{<". It returns the index just past the closing ">}}".
func parseShortcodeTag(source string, start int) (shortcodeTag, int, bool) {
	rel := strings.Index(source[start+3:], ">}}")
	if rel < 0 {
		return shortcodeTag{}, 0, false
	}
	body := strings.TrimSpace(source[start+3 : start+3+rel])
	end := start + 3 + rel + 3

	tag := shortcodeTag{args: map[string]string{}}
	if strings.HasPrefix(body, "/") {
		tag.closing = true
		body = strings.TrimSpace(body[1:])
	}
	if strings.HasSuffix(body, "/") {
		tag.selfClosing = true
		body = strings.TrimSpace(strings.TrimSuffix(body, "/"))
	}

	tokens, ok := tokenizeShortcodeArgs(body)
	if !ok || len(tokens) == 0 || !validShortcodeName(tokens[0]) {
		return shortcodeTag{}, 0, false
	}
	tag.name = tokens[0]
	for _, token := range tokens[1:] {
		if key, value, isNamed := strings.Cut(token, "="); isNamed && validShortcodeName(key) {
			tag.args[key] = unquote(value)
		} else {
			tag.pos = append(tag.pos, unquote(token))
		}
	}
	return tag, end, true
}

func validShortcodeName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		valid := r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !valid {
			return false
		}
	}
	return true
}

// tokenizeShortcodeArgs splits on whitespace while keeping double-quoted
// strings, including key="value with spaces", together.
func tokenizeShortcodeArgs(body string) ([]string, bool) {
	var tokens []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '\\' && inQuote && i+1 < len(body):
			b.WriteByte(c)
			b.WriteByte(body[i+1])
			i++
		case c == '"':
			inQuote = !inQuote
			b.WriteByte(c)
		case (c == ' ' || c == '\t' || c == '\n') && !inQuote:
			if b.Len() > 0 {
				tokens = append(tokens, b.String())
				b.Reset()
			}
		default:
			b.WriteByte(c)
		}
	}
	if inQuote {
		return nil, false
	}
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
	}
	return tokens, true
}

func unquote(value string) string {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
		return value[1 : len(value)-1]
	}
	return value
}

// findShortcodeClose finds the {{< /name >}} that closes a tag opened before
// from, allowing nested tags with the same name.
func findShortcodeClose(source string, from int, name string, code codeSpans) (int, int, bool) {
	depth := 0
	for i := from; i < len(source); {
		if code.contains(i) {
			i = code.skip(i)
			continue
		}
		if !strings.HasPrefix(source[i:], "{{<") {
			i++
			continue
		}
		tag, end, ok := parseShortcodeTag(source, i)
		if !ok {
			i += 3
			continue
		}
		if tag.name == name {
			switch {
			case tag.closing && depth == 0:
				return i, end, true
			case tag.closing:
				depth--
			case !tag.selfClosing:
				depth++
			}
		}
		i = end
	}
	return 0, 0, false
}

func findOutsideCode(source string, from int, re *regexp.Regexp, code codeSpans) (int, int) {
	for from < len(source) {
		match := re.FindStringIndex(source[from:])
		if match == nil {
			return -1, -1
		}
		start := from + match[0]
		if !code.contains(start) {
			return start, from + match[1]
		}
		from = code.skip(start)
	}
	return -1, -1
}

// standsAlone reports whether source[start:end] is the only thing on its
// lines, in which case it is rendered as a block rather than inline.
func standsAlone(source string, start, end int) bool {
	lineStart := strings.LastIndexByte(source[:start], '\n') + 1
	if strings.TrimSpace(source[lineStart:start]) != "" {
		return false
	}
	lineEnd := strings.IndexByte(source[end:], '\n')
	if lineEnd < 0 {
		lineEnd = len(source) - end
	}
	return strings.TrimSpace(source[end:end+lineEnd]) == ""
}

// codeSpans holds sorted, non-overlapping [start, end) ranges of fenced code
// blocks and inline code, where shortcode syntax is left untouched.
type codeSpans [][2]int

func (c codeSpans) find(pos int) int {
	i := sort.Search(len(c), func(i int) bool { return c[i][1] > pos })
	if i < len(c) && c[i][0] <= pos {
		return i
	}
	return -1
}

func (c codeSpans) contains(pos int) bool { return c.find(pos) >= 0 }

// skip returns the end of the code range containing pos.
func (c codeSpans) skip(pos int) int {
	if i := c.find(pos); i >= 0 {
		return c[i][1]
	}
	return pos + 1
}

func codeRanges(source string) codeSpans {
	var spans codeSpans
	fenceChar, fenceLen, fenceStart := byte(0), 0, 0
	textStart := 0

	for lineStart := 0; lineStart < len(source); {
		lineEnd := strings.IndexByte(source[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(source)
		} else {
			lineEnd += lineStart + 1
		}
		line := strings.TrimLeft(source[lineStart:lineEnd], " ")
		indent := lineEnd - lineStart - len(line)

		if fenceChar == 0 {
			if n := fenceRun(line); n >= 3 && indent < 4 {
				spans = append(spans, inlineCodeRanges(source, textStart, lineStart)...)
				fenceChar, fenceLen, fenceStart = line[0], n, lineStart
			}
		} else if n := fenceRun(line); n >= fenceLen && line[0] == fenceChar && strings.TrimSpace(line[n:]) == "" {
			spans = append(spans, [2]int{fenceStart, lineEnd})
			fenceChar = 0
			textStart = lineEnd
		}
		lineStart = lineEnd
	}
	if fenceChar != 0 {
		spans = append(spans, [2]int{fenceStart, len(source)})
	} else {
		spans = append(spans, inlineCodeRanges(source, textStart, len(source))...)
	}
	return spans
}

func fenceRun(line string) int {
	if line == "" || (line[0] != '`' && line[0] != '~') {
		return 0
	}
	n := 0
	for n < len(line) && line[n] == line[0] {
		n++
	}
	return n
}

// inlineCodeRanges finds `code` spans: a backtick run closed by a run of the
// same length.
func inlineCodeRanges(source string, start, end int) codeSpans {
	var spans codeSpans
	for i := start; i < end; {
		if source[i] != '`' {
			i++
			continue
		}
		n := 0
		for i+n < end && source[i+n] == '`' {
			n++
		}
		closeAt := -1
		for j := i + n; j < end; {
			if source[j] != '`' {
				j++
				continue
			}
			m := 0
			for j+m < end && source[j+m] == '`' {
				m++
			}
			if m == n {
				closeAt = j + m
				break
			}
			j += m
		}
		if closeAt < 0 {
			i += n
			continue
		}
		spans = append(spans, [2]int{i, closeAt})
		i = closeAt
	}
	return spans
}

func renderShortcodeBody(source string, cfg config.SiteConfig, env Env) (string, error) {
	processed, err := processShortcodes(source, cfg, env)
	if err != nil {
		return "", err
	}
	result, err := renderMarkdownCore(processed.Source, cfg, env)
	if err != nil {
		return "", fmt.Errorf("render shortcode body: %w", err)
	}
	return restoreShortcodeHTML(result.HTML, processed.Replacements), nil
}

func renderDetailsShortcode(summary, body string, cfg config.SiteConfig, env Env) (string, error) {
	renderedBody, err := renderShortcodeBody(body, cfg, env)
	if err != nil {
		return "", err
	}
	return renderDetails(summary, renderedBody), nil
}

func summaryValue(args string) string {
	match := summaryAttrRe.FindStringSubmatch(args)
	if match == nil {
		return "Show details"
	}
	return strings.TrimSpace(match[1])
}

func renderDetails(summary, body string) string {
	var b strings.Builder
	b.WriteString("\n<details class=\"collapsible-details\">\n")
	b.WriteString("  <summary class=\"collapsible-details-summary\">\n")
	b.WriteString("    <span class=\"collapsible-details-icon\"></span>\n")
	b.WriteString("    <span class=\"collapsible-details-label\">")
	b.WriteString(html.EscapeString(summary))
	b.WriteString("</span>\n")
	b.WriteString("  </summary>\n")
	b.WriteString("  <div class=\"collapsible-details-body\">\n")
	b.WriteString(body)
	b.WriteString("  </div>\n")
	b.WriteString("</details>\n")
	return b.String()
}

func restoreShortcodeHTML(rendered string, replacements map[string]string) string {
	for placeholder, replacement := range replacements {
		rendered = strings.ReplaceAll(rendered, "<p>"+placeholder+"</p>\n", replacement)
		rendered = strings.ReplaceAll(rendered, "<p>"+placeholder+"</p>", replacement)
		rendered = strings.ReplaceAll(rendered, placeholder, replacement)
	}
	return rendered
}
