package migrate

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/alecthomas/chroma/v2/styles"
	toml "github.com/pelletier/go-toml/v2"

	"github.com/MohamedElashri/nida/internal/content"
)

// zolaSource migrates Zola sites. Zola is the closest generator to Nida:
// TOML front matter, _index.md sections, page bundles, and @/ links all
// carry over, so most of the work is config keys and shortcode syntax.
type zolaSource struct{}

func (zolaSource) Name() string  { return "zola" }
func (zolaSource) Title() string { return "Zola" }

func (zolaSource) Detect(root string) int {
	p := &Plan{Root: root}
	data, err := p.ReadFile("config.toml")
	if err != nil {
		return 0
	}
	var cfg map[string]any
	if toml.Unmarshal(data, &cfg) != nil {
		return 0
	}
	if _, ok := cfg["base_url"]; !ok {
		return 0 // Hugo uses baseURL
	}
	score := 1
	for _, key := range []string{"compile_sass", "build_search_index", "generate_feeds", "generate_feed", "default_language", "feed_filenames"} {
		if _, ok := cfg[key]; ok {
			score++
		}
	}
	if p.Exists("templates") || p.Exists("sass") {
		score++
	}
	if p.Exists("content") {
		score++
	}
	return score
}

func (z zolaSource) Plan(p *Plan) error {
	data, err := p.ReadFile("config.toml")
	if err != nil {
		return err
	}
	var cfg map[string]any
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config.toml: %w", err)
	}
	slugMode := "on"
	if slugify, ok := cfg["slugify"].(map[string]any); ok {
		if mode, ok := slugify["paths"].(string); ok {
			slugMode = mode
		}
	}
	z.planConfig(p, cfg)
	if err := z.planContent(p, slugMode); err != nil {
		return err
	}
	if err := p.Walk("static", func(rel string) error {
		p.Copy(rel, rel)
		return nil
	}); err != nil {
		return err
	}
	z.planSass(p, cfg)
	for _, dir := range []string{"templates", "themes"} {
		found := false
		if err := p.Walk(dir, func(rel string) error {
			found = true
			p.Preserve(rel)
			return nil
		}); err != nil {
			return err
		}
		if found {
			p.Report.Add(Unsupported, dir, 0, "Tera templates in `%s/` are not translated; the site uses Nida's starter templates. The originals are in `%s/%s/`. See the template guide to port them.", dir, preservedDir, dir)
		}
	}
	return nil
}

