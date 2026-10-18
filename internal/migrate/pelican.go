package migrate

import (
	"fmt"
	"html"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/content"
	routes "github.com/MohamedElashri/nida/internal/route"
)

// pelicanSource migrates Pelican sites. Pelican's settings are Python, so
// they are read as literals and never executed. Articles become
// content/posts/, pages content/pages/, and URLs are computed from the
// *_SAVE_AS patterns, which name the files Pelican actually wrote.
type pelicanSource struct{}

func (pelicanSource) Name() string  { return "pelican" }
func (pelicanSource) Title() string { return "Pelican" }

func (pelicanSource) Detect(root string) int {
	p := &Plan{Root: root}
	score := 0
	if p.Exists("pelicanconf.py") {
		score += 3
	}
	if p.Exists("publishconf.py") {
		score++
	}
	for _, name := range []string{"tasks.py", "Makefile", "requirements.txt", "pyproject.toml"} {
		if data, err := p.ReadFile(name); err == nil && strings.Contains(strings.ToLower(string(data)), "pelican") {
			score++
		}
	}
	if score == 0 {
		return 0
	}
	if p.Exists("content") {
		score++
	}
	return score
}

const (
	pelicanArticle = "article"
	pelicanPage    = "page"
)

// pelicanSite is the state of one Pelican migration.
type pelicanSite struct {
	p      *Plan
	conf   *pyConfig
	path   string // content directory, source-relative
	loc    *time.Location
	slug   pelicanSlugger
	items  []*pelicanItem
	byRel  map[string]*pelicanItem // source-relative path -> item
	static map[string]string       // content-relative static file -> output path
	dests  map[string]bool
	// attached maps content-relative files moved by {attach} to their
	// output path.
	attached map[string]string
	// unpublished are source files outside every Pelican path.
	unpublished []string
	usesMath    bool
}

// pelicanItem is one article or page.
type pelicanItem struct {
	rel      string // source-relative
	inner    string // relative to the content directory
	kind     string
	format   string
	meta     map[string]any
	order    []string
	body     string
	bodyLine int
	rstNotes []rstNote

	title    string
	slug     string
	category string
	date     time.Time
	dateOnly bool
	draft    bool
	dest     string
	oldURL   string
	// saveAs is the output file path (content-relative URL without the
	// leading slash), which {attach} files are placed next to.
	saveAs string
}

func (it *pelicanItem) set(key string, value any) {
	key = strings.ToLower(strings.TrimSpace(key))
	if _, ok := it.meta[key]; !ok {
		it.order = append(it.order, key)
	}
	it.meta[key] = value
}

