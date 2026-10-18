package migrate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// liquidTags are Liquid and Nunjucks tags that are template logic rather
// than shortcodes. They have no Nida equivalent in content.
var liquidTags = map[string]bool{
	"if": true, "elsif": true, "elif": true, "else": true, "unless": true, "for": true, "case": true, "when": true,
	"assign": true, "capture": true, "include": true, "render": true, "set": true, "block": true, "extends": true,
	"macro": true, "call": true, "filter": true, "import": true, "from": true, "tablerow": true, "cycle": true,
	"increment": true, "decrement": true, "liquid": true, "echo": true, "break": true, "continue": true,
	"layout": true, "ifchanged": true, "asyncEach": true, "asyncAll": true, "with": true,
}

var (
	liquidTag      = regexp.MustCompile(`\{%-?\s*([A-Za-z_][\w-]*)\s*((?:[^%]|%[^}])*?)\s*-?%\}`)
	liquidRawBlock = regexp.MustCompile(`(?s)\{%-?\s*(raw|verbatim)\s*-?%\}(.*?)\{%-?\s*end(?:raw|verbatim)\s*-?%\}`)
	liquidComment  = regexp.MustCompile(`(?s)\{%-?\s*comment\s*-?%\}.*?\{%-?\s*endcomment\s*-?%\}`)
	njkComment     = regexp.MustCompile(`(?s)\{#.*?#\}`)
	liquidAny      = regexp.MustCompile(`\{\{.*?\}\}|\{%.*?%\}`)
)

// eleventyBody converts one template's Markdown body.
type eleventyBody struct {
	p      *Plan
	cfg    *eleventyConfig
	file   string // source-relative path, for the report
	doc    *Document
	page   eleventyPage
	global map[string]any
	// asset resolves a shortcode's image source to its new URL.
	asset func(src string) (string, bool)
}

const rawSentinel = "\x00NIDARAW%d\x00"

// convert rewrites the body for Nida. Liquid (or Nunjucks) runs over the
// whole file in Eleventy, code blocks included, so raw blocks and comments
// are handled everywhere; shortcodes and variables are converted only in
// prose, and template syntax left inside code is reported.
func (b *eleventyBody) convert() {
	body := b.doc.Body
	if b.cfg.mdEngine == "" {
		b.doc.SetBody(b.excerpt(body))
		return
	}
	njk := b.cfg.mdEngine == "njk"

	var raws []string
	body = replaceRawBlocks(body, &raws)
	if len(raws) > 0 {
		b.p.Report.Add(Changed, b.file, 0, "Unwrapped `{%% raw %%}` blocks; Nida does not run Liquid, so their contents are shown as written.")
	}
	if liquidComment.MatchString(body) || (njk && njkComment.MatchString(body)) {
		body = liquidComment.ReplaceAllString(body, "")
		if njk {
			body = njkComment.ReplaceAllString(body, "")
		}
		b.p.Report.Add(Changed, b.file, 0, "Removed template comments, which Eleventy did not output.")
	}

	var proseRanges [][2]int
	converted := RewriteProse(body, func(prose string, offset int) string {
		proseRanges = append(proseRanges, [2]int{offset, offset + len(prose)})
		return b.prose(prose, func(at int) int { return b.line(body, offset+at) })
	})
	// Template syntax inside code blocks ran in Eleventy too.
	for _, m := range liquidAny.FindAllStringIndex(body, -1) {
		inProse := slices.ContainsFunc(proseRanges, func(r [2]int) bool { return m[0] >= r[0] && m[0] < r[1] })
		if !inProse {
			b.p.Report.Add(Review, b.file, b.line(body, m[0]), "`%s` is inside code. Eleventy ran it as a template without `{%% raw %%}`; Nida shows it as written.", truncate(body[m[0]:m[1]], 50))
		}
	}
	for i, raw := range raws {
		converted = strings.Replace(converted, fmt.Sprintf(rawSentinel, i), raw, 1)
	}
	b.doc.SetBody(b.excerpt(converted))
}

// replaceRawBlocks swaps each raw block for a sentinel and records its
// contents. A raw tag on a line of its own takes its line break with it,
// so a code fence wrapped in raw tags keeps no stray blank lines.
func replaceRawBlocks(body string, raws *[]string) string {
	var out strings.Builder
	last := 0
	for _, m := range liquidRawBlock.FindAllStringSubmatchIndex(body, -1) {
		inner := body[m[4]:m[5]]
		ownLineStart := m[0] == 0 || body[m[0]-1] == '\n'
		if ownLineStart && strings.HasPrefix(inner, "\n") {
			inner = inner[1:]
		}
		if strings.HasSuffix(inner, "\n") {
			// the end tag starts its own line
			end := m[1]
			if end == len(body) || body[end] == '\n' {
				inner = strings.TrimSuffix(inner, "\n")
			}
		}
		*raws = append(*raws, inner)
		out.WriteString(body[last:m[0]])
		fmt.Fprintf(&out, rawSentinel, len(*raws)-1)
		last = m[1]
	}
	out.WriteString(body[last:])
	return out.String()
}