// planConfig maps Zola's config.toml onto Nida's.
func (zolaSource) planConfig(p *Plan, cfg map[string]any) {
	c := p.Config
	r := p.Report
	const file = "config.toml"
	handled := map[string]bool{}
	take := func(key string) (any, bool) {
		handled[key] = true
		v, ok := cfg[key]
		return v, ok
	}

	for _, key := range []string{"base_url", "title", "description", "minify_html"} {
		if v, ok := take(key); ok {
			c.Set(key, v)
		}
	}
	if v, ok := take("default_language"); ok {
		c.Set("language", v)
	}
	if v, ok := take("output_dir"); ok {
		c.Set("output_dir", v)
	}
	if v, ok := take("author"); ok {
		c.Set("author", v)
	}
	if v, ok := take("theme"); ok {
		r.Add(Unsupported, file, 0, "Zola theme `%v` is made of Tera templates, which Nida cannot use; the site uses Nida's starter templates instead.", v)
	}

	// Feeds. Zola 0.19 renamed generate_feed and feed_filename to plurals.
	generate := false
	for _, key := range []string{"generate_feeds", "generate_feed"} {
		if v, ok := take(key); ok {
			generate, _ = v.(bool)
		}
	}
	filenames := []string{"atom.xml"}
	if v, ok := take("feed_filenames"); ok {
		filenames = stringList(v)
	}
	if v, ok := take("feed_filename"); ok {
		filenames = stringList(v)
	}
	feed := c.Table("feed")
	feed.Set("enabled", false)
	if generate {
		for _, name := range filenames {
			if strings.Contains(name, "rss") {
				rss := c.Table("rss")
				rss.Set("enabled", true)
				rss.Set("filename", name)
			} else {
				feed.Set("enabled", true)
				feed.Set("filename", name)
			}
		}
		r.Add(Changed, file, 0, "Feeds map to `[feed]` (Atom) and `[rss]`, keeping their file names.")
	}

	if v, ok := take("build_search_index"); ok && v == true {
		c.Table("search").Set("enabled", true)
		r.Add(Review, file, 0, "The search index is `search_index.en.js` with a `documentStore.docs` map like Zola's, but its keys are site-relative URLs. Check your search script.")
	}
	if v, ok := take("generate_sitemap"); ok && v == false {
		c.Table("sitemap").Set("enabled", false)
	}
	if v, ok := take("generate_robots_txt"); ok && v == false {
		c.Table("robots").Set("enabled", false)
	}

	if v, ok := take("taxonomies"); ok {
		for _, raw := range anyList(v) {
			tax, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			name, _ := tax["name"].(string)
			if name == "" {
				continue
			}
			t := c.Taxonomy(name)
			t.Set("render", true)
			if pb, ok := tax["paginate_by"]; ok {
				t.Set("paginate_by", pb)
			}
			if pp, ok := tax["paginate_path"]; ok {
				t.Set("paginate_path", pp)
			}
			if f, ok := tax["feed"]; ok && f == true {
				t.Set("feed", true)
			}
			if lang, ok := tax["lang"]; ok {
				r.Add(Unsupported, file, 0, "Taxonomy `%s` is for language `%v`; Nida sites have one language.", name, lang)
			}
		}
	}

	if v, ok := take("markdown"); ok {
		md, _ := v.(map[string]any)
		for key, value := range md {
			switch key {
			case "external_links_target_blank", "external_links_no_follow", "external_links_no_referrer":
				c.Table("markdown").Set(key, value)
			case "highlight_code":
				if value == false {
					r.Add(Unsupported, file, 0, "`markdown.highlight_code = false` has no equivalent; Nida always highlights code blocks.")
				}
			case "highlight_theme":
				theme, _ := value.(string)
				if styles.Registry[theme] != nil {
					c.Set("syntax_theme", theme)
				} else if theme != "" && theme != "css" {
					r.Add(Review, file, 0, "Highlight theme `%s` is not a Chroma style, so Nida's default is used. Set `syntax_theme` to one of Chroma's styles.", theme)
				}
			case "smart_punctuation", "render_emoji", "bottom_footnotes", "highlight_themes_css", "extra_syntaxes_and_themes", "lazy_async_image", "definition_list", "github_alerts":
				if value != false {
					r.Add(Unsupported, file, 0, "`markdown.%s` has no Nida equivalent and was dropped.", key)
				}
			default:
				r.Add(Unsupported, file, 0, "`markdown.%s` has no Nida equivalent and was dropped.", key)
			}
		}
	}

	if v, ok := take("compile_sass"); ok && v == true {
		handled["compile_sass"] = true // planned with the sass/ directory
	}
	if v, ok := take("extra"); ok {
		if extra, ok := v.(map[string]any); ok && len(extra) > 0 {
			for key, value := range extra {
				c.Table("extra").Set(key, value)
			}
			sortTableKeys(c.Table("extra"))
		}
	}
	if v, ok := take("languages"); ok {
		if langs, ok := v.(map[string]any); ok && len(langs) > 0 {
			r.Add(Unsupported, file, 0, "Zola `languages` (%s) are not migrated: Nida builds one language per site. Translated content files such as `page.fr.md` are kept but build as separate pages.", strings.Join(sortedKeys(langs), ", "))
		}
	}
	if v, ok := take("ignored_content"); ok {
		r.Add(Unsupported, file, 0, "`ignored_content` (%v) has no Nida equivalent; matching files were migrated. Delete the ones you do not want.", v)
	}

	for _, key := range sortedKeys(cfg) {
		if handled[key] {
			continue
		}
		switch key {
		case "generate_feed", "feed_limit", "hard_link_static", "check_external_links":
			continue
		case "link_checker", "slugify", "search", "translations":
			r.Add(Unsupported, file, 0, "Zola `[%s]` settings have no Nida equivalent and were dropped.", key)
		default:
			r.Add(Unsupported, file, 0, "Config key `%s` has no Nida equivalent and was dropped.", key)
		}
	}
}