func (it *pelicanItem) str(key string) string {
	switch v := it.meta[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func (it *pelicanItem) has(key string) bool {
	_, ok := it.meta[key]
	return ok
}

func (s pelicanSource) Plan(p *Plan) error {
	conf := newPyConfig()
	for _, name := range []string{"pelicanconf.py", "publishconf.py"} {
		data, err := p.ReadFile(name)
		if err != nil {
			continue
		}
		file := readPyConfig(string(data), conf)
		conf.merge(file)
		p.Preserve(name)
	}
	for _, name := range []string{"tasks.py", "Makefile"} {
		if p.Exists(name) {
			p.Preserve(name)
		}
	}
	site := &pelicanSite{
		p: p, conf: conf,
		byRel:    map[string]*pelicanItem{},
		static:   map[string]string{},
		dests:    map[string]bool{},
		attached: map[string]string{},
	}
	site.path = strings.Trim(path.Clean(filepath.ToSlash(site.str("PATH", "content"))), "/")
	if site.path == "" || site.path == "." {
		site.path = "."
	}
	if strings.HasPrefix(site.path, "..") || filepath.IsAbs(site.str("PATH", "content")) {
		p.Report.Add(Manual, "pelicanconf.py", 0, "`PATH` points outside the site (%s); copy the content into the site and migrate again.", site.str("PATH", ""))
		site.path = "content"
	}
	site.setupSlugs()
	site.setupTimezone()
	site.planConfig()
	if err := site.collect(); err != nil {
		return err
	}
	slices.SortFunc(site.items, func(a, b *pelicanItem) int { return strings.Compare(a.rel, b.rel) })
	for _, it := range site.items {
		site.resolve(it)
	}
	site.planStatic()
	site.reportUnpublished()
	site.planItems()
	site.planTheme()
	if site.usesMath {
		p.Config.Table("markdown").Set("math", true)
		p.Report.Add(Review, "", 0, "Content uses math, so `markdown.math` is on. It passes TeX through; load KaTeX or MathJax from your templates (the built-in `nida-scripts` partial does).")
	}
	return nil
}

func (s *pelicanSite) get(key string) (any, bool) {
	v, ok := s.conf.values[key]
	return v, ok
}

// str returns a string setting, or def when it is unset. None is "".
func (s *pelicanSite) str(key, def string) string {
	v, ok := s.conf.values[key]
	if !ok {
		return def
	}
	if _, none := v.(pyNone); none {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return def
}

func (s *pelicanSite) boolean(key string, def bool) bool {
	v, ok := s.conf.values[key]
	if !ok {
		return def
	}
	return pyTruthy(v)
}

func (s *pelicanSite) list(key string, def []string) []string {
	v, ok := s.conf.values[key]
	if !ok {
		return def
	}
	return pyStrings(v)
}

func (s *pelicanSite) setupSlugs() {
	s.slug = pelicanSlugger{
		useUnicode:   s.boolean("SLUGIFY_USE_UNICODE", false),
		preserveCase: s.boolean("SLUGIFY_PRESERVE_CASE", false),
	}
	if s.slug.useUnicode {
		s.slug.subs = unicodePelicanSubs
	}
	v, ok := s.get("SLUG_REGEX_SUBSTITUTIONS")
	if !ok {
		return
	}
	pairs, _ := v.([]any)
	var subs []pelicanSub
	for _, pair := range pairs {
		items, _ := pair.([]any)
		if len(items) != 2 {
			continue
		}
		pattern, repl := pyString(items[0]), pyString(items[1])
		re, err := regexp.Compile(pythonRegex(pattern))
		if err != nil {
			s.p.Report.Add(Review, "pelicanconf.py", 0, "`SLUG_REGEX_SUBSTITUTIONS` pattern %q is not supported by Go regular expressions (%v) and was skipped; check slugs.", pattern, err)
			continue
		}
		subs = append(subs, pelicanSub{re, pythonReplacement(repl)})
	}
	s.slug.subs = subs
}

// pythonRegex adapts Python regular expression syntax for Go.
func pythonRegex(p string) string {
	p = strings.ReplaceAll(p, "(?u)", "")
	p = strings.ReplaceAll(p, `\Z`, `\z`)
	return p
}

func pythonReplacement(r string) string {
	return regexp.MustCompile(`\\(\d)`).ReplaceAllString(r, `$${$1}`)
}

func (s *pelicanSite) setupTimezone() {
	tz := s.str("TIMEZONE", "")
	if tz == "" {
		s.loc = time.UTC
		s.p.Report.Add(Review, "pelicanconf.py", 0, "`TIMEZONE` is not set; Pelican requires it, so dates without an offset were read as UTC.")
		return
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		s.loc = time.UTC
		s.p.Report.Add(Review, "pelicanconf.py", 0, "Time zone %q is not available on this system, so dates without an offset were read as UTC. Check dates near midnight.", tz)
		return
	}
	s.loc = loc
}

// pelicanHandled lists settings the migration reads or deliberately drops
// without a note of their own.
var pelicanHandled = map[string]bool{}

func init() {
	for _, k := range strings.Fields(`PATH ARTICLE_PATHS ARTICLE_EXCLUDES PAGE_PATHS PAGE_EXCLUDES STATIC_PATHS
		STATIC_EXCLUDES STATIC_EXCLUDE_SOURCES EXTRA_PATH_METADATA ARTICLE_URL ARTICLE_SAVE_AS PAGE_URL PAGE_SAVE_AS
		DRAFT_URL DRAFT_SAVE_AS DRAFT_PAGE_URL DRAFT_PAGE_SAVE_AS CATEGORY_URL CATEGORY_SAVE_AS TAG_URL TAG_SAVE_AS
		SITEURL SITENAME SITESUBTITLE AUTHOR DEFAULT_LANG TIMEZONE FEED_ALL_ATOM FEED_ATOM FEED_ALL_RSS FEED_RSS
		FEED_MAX_ITEMS FEED_DOMAIN FEED_APPEND_REF CATEGORY_FEED_ATOM CATEGORY_FEED_RSS TAG_FEED_ATOM TAG_FEED_RSS
		AUTHOR_FEED_ATOM AUTHOR_FEED_RSS TRANSLATION_FEED_ATOM TRANSLATION_FEED_RSS DEFAULT_PAGINATION PAGINATION_PATTERNS
		THEME PLUGINS PLUGIN_PATHS MENUITEMS LINKS SOCIAL DISPLAY_PAGES_ON_MENU DISPLAY_CATEGORIES_ON_MENU DEFAULT_DATE
		SLUGIFY_SOURCE SLUG_REGEX_SUBSTITUTIONS SLUGIFY_USE_UNICODE SLUGIFY_PRESERVE_CASE USE_FOLDER_AS_CATEGORY
		DEFAULT_CATEGORY MARKDOWN WITH_FUTURE_DATES ARTICLE_ORDER_BY PAGE_ORDER_BY OUTPUT_PATH DEFAULT_METADATA
		FILENAME_METADATA PATH_METADATA IGNORE_FILES RELATIVE_URLS LOAD_CONTENT_CACHE CACHE_CONTENT CACHE_PATH
		DELETE_OUTPUT_DIRECTORY OUTPUT_RETENTION AUTHOR_URL AUTHOR_SAVE_AS AUTHORS_SAVE_AS CATEGORIES_SAVE_AS TAGS_SAVE_AS
		ARCHIVES_SAVE_AS YEAR_ARCHIVE_SAVE_AS MONTH_ARCHIVE_SAVE_AS DAY_ARCHIVE_SAVE_AS YEAR_ARCHIVE_URL MONTH_ARCHIVE_URL
		DAY_ARCHIVE_URL DIRECT_TEMPLATES TYPOGRIFY TYPOGRIFY_IGNORE_TAGS TYPOGRIFY_DASHES SUMMARY_MAX_LENGTH
		SUMMARY_MAX_PARAGRAPHS SUMMARY_END_SUFFIX DEFAULT_DATE_FORMAT DATE_FORMATS LOCALE JINJA_ENVIRONMENT JINJA_FILTERS
		JINJA_GLOBALS JINJA_TESTS THEME_STATIC_DIR THEME_STATIC_PATHS THEME_TEMPLATES_OVERRIDES READERS
		PYGMENTS_RST_OPTIONS DOCUTILS_SETTINGS LOG_FILTER ARTICLE_TRANSLATION_ID PAGE_TRANSLATION_ID ARTICLE_LANG_URL
		ARTICLE_LANG_SAVE_AS PAGE_LANG_URL PAGE_LANG_SAVE_AS DRAFT_LANG_URL DRAFT_LANG_SAVE_AS STATIC_URL STATIC_SAVE_AS
		STATIC_CHECK_IF_MODIFIED STATIC_CREATE_LINKS WRITE_SELECTED PORT BIND FORMATTED_FIELDS CSS_FILE
		NEWEST_FIRST_ARCHIVES REVERSE_CATEGORY_ORDER DEFAULT_ORPHANS INTRASITE_LINK_REGEX GZIP_CACHE
		CHECK_MODIFIED_METHOD CONTENT_CACHING_LAYER OUTPUT_SOURCES OUTPUT_SOURCES_EXTENSION DEFAULT_STATUS
		PAGINATED_TEMPLATES PAGINATED_DIRECT_TEMPLATES TEMPLATE_PAGES TEMPLATE_EXTENSIONS DELETE_OUTPUT_DIRECTORY
		ARTICLE_PERMALINK_STRUCTURE FEED_ALL_ATOM_URL FEED_ALL_RSS_URL FEED_ATOM_URL FEED_RSS_URL`) {
		pelicanHandled[k] = true
	}
}

func (s *pelicanSite) planConfig() {
	p, c, r := s.p, s.p.Config, s.p.Report
	const file = "pelicanconf.py"

	if url := strings.TrimRight(s.str("SITEURL", ""), "/"); url != "" {
		c.Set("base_url", url+"/")
	} else {
		r.Add(Manual, file, 0, "`SITEURL` is empty (Pelican uses relative URLs in development); set `base_url` to the published address.")
	}
	if v := s.str("SITENAME", ""); v != "" {
		c.Set("title", v)
	}
	if v := s.str("SITESUBTITLE", ""); v != "" {
		c.Set("description", v)
	}
	if v := s.str("AUTHOR", ""); v != "" {
		c.Set("author", v)
	}
	if v := s.str("DEFAULT_LANG", "en"); v != "" && v != "en" {
		c.Set("language", v)
	}
	if v := strings.Trim(s.str("OUTPUT_PATH", "output"), "/"); v != "" && v != "public" && !strings.HasPrefix(v, "..") && !filepath.IsAbs(v) {
		c.Set("output_dir", v)
	}
	// Pelican publishes future-dated articles unless told not to.
	if s.boolean("WITH_FUTURE_DATES", true) {
		c.Set("future", true)
	}
	if v, ok := s.get("DEFAULT_PAGINATION"); ok {
		if n, isInt := v.(int64); isInt && n > 0 {
			c.Set("paginate", n)
			r.Add(Review, file, 0, "Pagination is %d per page, but later pages move from Pelican's `index2.html` to Nida's `/page/2/`.", n)
		}
	}
	if s.has("PAGINATION_PATTERNS") {
		r.Add(Unsupported, file, 0, "`PAGINATION_PATTERNS` has no equivalent; paginated pages use `/page/N/`.")
	}

	// Feeds. Pelican's own default is FEED_ALL_ATOM = feeds/all.atom.xml.
	feed := c.Table("feed")
	atom := s.str("FEED_ALL_ATOM", "feeds/all.atom.xml")
	if atom == "" {
		atom = s.str("FEED_ATOM", "")
	} else if s.str("FEED_ATOM", "") != "" {
		r.Add(Review, file, 0, "Both `FEED_ALL_ATOM` and `FEED_ATOM` are set; Nida writes one Atom feed, at `%s`.", atom)
	}
	if atom != "" {
		feed.Set("enabled", true)
		feed.Set("filename", strings.TrimPrefix(atom, "/"))
		feed.Set("sections", []any{"posts"})
		s.setFeedLimit(feed)
	} else {
		feed.Set("enabled", false)
	}
	rss := s.str("FEED_ALL_RSS", "")
	if rss == "" {
		rss = s.str("FEED_RSS", "")
	}
	if rss != "" {
		t := c.Table("rss")
		t.Set("enabled", true)
		t.Set("filename", strings.TrimPrefix(rss, "/"))
		t.Set("sections", []any{"posts"})
		s.setFeedLimit(t)
	}
	if s.str("TRANSLATION_FEED_ATOM", "") != "" || s.str("TRANSLATION_FEED_RSS", "") != "" {
		r.Add(Unsupported, file, 0, "Translation feeds have no equivalent; Nida sites have one language.")
	}

	// Taxonomies: Pelican always has tags and categories.
	for _, tax := range []struct{ name, singular, key, def string }{
		{"tags", "Tag", "TAG_SAVE_AS", "tag/{slug}.html"},
		{"categories", "Category", "CATEGORY_SAVE_AS", "category/{slug}.html"},
	} {
		t := c.Taxonomy(tax.name)
		saveAs := s.str(tax.key, tax.def)
		if saveAs == "" {
			t.Set("render", false)
			continue
		}
		t.Set("render", true)
		dir := strings.TrimSuffix(saveAs, "index.html")
		if strings.Count(saveAs, "{slug}") == 1 && strings.HasSuffix(dir, "{slug}/") && !strings.Contains(strings.ReplaceAll(saveAs, "{slug}", ""), "{") {
			c.Table("permalinks").Set(tax.name, "/"+strings.TrimPrefix(dir, "/"))
			continue
		}
		if pattern := strings.TrimSuffix(saveAs, ".html"); strings.HasSuffix(pattern, "{slug}") && !strings.Contains(strings.ReplaceAll(pattern, "{slug}", ""), "{") {
			c.Table("permalinks").Set(tax.name, "/"+strings.TrimPrefix(pattern, "/")+"/")
			old := "/" + strings.ReplaceAll(saveAs, "{slug}", ":slug")
			r.Add(Unsupported, file, 0, "%s pages were `/%s`; Nida writes `/%s/`. Add a redirect on your host, such as Netlify's `%s %s 301`.",
				tax.singular, saveAs, pattern, old, "/"+strings.ReplaceAll(pattern, "{slug}", ":slug")+"/")
			continue
		}
		r.Add(Unsupported, file, 0, "`%s = %q` cannot be reproduced; %s pages use Nida's default URLs.", tax.key, saveAs, tax.name)
	}
	for _, key := range []string{"CATEGORY_FEED_ATOM", "CATEGORY_FEED_RSS", "TAG_FEED_ATOM", "TAG_FEED_RSS", "AUTHOR_FEED_ATOM", "AUTHOR_FEED_RSS"} {
		def := ""
		switch key {
		case "CATEGORY_FEED_ATOM", "AUTHOR_FEED_ATOM":
			def = "feeds/{slug}.atom.xml"
		case "AUTHOR_FEED_RSS":
			def = "feeds/{slug}.rss.xml"
		}
		v := s.str(key, def)
		if v == "" {
			continue
		}
		tax := "tags"
		switch {
		case strings.HasPrefix(key, "AUTHOR"):
			r.Add(Unsupported, file, 0, "`%s` (`%s`): Nida has no author feeds.", key, v)
			continue
		case strings.HasPrefix(key, "CATEGORY"):
			tax = "categories"
		}
		c.Taxonomy(tax).Set("feed", true)
		feedName := "feed.xml"
		if atom != "" {
			feedName = strings.TrimPrefix(atom, "/")
		}
		r.Add(Review, file, 0, "%s feeds move from `/%s` to `%s<slug>/%s`.", map[string]string{"tags": "Tag", "categories": "Category"}[tax], v, s.termPrefix(tax), feedName)
	}
	if s.str("AUTHOR_SAVE_AS", "author/{slug}.html") != "" {
		r.Add(Unsupported, file, 0, "Author pages (`%s`) have no equivalent in Nida.", s.str("AUTHOR_SAVE_AS", "author/{slug}.html"))
	}
	var archives []string
	for _, k := range []string{"ARCHIVES_SAVE_AS", "AUTHORS_SAVE_AS", "CATEGORIES_SAVE_AS", "TAGS_SAVE_AS"} {
		def := map[string]string{"ARCHIVES_SAVE_AS": "archives.html", "AUTHORS_SAVE_AS": "authors.html", "CATEGORIES_SAVE_AS": "categories.html", "TAGS_SAVE_AS": "tags.html"}[k]
		if v := s.str(k, def); v != "" {
			archives = append(archives, "`"+v+"`")
		}
	}
	for _, k := range []string{"YEAR_ARCHIVE_SAVE_AS", "MONTH_ARCHIVE_SAVE_AS", "DAY_ARCHIVE_SAVE_AS"} {
		if v := s.str(k, ""); v != "" {
			archives = append(archives, "`"+v+"`")
		}
	}
	if len(archives) > 0 {
		r.Add(Unsupported, file, 0, "Pelican's index pages %s have no direct equivalent. Nida lists taxonomies at `%s` and `%s`; build an archive page with `groupByMonth` if you need one.", strings.Join(archives, ", "), s.termPrefix("tags"), s.termPrefix("categories"))
	}

	// Home and section listing.
	c.Table("home").Set("sections", []any{"posts"})
	c.Table("home").Set("limit", int64(0))

	// Menus and links for templates.
	for _, k := range []struct{ key, extra string }{{"MENUITEMS", "menu"}, {"LINKS", "links"}, {"SOCIAL", "social"}} {
		if v, ok := s.get(k.key); ok {
			if items := pelicanLinks(v); len(items) > 0 {
				c.Table("extra").Set(k.extra, items)
				r.Add(Review, file, 0, "`%s` is kept as `extra.%s` (a list of name and url); the starter templates do not show it.", k.key, k.extra)
			}
		}
	}
	for _, k := range []string{"DISPLAY_PAGES_ON_MENU", "DISPLAY_CATEGORIES_ON_MENU"} {
		if v, ok := s.get(k); ok {
			c.Table("extra").Set(strings.ToLower(k), tomlSafe(v))
		}
	}

	if v, ok := s.get("PLUGINS"); ok {
		if names := pyStrings(v); len(names) > 0 {
			r.Add(Unsupported, file, 0, "Pelican plugins do not run in Nida: %s.%s", strings.Join(names, ", "), pluginHints(names))
		}
	}
	if v, ok := s.get("MARKDOWN"); ok {
		var exts []string
		if m, ok := v.(map[string]any); ok {
			if cfg, ok := m["extension_configs"].(map[string]any); ok {
				exts = append(exts, sortedKeys(cfg)...)
			}
			exts = append(exts, pyStrings(m["extensions"])...)
		}
		slices.Sort(exts)
		exts = slices.Compact(exts)
		r.Add(Unsupported, file, 0, "The `MARKDOWN` settings do not carry over%s. Nida renders CommonMark with GFM tables, footnotes, heading ids, and highlighting built in; tables of contents come from `.Page.TOC`.", optionalList(exts))
	}
	if s.boolean("TYPOGRIFY", false) {
		r.Add(Unsupported, file, 0, "`TYPOGRIFY` has no equivalent; quotes and dashes are not curled.")
	}
	if s.has("SUMMARY_MAX_LENGTH") {
		r.Add(Unsupported, file, 0, "`SUMMARY_MAX_LENGTH` has no equivalent; Nida summaries are the first paragraph or the text before `<!-- more -->`.")
	}
	if s.has("DEFAULT_DATE_FORMAT") || s.has("DATE_FORMATS") || s.has("LOCALE") {
		r.Add(Review, file, 0, "Date formats (`DEFAULT_DATE_FORMAT`, `DATE_FORMATS`, `LOCALE`) are a template concern; set them with `formatDate` in your templates.")
	}
	switch s.str("ARTICLE_ORDER_BY", "reversed-date") {
	case "reversed-date", "date":
	default:
		r.Add(Review, file, 0, "`ARTICLE_ORDER_BY = %q`: articles are listed newest first.", s.str("ARTICLE_ORDER_BY", ""))
	}

	for _, name := range s.conf.order {
		if reason, bad := s.conf.bad[name]; bad {
			r.Add(Unsupported, file, 0, "`%s` is not a literal value (it %s), so it was not read. Set its equivalent by hand if it matters.", name, reason)
			continue
		}
		if pelicanHandled[name] || name != strings.ToUpper(name) {
			continue
		}
		c.Table("extra").Set(strings.ToLower(name), tomlSafe(s.conf.values[name]))
		r.Add(Unsupported, file, 0, "`%s` has no Nida meaning; it is kept as `extra.%s` for templates.", name, strings.ToLower(name))
	}
	if c.HasTable("extra") {
		sortTableKeys(c.Table("extra"))
	}
	_ = p
}

// termPrefix returns the URL prefix of a taxonomy's term pages, such as
// "/tag/" for a permalink of "/tag/{slug}/".
func (s *pelicanSite) termPrefix(tax string) string {
	if v, ok := s.p.Config.Table("permalinks").Get(tax); ok {
		return strings.TrimSuffix(fmt.Sprint(v), "{slug}/")
	}
	return "/" + tax + "/"
}

func (s *pelicanSite) has(key string) bool {
	_, ok := s.conf.values[key]
	_, bad := s.conf.bad[key]
	return ok || bad
}

// setFeedLimit copies FEED_MAX_ITEMS. Pelican puts every article in its
// feeds by default; Nida needs a limit, so its default of 20 stays.
func (s *pelicanSite) setFeedLimit(t *Table) {
	if v, ok := s.get("FEED_MAX_ITEMS"); ok {
		if n, isInt := v.(int64); isInt && n > 0 {
			t.Set("limit", n)
			return
		}
	}
	s.p.Report.Add(Review, "pelicanconf.py", 0, "Pelican feeds list every article; Nida's list the newest 20. Raise `limit` in `[feed]` if you want more.")
}

func pelicanLinks(v any) []any {
	items, _ := v.([]any)
	var out []any
	for _, item := range items {
		pair, _ := item.([]any)
		if len(pair) < 2 {
			continue
		}
		out = append(out, map[string]any{"name": fmt.Sprint(tomlSafe(pair[0])), "url": fmt.Sprint(tomlSafe(pair[1]))})
	}
	return out
}

func pluginHints(names []string) string {
	hints := map[string]string{
		"sitemap":               "Nida writes sitemap.xml itself",
		"neighbors":             "use `.Page.PrevURL` and `.Page.NextURL`",
		"render_math":           "set `markdown.math` and load KaTeX",
		"related_posts":         "use `.Index.Related`",
		"i18n_subsites":         "Nida sites have one language",
		"pelican-yaml-metadata": "YAML front matter is read natively",
		"yaml_metadata":         "YAML front matter is read natively",
		"tag_cloud":             "build it from `.Index.Taxonomies`",
		"series":                "use `series` front matter and `.Index.Series`",
		"readtime":              "use `.Page.ReadingTime`",
		"post_stats":            "use `.Page.WordCount` and `.Page.ReadingTime`",
		"simple_footnotes":      "use Markdown footnotes",
		"pelican_youtube":       "use the `youtube` shortcode",
		"liquid_tags":           "rewrite the tags as shortcodes",
		"search":                "set `[search] enabled = true`",
		"tipue_search":          "set `[search] enabled = true`",
		"assets":                "use `[pipeline]`",
		"webassets":             "use `[pipeline]`",
	}
	var out []string
	for _, n := range names {
		base := n[strings.LastIndexAny(n, "./")+1:]
		if h, ok := hints[base]; ok {
			out = append(out, fmt.Sprintf("%s: %s", base, h))
		}
	}
	if len(out) == 0 {
		return ""
	}
	return " Equivalents: " + strings.Join(out, "; ") + "."
}

func optionalList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = "`" + s + "`"
	}
	return " (" + strings.Join(quoted, ", ") + ")"
}

// contentRel joins a content-relative path onto the content directory.
func (s *pelicanSite) contentRel(inner string) string {
	if s.path == "." {
		return inner
	}
	return path.Join(s.path, inner)
}

var pelicanExts = map[string]string{
	".md": "md", ".markdown": "md", ".mkd": "md", ".mdown": "md",
	".rst": "rst", ".html": "html", ".htm": "html",
}

var pelicanUnreadable = map[string]string{
	".ipynb": "Jupyter notebooks (pelican-jupyter)", ".org": "Org mode files", ".adoc": "AsciiDoc files", ".asciidoc": "AsciiDoc files",
}

// under reports whether a content-relative path is inside dir, where ""
// or "." is the whole content directory.
func under(inner, dir string) bool {
	dir = strings.Trim(path.Clean(dir), "/")
	if dir == "" || dir == "." {
		return true
	}
	return inner == dir || strings.HasPrefix(inner, dir+"/")
}

func underAny(inner string, dirs []string) bool {
	for _, d := range dirs {
		if under(inner, d) {
			return true
		}
	}
	return false
}

// collect reads every article and page.
func (s *pelicanSite) collect() error {
	articlePaths := s.list("ARTICLE_PATHS", []string{""})
	pagePaths := s.list("PAGE_PATHS", []string{"pages"})
	articleExcludes := append(s.list("ARTICLE_EXCLUDES", nil), pagePaths...)
	pageExcludes := s.list("PAGE_EXCLUDES", nil)
	for _, ap := range articlePaths {
		if ap != "" && ap != "." {
			pageExcludes = append(pageExcludes, ap)
		}
	}
	staticPaths := s.list("STATIC_PATHS", []string{"images"})
	ignore := s.list("IGNORE_FILES", []string{".#*"})
	var unreadable, unpublished []string

	err := s.p.Walk(s.contentDir(), func(rel string) error {
		inner := s.inner(rel)
		base := path.Base(inner)
		for _, pattern := range ignore {
			if ok, _ := path.Match(pattern, base); ok {
				return nil
			}
		}
		ext := strings.ToLower(path.Ext(inner))
		format, readable := pelicanExts[ext]
		if !readable {
			if what, ok := pelicanUnreadable[ext]; ok {
				unreadable = append(unreadable, what)
				s.p.Report.Add(Unsupported, rel, 0, "%s need a Pelican plugin to read and are not migrated.", what)
				return nil
			}
			if !underAny(inner, staticPaths) {
				unpublished = append(unpublished, rel)
			}
			return nil
		}
		kind := ""
		switch {
		case underAny(inner, pagePaths) && !underAnyExcept(inner, pageExcludes, pagePaths):
			kind = pelicanPage
		case underAny(inner, articlePaths) && !underAny(inner, articleExcludes):
			kind = pelicanArticle
		default:
			if !underAny(inner, staticPaths) {
				unpublished = append(unpublished, rel)
			}
			return nil
		}
		item, err := s.read(rel, inner, kind, format)
		if err != nil {
			s.p.Report.Add(Manual, rel, 0, "Could not read this file (%v); it was not migrated.", err)
			return nil
		}
		if item == nil {
			return nil
		}
		s.items = append(s.items, item)
		s.byRel[inner] = item
		return nil
	})
	if err != nil {
		return err
	}
	s.unpublished = unpublished
	return nil
}

// reportUnpublished notes files Pelican ignored, once {attach} links
// have claimed the files they move.
func (s *pelicanSite) reportUnpublished() {
	for _, rel := range s.unpublished {
		if _, attached := s.attached[s.inner(rel)]; attached {
			continue
		}
		s.p.Report.Add(Review, rel, 0, "Pelican does not publish this file (it is outside the article, page, and static paths), so it was not copied.")
	}
}

// underAnyExcept is underAny for excludes, ignoring an exclude that is a
// parent of one of keep (a page path inside the whole-content article
// path).
func underAnyExcept(inner string, excludes, keep []string) bool {
	for _, e := range excludes {
		if under(inner, e) && !slices.Contains(keep, e) {
			return true
		}
	}
	return false
}

func (s *pelicanSite) contentDir() string {
	if s.path == "." {
		return ""
	}
	return s.path
}

func (s *pelicanSite) inner(rel string) string {
	if s.path == "." {
		return rel
	}
	return strings.TrimPrefix(rel, s.path+"/")
}

// read parses one content file into an item. It returns nil for a file
// Pelican would not publish as content, such as an HTML file without a
// title inside a static path.
func (s *pelicanSite) read(rel, inner, kind, format string) (*pelicanItem, error) {
	data, err := s.p.ReadFile(rel)
	if err != nil {
		return nil, err
	}
	it := &pelicanItem{rel: rel, inner: inner, kind: kind, format: format, meta: map[string]any{}, bodyLine: 1}
	text := strings.TrimPrefix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\xef\xbb\xbf")
	switch format {
	case "md":
		s.readMarkdown(it, text)
	case "rst":
		doc := convertRST(text)
		for _, k := range doc.metaOrder {
			it.set(k, doc.meta[k])
		}
		it.body = doc.body
		it.rstNotes = doc.notes
		if doc.usesMath {
			s.usesMath = true
		}
		s.p.Preserve(rel)
	case "html":
		if !s.readHTML(it, text) {
			return nil, nil
		}
	}
	if it.str("title") == "" && underAny(inner, s.list("STATIC_PATHS", []string{"images"})) {
		return nil, nil // Pelican skips it as content; it is a static file
	}
	return it, nil
}

var pelicanMetaLine = regexp.MustCompile(`^ {0,3}([A-Za-z0-9_-]+):\s*(.*)$`)

// readMarkdown reads Python-Markdown "meta" headers, or YAML or TOML front
// matter, and the body.
func (s *pelicanSite) readMarkdown(it *pelicanItem, text string) {
	if strings.HasPrefix(text, "---\n") || strings.HasPrefix(text, "+++\n") {
		if doc, err := ParseDocument([]byte(text)); err == nil && doc.Format != NoFrontMatter {
			for _, k := range doc.Keys() {
				v, _ := doc.Get(k)
				it.set(k, v)
			}
			it.body, it.bodyLine = doc.Body, doc.BodyLine
			return
		}
		// Python-Markdown's meta extension also allows --- delimiters
		// around Key: value lines that are not valid YAML.
		if strings.HasPrefix(text, "---\n") {
			rest := text[4:]
			end := regexp.MustCompile(`(?m)^(---|\.\.\.)\s*$`).FindStringIndex(rest)
			if end != nil {
				s.metaHeaders(it, rest[:end[0]])
				it.body = strings.TrimPrefix(rest[end[1]:], "\n")
				it.bodyLine = strings.Count(text[:4+end[1]], "\n") + 2
				return
			}
		}
	}
	lines := strings.SplitAfter(text, "\n")
	if len(lines) == 0 || !pelicanMetaLine.MatchString(strings.TrimRight(lines[0], "\n")) {
		it.body = text
		return
	}
	n := 0
	for n < len(lines) && strings.TrimSpace(lines[n]) != "" {
		n++
	}
	s.metaHeaders(it, strings.Join(lines[:n], ""))
	body := strings.Join(lines[n:], "")
	it.body = strings.TrimPrefix(body, "\n")
	it.bodyLine = n + 2
}

func (s *pelicanSite) metaHeaders(it *pelicanItem, header string) {
	key := ""
	for _, line := range strings.Split(header, "\n") {
		if m := pelicanMetaLine.FindStringSubmatch(line); m != nil && !strings.HasPrefix(line, "    ") {
			key = strings.ToLower(m[1])
			it.set(key, strings.TrimSpace(m[2]))
			continue
		}
		if key != "" && strings.TrimSpace(line) != "" {
			// Continuation lines join the previous value.
			it.set(key, strings.TrimSpace(it.str(key)+" "+strings.TrimSpace(line)))
		}
	}
}

var (
	htmlTitleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	htmlMetaRe  = regexp.MustCompile(`(?is)<meta\s+([^>]*?)/?>`)
	htmlAttrRe  = regexp.MustCompile(`(?is)(name|content)\s*=\s*("([^"]*)"|'([^']*)')`)
	htmlBodyRe  = regexp.MustCompile(`(?is)<body[^>]*>(.*)</body>`)
)

// readHTML reads Pelican's HTML format: metadata from <title> and <meta>
// tags, content from <body>.
func (s *pelicanSite) readHTML(it *pelicanItem, text string) bool {
	if m := htmlTitleRe.FindStringSubmatch(text); m != nil {
		it.set("title", html.UnescapeString(strings.TrimSpace(m[1])))
	}
	for _, m := range htmlMetaRe.FindAllStringSubmatch(text, -1) {
		var name, value string
		for _, a := range htmlAttrRe.FindAllStringSubmatch(m[1], -1) {
			v := a[3] + a[4]
			if strings.EqualFold(a[1], "name") {
				name = strings.ToLower(v)
			} else {
				value = html.UnescapeString(v)
			}
		}
		if name == "keywords" {
			name = "tags"
		}
		if name != "" {
			it.set(name, value)
		}
	}
	if it.str("title") == "" {
		return false
	}
	if m := htmlBodyRe.FindStringSubmatch(text); m != nil {
		it.body = strings.TrimSpace(m[1]) + "\n"
	} else {
		s.p.Report.Add(Manual, it.rel, 0, "No `<body>` found; the content was not migrated.")
	}
	return true
}

// planItems resolves each item's metadata, destination, and old URL,
// then converts its body and plans it.
func (s *pelicanSite) planItems() {
	if !s.dests["content/_index.md"] {
		s.p.Page("content/_index.md", s.homeDoc(), "", "")
	}
	for _, sec := range []struct{ dir, title, template string }{{"posts", "Posts", ""}, {"pages", "Pages", "page"}} {
		used := false
		for _, it := range s.items {
			if strings.HasPrefix(it.dest, "content/"+sec.dir+"/") {
				used = true
			}
		}
		if !used {
			continue
		}
		doc := NewDocument(TOML, "")
		_ = doc.Set("title", sec.title)
		if sec.template != "" {
			_ = doc.Set("page_template", sec.template)
			_ = doc.Set("sort_by", "title")
		}
		s.p.Page("content/"+sec.dir+"/_index.md", doc, "", "")
	}
	s.planPermalinks()
	for _, it := range s.items {
		doc := s.document(it)
		s.p.Page(it.dest, doc, it.rel, it.oldURL)
		for _, n := range it.rstNotes {
			s.p.Report.Add(n.sev, it.rel, n.line, "%s", n.message)
		}
		if it.format == "rst" {
			s.p.Report.Add(Review, it.rel, 0, "Converted from reStructuredText. Compare the result with the original in `%s/`.", preservedDir)
		}
	}
}

func (s *pelicanSite) homeDoc() *Document {
	doc := NewDocument(TOML, "")
	title := s.str("SITENAME", "Home")
	if title == "" {
		title = "Home"
	}
	_ = doc.Set("title", title)
	return doc
}

var pelicanFilenameDefault = `(?P<date>\d{4}-\d{2}-\d{2}).*`

// resolve fills in an item's title, slug, date, category, status,
// destination, and old URL, following Pelican's rules.
func (s *pelicanSite) resolve(it *pelicanItem) {
	r := s.p.Report
	stem := strings.TrimSuffix(path.Base(it.inner), path.Ext(it.inner))

	// Metadata from the file name and path, then EXTRA_PATH_METADATA and
	// DEFAULT_METADATA, each only where the file sets nothing.
	s.regexMetadata(it, s.str("FILENAME_METADATA", pelicanFilenameDefault), stem)
	s.regexMetadata(it, s.str("PATH_METADATA", ""), it.inner)
	if extra, ok := s.get("EXTRA_PATH_METADATA"); ok {
		if m, ok := extra.(map[string]any); ok {
			if values, ok := m[it.inner].(map[string]any); ok {
				for k, v := range values {
					if !it.has(strings.ToLower(k)) {
						it.set(k, tomlSafe(v))
					}
				}
			}
		}
	}
	if def, ok := s.get("DEFAULT_METADATA"); ok {
		if m, ok := def.(map[string]any); ok {
			for _, k := range sortedKeys(m) {
				if !it.has(strings.ToLower(k)) {
					it.set(k, tomlSafe(m[k]))
				}
			}
		}
	}

	it.title = it.str("title")
	if it.title == "" {
		it.title = titleFromSlug(stem)
		it.draft = true
		r.Add(Review, it.rel, 0, "Pelican skips files without a title, so this one became a draft titled %q. Publish it or delete it.", it.title)
	}

	switch status := strings.ToLower(it.str("status")); status {
	case "", "published":
		if status == "" && strings.ToLower(s.str("DEFAULT_STATUS", "published")) == "draft" {
			it.draft = true
		}
	case "draft":
		it.draft = true
	case "hidden":
		r.Add(Review, it.rel, 0, "`Status: hidden` has no equivalent; the page is published and listed in its section.")
	default:
		r.Add(Review, it.rel, 0, "Unknown status %q was ignored.", status)
	}

	// Slug: metadata, or slugified from the title (SLUGIFY_SOURCE).
	if slug := it.str("slug"); slug != "" {
		it.slug = slug
	} else {
		source := it.title
		if s.str("SLUGIFY_SOURCE", "title") == "basename" {
			source = stem
		}
		slug, exact := s.slug.slugify(source)
		it.slug = slug
		if !exact {
			r.Add(Review, it.rel, 0, "The slug %q was transliterated approximately from %q; Pelican's unidecode may have produced a different URL. Compare with the old site.", slug, source)
		}
	}
	if it.slug == "" {
		it.slug = content.DeriveSlug(stem)
	}

	// Date, in TIMEZONE when it has no offset.
	if raw, ok := it.meta["date"]; ok {
		if t, dateOnly, err := s.parseDate(raw); err == nil {
			it.date, it.dateOnly = t, dateOnly
		} else {
			r.Add(Manual, it.rel, 0, "Date %v could not be read; set `date` by hand.", raw)
		}
	} else if it.kind == pelicanArticle {
		s.defaultDate(it)
	}
	if it.kind == pelicanArticle && !it.date.IsZero() && !s.boolean("WITH_FUTURE_DATES", true) && it.date.After(time.Now()) {
		it.draft = true
		r.Add(Review, it.rel, 0, "Future-dated, and `WITH_FUTURE_DATES = False`, so Pelican treated it as a draft; so does the migrated site.")
	}

	if it.kind == pelicanArticle {
		it.category = it.str("category")
		if it.category == "" && s.boolean("USE_FOLDER_AS_CATEGORY", true) {
			if dir := path.Base(path.Dir(it.inner)); path.Dir(it.inner) != "." {
				it.category = dir
			}
		}
		if it.category == "" {
			it.category = s.str("DEFAULT_CATEGORY", "misc")
		}
	}
	if lang := it.str("lang"); lang != "" && lang != s.str("DEFAULT_LANG", "en") {
		r.Add(Unsupported, it.rel, 0, "This is a `%s` translation; Nida sites have one language, so it became a separate page.", lang)
	}
	if it.has("translation") {
		r.Add(Unsupported, it.rel, 0, "`Translation: %s` has no equivalent; it is kept in `extra`.", it.str("translation"))
	}

	s.oldURL(it)
	s.destination(it)
}

func (s *pelicanSite) regexMetadata(it *pelicanItem, pattern, subject string) {
	if pattern == "" {
		return
	}
	re, err := regexp.Compile(pythonRegex(pattern))
	if err != nil {
		s.p.Report.Add(Review, "pelicanconf.py", 0, "Metadata pattern %q is not supported by Go regular expressions; it was not applied.", pattern)
		return
	}
	m := re.FindStringSubmatch(subject)
	if m == nil || re.FindStringIndex(subject)[0] != 0 {
		return // Python's re.match anchors at the start
	}
	for i, name := range re.SubexpNames() {
		if name != "" && m[i] != "" && !it.has(name) {
			it.set(name, m[i])
		}
	}
}

func (s *pelicanSite) parseDate(raw any) (time.Time, bool, error) {
	if t, ok := raw.(time.Time); ok {
		return t, false, nil
	}
	str := strings.TrimSpace(fmt.Sprint(raw))
	t, dateOnly, err := ParseDate(str)
	if err != nil {
		return t, false, err
	}
	if !hasOffset(str) {
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), s.loc)
	}
	return t, dateOnly, nil
}

