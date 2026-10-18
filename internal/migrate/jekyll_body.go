package migrate

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Liquid runs over the whole file before Markdown, code blocks included,
// so raw blocks, comments, and highlight blocks are handled everywhere;
// the remaining Liquid is converted only in prose, and Liquid left inside
// code blocks is reported, since Jekyll would have rendered it.

var (
	liquidRaw           = regexp.MustCompile(`(?s)\{%-?\s*raw\s*-?%\}(.*?)\{%-?\s*endraw\s*-?%\}`)
	jekyllLiquidComment = regexp.MustCompile(`(?s)\{%-?\s*comment\s*-?%\}.*?\{%-?\s*endcomment\s*-?%\}`)
	liquidHighlight     = regexp.MustCompile(`(?s)\{%-?\s*highlight\s+([A-Za-z0-9_+#.-]+)([^%]*?)-?%\}(.*?)\{%-?\s*endhighlight\s*-?%\}`)
	liquidPostURL       = regexp.MustCompile(`\{%-?\s*post_url\s+([^\s%]+)\s*-?%\}`)
	liquidLink          = regexp.MustCompile(`\{%-?\s*link\s+([^\s%]+)\s*-?%\}`)
	liquidInclude       = regexp.MustCompile(`\{%-?\s*include\s+([^\s%]+)((?:\s+[A-Za-z0-9_-]+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s%]+))*)\s*-?%\}`)
	liquidIncArg        = regexp.MustCompile(`([A-Za-z0-9_-]+)\s*=\s*("[^"]*"|'[^']*'|[^\s%]+)`)
	liquidGist          = regexp.MustCompile(`\{%-?\s*gist\s+([^\s%]+)(?:\s+([^\s%]+))?\s*-?%\}`)
	liquidURLFilter     = regexp.MustCompile(`\{\{-?\s*(?:site\.url\s*\}\}\s*\{\{-?\s*)?["']([^"']*)["']\s*\|\s*(?:relative_url|absolute_url|prepend:\s*site\.baseurl)\s*-?\}\}`)
	liquidBaseURL       = regexp.MustCompile(`\{\{-?\s*site\.url\s*-?\}\}\s*\{\{-?\s*site\.baseurl\s*-?\}\}|\{\{-?\s*site\.(?:baseurl|url)\s*-?\}\}`)
	liquidVariable      = regexp.MustCompile(`\{\{-?\s*(page|site)\.([A-Za-z0-9_]+)\s*-?\}\}`)
	jekyllLiquidAny     = regexp.MustCompile(`\{\{[^<].*?\}\}|\{%.*?%\}`)
	mdLink              = regexp.MustCompile(`\]\(([^)\s#?]+\.(?:md|markdown))((?:#|\?)[^)\s]*)?\)`)

	kramdownTOC        = regexp.MustCompile(`(?m)^[ \t]*(?:[*+-]|\d+\.)[ \t]+.*\n[ \t]*\{:[ \t]*(?:\.?toc|#markdown-toc)[^}]*\}[ \t]*\n?`)
	kramdownOptions    = regexp.MustCompile(`\{::options\b[^}]*/\}\n?`)
	kramdownComment    = regexp.MustCompile(`(?s)\{::comment\}.*?\{:/(?:comment)?\}\n?`)
	kramdownNoMarkdown = regexp.MustCompile(`(?s)\{::nomarkdown\}(.*?)\{:/(?:nomarkdown)?\}`)
	kramdownHeadingIAL = regexp.MustCompile(`^(#{1,6}[ \t].*?)[ \t]*\{:?[ \t]*([^}]*)\}[ \t]*$`)
	kramdownBlockIAL   = regexp.MustCompile(`^[ \t]*\{:[^}]*\}[ \t]*$`)
	kramdownSpanIAL    = regexp.MustCompile(`([)\]*_` + "`" + `])\{:[^}]*\}`)
	kramdownALD        = regexp.MustCompile(`^[ \t]*\{:[A-Za-z][\w-]*:[^}]*\}[ \t]*$`)
	kramdownDefList    = regexp.MustCompile(`(?m)^\S.*\n:[ \t]+\S`)
	kramdownAbbr       = regexp.MustCompile(`(?m)^\*\[[^\]]+\]:`)
	setextUnderline    = regexp.MustCompile(`^(=+|-+)[ \t]*$`)
	blankRuns          = regexp.MustCompile(`\n{3,}`)
)

