package migrate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/MohamedElashri/nida/internal/content"
)

// rstNote is a problem found while converting reStructuredText.
type rstNote struct {
	sev     Severity
	line    int
	message string
}

// rstDoc is a reStructuredText document converted to Markdown.
type rstDoc struct {
	// meta holds the document title and its bibliographic fields, with
	// lowercase keys, as Pelican reads them.
	meta      map[string]string
	metaOrder []string
	body      string
	notes     []rstNote
	// usesMath is set when the document has math, which needs
	// markdown.math in Nida.
	usesMath bool
}

type rstLine struct {
	text string
	num  int // 1-based line in the source file
}

type rstConverter struct {
	doc       *rstDoc
	targets   map[string]string // reference name -> URL
	anonymous []string          // anonymous target URLs, in order
	anonUsed  int
	sections  map[string]string // section title -> anchor
	styles    []string          // heading adornment styles, in order
	titleKey  string            // style of the promoted document title
	footnotes int               // auto-numbered footnote counter
	footRefs  int
	highlight string // language from .. highlight::
}

// convertRST converts the common subset of reStructuredText that Pelican
// articles use into Markdown. Constructs outside the subset are kept as
// fenced rst blocks and reported, so nothing is silently lost.
func convertRST(src string) *rstDoc {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.TrimPrefix(src, "\xef\xbb\xbf")
	var lines []rstLine
	for i, text := range strings.Split(src, "\n") {
		lines = append(lines, rstLine{text: strings.TrimRight(expandTabs(text), " "), num: i + 1})
	}
	c := &rstConverter{
		doc:      &rstDoc{meta: map[string]string{}},
		targets:  map[string]string{},
		sections: map[string]string{},
	}
	c.collectTargets(lines)
	lines = c.docTitle(lines)
	lines = c.docinfo(lines)
	out := c.blocks(lines)
	c.doc.body = strings.TrimSpace(strings.Join(out, "\n")) + "\n"
	if strings.TrimSpace(c.doc.body) == "" {
		c.doc.body = ""
	}
	return c.doc
}

func expandTabs(s string) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			n := 8 - col%8
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

func (c *rstConverter) note(sev Severity, line int, format string, args ...any) {
	c.doc.notes = append(c.doc.notes, rstNote{sev: sev, line: line, message: fmt.Sprintf(format, args...)})
}

func (c *rstConverter) setMeta(key, value string) {
	key = strings.ToLower(strings.TrimSpace(key))
	if _, ok := c.doc.meta[key]; !ok {
		c.doc.metaOrder = append(c.doc.metaOrder, key)
	}
	c.doc.meta[key] = value
}

var (
	rstTargetRe     = regexp.MustCompile(`^\.\. _([^:]+|` + "`[^`]+`" + `):\s*(\S*)\s*$`)
	rstAnonTargetRe = regexp.MustCompile(`^(?:\.\. __:|__)\s+(\S+)\s*$`)
)

// collectTargets reads hyperlink targets and section titles up front, so
// references can be resolved wherever they appear.
func (c *rstConverter) collectTargets(lines []rstLine) {
	for i := 0; i < len(lines); i++ {
		text := strings.TrimSpace(lines[i].text)
		if m := rstTargetRe.FindStringSubmatch(text); m != nil {
			url := m[2]
			// A long URL may continue on indented lines.
			for i+1 < len(lines) && indentOf(lines[i+1].text) > 0 && strings.TrimSpace(lines[i+1].text) != "" {
				i++
				url += strings.TrimSpace(lines[i].text)
			}
			name := refName(strings.Trim(m[1], "`"))
			if url != "" {
				c.targets[name] = url
			}
			continue
		}
		if m := rstAnonTargetRe.FindStringSubmatch(text); m != nil {
			c.anonymous = append(c.anonymous, m[1])
		}
		if i+1 < len(lines) && isAdornment(lines[i+1].text) && text != "" && !isAdornment(text) && indentOf(lines[i].text) == 0 {
			c.sections[refName(text)] = "#" + content.DeriveSlug(text)
		}
	}
	// Targets that point at other targets ("alias_") resolve through.
	for name, url := range c.targets {
		if strings.HasSuffix(url, "_") {
			if next, ok := c.targets[refName(strings.Trim(strings.TrimSuffix(url, "_"), "`"))]; ok {
				c.targets[name] = next
			}
		}
	}
}