var offsetRe = regexp.MustCompile(`(Z|[+-]\d{2}:?\d{2}|[A-Z]{3,4})$`)

func hasOffset(s string) bool {
	return offsetRe.MatchString(strings.TrimSpace(s))
}

func (s *pelicanSite) defaultDate(it *pelicanItem) {
	v, ok := s.get("DEFAULT_DATE")
	switch {
	case !ok:
		it.draft = true
		s.p.Report.Add(Review, it.rel, 0, "The article has no date, which Pelican requires, so it became a draft. Add a `date`.")
	case pyString(v) == "fs":
		if info, err := os.Stat(filepath.Join(s.p.Root, filepath.FromSlash(it.rel))); err == nil {
			it.date = info.ModTime().In(s.loc).Truncate(time.Second)
			s.p.Report.Add(Review, it.rel, 0, "`DEFAULT_DATE = 'fs'` dated this article by its file modification time, which changes on checkout; the date is now fixed in front matter.")
		}
	default:
		if parts, ok := v.([]any); ok && len(parts) >= 3 {
			var n [6]int
			for i := 0; i < len(parts) && i < 6; i++ {
				if x, ok := parts[i].(int64); ok {
					n[i] = int(x)
				}
			}
			it.date = time.Date(n[0], time.Month(n[1]), n[2], n[3], n[4], n[5], 0, s.loc)
			it.dateOnly = len(parts) == 3
		}
	}
}