func (s *jekyllSite) convertBody(d *jekyllDoc) {
	orig := d.doc.Body
	body := orig
	rel := d.origin
	at := func(snippet string) int { return d.doc.BodyLine + lineOfSnippet(orig, snippet) - 1 }

	if d.skipLiquid {
		// render_with_liquid: false shows Liquid as text, which Nida does.
		s.convertKramdown(d, body, at)
		return
	}

	// Raw blocks keep their content verbatim; hide it until the end.
	var raws []string
	body = liquidRaw.ReplaceAllStringFunc(body, func(m string) string {
		// Tags on their own lines leave the newlines just inside them.
		inner := strings.TrimSuffix(strings.TrimPrefix(liquidRaw.FindStringSubmatch(m)[1], "\n"), "\n")
		raws = append(raws, inner)
		return fmt.Sprintf("\x00RAW%d\x00", len(raws)-1)
	})
	if len(raws) > 0 {
		s.p.Report.Add(Changed, rel, 0, "Removed `{%% raw %%}` tags; Nida does not run Liquid, so their content shows as written.")
	}
	if jekyllLiquidComment.MatchString(body) {
		body = jekyllLiquidComment.ReplaceAllString(body, "")
		s.p.Report.Add(Changed, rel, 0, "Removed `{%% comment %%}` blocks.")
	}
	body = liquidHighlight.ReplaceAllStringFunc(body, func(m string) string {
		sub := liquidHighlight.FindStringSubmatch(m)
		s.p.Report.Add(Changed, rel, 0, "Converted `{%% highlight %%}` blocks to fenced code blocks.")
		return jekyllHighlightFence(sub[1], sub[2], sub[3])
	})
	body = s.convertKramdown(d, body, at)

	body = RewriteProse(body, func(prose string, _ int) string {
		return s.convertLiquid(d, prose, at)
	})

	// Report what is left. Shortcodes we wrote start with {{<.
	RewriteProse(body, func(prose string, _ int) string {
		for _, m := range jekyllLiquidAny.FindAllString(prose, -1) {
			s.p.Report.Add(Manual, rel, at(m), "Liquid `%s` is left as text; Nida does not run Liquid. Replace it with content or a shortcode.", truncate(m, 60))
		}
		return prose
	})
	proseLiquid := 0
	RewriteProse(body, func(prose string, _ int) string {
		proseLiquid += len(jekyllLiquidAny.FindAllString(prose, -1))
		return prose
	})
	if len(jekyllLiquidAny.FindAllString(body, -1)) > proseLiquid {
		s.p.Report.Add(Review, rel, 0, "Code blocks contain Liquid, which Jekyll rendered and Nida shows as written. Replace it with its output, or keep it if it was meant as an example.")
	}

	if body != orig {
		// Removed tags and blocks leave runs of blank lines behind; outside
		// code they mean nothing in Markdown.
		body = RewriteProse(body, func(prose string, _ int) string {
			return blankRuns.ReplaceAllString(prose, "\n\n")
		})
		body = strings.TrimLeft(body, "\n")
	}
	for i, raw := range raws {
		body = strings.Replace(body, fmt.Sprintf("\x00RAW%d\x00", i), raw, 1)
	}
	d.doc.SetBody(body)
}