func refName(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func indentOf(s string) int {
	return len(s) - len(strings.TrimLeft(s, " "))
}

const rstPunct = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

// isAdornment reports whether a line is a section adornment: one
// punctuation character repeated, at least three times.
func isAdornment(s string) bool {
	if len(s) < 3 || indentOf(s) > 0 || !strings.ContainsRune(rstPunct, rune(s[0])) {
		return false
	}
	return strings.Trim(s, s[:1]) == ""
}

// title reports whether lines[i] starts a section title, returning the
// title text, its adornment style, and how many lines it spans.
func titleAt(lines []rstLine, i int) (string, string, int, bool) {
	if i >= len(lines) {
		return "", "", 0, false
	}
	cur := lines[i].text
	// Overline, text, underline.
	if isAdornment(cur) && i+2 < len(lines) && strings.TrimSpace(lines[i+1].text) != "" && lines[i+2].text == cur {
		return strings.TrimSpace(lines[i+1].text), "o" + cur[:1], 3, true
	}
	if strings.TrimSpace(cur) == "" || indentOf(cur) > 0 || isAdornment(cur) {
		return "", "", 0, false
	}
	if i+1 < len(lines) && isAdornment(lines[i+1].text) && utf8.RuneCountInString(lines[i+1].text) >= min(utf8.RuneCountInString(cur), 3) {
		// A one-line paragraph followed by a separator line after a blank
		// is a transition, not a title; titles need the underline directly.
		return strings.TrimSpace(cur), "u" + lines[i+1].text[:1], 2, true
	}
	return "", "", 0, false
}

// docTitle promotes a lone top-level title to the document title, as
// docutils does, and records it as the title metadata.
func (c *rstConverter) docTitle(lines []rstLine) []rstLine {
	i := skipBlankAndComments(lines, 0)
	text, style, n, ok := titleAt(lines, i)
	if !ok {
		return lines
	}
	// Promoted only if no other title uses the same style.
	for j := i + n; j < len(lines); j++ {
		if _, s, m, ok := titleAt(lines, j); ok {
			if s == style {
				return lines
			}
			j += m - 1
		}
	}
	c.setMeta("title", c.inline(text, lines[i].num))
	c.titleKey = style
	rest := lines[i+n:]
	// A subtitle directly after the title is promoted too.
	k := skipBlankAndComments(rest, 0)
	if sub, subStyle, m, ok := titleAt(rest, k); ok {
		unique := true
		for j := k + m; j < len(rest); j++ {
			if _, s, mm, ok := titleAt(rest, j); ok {
				if s == subStyle {
					unique = false
					break
				}
				j += mm - 1
			}
		}
		if unique {
			c.setMeta("subtitle", sub)
			c.note(Review, rest[k].num, "The subtitle %q is kept as `subtitle` front matter; templates do not show it yet.", sub)
			rest = rest[k+m:]
		}
	}
	return rest
}

func skipBlankAndComments(lines []rstLine, i int) int {
	for i < len(lines) {
		t := lines[i].text
		if strings.TrimSpace(t) == "" {
			i++
			continue
		}
		if isComment(t) {
			i++
			for i < len(lines) && (strings.TrimSpace(lines[i].text) == "" || indentOf(lines[i].text) > 0) {
				i++
			}
			continue
		}
		return i
	}
	return i
}

var (
	rstFieldRe     = regexp.MustCompile(`^:([^:\s][^:]*):(?:\s+(.*))?$`)
	rstDirectiveRe = regexp.MustCompile(`^\.\.\s+([A-Za-z0-9_:+.-]+)::(?:\s+(.*))?$`)
	rstSubDefRe    = regexp.MustCompile(`^\.\.\s+\|[^|]+\|\s+`)
	rstFootDefRe   = regexp.MustCompile(`^\.\.\s+\[([^\]]+)\](?:\s+(.*))?$`)
)

func isComment(t string) bool {
	if !strings.HasPrefix(t, "..") {
		return false
	}
	if t == ".." {
		return true
	}
	if !strings.HasPrefix(t, ".. ") {
		return false
	}
	return !rstDirectiveRe.MatchString(t) && !rstTargetRe.MatchString(t) && !rstSubDefRe.MatchString(t) &&
		!rstFootDefRe.MatchString(t) && !strings.HasPrefix(t, ".. __:")
}

// docinfo reads the bibliographic field list at the top of the document
// into metadata.
func (c *rstConverter) docinfo(lines []rstLine) []rstLine {
	i := skipBlankAndComments(lines, 0)
	if i >= len(lines) || !rstFieldRe.MatchString(lines[i].text) {
		return lines
	}
	for i < len(lines) {
		m := rstFieldRe.FindStringSubmatch(lines[i].text)
		if m == nil {
			break
		}
		value := []string{strings.TrimSpace(m[2])}
		i++
		for i < len(lines) && indentOf(lines[i].text) > 0 && strings.TrimSpace(lines[i].text) != "" {
			value = append(value, strings.TrimSpace(lines[i].text))
			i++
		}
		c.setMeta(m[1], strings.TrimSpace(strings.Join(value, " ")))
	}
	return lines[i:]
}

// headingLevel returns the Markdown level for an adornment style. Pelican
// renders the first level below the document title as <h2>.
func (c *rstConverter) headingLevel(style string) int {
	for i, s := range c.styles {
		if s == style {
			return min(i+2, 6)
		}
	}
	c.styles = append(c.styles, style)
	return min(len(c.styles)+1, 6)
}

var (
	rstBulletRe   = regexp.MustCompile(`^([-*+•‣⁃])( +)\S`)
	rstEnumRe     = regexp.MustCompile(`^(\(?)(\d+|#|[A-Za-z]|[ivxlcdm]+|[IVXLCDM]+)([.)])( +)\S`)
	rstLineBlock  = regexp.MustCompile(`^\|( |$)`)
	rstTableRule  = regexp.MustCompile(`^=+( +=+)+$`)
	rstGridRule   = regexp.MustCompile(`^\+[-=+]+\+$`)
	rstDoctest    = regexp.MustCompile(`^>>> `)
	rstOptionRe   = regexp.MustCompile(`^:([A-Za-z0-9_-]+):(?:\s+(.*))?$`)
	rstAdmonition = map[string]string{
		"note": "NOTE", "tip": "TIP", "hint": "TIP", "important": "IMPORTANT",
		"warning": "WARNING", "attention": "WARNING", "caution": "CAUTION",
		"danger": "CAUTION", "error": "CAUTION", "seealso": "NOTE",
	}
)

// blocks converts a run of lines whose base indentation is zero.
func (c *rstConverter) blocks(lines []rstLine) []string {
	var out []string
	emit := func(s ...string) {
		if len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
		out = append(out, s...)
	}
	for i := 0; i < len(lines); {
		line := lines[i]
		t := line.text
		switch {
		case strings.TrimSpace(t) == "":
			i++
			continue

		case indentOf(t) > 0:
			// An indented block not introduced by "::" is a block quote.
			block, next := indentedBlock(lines, i)
			inner := c.blocks(dedent(block))
			var quoted []string
			for _, l := range inner {
				if l == "" {
					quoted = append(quoted, ">")
				} else {
					quoted = append(quoted, "> "+l)
				}
			}
			emit(quoted...)
			i = next
			continue
		}

		if text, style, n, ok := titleAt(lines, i); ok {
			level := c.headingLevel(style)
			emit(strings.Repeat("#", level) + " " + c.inline(text, line.num))
			i += n
			continue
		}
		if isAdornment(t) && len(t) >= 4 {
			emit("---")
			i++
			continue
		}

		switch {
		case t == ".." || strings.HasPrefix(t, ".. "):
			i = c.explicit(lines, i, emit)
		case rstAnonTargetRe.MatchString(t):
			i++
		case rstGridRule.MatchString(t):
			block, next := untilBlank(lines, i)
			c.note(Manual, line.num, "Grid table kept as reStructuredText; rewrite it as a Markdown table.")
			emit(fenceRST(block)...)
			i = next
		case rstTableRule.MatchString(t):
			block, next := simpleTable(lines, i)
			if table, ok := c.simpleTable(block); ok {
				emit(table...)
			} else {
				c.note(Manual, line.num, "Table kept as reStructuredText; rewrite it as a Markdown table.")
				emit(fenceRST(block)...)
			}
			i = next
		case rstBulletRe.MatchString(t):
			var items []string
			for i < len(lines) {
				m := rstBulletRe.FindStringSubmatch(lines[i].text)
				if m == nil {
					break
				}
				var item []string
				item, i = c.listItem(lines, i, len(m[1])+len(m[2]), "- ")
				items = append(items, item...)
				i = skipBlankBetweenItems(lines, i, rstBulletRe)
			}
			emit(items...)
		case rstEnumRe.MatchString(t) && enumListContinues(lines, i):
			var items []string
			n := 0
			for i < len(lines) {
				m := rstEnumRe.FindStringSubmatch(lines[i].text)
				if m == nil {
					break
				}
				n++
				num := n
				if v, err := strconv.Atoi(m[2]); err == nil {
					if n == 1 {
						num = v
						n = v
					}
				} else if m[2] != "#" {
					c.note(Review, lines[i].num, "A lettered or roman-numeral list became a numbered list.")
				}
				marker := strconv.Itoa(num) + ". "
				width := len(m[1]) + len(m[2]) + len(m[3]) + len(m[4])
				var item []string
				item, i = c.listItem(lines, i, width, marker)
				items = append(items, item...)
				i = skipBlankBetweenItems(lines, i, rstEnumRe)
			}
			emit(items...)
		case rstLineBlock.MatchString(t):
			var block []string
			for i < len(lines) && rstLineBlock.MatchString(lines[i].text) {
				block = append(block, c.escapeLine(c.inline(strings.TrimPrefix(strings.TrimPrefix(lines[i].text, "|"), " "), lines[i].num)))
				i++
			}
			for j := 0; j < len(block)-1; j++ {
				block[j] += `\`
			}
			emit(block...)
		case rstDoctest.MatchString(t):
			block, next := untilBlank(lines, i)
			emit(fence("python", texts(block))...)
			i = next
		case rstFieldRe.MatchString(t):
			var fields []string
			for i < len(lines) {
				m := rstFieldRe.FindStringSubmatch(lines[i].text)
				if m == nil {
					break
				}
				value := []string{m[2]}
				i++
				for i < len(lines) && indentOf(lines[i].text) > 0 && strings.TrimSpace(lines[i].text) != "" {
					value = append(value, strings.TrimSpace(lines[i].text))
					i++
				}
				fields = append(fields, "**"+m[1]+":** "+c.inline(strings.Join(value, " "), lines[i-1].num)+`\`)
			}
			if len(fields) > 0 {
				fields[len(fields)-1] = strings.TrimSuffix(fields[len(fields)-1], `\`)
			}
			c.note(Review, line.num, "A field list became bold labels.")
			emit(fields...)
		default:
			i = c.paragraph(lines, i, emit)
		}
	}
	return out
}

// enumListContinues guards against paragraphs that start like "A. Smith":
// an enumerated item must be followed by a blank line, an indented line,
// another item, or the end.
func enumListContinues(lines []rstLine, i int) bool {
	if i+1 >= len(lines) {
		return true
	}
	next := lines[i+1].text
	return strings.TrimSpace(next) == "" || indentOf(next) > 0 || rstEnumRe.MatchString(next)
}

func skipBlankBetweenItems(lines []rstLine, i int, re *regexp.Regexp) int {
	j := i
	for j < len(lines) && strings.TrimSpace(lines[j].text) == "" {
		j++
	}
	if j < len(lines) && re.MatchString(lines[j].text) {
		return j
	}
	return i
}

// listItem converts one list item whose text starts at column width.
func (c *rstConverter) listItem(lines []rstLine, i, width int, marker string) ([]string, int) {
	first := rstLine{text: lines[i].text[width:], num: lines[i].num}
	body := []rstLine{first}
	j := i + 1
	for j < len(lines) {
		t := lines[j].text
		if strings.TrimSpace(t) != "" && indentOf(t) < width {
			break
		}
		if strings.TrimSpace(t) == "" {
			// A blank line ends the item unless indented text follows.
			k := j
			for k < len(lines) && strings.TrimSpace(lines[k].text) == "" {
				k++
			}
			if k >= len(lines) || indentOf(lines[k].text) < width {
				break
			}
		}
		if len(t) >= width {
			t = t[width:]
		} else {
			t = ""
		}
		body = append(body, rstLine{text: t, num: lines[j].num})
		j++
	}
	inner := c.blocks(body)
	pad := strings.Repeat(" ", len(marker))
	var out []string
	for k, l := range inner {
		switch {
		case k == 0:
			out = append(out, marker+l)
		case l == "":
			out = append(out, "")
		default:
			out = append(out, pad+l)
		}
	}
	return out, j
}

// paragraph converts a paragraph, a definition list item, or a paragraph
// that introduces a literal block with "::".
func (c *rstConverter) paragraph(lines []rstLine, i int, emit func(...string)) int {
	start := i
	var text []rstLine
	for i < len(lines) && strings.TrimSpace(lines[i].text) != "" && indentOf(lines[i].text) == 0 {
		if i > start {
			if _, _, _, ok := titleAt(lines, i); ok {
				break
			}
		}
		text = append(text, lines[i])
		i++
	}
	// A line followed directly by an indented one is a definition list
	// term.
	if len(text) == 1 && i < len(lines) && indentOf(lines[i].text) > 0 && strings.TrimSpace(lines[i].text) != "" {
		block, next := indentedBlock(lines, i)
		def := c.blocks(dedent(block))
		term := text[0].text
		classifier := ""
		if t, cls, ok := strings.Cut(term, " : "); ok {
			term, classifier = t, cls
		}
		head := "**" + c.inline(term, text[0].num) + "**"
		if classifier != "" {
			head += " *(" + c.inline(classifier, text[0].num) + ")*"
		}
		c.note(Review, text[0].num, "A definition list became bold terms followed by their definitions.")
		emit(append([]string{head + `\`}, def...)...)
		return next
	}

	literal := false
	last := text[len(text)-1].text
	if strings.HasSuffix(last, "::") {
		literal = true
		switch {
		case strings.TrimSpace(last) == "::":
			text = text[:len(text)-1]
		case strings.HasSuffix(last, " ::"):
			text[len(text)-1].text = strings.TrimSuffix(last, " ::")
		default:
			text[len(text)-1].text = strings.TrimSuffix(last, ":")
		}
	}
	if len(text) > 0 {
		var md []string
		for _, l := range text {
			md = append(md, c.escapeLine(c.inline(l.text, l.num)))
		}
		emit(md...)
	}
	if literal {
		j := i
		for j < len(lines) && strings.TrimSpace(lines[j].text) == "" {
			j++
		}
		if j < len(lines) && indentOf(lines[j].text) > 0 {
			block, next := indentedBlock(lines, j)
			emit(fence(c.highlight, texts(dedent(block)))...)
			return next
		}
	}
	return i
}

// explicit converts a line starting with "..": a directive, a target, a
// footnote, a substitution definition, or a comment.
func (c *rstConverter) explicit(lines []rstLine, i int, emit func(...string)) int {
	line := lines[i]
	t := line.text
	block, next := explicitBlock(lines, i)
	switch {
	case rstTargetRe.MatchString(strings.TrimSpace(t)) || strings.HasPrefix(t, ".. __:"):
		return next
	case rstSubDefRe.MatchString(t):
		c.note(Manual, line.num, "Substitution definitions have no Markdown equivalent; the definition is kept as reStructuredText and `|name|` references are left as text.")
		emit(fenceRST(block)...)
		return next
	case rstFootDefRe.MatchString(t):
		m := rstFootDefRe.FindStringSubmatch(t)
		label := c.footnoteLabel(m[1], true)
		body := []rstLine{{text: m[2], num: line.num}}
		body = append(body, dedent(block[1:])...)
		inner := c.blocks(body)
		for k := range inner {
			if k > 0 && inner[k] != "" {
				inner[k] = "    " + inner[k]
			}
		}
		if len(inner) == 0 {
			inner = []string{""}
		}
		inner[0] = "[^" + label + "]: " + inner[0]
		emit(inner...)
		return next
	case isComment(t):
		return next
	}

	m := rstDirectiveRe.FindStringSubmatch(t)
	if m == nil {
		// An explicit markup start that is not a directive, footnote,
		// target, or comment; keep it as reStructuredText for review.
		c.note(Manual, line.num, "Explicit markup `%s` is not converted; it is kept as reStructuredText.", truncate(t, 40))
		emit(fenceRST(block)...)
		return next
	}
	name := strings.ToLower(m[1])
	arg := strings.TrimSpace(m[2])
	opts, body := directiveParts(block[1:])
	// A directive argument may continue on the next lines.
	switch name {
	case "code-block", "code", "sourcecode":
		lang := arg
		info := lang
		var attrs []string
		if _, ok := opts["linenos"]; ok {
			attrs = append(attrs, "linenos")
		}
		if hl, ok := opts["emphasize-lines"]; ok {
			attrs = append(attrs, `hl_lines="`+strings.ReplaceAll(hl, ",", " ")+`"`)
		}
		if hl, ok := opts["hl_lines"]; ok {
			attrs = append(attrs, `hl_lines="`+hl+`"`)
		}
		if cap, ok := opts["caption"]; ok {
			attrs = append(attrs, `title="`+strings.ReplaceAll(cap, `"`, `'`)+`"`)
		}
		if len(attrs) > 0 {
			info += " {" + strings.Join(attrs, ", ") + "}"
		}
		emit(fence(info, texts(dedent(body)))...)
	case "highlight":
		c.highlight = arg
	case "image", "figure":
		src := arg
		alt := opts["alt"]
		var caption []string
		if name == "figure" {
			caption = c.blocks(dedent(body))
		}
		if name == "image" && len(opts) <= 1 && (len(opts) == 0 || alt != "") {
			emit(fmt.Sprintf("![%s](%s)", alt, src))
			break
		}
		args := []Arg{{Key: "src", Value: src}}
		if alt != "" {
			args = append(args, Arg{Key: "alt", Value: alt})
		}
		if len(caption) > 0 {
			args = append(args, Arg{Key: "caption", Value: strings.Join(caption, " ")})
		}
		for _, k := range []string{"width", "height"} {
			if v, ok := opts[k]; ok {
				args = append(args, Arg{Key: k, Value: strings.TrimSuffix(v, "px")})
				if strings.HasSuffix(v, "%") {
					c.note(Review, line.num, "Image %s %q is a percentage; check it renders as intended.", k, v)
				}
			}
		}
		if v, ok := opts["target"]; ok {
			args = append(args, Arg{Key: "link", Value: v})
		}
		if v, ok := opts["align"]; ok {
			args = append(args, Arg{Key: "class", Value: "align-" + v})
		}
		if v, ok := opts["class"]; ok {
			args = append(args, Arg{Key: "class", Value: v})
		}
		if _, ok := opts["scale"]; ok {
			c.note(Review, line.num, "Image `:scale:` has no equivalent and was dropped.")
		}
		emit(Shortcode("figure", args))
	case "note", "tip", "hint", "important", "warning", "attention", "caution", "danger", "error", "seealso", "admonition":
		kind := rstAdmonition[name]
		title := ""
		if name == "admonition" {
			kind, title = "NOTE", arg
		} else if arg != "" {
			// "Directive content may start on the first line."
			body = append([]rstLine{{text: arg, num: line.num}}, dedent(body)...)
		}
		inner := c.blocks(dedent(body))
		head := "> [!" + kind + "]"
		if title != "" {
			head += " " + c.inline(title, line.num)
		}
		out := []string{head}
		for _, l := range inner {
			if l == "" {
				out = append(out, ">")
			} else {
				out = append(out, "> "+l)
			}
		}
		emit(out...)
	case "raw":
		if strings.EqualFold(arg, "html") {
			emit(texts(dedent(body))...)
		} else {
			c.note(Unsupported, line.num, "`.. raw:: %s` content was dropped; only HTML can be kept.", arg)
		}
	case "youtube":
		emit(Shortcode("youtube", []Arg{{Value: arg}}))
	case "contents":
		c.note(Review, line.num, "`.. contents::` was removed; build a table of contents from `.Page.TOC` in the template.")
	case "math":
		c.doc.usesMath = true
		math := texts(dedent(body))
		if arg != "" {
			math = append([]string{arg}, math...)
		}
		emit(append(append([]string{"$$"}, math...), "$$")...)
	case "topic", "sidebar", "rubric":
		inner := c.blocks(dedent(body))
		emit(append([]string{"**" + c.inline(arg, line.num) + "**"}, inner...)...)
		c.note(Review, line.num, "`.. %s::` became a bold heading and paragraphs.", name)
	case "epigraph", "highlights", "pull-quote":
		inner := c.blocks(dedent(body))
		var quoted []string
		for _, l := range inner {
			quoted = append(quoted, strings.TrimRight("> "+l, " "))
		}
		emit(quoted...)
	default:
		c.note(Manual, line.num, "Directive `.. %s::` is not converted; it is kept as reStructuredText.", name)
		emit(fenceRST(block)...)
	}
	return next
}

// explicitBlock returns an explicit markup block: its first line plus the
// indented lines (and blank lines between them) that follow.
func explicitBlock(lines []rstLine, i int) ([]rstLine, int) {
	block := []rstLine{lines[i]}
	j := i + 1
	for j < len(lines) {
		t := lines[j].text
		if strings.TrimSpace(t) == "" {
			k := j
			for k < len(lines) && strings.TrimSpace(lines[k].text) == "" {
				k++
			}
			if k >= len(lines) || indentOf(lines[k].text) == 0 {
				break
			}
		} else if indentOf(t) == 0 {
			break
		}
		block = append(block, lines[j])
		j++
	}
	return block, j
}

// directiveParts splits a directive's indented lines into its options and
// its content.
func directiveParts(rest []rstLine) (map[string]string, []rstLine) {
	opts := map[string]string{}
	i := 0
	for i < len(rest) {
		m := rstOptionRe.FindStringSubmatch(strings.TrimSpace(rest[i].text))
		if strings.TrimSpace(rest[i].text) == "" || m == nil {
			break
		}
		opts[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
		i++
	}
	for i < len(rest) && strings.TrimSpace(rest[i].text) == "" {
		i++
	}
	return opts, rest[i:]
}

// indentedBlock returns the indented lines starting at i, including blank
// lines inside it.
func indentedBlock(lines []rstLine, i int) ([]rstLine, int) {
	var block []rstLine
	j := i
	for j < len(lines) {
		t := lines[j].text
		if strings.TrimSpace(t) != "" && indentOf(t) == 0 {
			break
		}
		block = append(block, lines[j])
		j++
	}
	for len(block) > 0 && strings.TrimSpace(block[len(block)-1].text) == "" {
		block = block[:len(block)-1]
	}
	return block, j
}

func untilBlank(lines []rstLine, i int) ([]rstLine, int) {
	j := i
	for j < len(lines) && strings.TrimSpace(lines[j].text) != "" {
		j++
	}
	return lines[i:j], j
}

// simpleTable returns a simple table: from its top rule to the rule that
// is followed by a blank line or the end.
func simpleTable(lines []rstLine, i int) ([]rstLine, int) {
	j := i + 1
	for j < len(lines) {
		if rstTableRule.MatchString(lines[j].text) && (j+1 >= len(lines) || strings.TrimSpace(lines[j+1].text) == "") {
			return lines[i : j+1], j + 1
		}
		j++
	}
	return untilBlank(lines, i)
}

// simpleTable converts a simple table with one-line cells into a Markdown
// pipe table.
func (c *rstConverter) simpleTable(block []rstLine) ([]string, bool) {
	rule := block[0].text
	var cols [][2]int
	for _, m := range regexp.MustCompile(`=+`).FindAllStringIndex(rule, -1) {
		cols = append(cols, [2]int{m[0], m[1]})
	}
	var rows [][]string
	header := -1
	for k, l := range block[1:] {
		if rstTableRule.MatchString(l.text) {
			if header < 0 && k+2 < len(block) {
				header = len(rows)
			}
			continue
		}
		if strings.TrimSpace(l.text) == "" {
			return nil, false
		}
		var row []string
		for ci, col := range cols {
			start := col[0]
			end := len(l.text)
			if ci+1 < len(cols) {
				end = min(cols[ci+1][0], len(l.text))
			}
			cell := ""
			if start < len(l.text) {
				cell = strings.TrimSpace(l.text[start:max(start, end)])
			}
			row = append(row, cell)
		}
		// A row whose first cell is empty continues the previous row,
		// which a pipe table cannot express.
		if row[0] == "" && len(rows) > 0 {
			return nil, false
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, false
	}
	cell := func(s string, num int) string {
		return strings.ReplaceAll(c.inline(s, num), "|", `\|`)
	}
	var out []string
	sep := make([]string, len(cols))
	for k := range sep {
		sep[k] = "---"
	}
	if header != 1 {
		// No header row: Markdown tables need one, so use empty headers.
		out = append(out, "|"+strings.Repeat("   |", len(cols)), "| "+strings.Join(sep, " | ")+" |")
		header = 0
	}
	for k, row := range rows {
		for ci := range row {
			row[ci] = cell(row[ci], block[0].num)
		}
		out = append(out, "| "+strings.Join(row, " | ")+" |")
		if k == 0 && header == 1 {
			out = append(out, "| "+strings.Join(sep, " | ")+" |")
		}
	}
	return out, true
}

func dedent(lines []rstLine) []rstLine {
	least := -1
	for _, l := range lines {
		if strings.TrimSpace(l.text) == "" {
			continue
		}
		if n := indentOf(l.text); least < 0 || n < least {
			least = n
		}
	}
	out := make([]rstLine, len(lines))
	for k, l := range lines {
		if len(l.text) >= least && least > 0 {
			l.text = l.text[least:]
		} else if strings.TrimSpace(l.text) == "" {
			l.text = ""
		}
		out[k] = l
	}
	return out
}

func texts(lines []rstLine) []string {
	out := make([]string, len(lines))
	for k, l := range lines {
		out[k] = l.text
	}
	return out
}

func fence(info string, body []string) []string {
	ticks := "```"
	for _, l := range body {
		for strings.Contains(l, ticks) {
			ticks += "`"
		}
	}
	return append(append([]string{ticks + info}, body...), ticks)
}

func fenceRST(block []rstLine) []string {
	return fence("rst", texts(block))
}

var (
	rstLiteral      = regexp.MustCompile("``(.+?)``")
	rstRolePrefix   = regexp.MustCompile(":([A-Za-z][\\w.+-]*):`([^`]+)`")
	rstRoleSuffix   = regexp.MustCompile("`([^`]+)`:([A-Za-z][\\w.+-]*):")
	rstEmbeddedLink = regexp.MustCompile("`([^`<]*?)\\s*<([^`>]+)>`(__?)")
	rstNamedRef     = regexp.MustCompile("`([^`]+)`(__?)")
	rstWordRef      = regexp.MustCompile(`(^|[\s(])([A-Za-z0-9][A-Za-z0-9.+-]*)_([\s).,;:!?]|$)`)
	rstFootRef      = regexp.MustCompile(`\[(#[\w-]*|\d+|\*|[A-Za-z][\w.-]*)\]_`)
	rstDefaultRole  = regexp.MustCompile("`([^`]+)`")
	rstSubstRef     = regexp.MustCompile(`\|([^|\s][^|]*)\|`)
)

// inline converts inline markup on one line of text.
func (c *rstConverter) inline(s string, num int) string {
	var saved []string
	keep := func(v string) string {
		saved = append(saved, v)
		return fmt.Sprintf("\x00%d\x00", len(saved)-1)
	}
	s = strings.ReplaceAll(s, `\ `, "")
	s = rstLiteral.ReplaceAllStringFunc(s, func(m string) string {
		return keep(mdCode(rstLiteral.FindStringSubmatch(m)[1]))
	})
	s = rstRolePrefix.ReplaceAllStringFunc(s, func(m string) string {
		sub := rstRolePrefix.FindStringSubmatch(m)
		return keep(c.role(sub[1], sub[2], num))
	})
	s = rstRoleSuffix.ReplaceAllStringFunc(s, func(m string) string {
		sub := rstRoleSuffix.FindStringSubmatch(m)
		return keep(c.role(sub[2], sub[1], num))
	})
	s = rstEmbeddedLink.ReplaceAllStringFunc(s, func(m string) string {
		sub := rstEmbeddedLink.FindStringSubmatch(m)
		text, url := strings.TrimSpace(sub[1]), sub[2]
		if text == "" {
			text = url
		}
		if strings.HasSuffix(url, "_") && !strings.Contains(url, "/") {
			// `text <other_>`_ refers to another target.
			if target, ok := c.targets[refName(strings.TrimSuffix(url, "_"))]; ok {
				url = target
			}
		}
		if sub[3] == "_" {
			c.targets[refName(text)] = url
		}
		return keep("[" + text + "](" + url + ")")
	})
	s = rstNamedRef.ReplaceAllStringFunc(s, func(m string) string {
		sub := rstNamedRef.FindStringSubmatch(m)
		return keep(c.reference(sub[1], sub[2] == "__", num))
	})
	s = rstFootRef.ReplaceAllStringFunc(s, func(m string) string {
		label := rstFootRef.FindStringSubmatch(m)[1]
		return keep("[^" + c.footnoteLabel(label, false) + "]")
	})
	s = rstWordRef.ReplaceAllStringFunc(s, func(m string) string {
		sub := rstWordRef.FindStringSubmatch(m)
		name := refName(sub[2])
		if _, ok := c.targets[name]; !ok {
			if _, ok := c.sections[name]; !ok {
				return m
			}
		}
		return sub[1] + keep(c.reference(sub[2], false, num)) + sub[3]
	})
	s = rstDefaultRole.ReplaceAllStringFunc(s, func(m string) string {
		return keep("*" + rstDefaultRole.FindStringSubmatch(m)[1] + "*")
	})
	s = rstSubstRef.ReplaceAllStringFunc(s, func(m string) string {
		name := rstSubstRef.FindStringSubmatch(m)[1]
		switch name {
		case "filename", "static", "attach", "tag", "category", "author", "index":
			return m // Pelican's legacy link syntax, converted later
		}
		c.note(Manual, num, "Substitution `|%s|` is not converted.", name)
		return m
	})
	// Raw HTML is literal text in reStructuredText.
	s = strings.ReplaceAll(s, "<", "&lt;")
	for k := len(saved) - 1; k >= 0; k-- {
		s = strings.Replace(s, fmt.Sprintf("\x00%d\x00", k), saved[k], 1)
	}
	return s
}

func mdCode(s string) string {
	ticks := "`"
	for strings.Contains(s, ticks) {
		ticks += "`"
	}
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return ticks + " " + s + " " + ticks
	}
	return ticks + s + ticks
}

func (c *rstConverter) role(name, text string, num int) string {
	switch name {
	case "code", "literal", "samp", "command", "file", "kbd":
		return mdCode(text)
	case "emphasis", "title-reference", "t", "title", "dfn":
		return "*" + text + "*"
	case "strong":
		return "**" + text + "**"
	case "sub", "subscript":
		return "<sub>" + text + "</sub>"
	case "sup", "superscript":
		return "<sup>" + text + "</sup>"
	case "math":
		c.doc.usesMath = true
		return "$" + text + "$"
	case "abbr":
		if word, _, ok := strings.Cut(text, " ("); ok {
			return word
		}
		return text
	case "pep", "pep-reference":
		return fmt.Sprintf("[PEP %s](https://peps.python.org/pep-%04s/)", text, text)
	case "rfc", "rfc-reference":
		return fmt.Sprintf("[RFC %s](https://www.rfc-editor.org/rfc/rfc%s)", text, text)
	}
	c.note(Review, num, "Role `:%s:` has no Markdown equivalent; its text was kept.", name)
	return text
}

// reference resolves a named or anonymous hyperlink reference.
func (c *rstConverter) reference(text string, anonymous bool, num int) string {
	if anonymous {
		if c.anonUsed < len(c.anonymous) {
			url := c.anonymous[c.anonUsed]
			c.anonUsed++
			return "[" + text + "](" + url + ")"
		}
		c.note(Manual, num, "Anonymous reference `%s` has no matching `__` target.", text)
		return text
	}
	name := refName(text)
	if url, ok := c.targets[name]; ok {
		return "[" + text + "](" + url + ")"
	}
	if anchor, ok := c.sections[name]; ok {
		c.note(Review, num, "The link to section %q uses the anchor `%s`; check it matches the heading's id.", text, anchor)
		return "[" + text + "](" + anchor + ")"
	}
	c.note(Manual, num, "Reference `%s` has no target and was left as text.", text)
	return text
}

func (c *rstConverter) footnoteLabel(label string, definition bool) string {
	switch {
	case label == "#" || label == "*":
		if definition {
			c.footnotes++
			return strconv.Itoa(c.footnotes)
		}
		c.footRefs++
		return strconv.Itoa(c.footRefs)
	case strings.HasPrefix(label, "#"):
		return label[1:]
	}
	return label
}

var mdLineStart = regexp.MustCompile(`^(#|>|[-+*] |\d+[.)] |=+$|-+$)`)

// escapeLine escapes a paragraph line that Markdown would otherwise read
// as a heading, quote, list item, or setext underline.
func (c *rstConverter) escapeLine(s string) string {
	if m := mdLineStart.FindStringIndex(s); m != nil {
		if d := strings.IndexAny(s, ".)"); d > 0 && d < m[1] && s[0] >= '0' && s[0] <= '9' {
			return s[:d] + `\` + s[d:]
		}
		return `\` + s
	}
	return s
}