// oldURL computes the URL Pelican wrote the item to, from save_as
// metadata or the *_SAVE_AS setting.
func (s *pelicanSite) oldURL(it *pelicanItem) {
	if it.draft {
		return // drafts were not published at a stable URL
	}
	saveAs, explicit := "", false
	if v, ok := it.meta["save_as"]; ok {
		saveAs, explicit = strings.TrimSpace(fmt.Sprint(tomlSafe(v))), true
		if saveAs == "" {
			s.p.Report.Add(Review, it.rel, 0, "`save_as` is empty, so Pelican did not write this page; it is published in Nida. Make it a draft if it should stay unpublished.")
			return
		}
	}
	key, def := "ARTICLE_SAVE_AS", "{slug}.html"
	urlKey, urlDef := "ARTICLE_URL", "{slug}.html"
	if it.kind == pelicanPage {
		key, def, urlKey, urlDef = "PAGE_SAVE_AS", "pages/{slug}.html", "PAGE_URL", "pages/{slug}.html"
	}
	if !explicit {
		saveAs = s.str(key, def)
		if saveAs == "" {
			s.p.Report.Add(Review, it.rel, 0, "`%s` is empty, so Pelican did not write this %s; Nida publishes it.", key, it.kind)
			return
		}
	}
	fields := s.urlFields(it)
	out, err := pelicanFormat(saveAs, fields, it.date)
	if err != nil {
		s.p.Report.Add(Manual, it.rel, 0, "Could not compute the old URL from `%s` (%v); check this page's URL.", saveAs, err)
		return
	}
	it.saveAs = strings.TrimPrefix(out, "/")
	it.oldURL = "/" + it.saveAs
	if !explicit && !it.has("url") {
		if url, err := pelicanFormat(s.str(urlKey, urlDef), fields, it.date); err == nil && !sameTarget(url, it.saveAs) {
			s.p.Report.Add(Review, it.rel, 0, "Pelican linked to `/%s` but wrote the file to `/%s` (`%s` and `%s` differ); the migrated page is at the file's location.", strings.TrimPrefix(url, "/"), it.saveAs, urlKey, key)
		}
	}
}