// jekyllHighlightFence turns a highlight block into a fenced code block with
// Nida's info string options.
func jekyllHighlightFence(lang, opts, code string) string {
	code = strings.TrimPrefix(strings.TrimPrefix(code, "\r"), "\n")
	code = strings.TrimRight(code, " \t")
	code = strings.TrimSuffix(code, "\n")
	var attrs []string
	for _, opt := range strings.Fields(opts) {
		key, value, _ := strings.Cut(opt, "=")
		value = strings.Trim(value, `"'`)
		switch key {
		case "linenos":
			attrs = append(attrs, "linenos=true")
		case "mark_lines", "hl_lines":
			attrs = append(attrs, fmt.Sprintf("hl_lines=%q", value))
		}
	}
	// A quoted multi-word mark_lines value was split by Fields; rejoin it.
	if m := regexp.MustCompile(`(?:mark_lines|hl_lines)\s*=\s*"([^"]*)"`).FindStringSubmatch(opts); m != nil {
		attrs = slices.DeleteFunc(attrs, func(a string) bool { return strings.HasPrefix(a, "hl_lines=") })
		attrs = append(attrs, fmt.Sprintf("hl_lines=%q", m[1]))
	}
	fence := "```"
	for strings.Contains(code, fence) {
		fence += "`"
	}
	info := lang
	if len(attrs) > 0 {
		info += " {" + strings.Join(attrs, ", ") + "}"
	}
	return fence + info + "\n" + code + "\n" + fence
}

// convertKramdown handles kramdown syntax that CommonMark lacks, line by
// line outside fenced code.
func (s *jekyllSite) convertKramdown(d *jekyllDoc, body string, at func(string) int) string {
	rel := d.origin
	r := s.p.Report
	if kramdownTOC.MatchString(body) {
		body = kramdownTOC.ReplaceAllString(body, "")
		r.Add(Review, rel, 0, "Removed the kramdown `{:toc}` list. Templates can render a table of contents from `.Page.TOC`.")
	}
	if kramdownOptions.MatchString(body) {
		body = kramdownOptions.ReplaceAllString(body, "")
		r.Add(Review, rel, 0, "Removed `{::options}`; Nida has no per-page Markdown options.")
	}
	if kramdownComment.MatchString(body) {
		body = kramdownComment.ReplaceAllString(body, "")
		r.Add(Changed, rel, 0, "Removed kramdown `{::comment}` blocks.")
	}
	if kramdownNoMarkdown.MatchString(body) {
		body = kramdownNoMarkdown.ReplaceAllString(body, "$1")
		s.p.UsesRawHTML()
		r.Add(Review, rel, 0, "Removed `{::nomarkdown}` markers; their content is now ordinary Markdown with raw HTML.")
	}

	lines := strings.SplitAfter(body, "\n")
	fence := "" // the open fence's run, such as "```"
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r\n")
		nl := lines[i][len(line):]
		if fence != "" {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, fence) && strings.Trim(trimmed, fence[:1]) == "" {
				fence = ""
			}
			continue
		}
		if m := fenceOpen.FindStringSubmatch(line); m != nil {
			fence = m[2]
			continue
		}
		switch {
		case kramdownALD.MatchString(line):
			lines[i] = ""
			r.Add(Review, rel, at(line), "Removed a kramdown attribute list definition `%s`.", truncate(strings.TrimSpace(line), 50))
		case kramdownHeadingIAL.MatchString(line):
			m := kramdownHeadingIAL.FindStringSubmatch(line)
			lines[i] = m[1] + nidaAttrs(m[2]) + nl
			r.Add(Changed, rel, 0, "Converted kramdown heading attributes to Nida's `{#id .class}` form.")
		case kramdownBlockIAL.MatchString(line):
			attrs := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(line), "{:"), "}")
			// An IAL right under an ATX or setext heading belongs to it.
			if i > 0 {
				prev := strings.TrimRight(lines[i-1], "\r\n")
				if strings.HasPrefix(prev, "#") && !strings.Contains(prev, "{#") {
					lines[i-1] = prev + nidaAttrs(attrs) + lines[i-1][len(prev):]
					lines[i] = ""
					r.Add(Changed, rel, 0, "Converted kramdown heading attributes to Nida's `{#id .class}` form.")
					continue
				}
				if setextUnderline.MatchString(prev) && i > 1 {
					title := strings.TrimRight(lines[i-2], "\r\n")
					level := "#"
					if strings.HasPrefix(prev, "-") {
						level = "##"
					}
					lines[i-2] = level + " " + title + nidaAttrs(attrs) + lines[i-2][len(title):]
					lines[i-1], lines[i] = "", ""
					r.Add(Changed, rel, 0, "Converted kramdown heading attributes to Nida's `{#id .class}` form.")
					continue
				}
			}
			lines[i] = ""
			r.Add(Review, rel, at(line), "Removed kramdown block attributes `%s`; Nida cannot attach classes or attributes to paragraphs, lists, or tables.", truncate(strings.TrimSpace(line), 50))
		case kramdownSpanIAL.MatchString(line):
			lines[i] = kramdownSpanIAL.ReplaceAllString(line, "$1") + nl
			r.Add(Review, rel, at(line), "Removed kramdown span attributes after a link, image, or emphasis; Nida cannot add attributes there.")
		}
	}
	body = strings.Join(lines, "")

	if kramdownDefList.MatchString(body) {
		r.Add(Review, rel, 0, "This may use kramdown definition lists (`term` then `: definition`), which Nida renders as plain paragraphs.")
	}
	if kramdownAbbr.MatchString(body) {
		r.Add(Review, rel, 0, "kramdown abbreviations (`*[ABBR]: ...`) are not supported and show as text.")
	}
	if strings.Contains(body, "$$") {
		s.math = true
	}
	if sep := d.excerptSep; sep != "" && sep != "\n\n" && !strings.Contains(sep, "more") {
		if i := strings.Index(body, sep); i >= 0 {
			body = body[:i] + "\n\n<!-- more -->\n\n" + body[i+len(sep):]
			r.Add(Changed, rel, 0, "Replaced the excerpt separator `%s` with `<!-- more -->`.", strings.TrimSpace(sep))
		}
	}
	return body
}