func (b *eleventyBody) line(body string, offset int) int {
	return b.doc.BodyLine + LineOf(body, offset) - 1
}

// prose converts shortcodes and variables in one run of prose.
func (b *eleventyBody) prose(prose string, line func(int) int) string {
	var out strings.Builder
	last := 0
	matches := liquidTag.FindAllStringSubmatchIndex(prose, -1)
	for i := 0; i < len(matches); i++ {
		m := matches[i]
		if m[0] < last {
			continue
		}
		name, args := prose[m[2]:m[3]], prose[m[4]:m[5]]
		if strings.HasPrefix(name, "end") || name == "else" || name == "elsif" || name == "elif" || name == "when" {
			continue // reported with the tag that opens the block
		}
		if liquidTags[name] {
			b.p.Report.Add(Manual, b.file, line(m[0]), "Template tag `%s` was left as it was; Nida content has no template logic. Rewrite it as Markdown or a shortcode.", truncate(prose[m[0]:m[1]], 50))
			continue
		}
		out.WriteString(prose[last:m[0]])
		last = m[1]

		closeRe := regexp.MustCompile(`\{%-?\s*end` + regexp.QuoteMeta(name) + `\s*-?%\}`)
		paired := slices.Contains(b.cfg.paired, name) || name == "highlight"
		var inner string
		hasInner := false
		if paired || !slices.Contains(b.cfg.shortcodes, name) {
			if loc := closeRe.FindStringIndex(prose[m[1]:]); loc != nil {
				inner = prose[m[1] : m[1]+loc[0]]
				last = m[1] + loc[1]
				hasInner = true
			}
		}

		if name == "highlight" && hasInner {
			out.WriteString(highlightFence(args, inner))
			b.p.Report.Add(Changed, b.file, 0, "`{%% highlight %%}` blocks became fenced code blocks; Nida highlights code natively.")
			continue
		}
		parsed, ok := b.args(args)
		if !ok {
			b.p.Report.Add(Manual, b.file, line(m[0]), "Could not read the arguments of `%s`; it was left as it was.", truncate(prose[m[0]:m[1]], 50))
			out.WriteString(prose[m[0]:last])
			continue
		}
		switch {
		case name == "image" && !hasInner && len(parsed) > 0:
			src := parsed[0].Value
			if newSrc, found := b.asset(src); found {
				src = newSrc
			} else {
				b.p.Report.Add(Review, b.file, line(m[0]), "Image `%s` was not found; check the figure's `src`.", parsed[0].Value)
			}
			fig := []Arg{{Key: "src", Value: src}}
			if len(parsed) > 1 {
				fig = append(fig, Arg{Key: "alt", Value: parsed[1].Value})
			}
			out.WriteString(Shortcode("figure", fig))
			b.p.Report.Add(Review, b.file, 0, "`{%% image %%}` became Nida's `figure` shortcode. Turn on `[pipeline.images]` for responsive sizes like eleventy-img's.")
		case name == "youtube" && !hasInner && len(parsed) > 0:
			out.WriteString(Shortcode("youtube", parsed[:1]))
			b.p.Report.Add(Changed, b.file, 0, "Converted Eleventy shortcodes to Nida's `{{< name >}}` syntax.")
		default:
			out.WriteString(Shortcode(name, parsed))
			if hasInner {
				out.WriteString(inner)
				out.WriteString(ClosingShortcode(name))
			}
			b.p.Placeholder(name, b.file, line(m[0]))
			b.p.Report.Add(Changed, b.file, 0, "Converted Eleventy shortcodes to Nida's `{{< name >}}` syntax.")
		}
	}
	out.WriteString(prose[last:])
	return b.outputs(out.String(), line)
}

// outputs replaces {{ ... }} that can be worked out from the page and
// global data with their values, and reports the rest.
func (b *eleventyBody) outputs(prose string, line func(int) int) string {
	pg := b.page
	data := map[string]any{}
	for k, v := range b.global {
		data[k] = v
	}
	for k, v := range pg.data {
		data[k] = v
	}
	pg.data = data
	return liquidOutput.ReplaceAllStringFunc(prose, func(m string) string {
		expr := liquidOutput.FindStringSubmatch(m)[1]
		if strings.HasPrefix(strings.TrimSpace(expr), "<") {
			return m // a Nida shortcode written above
		}
		value, err := evalLiquidExpr(expr, pg)
		if err != nil {
			b.p.Report.Add(Manual, b.file, line(strings.Index(prose, m)), "`%s` was left as it was; Nida content cannot run template expressions. Replace it with its text.", truncate(m, 50))
			return m
		}
		b.p.Report.Add(Changed, b.file, 0, "Replaced template variables such as `{{ title }}` with their values from the front matter and global data.")
		if b.cfg.pathPrefix != "" && strings.Contains(expr, "url") {
			b.p.Report.Add(Review, b.file, 0, "Links that used the `url` filter no longer include the path prefix in raw HTML; Markdown links get the base path from `base_url`.")
		}
		return value
	})
}