func sameTarget(url, saveAs string) bool {
	url = strings.TrimPrefix(url, "/")
	return url == saveAs || url+"index.html" == saveAs || url+"/index.html" == saveAs || url+".html" == saveAs
}

func (s *pelicanSite) urlFields(it *pelicanItem) map[string]string {
	fields := map[string]string{}
	for k := range it.meta {
		fields[k] = it.str(k)
	}
	fields["slug"] = it.slug
	lang := it.str("lang")
	if lang == "" {
		lang = s.str("DEFAULT_LANG", "en")
	}
	fields["lang"] = lang
	if it.category != "" {
		fields["category"], _ = s.slug.slugify(it.category)
	}
	authors := s.authors(it)
	if len(authors) > 0 {
		fields["author"], _ = s.slug.slugify(authors[0])
	}
	fields["path"] = it.inner
	fields["filename"] = path.Base(it.inner)
	return fields
}

func (s *pelicanSite) authors(it *pelicanItem) []string {
	if it.has("authors") {
		return pelicanList(it.meta["authors"])
	}
	if a := it.str("author"); a != "" {
		return []string{a}
	}
	if a := s.str("AUTHOR", ""); a != "" {
		return []string{a}
	}
	return nil
}

// pelicanList splits a multi-value metadata field: on semicolons if there
// are any, otherwise on commas, as Pelican does.
func pelicanList(v any) []string {
	switch x := v.(type) {
	case []any:
		var out []string
		for _, item := range x {
			out = append(out, strings.TrimSpace(fmt.Sprint(item)))
		}
		return out
	case nil:
		return nil
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	sep := ","
	if strings.Contains(s, ";") {
		sep = ";"
	}
	var out []string
	for _, part := range strings.Split(s, sep) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// destination picks the item's content path from its slug.
func (s *pelicanSite) destination(it *pelicanItem) {
	if it.kind == pelicanPage && it.oldURL != "" {
		if u, err := cleanOldURL(it.oldURL); err == nil && u == "/" && !s.dests["content/_index.md"] {
			// A page saved as index.html replaces Pelican's home page.
			it.dest = "content/_index.md"
			s.dests[it.dest] = true
			it.oldURL = ""
			s.p.Report.Add(Changed, it.rel, 0, "This page was saved as the home page, so it became the home section `content/_index.md`.")
			return
		}
	}
	dir := "content/posts/"
	if it.kind == pelicanPage {
		dir = "content/pages/"
	}
	stem := it.slug
	if routes.ValidateComponent("slug", stem) != nil || strings.ContainsAny(stem, " ") {
		stem = content.DeriveSlug(stem)
	}
	if stem == "" || stem == "index" || stem == "_index" {
		stem = content.DeriveSlug(it.title) + "-page"
	}
	dest := dir + stem + ".md"
	for n := 2; s.dests[dest]; n++ {
		dest = fmt.Sprintf("%s%s-%d.md", dir, stem, n)
	}
	if dest != dir+stem+".md" {
		s.p.Report.Add(Review, it.rel, 0, "Another %s has the same slug %q, so this one was written to `%s`.", it.kind, it.slug, dest)
	}
	s.dests[dest] = true
	it.dest = dest
}

// planPermalinks maps the *_SAVE_AS patterns onto Nida permalinks, so
// most pages keep their URL without a per-page path.
func (s *pelicanSite) planPermalinks() {
	for _, sec := range []struct{ dir, key, def string }{{"posts", "ARTICLE_SAVE_AS", "{slug}.html"}, {"pages", "PAGE_SAVE_AS", "pages/{slug}.html"}} {
		pattern := s.str(sec.key, sec.def)
		if pattern == "" {
			continue
		}
		if link, ok := pelicanPermalink(pattern, true); ok {
			s.p.Config.Table("permalinks").Set(sec.dir, link)
		} else {
			s.p.Report.Add(Review, "pelicanconf.py", 0, "`%s = %q` has fields Nida permalinks lack, so each %s gets a `path` to keep its URL.", sec.key, pattern, strings.TrimSuffix(sec.dir, "s"))
		}
	}
}

// document builds the item's Markdown file.
func (s *pelicanSite) document(it *pelicanItem) *Document {
	body := it.body
	if it.format == "md" {
		body = s.pythonMarkdown(it, body)
	}
	body = strings.TrimLeft(s.convertLinks(it, body), "\n")
	doc := NewDocument(YAML, body)
	set := func(k string, v any) { _ = doc.Set(k, v) }
	set("title", it.title)
	if !it.date.IsZero() {
		set("date", s.dateString(it.date, it.dateOnly))
	}
	if raw, ok := it.meta["modified"]; ok {
		if t, dateOnly, err := s.parseDate(raw); err == nil {
			set("updated", s.dateString(t, dateOnly))
		} else {
			s.p.Report.Add(Manual, it.rel, 0, "`Modified: %v` could not be read.", raw)
		}
	}
	stem := strings.TrimSuffix(path.Base(it.dest), ".md")
	if it.dest == "content/_index.md" {
		stem = ""
	}
	if stem != "" && content.DeriveSlug(stem) != it.slug && routes.ValidateComponent("slug", it.slug) == nil {
		set("slug", it.slug)
	}
	if v := it.str("summary"); v != "" {
		set("description", v)
	} else if v := it.str("description"); v != "" {
		set("description", v)
	}
	if it.draft {
		set("draft", true)
	}
	if t := it.str("template"); t != "" {
		set("template", t)
		s.p.Report.Add(Review, it.rel, 0, "Template `%s` is a Jinja template; the page renders with the default template until you port it.", t)
	}
	if it.has("tags") {
		var tags []any
		for _, t := range pelicanList(it.meta["tags"]) {
			tags = append(tags, t)
		}
		if len(tags) > 0 {
			set("tags", tags)
		}
	}
	if it.kind == pelicanArticle && it.category != "" {
		set("categories", []any{it.category})
	}
	if authors := pelicanList(it.meta["authors"]); len(authors) > 0 || it.str("author") != "" {
		if len(authors) == 0 {
			authors = []string{it.str("author")}
		}
		var list []any
		for _, a := range authors {
			list = append(list, a)
		}
		set("authors", list)
	}
	skip := map[string]bool{
		"title": true, "date": true, "modified": true, "slug": true, "summary": true, "description": true,
		"status": true, "template": true, "tags": true, "category": true, "author": true, "authors": true,
		"save_as": true, "url": true, "lang": true, "keywords": true,
	}
	for _, k := range it.order {
		if skip[k] {
			continue
		}
		if k == "translation" {
			set("translation", it.str(k))
			continue
		}
		set(k, tomlSafe(it.meta[k]))
	}
	return doc
}

func (s *pelicanSite) dateString(t time.Time, dateOnly bool) string {
	if dateOnly {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339)
}

var (
	pmdIndentedLang = regexp.MustCompile(`^    (:::|#!)\s*([\w+#.-]+)\s*$`)
	pmdFenceAttr    = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})\\s*\\{\\s*\\.([\\w+#-]+)\\s*([^}]*)\\}\\s*$")
	pmdFenceDot     = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})\\s*\\.([\\w+#-]+)\\s*$")
	pmdHeadingAttr  = regexp.MustCompile(`^(#{1,6} .*?)\s*\{:\s*([^}]*)\}\s*$`)
	pmdBlockAttr    = regexp.MustCompile(`^\{:\s*[^}]*\}\s*$`)
	pmdInlineAttr   = regexp.MustCompile(`([)\]*_])\{:\s*[^}]*\}`)
	pmdAbbr         = regexp.MustCompile(`^\*\[[^\]]+\]:`)
	pmdTOC          = regexp.MustCompile(`^\s*\[TOC\]\s*$`)
	pmdDefinition   = regexp.MustCompile(`^:\s{1,3}\S`)
)

// pythonMarkdown rewrites Python-Markdown extensions that Pelican enables
// by default (codehilite, extra, toc) into CommonMark Nida reads.
func (s *pelicanSite) pythonMarkdown(it *pelicanItem, body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	inFence := ""
	prevBlank := true
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		num := it.bodyLine + i
		if inFence != "" {
			out = append(out, line)
			if strings.HasPrefix(strings.TrimSpace(line), inFence) {
				inFence = ""
			}
			prevBlank = false
			continue
		}
		if m := pmdFenceAttr.FindStringSubmatch(line); m != nil {
			info := m[3]
			if extra := strings.TrimSpace(m[4]); extra != "" {
				info += " {" + strings.ReplaceAll(extra, " ", ", ") + "}"
			}
			out = append(out, m[1]+m[2]+info)
			inFence = m[2][:3]
			s.p.Report.Add(Changed, it.rel, 0, "Rewrote Python-Markdown code fence attributes (`{.lang}`) as plain info strings.")
			continue
		}
		if m := pmdFenceDot.FindStringSubmatch(line); m != nil {
			out = append(out, m[1]+m[2]+m[3])
			inFence = m[2][:3]
			continue
		}
		if fenceOpen.MatchString(line) {
			inFence = strings.TrimLeft(line, " ")[:3]
			out = append(out, line)
			continue
		}
		// codehilite: an indented code block starting with :::lang or
		// #!lang names its language.
		if prevBlank {
			if m := pmdIndentedLang.FindStringSubmatch(line); m != nil {
				info := m[2]
				if m[1] == "#!" {
					info += " {linenos}"
				}
				var code []string
				j := i + 1
				for j < len(lines) && (strings.HasPrefix(lines[j], "    ") || strings.TrimSpace(lines[j]) == "") {
					code = append(code, strings.TrimPrefix(lines[j], "    "))
					j++
				}
				for len(code) > 0 && strings.TrimSpace(code[len(code)-1]) == "" {
					code = code[:len(code)-1]
				}
				out = append(out, fence(info, code)...)
				out = append(out, "")
				s.p.Report.Add(Changed, it.rel, 0, "Converted codehilite `:::lang` code blocks to fenced code.")
				i = j - 1
				prevBlank = true
				continue
			}
		}
		// Skip ordinary indented code blocks untouched.
		if prevBlank && strings.HasPrefix(line, "    ") {
			for i < len(lines) && (strings.HasPrefix(lines[i], "    ") || strings.TrimSpace(lines[i]) == "") {
				out = append(out, lines[i])
				i++
			}
			i--
			prevBlank = strings.TrimSpace(lines[i]) == ""
			continue
		}
		removed := true
		switch {
		case pmdTOC.MatchString(line):
			s.p.Report.Add(Review, it.rel, num, "The `[TOC]` marker was removed; build a table of contents from `.Page.TOC` in the template.")
		case pmdAbbr.MatchString(line):
			s.p.Report.Add(Review, it.rel, num, "Abbreviation definitions (`*[ABBR]: ...`) are not supported and were removed.")
		case pmdBlockAttr.MatchString(line):
			s.p.Report.Add(Review, it.rel, num, "Attribute list `%s` is not supported and was removed.", strings.TrimSpace(line))
		default:
			removed = false
		}
		if removed {
			// Do not leave two blank lines where the line was.
			if (len(out) == 0 || strings.TrimSpace(out[len(out)-1]) == "") && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				i++
			}
			continue
		}
		if m := pmdHeadingAttr.FindStringSubmatch(line); m != nil {
			line = m[1] + " {" + strings.TrimSpace(m[2]) + "}"
		}
		if pmdInlineAttr.MatchString(line) {
			s.p.Report.Add(Review, it.rel, num, "Inline attribute lists (`{: ...}`) are not supported and were removed.")
			line = pmdInlineAttr.ReplaceAllString(line, "$1")
		}
		if pmdDefinition.MatchString(line) && i > 0 && strings.TrimSpace(lines[i-1]) != "" {
			s.p.Report.Add(Review, it.rel, num, "Definition lists are not supported; the term and `: definition` lines render as a paragraph.")
		}
		if strings.Contains(line, `markdown="1"`) || strings.Contains(line, `markdown="block"`) || strings.Contains(line, `markdown="span"`) {
			s.p.Report.Add(Review, it.rel, num, "Markdown inside HTML (`markdown=\"1\"`) is not processed by Nida; the HTML block renders as written.")
		}
		out = append(out, line)
		prevBlank = strings.TrimSpace(line) == ""
	}
	return strings.Join(out, "\n")
}

var pelicanLinkRe = regexp.MustCompile(`[{|](filename|static|attach|tag|category|author|index)[}|]([^\s"'<>)\]#?]*)([#?][^\s"'<>)\]]*)?`)

// convertLinks rewrites Pelican's intrasite link syntax: {filename} links
// become @/ links to the migrated file, {static} and {attach} links become
// the file's URL, and {tag} and {category} links point at Nida's term pages.
func (s *pelicanSite) convertLinks(it *pelicanItem, body string) string {
	return RewriteProse(body, func(prose string, offset int) string {
		return pelicanLinkRe.ReplaceAllStringFunc(prose, func(m string) string {
			sub := pelicanLinkRe.FindStringSubmatch(m)
			kind, target, suffix := sub[1], sub[2], sub[3]
			line := it.bodyLine
			if i := strings.Index(prose, m); i >= 0 {
				line += LineOf(body, offset+i) - 1
			}
			switch kind {
			case "index":
				return "/" + suffix
			case "tag", "category":
				tax := "tags"
				if kind == "category" {
					tax = "categories"
				}
				s.p.Report.Add(Review, it.rel, line, "`{%s}` links now point at Nida's %s pages.", kind, tax)
				return s.termPrefix(tax) + content.DeriveSlug(target) + "/" + suffix
			case "author":
				s.p.Report.Add(Manual, it.rel, line, "`{author}%s` links to an author page, which Nida does not have; it now links to the home page.", target)
				return "/" + suffix
			}
			resolved := s.resolveLink(it, target)
			if kind == "filename" {
				if item, ok := s.byRel[resolved]; ok && item.dest != "" {
					return "@/" + strings.TrimPrefix(item.dest, "content/") + suffix
				}
			}
			if kind == "attach" {
				if out, ok := s.attached[resolved]; ok {
					return "/" + out + suffix
				}
			}
			if out, ok := s.static[resolved]; ok {
				return "/" + out + suffix
			}
			s.p.Report.Add(Manual, it.rel, line, "`{%s}%s` points at a file that was not found; the link was left as it was.", kind, target)
			return m
		})
	})
}

// resolveLink resolves a link target relative to the linking file, or to
// the content root when it starts with "/".
func (s *pelicanSite) resolveLink(it *pelicanItem, target string) string {
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(path.Clean(target), "/")
	}
	return path.Clean(path.Join(path.Dir(it.inner), target))
}