// nidaAttrs turns kramdown attributes like "#id .cls" or ": #id" into
// Nida's " {#id .cls}". Key-value attributes are kept, which goldmark
// accepts.
func nidaAttrs(attrs string) string {
	attrs = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(attrs), ":"))
	var keep []string
	for _, f := range strings.Fields(attrs) {
		if f == ".no_toc" {
			continue
		}
		keep = append(keep, f)
	}
	if len(keep) == 0 {
		return ""
	}
	return " {" + strings.Join(keep, " ") + "}"
}

// convertLiquid converts the Liquid Nida can express in a run of prose.
func (s *jekyllSite) convertLiquid(d *jekyllDoc, prose string, at func(string) int) string {
	rel := d.origin
	r := s.p.Report

	prose = liquidURLFilter.ReplaceAllStringFunc(prose, func(m string) string {
		target := liquidURLFilter.FindStringSubmatch(m)[1]
		r.Add(Changed, rel, 0, "Replaced `relative_url` and `absolute_url` filters with site-relative paths; Nida adds the base path itself.")
		return "/" + strings.TrimPrefix(target, "/")
	})
	if liquidBaseURL.MatchString(prose) {
		prose = liquidBaseURL.ReplaceAllString(prose, "")
		r.Add(Changed, rel, 0, "Removed `{{ site.baseurl }}` and `{{ site.url }}` before links; Nida adds the base path to site-relative links itself.")
	}
	prose = liquidPostURL.ReplaceAllStringFunc(prose, func(m string) string {
		name := liquidPostURL.FindStringSubmatch(m)[1]
		if dest := s.findPost(name); dest != "" {
			r.Add(Changed, rel, 0, "Converted `{%% post_url %%}` and `{%% link %%}` tags to `@/` links.")
			return "@/" + dest
		}
		r.Add(Manual, rel, at(m), "`%s` names no post; fix the link by hand.", m)
		return m
	})
	prose = liquidLink.ReplaceAllStringFunc(prose, func(m string) string {
		target := strings.TrimPrefix(liquidLink.FindStringSubmatch(m)[1], "/")
		if strings.Contains(target, "{{") {
			return m
		}
		if dest, ok := s.linkTarget(target); ok {
			r.Add(Changed, rel, 0, "Converted `{%% post_url %%}` and `{%% link %%}` tags to `@/` links.")
			return dest
		}
		r.Add(Manual, rel, at(m), "`%s` names no file in the site; fix the link by hand.", m)
		return m
	})
	prose = liquidGist.ReplaceAllStringFunc(prose, func(m string) string {
		sub := liquidGist.FindStringSubmatch(m)
		args := []Arg{{Value: sub[1]}}
		if sub[2] != "" {
			args = append(args, Arg{Value: sub[2]})
		}
		s.p.Placeholder("gist", rel, at(m))
		r.Add(Changed, rel, 0, "Converted Liquid tags to Nida shortcodes.")
		return Shortcode("gist", args)
	})
	prose = liquidInclude.ReplaceAllStringFunc(prose, func(m string) string {
		sub := liquidInclude.FindStringSubmatch(m)
		file := sub[1]
		if strings.Contains(file, "{{") {
			return m
		}
		var args []Arg
		for _, a := range liquidIncArg.FindAllStringSubmatch(sub[2], -1) {
			v := a[2]
			if !strings.HasPrefix(v, `"`) && !strings.HasPrefix(v, "'") {
				r.Add(Manual, rel, at(m), "`%s` passes a variable; it was left as Liquid.", truncate(m, 60))
				return m
			}
			args = append(args, Arg{Key: a[1], Value: v[1 : len(v)-1]})
		}
		name := includeShortcodeName(file)
		if slices.Contains(builtinShortcodes, name) {
			r.Add(Review, rel, at(m), "`%s` now uses Nida's built-in `%s` shortcode; check that its arguments match.", truncate(m, 60), name)
		} else {
			s.p.Placeholder(name, rel, at(m))
		}
		r.Add(Changed, rel, 0, "Converted `{%% include %%}` tags to shortcodes named after the include file. Port each include from `_migration/original/_includes/` to `templates/shortcodes/`; its parameters arrive as `.Get \"name\"` instead of `include.name`.")
		return Shortcode(name, args)
	})
	prose = liquidVariable.ReplaceAllStringFunc(prose, func(m string) string {
		sub := liquidVariable.FindStringSubmatch(m)
		var value any
		var ok bool
		if sub[1] == "page" {
			value, ok = d.vars[sub[2]]
		} else {
			value, ok = s.cfg[sub[2]]
		}
		if !ok {
			return m
		}
		switch v := value.(type) {
		case string, int, int64, float64, bool:
			r.Add(Changed, rel, 0, "Replaced simple `{{ page.x }}` and `{{ site.x }}` variables with their values.")
			return fmt.Sprint(v)
		}
		return m
	})
	prose = mdLink.ReplaceAllStringFunc(prose, func(m string) string {
		sub := mdLink.FindStringSubmatch(m)
		target := sub[1]
		if strings.Contains(target, "://") {
			return m
		}
		var src string
		if strings.HasPrefix(target, "/") {
			src = strings.TrimPrefix(target, "/")
		} else {
			src = path.Join(path.Dir(d.origin), target)
		}
		if t, ok := s.byOrigin[src]; ok {
			r.Add(Changed, rel, 0, "Converted links to `.md` files into `@/` links.")
			return "](@/" + strings.TrimPrefix(t.dest, "content/") + sub[2] + ")"
		}
		return m
	})
	return prose
}

// includeShortcodeName names a shortcode after an include file:
// "figure.html" is figure, "media/video.html" is media-video.
func includeShortcodeName(file string) string {
	name := strings.TrimSuffix(file, path.Ext(file))
	name = strings.ReplaceAll(name, "/", "-")
	return strings.Trim(regexp.MustCompile(`[^A-Za-z0-9_-]+`).ReplaceAllString(name, "-"), "-")
}

// lineOfSnippet finds the line of the first occurrence of a snippet in
// the original body, for reports. Conversions shift offsets, so searching
// the original text is more accurate than tracking them.
func lineOfSnippet(body, snippet string) int {
	if i := strings.Index(body, snippet); i >= 0 {
		return LineOf(body, i)
	}
	return 1
}