// args parses shortcode arguments: quoted strings, numbers, booleans, and
// data variables, separated by commas or spaces, with optional key=value.
func (b *eleventyBody) args(raw string) ([]Arg, bool) {
	var out []Arg
	tokens, ok := splitArgs(raw)
	if !ok {
		return nil, false
	}
	for _, tok := range tokens {
		key := ""
		if k, v, found := strings.Cut(tok, "="); found && regexp.MustCompile(`^\w+$`).MatchString(k) && !strings.HasPrefix(tok, `"`) {
			key, tok = k, v
		}
		if s, ok := unquoteLiquid(tok); ok {
			out = append(out, Arg{Key: key, Value: s})
			continue
		}
		if regexp.MustCompile(`^-?[\d.]+$|^true$|^false$`).MatchString(tok) {
			out = append(out, Arg{Key: key, Value: tok})
			continue
		}
		pg := b.page
		data := map[string]any{}
		for k, v := range b.global {
			data[k] = v
		}
		for k, v := range pg.data {
			data[k] = v
		}
		pg.data = data
		v, err := evalLiquidExpr(tok, pg)
		if err != nil {
			return nil, false
		}
		out = append(out, Arg{Key: key, Value: v})
	}
	return out, true
}

func splitArgs(raw string) ([]string, bool) {
	var tokens []string
	var cur strings.Builder
	var quote byte
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case quote != 0:
			cur.WriteByte(c)
			if c == '\\' && i+1 < len(raw) {
				i++
				cur.WriteByte(raw[i])
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
			cur.WriteByte(c)
		case c == ',' || c == ' ' || c == '\t' || c == '\n':
			flush()
		case c == '(' || c == ')' || c == '[' || c == '{':
			return nil, false
		default:
			cur.WriteByte(c)
		}
	}
	if quote != 0 {
		return nil, false
	}
	flush()
	return tokens, true
}

// highlightFence turns {% highlight "js 1,3-4" %}code{% endhighlight %}
// into a fenced code block.
func highlightFence(args, inner string) string {
	args = strings.TrimSpace(args)
	if s, ok := unquoteLiquid(args); ok {
		args = s
	}
	fields := strings.Fields(args)
	info := ""
	if len(fields) > 0 {
		info = fields[0]
	}
	if len(fields) > 1 {
		ranges := strings.ReplaceAll(strings.Join(fields[1:], ","), ",", " ")
		info += fmt.Sprintf(` {hl_lines="%s"}`, strings.TrimSpace(ranges))
	}
	inner = strings.Trim(inner, "\n")
	fence := "```"
	for strings.Contains(inner, fence) {
		fence += "`"
	}
	return fence + info + "\n" + inner + "\n" + fence
}

// excerpt turns Eleventy's front matter excerpt separator into Nida's
// <!-- more --> summary marker.
func (b *eleventyBody) excerpt(body string) string {
	if !b.cfg.excerpt {
		return body
	}
	if sep := b.cfg.excerptSep; sep != "" {
		if strings.Contains(body, sep) && !strings.Contains(body, "<!-- more -->") {
			b.p.Report.Add(Changed, b.file, 0, "The excerpt separator became `<!-- more -->`.")
			return strings.Replace(body, sep, "<!-- more -->", 1)
		}
		return body
	}
	// Without a separator, gray-matter splits at the first --- line, which
	// still renders as a rule.
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			lines = slices.Insert(lines, i, "<!-- more -->", "")
			b.p.Report.Add(Changed, b.file, 0, "Marked the excerpt, which ended at the first `---`, with `<!-- more -->`.")
			return strings.Join(lines, "\n")
		}
	}
	return body
}

var htmlPreBlock = regexp.MustCompile(`(?is)<(pre|textarea|script|style)\b.*?</(pre|textarea|script|style)>`)

// htmlAsMarkdown prepares an HTML page body for a Markdown file. A blank
// line ends an HTML block in Markdown and indented lines after it become
// code, so blank lines outside <pre> and similar elements are removed and
// the common indentation is stripped.
func htmlAsMarkdown(body string) string {
	var keep []string
	body = htmlPreBlock.ReplaceAllStringFunc(body, func(m string) string {
		keep = append(keep, m)
		return fmt.Sprintf("\x00HTMLKEEP%d\x00", len(keep)-1)
	})
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimRight(line, " \t\r"))
		}
	}
	indent := -1
	for _, line := range lines {
		n := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	for i := range lines {
		if indent > 0 && len(lines[i]) >= indent {
			lines[i] = lines[i][indent:]
		}
	}
	out := strings.Join(lines, "\n") + "\n"
	for i, k := range keep {
		out = strings.Replace(out, fmt.Sprintf("\x00HTMLKEEP%d\x00", i), k, 1)
	}
	return out
}