// planStatic copies STATIC_PATHS and plans {attach} files. It runs after
// items are collected but before their links are converted.
func (s *pelicanSite) planStatic() {
	staticPaths := s.list("STATIC_PATHS", []string{"images"})
	excludes := s.list("STATIC_EXCLUDES", nil)
	extra := map[string]string{}
	if v, ok := s.get("EXTRA_PATH_METADATA"); ok {
		if m, ok := v.(map[string]any); ok {
			for k, meta := range m {
				if mm, ok := meta.(map[string]any); ok {
					if out := pyString(mm["path"]); out != "" {
						extra[strings.Trim(k, "/")] = strings.TrimPrefix(out, "/")
					}
				}
			}
		}
	}

	// {attach} files move next to the article that links them.
	for _, it := range s.items {
		for _, m := range pelicanLinkRe.FindAllStringSubmatch(it.body, -1) {
			if m[1] != "attach" {
				continue
			}
			src := s.resolveLink(it, m[2])
			if _, done := s.attached[src]; done || !s.p.Exists(s.contentRel(src)) {
				continue
			}
			rel, err := filepath.Rel(path.Dir(it.inner), src)
			rel = filepath.ToSlash(rel)
			if err != nil || strings.HasPrefix(rel, "..") || it.saveAs == "" {
				continue // falls back to the file's static location
			}
			out := path.Join(path.Dir(it.saveAs), rel)
			s.attached[src] = out
			s.p.Copy(path.Join("static", out), s.contentRel(src))
		}
	}

	seen := map[string]bool{}
	for _, sp := range staticPaths {
		sp = strings.Trim(path.Clean(sp), "/")
		walkRoot := s.contentRel(sp)
		if sp == "." || sp == "" {
			walkRoot = s.contentDir()
		}
		info, err := os.Stat(filepath.Join(s.p.Root, filepath.FromSlash(walkRoot)))
		if err != nil {
			continue
		}
		visit := func(rel string) error {
			inner := s.inner(rel)
			if seen[inner] || underAny(inner, excludes) {
				return nil
			}
			seen[inner] = true
			if _, isItem := s.byRel[inner]; isItem && s.boolean("STATIC_EXCLUDE_SOURCES", true) {
				return nil
			}
			if _, moved := s.attached[inner]; moved {
				return nil
			}
			out := inner
			if o, ok := extra[inner]; ok {
				out = o
			} else {
				for dir, o := range extra {
					if strings.HasPrefix(inner, dir+"/") {
						out = path.Join(o, strings.TrimPrefix(inner, dir+"/"))
					}
				}
			}
			s.static[inner] = out
			switch out {
			case "robots.txt":
				s.p.Config.Table("robots").Set("enabled", false)
				s.p.Report.Add(Changed, rel, 0, "The site has its own robots.txt, so Nida's generated one is off.")
			case "sitemap.xml":
				s.p.Config.Table("sitemap").Set("enabled", false)
				s.p.Report.Add(Review, rel, 0, "The site has a static sitemap.xml, so Nida's generated sitemap is off. Consider deleting the static one.")
			}
			s.p.Copy(path.Join("static", out), rel)
			return nil
		}
		if info.IsDir() {
			if err := s.p.Walk(walkRoot, visit); err != nil {
				continue
			}
		} else {
			_ = visit(walkRoot)
		}
	}
}

// planTheme keeps a local Jinja theme for reference.
func (s *pelicanSite) planTheme() {
	theme := s.str("THEME", "")
	dirs := s.list("THEME_TEMPLATES_OVERRIDES", nil)
	if theme != "" && !filepath.IsAbs(theme) && !strings.HasPrefix(path.Clean(theme), "..") && s.p.Exists(theme) {
		dirs = append(dirs, theme)
	} else if theme != "" {
		s.p.Report.Add(Unsupported, "pelicanconf.py", 0, "Theme `%s` is a Jinja theme installed outside the site; the migrated site uses Nida's starter templates.", theme)
	}
	for _, dir := range dirs {
		found := false
		_ = s.p.Walk(strings.Trim(path.Clean(dir), "/"), func(rel string) error {
			found = true
			s.p.Preserve(rel)
			return nil
		})
		if found {
			s.p.Report.Add(Unsupported, dir, 0, "Jinja templates in `%s/` are not translated; the site uses Nida's starter templates. The originals are in `%s/%s/`.", dir, preservedDir, dir)
		}
	}
}