// planSass moves Zola's sass/ directory into Nida's SCSS pipeline. Zola
// compiles sass/foo.scss to /foo.css, but Nida compiles the entry files in
// static/<entry_dir>/ to /<entry_dir>/ and cannot write to the site root,
// so sass/ becomes static/css/ and the stylesheets move to /css/.
func (zolaSource) planSass(p *Plan, cfg map[string]any) {
	if !p.Exists("sass") {
		return
	}
	var entries []string
	_ = p.Walk("sass", func(rel string) error {
		inner := strings.TrimPrefix(rel, "sass/")
		dest := path.Join("static/css", inner)
		if p.Planned(dest) {
			p.Report.Add(Manual, rel, 0, "Not migrated: `%s` already exists. Move the stylesheet by hand.", dest)
			return nil
		}
		p.Copy(dest, rel)
		base := path.Base(inner)
		if !strings.HasPrefix(base, "_") && (strings.HasSuffix(base, ".scss") || strings.HasSuffix(base, ".sass")) {
			if path.Dir(inner) == "." {
				entries = append(entries, base)
			} else {
				p.Report.Add(Manual, rel, 0, "Nida compiles SCSS entry files only at the top of `static/css/`; this one was copied but is not compiled. Move it up or import it from a top-level file.")
			}
		}
		return nil
	})
	if cfg["compile_sass"] != true || len(entries) == 0 {
		return
	}
	scss := p.Config.Table("pipeline.scss")
	scss.Set("enabled", true)
	scss.Set("entry_dir", "css")
	p.Report.Add(Review, "sass", 0, "SCSS compiles through Nida's pipeline, which needs the `sass` command installed (Zola has a built-in compiler).")
	for _, entry := range entries {
		css := strings.TrimSuffix(strings.TrimSuffix(entry, ".scss"), ".sass") + ".css"
		p.Report.Add(Review, "sass/"+entry, 0, "The stylesheet moved from `/%s` to `/css/%s`; update links to it when you port your templates.", css, css)
	}
}

func (z zolaSource) planContent(p *Plan, slugMode string) error {
	// Bundle directories hold an index.md; their other files are page
	// resources and stay beside it.
	bundles := map[string]bool{}
	_ = p.Walk("content", func(rel string) error {
		if path.Base(rel) == "index.md" && path.Dir(rel) != "content" {
			bundles[path.Dir(rel)] = true
		}
		return nil
	})
	templatesUsed := map[string]string{}
	defer func() {
		for _, name := range sortedKeys(templatesUsed) {
			p.Report.Add(Review, templatesUsed[name], 0, "Content asks for template `%s`, which the starter templates do not have, so it renders with the default template until you port it.", name)
		}
	}()

	return p.Walk("content", func(rel string) error {
		if !strings.EqualFold(path.Ext(rel), ".md") {
			if inBundle(rel, bundles, "content") {
				p.Copy(rel, rel)
			} else {
				// Zola serves assets beside section files at the same path;
				// Nida copies only bundle resources, so move them to static/.
				p.Copy(path.Join("static", strings.TrimPrefix(rel, "content/")), rel)
			}
			return nil
		}
		data, err := p.ReadFile(rel)
		if err != nil {
			return err
		}
		doc, err := ParseDocument(data)
		if err != nil {
			p.Report.Add(Manual, rel, 1, "Front matter could not be parsed (%v); the file was kept in `%s/`.", err, preservedDir)
			p.Preserve(rel)
			return nil
		}
		isSection := path.Base(rel) == "_index.md"
		for _, key := range []string{"template", "page_template"} {
			if name := doc.String(key); name != "" {
				if _, seen := templatesUsed[name]; !seen {
					templatesUsed[name] = rel
				}
			}
		}
		if doc.Format == NoFrontMatter {
			stem := strings.TrimSuffix(path.Base(rel), ".md")
			if stem == "index" || stem == "_index" {
				stem = path.Base(path.Dir(rel))
			}
			_ = doc.Set("title", titleFromSlug(stem))
			p.Report.Add(Review, rel, 0, "The file had no front matter, which Nida requires; its title was set from the file name.")
		}
		if isSection {
			z.convertSection(p, doc, rel)
		} else {
			z.convertPage(p, doc, rel)
		}
		z.convertBody(p, doc, rel)
		oldURL := ""
		if !isSection {
			oldURL = z.pageURL(doc, rel, slugMode)
		}
		p.Page(rel, doc, rel, oldURL)
		return nil
	})
}

// inBundle reports whether a directory between rel and stop, such as
// "content", holds a page bundle.
func inBundle(rel string, bundles map[string]bool, stop string) bool {
	for dir := path.Dir(rel); dir != "." && dir != stop; dir = path.Dir(dir) {
		if bundles[dir] {
			return true
		}
	}
	return false
}

var zolaDatePrefix = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})(?:[T ]\d{2}:\d{2}(?::\d{2})?(?:Z|[+-]\d{2}:?\d{2})?)?[_-]`)

func (zolaSource) convertPage(p *Plan, doc *Document, rel string) {
	// Zola reads a date and slug from file names like 2024-05-01-hello.md.
	stem := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	if path.Base(rel) == "index.md" {
		stem = path.Base(path.Dir(rel))
	}
	if m := zolaDatePrefix.FindStringSubmatchIndex(stem); m != nil {
		if !doc.Has("date") {
			_ = doc.Set("date", DateValue(doc, mustDate(stem[m[2]:m[3]]), true))
			p.Report.Add(Changed, rel, 0, "Set `date` from the file name, which Zola reads and Nida does not.")
		}
		if !doc.Has("slug") && !doc.Has("path") {
			_ = doc.Set("slug", stem[m[1]:])
			p.Report.Add(Changed, rel, 0, "Set `slug` without the date prefix, as Zola does.")
		}
	}
	for _, key := range []string{"date", "updated"} {
		NormalizeDate(doc, key, p.Report, rel)
	}
	if v, ok := doc.Get("in_search_index"); ok && v == false {
		p.Report.Add(Unsupported, rel, 0, "`in_search_index = false` has no equivalent; the page is indexed.")
	}
}

func (zolaSource) convertSection(p *Plan, doc *Document, rel string) {
	switch doc.String("sort_by") {
	case "", "date", "title", "weight", "none":
	case "title_bytes":
		_ = doc.Set("sort_by", "title")
		p.Report.Add(Changed, rel, 0, "`sort_by = \"title_bytes\"` became `\"title\"`.")
	case "update_date":
		_ = doc.Set("sort_by", "date")
		p.Report.Add(Review, rel, 0, "`sort_by = \"update_date\"` became `\"date\"`; Nida sorts by publication date.")
	default:
		p.Report.Add(Unsupported, rel, 0, "`sort_by = %q` is not supported; Nida sorts by date.", doc.String("sort_by"))
		doc.Delete("sort_by")
	}
	if v, ok := doc.Get("insert_anchor_links"); ok {
		if s, _ := v.(string); s != "" && s != "none" {
			p.Config.Table("markdown").Set("heading_anchors", true)
			p.Report.Add(Review, rel, 0, "`insert_anchor_links` became the site-wide `markdown.heading_anchors`, which adds a link after each heading.")
		}
		doc.Delete("insert_anchor_links")
	}
	for _, key := range []string{"paginate_reversed", "redirect_to", "render", "in_search_index"} {
		if v, ok := doc.Get(key); ok {
			p.Report.Add(Unsupported, rel, 0, "Section `%s = %v` has no Nida equivalent.", key, v)
		}
	}
	if doc.Has("aliases") {
		p.Report.Add(Unsupported, rel, 0, "Section `aliases` are not supported; add redirects for them on your web host.")
	}
}

var (
	zolaInline    = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\(((?:[^()"']|"[^"]*"|'[^']*'|\([^()]*\))*)\)\s*\}\}`)
	zolaBlockOpen = regexp.MustCompile(`\{%\s*([A-Za-z_][A-Za-z0-9_]*)\(((?:[^()"']|"[^"]*"|'[^']*'|\([^()]*\))*)\)\s*%\}`)
	zolaBlockEnd  = regexp.MustCompile(`\{%\s*end\s*%\}`)
	zolaEscInline = regexp.MustCompile(`\{\{/\*\s*(.*?)\s*\*/\}\}`)
	zolaEscBlock  = regexp.MustCompile(`\{%/\*\s*(.*?)\s*\*/%\}`)
)

// convertBody rewrites Zola shortcodes to Nida's syntax:
// {{ name(a=1) }} becomes {{< name a="1" >}}, and {% name() %}...{% end %}
// becomes a paired shortcode. details blocks are left alone, since Nida
// reads them natively.
func (zolaSource) convertBody(p *Plan, doc *Document, rel string) {
	body := RewriteProse(doc.Body, func(prose string, offset int) string {
		line := func(at int) int { return doc.BodyLine + LineOf(doc.Body, offset+at) - 1 }

		// Escaped shortcodes render as their literal source in Zola. Hide
		// them from the conversion below and restore them unescaped.
		var escaped []string
		hide := func(literal string) string {
			escaped = append(escaped, literal)
			return fmt.Sprintf("\x00ESC%d\x00", len(escaped)-1)
		}
		prose = zolaEscInline.ReplaceAllStringFunc(prose, func(m string) string {
			return hide("{{ " + zolaEscInline.FindStringSubmatch(m)[1] + " }}")
		})
		prose = zolaEscBlock.ReplaceAllStringFunc(prose, func(m string) string {
			return hide("{% " + zolaEscBlock.FindStringSubmatch(m)[1] + " %}")
		})

		var out strings.Builder
		last := 0
		for _, m := range zolaBlockOpen.FindAllStringSubmatchIndex(prose, -1) {
			if m[0] < last {
				continue
			}
			name := prose[m[2]:m[3]]
			if name == "details" {
				continue
			}
			end := zolaBlockEnd.FindStringIndex(prose[m[1]:])
			if end == nil {
				p.Report.Add(Manual, rel, line(m[0]), "Shortcode `%s` has no `{%% end %%}`; it was left as it was.", name)
				continue
			}
			args, ok := zolaArgs(p, prose[m[4]:m[5]], rel, line(m[0]))
			if !ok {
				continue
			}
			out.WriteString(prose[last:m[0]])
			out.WriteString(Shortcode(name, args))
			out.WriteString(prose[m[1] : m[1]+end[0]])
			out.WriteString(ClosingShortcode(name))
			last = m[1] + end[1]
			p.Placeholder(name, rel, line(m[0]))
			p.Report.Add(Changed, rel, 0, "Converted Zola shortcodes to Nida's `{{< name >}}` syntax.")
		}
		out.WriteString(prose[last:])
		prose = out.String()

		out.Reset()
		last = 0
		for _, m := range zolaInline.FindAllStringSubmatchIndex(prose, -1) {
			name := prose[m[2]:m[3]]
			args, ok := zolaArgs(p, prose[m[4]:m[5]], rel, line(m[0]))
			if !ok {
				continue
			}
			out.WriteString(prose[last:m[0]])
			out.WriteString(Shortcode(name, args))
			last = m[1]
			p.Placeholder(name, rel, line(m[0]))
			p.Report.Add(Changed, rel, 0, "Converted Zola shortcodes to Nida's `{{< name >}}` syntax.")
		}
		out.WriteString(prose[last:])
		prose = out.String()

		for i, literal := range escaped {
			prose = strings.Replace(prose, fmt.Sprintf("\x00ESC%d\x00", i), literal, 1)
		}
		return prose
	})
	doc.SetBody(body)
}

// zolaArgs parses Zola keyword arguments: a="x", b=3, c=true, d=[1, 2].
func zolaArgs(p *Plan, raw, file string, line int) ([]Arg, bool) {
	var args []Arg
	rest := strings.TrimSpace(raw)
	for rest != "" {
		eq := strings.IndexByte(rest, '=')
		if eq <= 0 {
			p.Report.Add(Manual, file, line, "Could not read shortcode arguments `%s`; the shortcode was left as it was.", truncate(raw, 60))
			return nil, false
		}
		key := strings.TrimSpace(rest[:eq])
		rest = strings.TrimSpace(rest[eq+1:])
		value, n, ok := zolaValue(rest)
		if !ok {
			p.Report.Add(Manual, file, line, "Could not read shortcode arguments `%s`; the shortcode was left as it was.", truncate(raw, 60))
			return nil, false
		}
		if strings.HasPrefix(rest, "[") {
			p.Report.Add(Review, file, line, "Shortcode argument `%s` is a list; Nida passes it as the text `%s`.", key, value)
		}
		args = append(args, Arg{Key: key, Value: value})
		rest = strings.TrimSpace(rest[n:])
		rest = strings.TrimSpace(strings.TrimPrefix(rest, ","))
	}
	return args, true
}

// zolaValue reads one value and returns it as text with its length.
func zolaValue(s string) (string, int, bool) {
	if s == "" {
		return "", 0, false
	}
	switch q := s[0]; q {
	case '"', '\'', '`':
		end := strings.IndexByte(s[1:], q)
		if end < 0 {
			return "", 0, false
		}
		return s[1 : end+1], end + 2, true
	case '[':
		end := strings.IndexByte(s, ']')
		if end < 0 {
			return "", 0, false
		}
		var items []string
		for _, item := range strings.Split(s[1:end], ",") {
			if item = strings.TrimSpace(item); item != "" {
				items = append(items, strings.Trim(item, "\"'`"))
			}
		}
		return strings.Join(items, ","), end + 1, true
	}
	end := strings.IndexAny(s, ", ")
	if end < 0 {
		end = len(s)
	}
	value := s[:end]
	if _, err := strconv.ParseFloat(value, 64); err != nil && value != "true" && value != "false" {
		return "", 0, false
	}
	return value, end, true
}

// pageURL predicts a page's URL on the Zola site.
func (zolaSource) pageURL(doc *Document, rel, slugMode string) string {
	if p := doc.String("path"); p != "" {
		return "/" + strings.Trim(p, "/") + "/"
	}
	dir := path.Dir(strings.TrimPrefix(rel, "content/"))
	stem := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	if path.Base(rel) == "index.md" {
		stem = path.Base(dir)
		dir = path.Dir(dir)
	}
	slug := doc.String("slug")
	if slug == "" {
		slug = stem
		if m := zolaDatePrefix.FindStringIndex(stem); m != nil {
			slug = stem[m[1]:]
		}
	}
	var parts []string
	if dir != "." {
		for _, seg := range strings.Split(dir, "/") {
			parts = append(parts, zolaSlug(seg, slugMode))
		}
	}
	parts = append(parts, zolaSlug(slug, slugMode))
	return "/" + strings.Join(parts, "/") + "/"
}

// zolaSlug approximates Zola's slugify strategies. "on" also transliterates
// non-ASCII letters, which this does not; the URL check reports pages
// whose URL then differs.
func zolaSlug(s, mode string) string {
	switch mode {
	case "off":
		return s
	case "safe":
		return strings.Map(func(r rune) rune {
			switch {
			case unicode.IsSpace(r):
				return '-'
			case strings.ContainsRune(`<>:"/\|?*#`, r):
				return -1
			}
			return r
		}, strings.TrimSpace(s))
	}
	return content.DeriveSlug(s)
}

func mustDate(s string) time.Time {
	t, _, _ := ParseDate(s)
	return t
}

func stringList(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return x
	}
	return nil
}

func anyList(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case []map[string]any:
		out := make([]any, len(x))
		for i, m := range x {
			out[i] = m
		}
		return out
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func sortTableKeys(t *Table) { slices.Sort(t.keys) }

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
