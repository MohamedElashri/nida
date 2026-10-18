package migrate

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	yaml "go.yaml.in/yaml/v3"

	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/frontmatter"
)

// jekyllSource migrates Jekyll sites. Jekyll differs from Nida more than
// Zola does: posts take their dates from file names, any directory can
// hold pages, Liquid runs over content, and URLs usually end in .html.
type jekyllSource struct{}

func (jekyllSource) Name() string  { return "jekyll" }
func (jekyllSource) Title() string { return "Jekyll" }

func (jekyllSource) Detect(root string) int {
	p := &Plan{Root: root}
	score := 0
	if jekyllConfigFile(p) != "" {
		score += 2
	}
	if p.Exists("_posts") {
		score += 2
	}
	for _, dir := range []string{"_layouts", "_includes", "_drafts"} {
		if p.Exists(dir) {
			score++
		}
	}
	if data, err := p.ReadFile("Gemfile"); err == nil && (bytes.Contains(data, []byte("jekyll")) || bytes.Contains(data, []byte("github-pages"))) {
		score++
	}
	return score
}

func jekyllConfigFile(p *Plan) string {
	for _, name := range []string{"_config.yml", "_config.yaml", "_config.toml"} {
		if p.Exists(name) {
			return name
		}
	}
	return ""
}

// jekyllDefaultExcludes is Jekyll 4's default exclude list.
var jekyllDefaultExcludes = []string{".sass-cache", ".jekyll-cache", "gemfiles", "Gemfile", "Gemfile.lock", "node_modules", "vendor/bundle/", "vendor/cache/", "vendor/gems/", "vendor/ruby/"}

// jekyllContentExts are the extensions Jekyll converts as Markdown, plus
// HTML, which Jekyll renders through Liquid and layouts.
var jekyllMarkdownExts = []string{".md", ".markdown", ".mkd", ".mkdn", ".mdown", ".mdwn", ".mkdown", ".mdtxt", ".mdtext", ".text"}

var jekyllDateName = regexp.MustCompile(`^(\d{2,4}-\d{1,2}-\d{1,2})-(.*)$`)

type jekyllCollection struct {
	label     string
	output    bool
	permalink string
}

type jekyllDefault struct {
	path   string
	typ    string
	values map[string]any
}

type jekyllSite struct {
	p              *Plan
	cfg            map[string]any
	permalink      string
	tz             *time.Location
	collectionsDir string
	collections    map[string]*jekyllCollection
	excludes       []string
	includes       []string
	defaults       []jekyllDefault
	excerptSep     string
	plugins        map[string]bool
	sassDir        string
	dataDir        string
	ghPages        bool

	docs        []*jekyllDoc
	byOrigin    map[string]*jekyllDoc
	staticPaths map[string]bool // source paths copied to static/
	usedDests   map[string]bool
	layouts     map[string]string // layout name -> first file using it
	sassEntries []string          // source paths of SCSS entry files
	feedName    string
	hasPosts    bool
	math        bool
	anchors     bool
}

type jekyllDoc struct {
	origin     string // source path
	kind       string // "posts", "drafts", "pages", or a collection label
	doc        *Document
	dest       string
	date       time.Time
	hasDate    bool
	slugData   string // Jekyll's data["slug"]: front matter or file name part
	fileTitle  string // the file name without date and extension
	basename   string // file name without extension
	collPath   string // path inside the collection, without extension
	categories []string
	tags       []string
	permalink  string
	isIndex    bool
	draft      bool
	skip       bool // kept aside in _migration, not migrated
	skipLiquid bool
	excerptSep string
	vars       map[string]any
	oldURL     string
}

func (j jekyllSource) Plan(p *Plan) error {
	s := &jekyllSite{
		p:           p,
		collections: map[string]*jekyllCollection{},
		plugins:     map[string]bool{},
		byOrigin:    map[string]*jekyllDoc{},
		staticPaths: map[string]bool{},
		usedDests:   map[string]bool{},
		layouts:     map[string]string{},
		sassDir:     "_sass",
		dataDir:     "_data",
		feedName:    "feed.xml",
	}
	if err := s.readConfig(); err != nil {
		return err
	}
	if data, err := p.ReadFile("Gemfile"); err == nil && bytes.Contains(data, []byte("github-pages")) {
		s.ghPages = true
		for _, name := range []string{"jekyll-optional-front-matter", "jekyll-titles-from-headings", "jekyll-relative-links", "jekyll-readme-index", "jekyll-default-layout"} {
			s.plugins[name] = true
		}
	}
	if err := s.walk(); err != nil {
		return err
	}
	s.finishDocs()
	s.convertDocs()
	s.planConfig()
	s.addDocs()
	s.planSass()
	return nil
}

func (s *jekyllSite) readConfig() error {
	name := jekyllConfigFile(s.p)
	s.cfg = map[string]any{}
	if name == "" {
		s.p.Report.Add(Review, "", 0, "No `_config.yml` found; Jekyll's defaults are assumed.")
		return nil
	}
	data, err := s.p.ReadFile(name)
	if err != nil {
		return err
	}
	if strings.HasSuffix(name, ".toml") {
		err = toml.Unmarshal(data, &s.cfg)
	} else {
		err = yaml.Unmarshal(data, &s.cfg)
	}
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	if s.cfg == nil {
		s.cfg = map[string]any{}
	}
	s.p.Preserve(name)

	s.permalink, _ = s.cfg["permalink"].(string)
	if tz, _ := s.cfg["timezone"].(string); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			s.tz = loc
		} else {
			s.p.Report.Add(Review, name, 0, "Time zone `%s` is unknown to this system, so dates without an offset are read as UTC.", tz)
		}
	}
	if v, _ := s.cfg["collections_dir"].(string); v != "" {
		s.collectionsDir = strings.Trim(v, "/")
	}
	s.excludes = append(slices.Clone(jekyllDefaultExcludes), stringList(s.cfg["exclude"])...)
	s.includes = stringList(s.cfg["include"])
	if sep, ok := s.cfg["excerpt_separator"].(string); ok {
		s.excerptSep = sep
	}
	for _, key := range []string{"plugins", "gems"} {
		for _, name := range stringList(s.cfg[key]) {
			s.plugins[name] = true
		}
	}
	if sass, ok := s.cfg["sass"].(map[string]any); ok {
		if dir, _ := sass["sass_dir"].(string); dir != "" {
			s.sassDir = strings.Trim(dir, "/")
		}
	}
	if dir, _ := s.cfg["data_dir"].(string); dir != "" {
		s.dataDir = strings.Trim(dir, "/")
	}
	if feed, ok := s.cfg["feed"].(map[string]any); ok {
		if p, _ := feed["path"].(string); p != "" {
			s.feedName = strings.TrimPrefix(p, "/")
		}
	}

	switch c := s.cfg["collections"].(type) {
	case map[string]any:
		for label, raw := range c {
			coll := &jekyllCollection{label: label}
			if meta, ok := raw.(map[string]any); ok {
				coll.output = meta["output"] == true
				coll.permalink, _ = meta["permalink"].(string)
			}
			s.collections[label] = coll
		}
	case []any:
		for _, label := range stringList(c) {
			s.collections[label] = &jekyllCollection{label: label}
		}
	}
	if posts, ok := s.collections["posts"]; ok {
		if posts.permalink != "" {
			s.permalink = posts.permalink
		}
		delete(s.collections, "posts")
	}

	if list, ok := s.cfg["defaults"].([]any); ok {
		for _, raw := range list {
			set, _ := raw.(map[string]any)
			values, _ := set["values"].(map[string]any)
			if values == nil {
				continue
			}
			d := jekyllDefault{values: values}
			if scope, ok := set["scope"].(map[string]any); ok {
				d.path, _ = scope["path"].(string)
				d.path = strings.Trim(d.path, "/")
				d.typ, _ = scope["type"].(string)
			}
			s.defaults = append(s.defaults, d)
		}
		// Later, more specific scopes win: longer paths, then typed ones.
		slices.SortStableFunc(s.defaults, func(a, b jekyllDefault) int {
			if len(a.path) != len(b.path) {
				return len(a.path) - len(b.path)
			}
			if (a.typ != "") != (b.typ != "") {
				if a.typ == "" {
					return -1
				}
				return 1
			}
			return 0
		})
	}
	return nil
}

// excluded applies Jekyll's exclude and include lists to a source path.
func (s *jekyllSite) excluded(rel string) bool {
	matches := func(patterns []string) bool {
		for _, pattern := range patterns {
			pattern = strings.TrimPrefix(pattern, "/")
			dir := strings.TrimSuffix(pattern, "/")
			if rel == dir || strings.HasPrefix(rel, dir+"/") {
				return true
			}
			if ok, _ := path.Match(pattern, rel); ok {
				return true
			}
			if ok, _ := path.Match(pattern, path.Base(rel)); ok && !strings.Contains(pattern, "/") {
				return true
			}
		}
		return false
	}
	if matches(s.includes) {
		return false
	}
	if matches(s.excludes) {
		return true
	}
	for _, seg := range strings.Split(rel, "/") {
		if strings.HasPrefix(seg, "#") || strings.HasSuffix(seg, "~") {
			return true
		}
	}
	return false
}

func (s *jekyllSite) included(seg string) bool {
	return slices.ContainsFunc(s.includes, func(p string) bool { return strings.Trim(p, "/") == seg })
}

// walk classifies every file in the site.
func (s *jekyllSite) walk() error {
	preservedNote := map[string]bool{}
	return s.p.Walk(".", func(rel string) error {
		if s.excluded(rel) {
			return nil
		}
		first, _, _ := strings.Cut(rel, "/")
		inner := rel
		if s.collectionsDir != "" {
			if rest, ok := strings.CutPrefix(rel, s.collectionsDir+"/"); ok {
				inner = rest
			}
		}
		innerFirst, _, _ := strings.Cut(inner, "/")
		segs := strings.Split(rel, "/")

		switch {
		case strings.HasPrefix(path.Base(rel), "_config.") && !strings.Contains(rel, "/"):
			return nil
		case slices.Contains(segs[:len(segs)-1], "_posts"):
			return s.addPost(rel, false)
		case innerFirst == "_drafts" && inner != rel || first == "_drafts":
			return s.addPost(rel, true)
		case strings.HasPrefix(innerFirst, "_") && s.collections[strings.TrimPrefix(innerFirst, "_")] != nil && strings.Contains(inner, "/"):
			return s.addCollectionDoc(rel, s.collections[strings.TrimPrefix(innerFirst, "_")], strings.TrimPrefix(inner, innerFirst+"/"))
		case first == "_site" || first == ".jekyll-metadata":
			return nil
		case first == s.dataDir:
			return s.addData(rel)
		case first == s.sassDir:
			return nil // planned with the SCSS entries
		case strings.HasPrefix(first, "_") && !s.included(first) && strings.Contains(rel, "/"):
			s.p.Preserve(rel)
			if !preservedNote[first] {
				preservedNote[first] = true
				switch first {
				case "_layouts", "_includes":
					s.p.Report.Add(Unsupported, first, 0, "Liquid templates in `%s/` are not translated; the site uses Nida's starter templates. The originals are in `%s/%s/`.", first, preservedDir, first)
				case "_plugins":
					s.p.Report.Add(Unsupported, first, 0, "Ruby plugins in `_plugins/` cannot run in Nida. Reimplement what they do with shortcodes or templates.")
				default:
					s.p.Report.Add(Review, first, 0, "Jekyll ignores `%s/` because it is not a collection or listed in `include`; it was kept in `%s/` only.", first, preservedDir)
				}
			}
			return nil
		case strings.HasPrefix(first, "_") && !s.included(first):
			return nil // an underscore file Jekyll ignores
		}
		return s.addRegular(rel)
	})
}

// readHead reports whether a file starts with YAML front matter, the way
// Utils.has_yaml_header? does.
func (s *jekyllSite) hasFrontMatter(rel string) bool {
	f, err := os.Open(filepath.Join(s.p.Root, filepath.FromSlash(rel)))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, 64)
	n, _ := io.ReadFull(f, head)
	head = bytes.TrimPrefix(head[:n], bomPrefix)
	line, _, found := bytes.Cut(head, []byte("\n"))
	return found && string(bytes.TrimRight(line, " \t\r")) == "---"
}

// parse reads a Jekyll document. Jekyll also closes front matter with
// "...", which Nida does not accept.
func (s *jekyllSite) parse(rel string) (*Document, []byte, error) {
	data, err := s.p.ReadFile(rel)
	if err != nil {
		return nil, nil, err
	}
	if s.hasFrontMatter(rel) {
		lines := strings.SplitAfter(string(data), "\n")
		for i := 1; i < len(lines); i++ {
			trimmed := strings.TrimRight(lines[i], " \t\r\n")
			if trimmed == "---" {
				break
			}
			if trimmed == "..." {
				lines[i] = strings.Replace(lines[i], "...", "---", 1)
				data = []byte(strings.Join(lines, ""))
				break
			}
		}
	} else {
		data = append([]byte("---\n---\n"), data...)
	}
	doc, err := ParseDocument(data)
	return doc, data, err
}

func isMarkdownExt(ext string) bool { return slices.Contains(jekyllMarkdownExts, strings.ToLower(ext)) }

func isHTMLExt(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".html" || ext == ".htm"
}

func (s *jekyllSite) addPost(rel string, draft bool) error {
	ext := path.Ext(rel)
	base := strings.TrimSuffix(path.Base(rel), ext)
	if !isMarkdownExt(ext) && !isHTMLExt(ext) {
		if s.hasFrontMatter(rel) {
			s.p.Preserve(rel)
			s.p.Report.Add(Manual, rel, 0, "Not migrated: Jekyll renders this %s file as a post, but Nida posts are Markdown. Convert it by hand.", ext)
		} else {
			s.p.Copy(path.Join("static", rel), rel)
			s.p.Report.Add(Review, rel, 0, "A file without front matter inside `_posts/` was copied to `static/%s`; check where the old site served it.", rel)
		}
		return nil
	}
	d := &jekyllDoc{origin: rel, kind: "posts", basename: base, draft: draft, excerptSep: s.excerptSep}
	if draft {
		d.kind = "drafts"
		d.slugData = base
	} else {
		m := jekyllDateName.FindStringSubmatch(base)
		if m == nil {
			s.p.Report.Add(Manual, rel, 0, "Not migrated: Jekyll skips posts whose file name has no `YYYY-MM-DD-` date, so this was never published. Rename it to migrate it.")
			s.p.Preserve(rel)
			return nil
		}
		t, _, err := ParseDate(m[1])
		if err != nil {
			t, err = time.Parse("2006-1-2", m[1])
		}
		if err == nil {
			d.date, d.hasDate = t, true
		}
		d.slugData = m[2]
	}
	d.slugData = strings.TrimRight(d.slugData, ".")
	d.fileTitle = d.slugData
	// Posts in blog/_posts/ get the category "blog" from their path.
	if i := strings.Index(rel, "_posts/"); i > 0 {
		for _, c := range strings.Split(strings.Trim(rel[:i], "/"), "/") {
			if c != "" && c != s.collectionsDir {
				d.categories = append(d.categories, c)
			}
		}
	}
	return s.addDoc(d, rel)
}

func (s *jekyllSite) addCollectionDoc(rel string, coll *jekyllCollection, inner string) error {
	ext := path.Ext(rel)
	if !isMarkdownExt(ext) && !isHTMLExt(ext) || !s.hasFrontMatter(rel) {
		if coll.output {
			s.p.Copy(path.Join("static", coll.label, inner), rel)
		} else {
			s.p.Preserve(rel)
		}
		return nil
	}
	if !coll.output {
		s.p.Preserve(rel)
		s.p.Report.Add(Unsupported, "_"+coll.label, 0, "Collection `%s` has `output: false`, so its documents are only used by templates. They were kept in `%s/`; move what templates need into `data/`.", coll.label, preservedDir)
		return nil
	}
	base := strings.TrimSuffix(path.Base(rel), ext)
	d := &jekyllDoc{origin: rel, kind: coll.label, basename: base, collPath: "/" + strings.TrimSuffix(inner, ext), excerptSep: s.excerptSep}
	d.slugData = base
	if m := jekyllDateName.FindStringSubmatch(base); m != nil {
		if t, _, err := ParseDate(m[1]); err == nil {
			d.date, d.hasDate = t, true
		}
		d.slugData = m[2]
	}
	d.slugData = strings.TrimRight(d.slugData, ".")
	return s.addDoc(d, rel)
}

func (s *jekyllSite) addData(rel string) error {
	inner := strings.TrimPrefix(rel, s.dataDir+"/")
	switch strings.ToLower(path.Ext(rel)) {
	case ".yml", ".yaml", ".json", ".toml":
		s.p.Copy(path.Join("data", inner), rel)
	default:
		s.p.Preserve(rel)
		s.p.Report.Add(Unsupported, rel, 0, "Nida data files are TOML, JSON, or YAML; this one was kept in `%s/`. Convert it to use it from templates.", preservedDir)
	}
	return nil
}

// addRegular handles a file outside the special directories: a page if it
// has front matter, a static file otherwise.
func (s *jekyllSite) addRegular(rel string) error {
	ext := path.Ext(rel)
	fm := s.hasFrontMatter(rel)
	optional := !fm && isMarkdownExt(ext) && s.plugins["jekyll-optional-front-matter"]
	if !fm && !optional {
		s.p.Copy(path.Join("static", rel), rel)
		s.staticPaths[rel] = true
		return nil
	}
	if !isMarkdownExt(ext) && !isHTMLExt(ext) {
		return s.addLiquidStatic(rel, ext)
	}
	base := strings.TrimSuffix(path.Base(rel), ext)
	if strings.EqualFold(base, "404") && !strings.Contains(rel, "/") {
		s.p.Preserve(rel)
		s.p.Report.Add(Review, rel, 0, "Nida writes `/404.html` from the `404` template, so this page was not migrated. Move its content into `templates/404.html`.")
		return nil
	}
	d := &jekyllDoc{origin: rel, kind: "pages", basename: base, slugData: base, excerptSep: s.excerptSep}
	d.isIndex = base == "index" || base == "README" && s.plugins["jekyll-readme-index"] && !s.siblingIndex(rel)
	if d.isIndex && base == "README" {
		s.p.Report.Add(Changed, rel, 0, "`README` became its directory's index page, as jekyll-readme-index does.")
	}
	return s.addDoc(d, rel)
}

func (s *jekyllSite) siblingIndex(rel string) bool {
	dir := path.Dir(rel)
	for _, ext := range append(slices.Clone(jekyllMarkdownExts), ".html") {
		if s.p.Exists(path.Join(dir, "index"+ext)) {
			return true
		}
	}
	return false
}

// addLiquidStatic handles a non-page file with front matter, which Jekyll
// runs through Liquid. Nida generates feeds, sitemaps, and robots.txt
// itself and copies other files unchanged.
func (s *jekyllSite) addLiquidStatic(rel, ext string) error {
	switch lower := strings.ToLower(rel); {
	case lower == "feed.xml" || lower == "atom.xml":
		s.feedName = rel
		s.p.Preserve(rel)
		s.p.Report.Add(Changed, rel, 0, "Nida writes the Atom feed at `/%s` itself; this Liquid template was kept for reference only.", rel)
		return nil
	case lower == "sitemap.xml" || lower == "robots.txt":
		s.p.Preserve(rel)
		s.p.Report.Add(Changed, rel, 0, "Nida writes `/%s` itself; this Liquid template was kept for reference only.", rel)
		return nil
	case ext == ".scss" || ext == ".sass":
		s.sassEntries = append(s.sassEntries, rel)
		return nil
	}
	s.p.Preserve(rel)
	s.p.Report.Add(Manual, rel, 0, "Not migrated: Jekyll runs Liquid over this file, but Nida copies static files unchanged. Write its output to `static/%s`, or generate it from a template.", rel)
	return nil
}

func (s *jekyllSite) addDoc(d *jekyllDoc, rel string) error {
	doc, raw, err := s.parse(rel)
	if err != nil {
		s.p.Report.Add(Manual, rel, 1, "Front matter could not be parsed (%v); the file was kept in `%s/`.", err, preservedDir)
		s.p.Preserve(rel)
		return nil
	}
	d.doc = doc
	if doc.Format == TOML {
		// Jekyll reads only YAML front matter; "+++" is plain text there.
		doc, _ = ParseDocument(append([]byte("---\n---\n"), raw...))
		d.doc = doc
	}
	s.applyDefaults(d)
	d.vars = cloneMap(doc.Values())
	if v, ok := doc.Get("date"); ok {
		if t, _, err := ParseDate(v); err == nil {
			d.date, d.hasDate = t, true
		}
	}
	if slug := doc.String("slug"); slug != "" {
		d.slugData = slug
	}
	d.permalink = doc.String("permalink")
	d.categories = append(d.categories, splitTerms(doc.Values()["categories"])...)
	d.categories = append(d.categories, splitTerms(doc.Values()["category"])...)
	d.categories = uniqueStrings(d.categories)
	d.tags = uniqueStrings(append(splitTerms(doc.Values()["tags"]), splitTerms(doc.Values()["tag"])...))
	if v, ok := doc.Get("render_with_liquid"); ok && v == false {
		d.skipLiquid = true
	}
	if sep, ok := doc.Get("excerpt_separator"); ok {
		d.excerptSep, _ = sep.(string)
	}
	if v, ok := doc.Get("published"); ok && isFalse(v) {
		d.draft = true
	}
	s.docs = append(s.docs, d)
	s.byOrigin[rel] = d
	return nil
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// isFalse reads YAML 1.1 booleans, which Jekyll's YAML parser accepts.
func isFalse(v any) bool {
	switch x := v.(type) {
	case bool:
		return !x
	case string:
		switch strings.ToLower(x) {
		case "false", "no", "off", "n":
			return true
		}
	}
	return false
}

// splitTerms reads categories or tags: a list, or a string split on
// whitespace, as Jekyll does.
func splitTerms(v any) []string {
	switch x := v.(type) {
	case string:
		return strings.Fields(x)
	case []any:
		var out []string
		for _, item := range x {
			if item != nil {
				out = append(out, fmt.Sprint(item))
			}
		}
		return out
	}
	return nil
}

func uniqueStrings(in []string) []string {
	var out []string
	for _, s := range in {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// applyDefaults copies matching front matter defaults into a document
// where it does not set the key itself.
func (s *jekyllSite) applyDefaults(d *jekyllDoc) {
	typ := d.kind
	if typ == "drafts" {
		typ = "posts"
	}
	applied := false
	merged := map[string]any{}
	for _, def := range s.defaults {
		if def.typ != "" && def.typ != typ {
			continue
		}
		if def.path != "" {
			matched, _ := path.Match(def.path, d.origin)
			if !matched && d.origin != def.path && !strings.HasPrefix(d.origin, def.path+"/") {
				if ok, _ := path.Match(def.path, path.Dir(d.origin)); !ok {
					continue
				}
			}
		}
		for k, v := range def.values {
			merged[k] = v
		}
	}
	for _, k := range sortedKeys(merged) {
		if !d.doc.Has(k) {
			_ = d.doc.Set(k, merged[k])
			applied = true
		}
	}
	if applied {
		s.p.Report.Add(Changed, d.origin, 0, "Wrote the `defaults` from the Jekyll config into the front matter, since Nida has no front matter defaults.")
	}
}

// finishDocs decides where each document goes and what its old URL was.
func (s *jekyllSite) finishDocs() {
	// Posts first, so pages cannot take their names.
	slices.SortStableFunc(s.docs, func(a, b *jekyllDoc) int {
		return kindOrder(a.kind) - kindOrder(b.kind)
	})
	for _, d := range s.docs {
		switch d.kind {
		case "posts", "drafts":
			s.hasPosts = true
			name := d.fileTitle
			if name == "" {
				name = d.basename
			}
			dest := "content/posts/" + name + ".md"
			if s.usedDests[dest] && d.hasDate {
				dest = "content/posts/" + d.date.Format("2006-01-02") + "-" + name + ".md"
			}
			d.dest = dest
		case "pages":
			// Pages in an included _pages/ directory go to content/pages/.
			dir := strings.TrimPrefix(path.Dir(d.origin), "_")
			if d.isIndex {
				d.dest = path.Join("content", dir, "_index.md")
			} else {
				d.dest = path.Join("content", dir, d.basename+".md")
			}
		default:
			d.dest = path.Join("content", d.kind, strings.TrimPrefix(d.collPath, "/")+".md")
		}
		s.usedDests[d.dest] = true
		if !d.draft && !d.isIndex {
			d.oldURL = s.oldURL(d)
		}
	}
}

func kindOrder(kind string) int {
	switch kind {
	case "posts":
		return 0
	case "drafts":
		return 1
	case "pages":
		return 3
	}
	return 2
}

// urlValues returns the placeholders for a post or collection document.
func (s *jekyllSite) urlValues(d *jekyllDoc) map[string]string {
	values := map[string]string{
		"collection": d.kind,
		"path":       d.collPath,
		"name":       jekyllSlugify(d.basename, "default", false),
		"title":      jekyllSlugify(d.slugData, "pretty", true),
		"slug":       jekyllSlugify(d.slugData, "default", false),
		"categories": jekyllCategoriesPath(d.categories),
		"output_ext": ".html",
	}
	if d.kind == "posts" {
		values["path"] = "/" + d.basename
	}
	if d.hasDate {
		jekyllDateValues(d.date, values)
	}
	return values
}

func (s *jekyllSite) oldURL(d *jekyllDoc) string {
	var template string
	var values map[string]string
	switch d.kind {
	case "pages":
		dir := path.Dir(d.origin)
		if dir == "." {
			dir = ""
		}
		values = map[string]string{"path": dir, "basename": d.basename, "output_ext": ".html"}
		template = jekyllAddSuffix("/:path/:basename", s.permalink)
		if d.permalink != "" {
			template = d.permalink
		}
	case "posts":
		values = s.urlValues(d)
		template = jekyllPostTemplate(s.permalink)
		if d.permalink != "" {
			template = d.permalink
		}
	default:
		values = s.urlValues(d)
		coll := s.collections[d.kind]
		template = coll.permalink
		if template == "" {
			template = jekyllAddSuffix("/:collection/:path", s.permalink)
		}
		if t, ok := jekyllStyles[template]; ok {
			template = t
		}
		if d.permalink != "" {
			template = d.permalink
		}
	}
	if !d.hasDate && regexp.MustCompile(`:(year|month|day|i_|short_|long_|y_day|week|w_|hour|minute|second)`).MatchString(template) {
		s.p.Report.Add(Manual, d.origin, 0, "The old URL uses the date, but this document has none, so it could not be worked out. Set `path` to its old URL.")
		return ""
	}
	u := jekyllServedURL(jekyllURL(template, values), ".html")
	if strings.HasSuffix(u, "/index.html") {
		u = strings.TrimSuffix(u, "index.html")
	}
	return u
}

// planConfig maps _config.yml onto Nida's config.
func (s *jekyllSite) planConfig() {
	c := s.p.Config
	r := s.p.Report
	cfg := s.cfg
	file := jekyllConfigFile(s.p)

	url, _ := cfg["url"].(string)
	baseurl, _ := cfg["baseurl"].(string)
	baseurl = strings.Trim(baseurl, "/")
	switch {
	case url != "":
		u := strings.TrimRight(url, "/")
		if baseurl != "" {
			u += "/" + baseurl
		}
		c.Set("base_url", u+"/")
	case baseurl != "":
		c.Set("base_url", "http://127.0.0.1:1702/"+baseurl+"/")
		r.Add(Manual, file, 0, "The old site sets `baseurl` but no `url`, so `base_url` is the local preview address with the base path. Set it to the site's real URL before deploying.")
	}
	for _, key := range []string{"title", "description"} {
		if v, ok := cfg[key].(string); ok {
			c.Set(key, v)
		}
	}
	for _, key := range []string{"lang", "locale"} {
		if v, ok := cfg[key].(string); ok && v != "" {
			c.Set("language", strings.ReplaceAll(v, "_", "-"))
			break
		}
	}
	switch a := cfg["author"].(type) {
	case string:
		c.Set("author", a)
	case map[string]any:
		if name, ok := a["name"].(string); ok {
			c.Set("author", name)
		}
		c.Table("extra").Set("author", a)
		r.Add(Review, file, 0, "`author` is a table; its name became `author`, and the whole table is `.Config.Extra.author` for templates.")
	}
	if v, ok := cfg["future"].(bool); ok && v {
		c.Set("future", true)
	}
	if v, ok := cfg["show_drafts"].(bool); ok && v {
		c.Set("drafts", true)
	}
	if v, ok := cfg["unpublished"].(bool); ok && v {
		r.Add(Review, file, 0, "`unpublished: true` built pages marked `published: false`; they are drafts now, so build with `--drafts` to include them.")
	}

	if s.hasPosts {
		home := c.Table("home")
		home.Set("sections", []any{"posts"})
		if n, ok := cfg["paginate"].(int); ok && n > 0 {
			home.Set("limit", n)
			c.Set("paginate", n)
			r.Add(Review, file, 0, "Pagination pages move from Jekyll's `/page2/` to Nida's `/posts/page/2/`, and the home page lists the latest %d posts.", n)
		}
	}
	feed := c.Table("feed")
	feed.Set("enabled", true)
	feed.Set("filename", s.feedName)
	if s.hasPosts {
		feed.Set("sections", []any{"posts"})
	}
	if !s.plugins["jekyll-feed"] && s.feedName == "feed.xml" && !s.p.Exists("feed.xml") {
		r.Add(Changed, "", 0, "Nida writes an Atom feed of the posts at `/feed.xml`.")
	} else if s.plugins["jekyll-feed"] {
		r.Add(Changed, file, 0, "jekyll-feed became Nida's Atom feed at `/%s`.", s.feedName)
	}
	if feedCfg, ok := cfg["feed"].(map[string]any); ok {
		for _, key := range []string{"categories", "collections", "tags", "posts_limit", "excerpt_only"} {
			if _, ok := feedCfg[key]; ok {
				r.Add(Unsupported, file, 0, "jekyll-feed `%s` has no Nida equivalent.", key)
			}
		}
	}

	if kd, ok := cfg["kramdown"].(map[string]any); ok {
		for _, key := range sortedKeys(kd) {
			v := kd[key]
			switch key {
			case "hard_wrap":
				if v == true {
					c.Table("markdown").Set("hard_wraps", true)
				}
			case "input":
				if fmt.Sprint(v) != "GFM" {
					r.Add(Review, file, 0, "kramdown `input: %v` is not GFM; Nida renders CommonMark with GitHub extensions, which treats some kramdown syntax differently.", v)
				}
			case "parse_block_html", "parse_span_html":
				if v == true {
					r.Add(Review, file, 0, "kramdown `%s` parsed Markdown inside HTML; Nida does not, so Markdown inside HTML blocks shows as text.", key)
				}
			case "auto_ids":
				if v == false {
					r.Add(Unsupported, file, 0, "kramdown `auto_ids: false` has no equivalent; Nida always gives headings IDs.")
				}
			case "math_engine", "syntax_highlighter", "syntax_highlighter_opts", "toc_levels", "footnote_nr", "show_warnings", "guess_lang":
			default:
				r.Add(Unsupported, file, 0, "kramdown `%s` has no Nida equivalent.", key)
			}
		}
	}
	if v, ok := cfg["highlighter"]; ok && (v == false || v == "none") {
		r.Add(Unsupported, file, 0, "`highlighter: none` has no equivalent; Nida always highlights code blocks.")
	}
	if v, ok := cfg["markdown"].(string); ok && !strings.EqualFold(v, "kramdown") && !strings.HasPrefix(strings.ToLower(v), "commonmark") {
		r.Add(Review, file, 0, "The site used the `%s` Markdown renderer; Nida renders CommonMark with GitHub extensions.", v)
	}

	for _, theme := range []string{"theme", "remote_theme"} {
		if v, ok := cfg[theme].(string); ok && v != "" {
			r.Add(Unsupported, file, 0, "`%s: %s` is a Ruby gem of Liquid layouts that Nida cannot use; the site uses Nida's starter templates.", theme, v)
		}
	}
	for _, name := range sortedKeys(s.plugins) {
		switch name {
		case "jekyll-feed", "jekyll-sitemap", "jekyll-optional-front-matter", "jekyll-relative-links", "jekyll-readme-index", "jekyll-titles-from-headings":
		case "jekyll-redirect-from":
			r.Add(Changed, file, 0, "jekyll-redirect-from `redirect_from` lists became `aliases`.")
		case "jekyll-seo-tag":
			r.Add(Changed, file, 0, "jekyll-seo-tag is replaced by Nida's `nida-seo` partial, which the starter templates use.")
		case "jekyll-paginate", "jekyll-paginate-v2":
		case "jekyll-default-layout":
			r.Add(Changed, file, 0, "jekyll-default-layout is not needed; Nida picks `post` or `page` templates by default.")
		default:
			r.Add(Unsupported, file, 0, "Plugin `%s` cannot run in Nida.", name)
		}
	}
	for _, key := range []string{"jekyll-archives", "jekyll_archives"} {
		if _, ok := cfg[key]; ok {
			r.Add(Unsupported, file, 0, "jekyll-archives pages are not migrated. Nida taxonomies render term pages at `/tags/<term>/` with `render = true`.")
		}
	}
	if v, _ := cfg["source"].(string); v != "" && v != "." {
		r.Add(Unsupported, file, 0, "`source: %s` is not followed; run `nida migrate` on that directory instead.", v)
	}

	// Taxonomies used by posts.
	usesTags, usesCats := false, false
	for _, d := range s.docs {
		usesTags = usesTags || len(d.tags) > 0
		usesCats = usesCats || len(d.categories) > 0
	}
	if usesTags {
		c.Taxonomy("tags")
	}
	if usesCats {
		c.Taxonomy("categories")
	}
	if s.math {
		c.Table("markdown").Set("math", true)
		r.Add(Review, "", 0, "Content uses `$$` math, so `markdown.math` is on. kramdown treats `$$...$$` inside a paragraph as inline math; Nida treats it as display math, so use `$...$` for inline formulas.")
	}
	if s.anchors {
		r.Add(Review, "", 0, "Content links to `#anchors`. kramdown and Nida build heading IDs differently for headings with numbers, punctuation, or non-ASCII letters, so check in-page links.")
	}

	s.planPermalinks()

	// Custom site variables become [extra].
	var extras []string
	for _, key := range sortedKeys(cfg) {
		if jekyllBuiltinKeys[key] {
			continue
		}
		c.Table("extra").Set(key, cfg[key])
		extras = append(extras, key)
	}
	if len(extras) > 0 {
		r.Add(Changed, file, 0, "Custom site settings (%s) moved to `[extra]`; templates read `site.x` as `.Config.Extra.x`.", strings.Join(extras, ", "))
	}

	for name, conflict := range map[string]string{"robots.txt": "robots", "sitemap.xml": "sitemap"} {
		if s.staticPaths[name] {
			c.Table(conflict).Set("enabled", false)
			r.Add(Changed, name, 0, "The site has its own `%s`, so Nida's generated one is off.", name)
		}
	}
	if s.staticPaths[s.feedName] {
		feed.Set("enabled", false)
		r.Add(Changed, s.feedName, 0, "The site has its own static `%s`, so Nida's feed is off.", s.feedName)
	}
}

// jekyllBuiltinKeys are Jekyll and plugin settings, which do not move to
// [extra].
var jekyllBuiltinKeys = map[string]bool{
	"url": true, "baseurl": true, "title": true, "description": true, "lang": true, "locale": true, "author": true,
	"timezone": true, "permalink": true, "collections": true, "collections_dir": true, "defaults": true,
	"exclude": true, "include": true, "keep_files": true, "plugins": true, "gems": true, "whitelist": true,
	"plugins_dir": true, "layouts_dir": true, "data_dir": true, "includes_dir": true, "sass": true,
	"markdown": true, "markdown_ext": true, "kramdown": true, "highlighter": true, "excerpt_separator": true,
	"paginate": true, "paginate_path": true, "pagination": true, "future": true, "show_drafts": true, "unpublished": true,
	"limit_posts": true, "theme": true, "remote_theme": true, "feed": true, "jekyll-archives": true, "jekyll_archives": true,
	"source": true, "destination": true, "safe": true, "incremental": true, "profile": true, "quiet": true, "verbose": true,
	"strict_front_matter": true, "liquid": true, "webrick": true, "show_dir_listing": true, "encoding": true, "lsi": true,
	"detach": true, "port": true, "host": true, "livereload": true, "livereload_port": true, "livereload_min_delay": true,
	"livereload_max_delay": true, "livereload_ignore": true, "watch": true, "force_polling": true, "cache_dir": true,
	"disable_disk_cache": true, "skip_initial_build": true, "serving": true, "relative_links": true, "readme_index": true,
	"titles_from_headings": true, "optional_front_matter": true, "redcarpet": true, "commonmark": true, "rdiscount": true,
	"sitemap": true, "seo": true, "jekyll_compose": true,
}

// planPermalinks writes a [permalinks] pattern for posts and collections
// when Jekyll's scheme can be written as one, and sets each document's
// slug to match. Documents the pattern cannot express keep their URL
// through the URL check, which sets `path`.
func (s *jekyllSite) planPermalinks() {
	groups := map[string][]*jekyllDoc{}
	for _, d := range s.docs {
		switch d.kind {
		case "pages":
		case "drafts":
			groups["posts"] = append(groups["posts"], d)
		default:
			groups[d.kind] = append(groups[d.kind], d)
		}
	}
	for _, section := range sortedKeys(groups) {
		docs := groups[section]
		var template string
		if section == "posts" {
			template = jekyllPostTemplate(s.permalink)
		} else {
			template = s.collections[section].permalink
			if template == "" {
				template = jekyllAddSuffix("/:collection/:path", s.permalink)
			}
			if t, ok := jekyllStyles[template]; ok {
				template = t
			}
		}
		// :path is the file path in the collection, unslugified; for files
		// at the top of the collection it can be a {slug}.
		usesPath := strings.Contains(template, ":path")
		if usesPath {
			if strings.Contains(template, ":name") || strings.Contains(template, ":title") || strings.Contains(template, ":slug") {
				continue
			}
			template = strings.ReplaceAll(template, ":path", ":name")
		}
		categories := ""
		single := len(docs) > 0
		for _, d := range docs {
			if !d.draft && (len(d.categories) != 1 || strings.ToLower(d.categories[0]) != content.DeriveSlug(d.categories[0])) {
				single = false
			}
		}
		if single && strings.Contains(template, ":categories") {
			categories = "{categories}"
		}
		pattern, token, ok := nidaPattern(template, section, categories)
		if !ok {
			continue
		}
		s.p.Config.Table("permalinks").Set(section, pattern)
		for _, d := range docs {
			value := s.urlValues(d)[token]
			if usesPath {
				value = strings.TrimPrefix(d.collPath, "/")
				if section == "posts" {
					value = d.basename
				}
				if strings.Contains(value, "/") {
					continue // nested; the URL check sets path
				}
			}
			stem := strings.TrimSuffix(path.Base(d.dest), ".md")
			if value != "" && value != content.DeriveSlug(stem) {
				_ = d.doc.Set("slug", value)
			}
		}
	}
}

// convertDocs converts each document's front matter and body.
func (s *jekyllSite) convertDocs() {
	for _, d := range s.docs {
		s.convertFrontMatter(d)
		if isHTMLExt(path.Ext(d.origin)) {
			d.skip = !s.convertHTML(d)
		} else {
			s.convertBody(d)
		}
		if strings.Contains(d.doc.Body, "](#") {
			s.anchors = true
		}
	}
}

// addDocs adds the converted documents to the plan, with a section for
// each directory of pages.
func (s *jekyllSite) addDocs() {
	pageDirs := map[string]bool{}
	for _, d := range s.docs {
		if d.skip {
			continue
		}
		if d.kind == "pages" {
			pageDirs[path.Dir(d.dest)] = true
		}
		s.p.Page(d.dest, d.doc, d.origin, d.oldURL)
	}
	// Sections of pages render them with the page template; Nida would
	// otherwise use post.html for every page.
	for _, dir := range sortedKeys(pageDirs) {
		index := dir + "/_index.md"
		if doc := s.p.Doc(index); doc != nil {
			if !doc.Has("page_template") {
				_ = doc.Set("page_template", "page")
			}
			continue
		}
		doc := NewDocument(TOML, "")
		title := "Home"
		if dir != "content" {
			title = titleFromSlug(path.Base(dir))
		}
		_ = doc.Set("title", title)
		_ = doc.Set("page_template", "page")
		s.p.Page(index, doc, "", "")
	}
	if s.hasPosts && !s.p.Planned("content/posts/_index.md") {
		doc := NewDocument(TOML, "")
		_ = doc.Set("title", "Posts")
		s.p.Page("content/posts/_index.md", doc, "", "")
	}
	for _, name := range sortedKeys(s.layouts) {
		if !slices.Contains([]string{"post", "page"}, name) {
			s.p.Report.Add(Review, s.layouts[name], 0, "Content uses layout `%s`, which the starter templates do not have, so it renders with the default template until you port it.", name)
		}
	}
}

// convertFrontMatter rewrites Jekyll front matter keys for Nida.
func (s *jekyllSite) convertFrontMatter(d *jekyllDoc) {
	doc := d.doc
	r := s.p.Report
	rel := d.origin

	if d.draft {
		_ = doc.Set("draft", true)
		if d.kind == "drafts" {
			r.Add(Changed, rel, 0, "Posts in `_drafts/` became posts with `draft = true`. Jekyll dated them by file time; give them a `date` when you publish.")
		}
	}
	doc.Delete("published")
	if !doc.Has("title") {
		switch {
		case d.kind == "posts" || d.kind == "drafts" || s.collections[d.kind] != nil:
			_ = doc.Set("title", jekyllTitleize(d.slugData))
			r.Add(Changed, rel, 0, "Set `title` from the file name, as Jekyll does.")
		case s.plugins["jekyll-titles-from-headings"]:
			if m := regexp.MustCompile(`(?m)^#[ \t]+(.+?)[ \t#]*$`).FindStringSubmatch(doc.Body); m != nil {
				_ = doc.Set("title", m[1])
				r.Add(Changed, rel, 0, "Set `title` from the first heading, as jekyll-titles-from-headings does.")
			}
		}
	}
	if d.hasDate && !doc.Has("date") && (d.kind == "posts" || s.collections[d.kind] != nil) {
		_ = doc.Set("date", DateValue(doc, d.date, true))
		r.Add(Changed, rel, 0, "Set `date` from the file name, which Jekyll reads and Nida does not.")
	}
	s.normalizeJekyllDate(d, "date")
	if doc.Rename("last_modified_at", "updated") {
		r.Add(Changed, rel, 0, "Renamed `last_modified_at` to `updated`.")
	}
	s.normalizeJekyllDate(d, "updated")

	if layout := doc.String("layout"); layout != "" {
		doc.Delete("layout")
		if d.isIndex {
			r.Add(Review, rel, 0, "Dropped `layout: %s` from an index page; Nida renders sections with the `index` or `list` template.", layout)
		} else if layout != "none" && layout != "null" {
			_ = doc.Set("template", layout)
			if _, seen := s.layouts[layout]; !seen {
				s.layouts[layout] = rel
			}
		}
	}
	if d.permalink != "" {
		doc.Delete("permalink")
		if d.isIndex {
			dir := "/" + strings.TrimPrefix(path.Dir(strings.TrimPrefix(d.dest, "content/")), ".")
			if want := strings.TrimSuffix(dir, "/") + "/"; strings.TrimSuffix(d.permalink, "index.html") != want && d.permalink != strings.TrimSuffix(want, "/") {
				r.Add(Unsupported, rel, 0, "Index pages become sections, which keep their directory URL `%s`; the old permalink `%s` was dropped. Add a redirect on your web host.", want, d.permalink)
			}
		}
	}
	if doc.Has("slug") && !d.isIndex {
		doc.Delete("slug") // planPermalinks sets Nida's slug where needed
	}

	// Taxonomies as lists.
	if len(d.categories) > 0 {
		_ = doc.Set("categories", toAny(d.categories))
	}
	doc.Delete("category")
	if len(d.tags) > 0 {
		_ = doc.Set("tags", toAny(d.tags))
	}
	doc.Delete("tag")

	if v, ok := doc.Get("redirect_from"); ok {
		aliases := frontmatter.StringList(doc.Values()["aliases"])
		for _, a := range splitTerms(v) {
			if !slices.Contains(aliases, a) {
				aliases = append(aliases, a)
			}
		}
		doc.Delete("redirect_from")
		_ = doc.Set("aliases", toAny(aliases))
		r.Add(Changed, rel, 0, "Converted `redirect_from` to `aliases`.")
	}
	if doc.Has("redirect_to") {
		r.Add(Unsupported, rel, 0, "`redirect_to` pages are not supported; add the redirect on your web host, or give the target page an alias.")
	}
	if ex := doc.String("excerpt"); ex != "" {
		if !doc.Has("description") {
			_ = doc.Set("description", ex)
			r.Add(Changed, rel, 0, "Moved `excerpt` to `description`.")
		}
		doc.Delete("excerpt")
	}
	if img, ok := doc.Values()["image"].(map[string]any); ok {
		if p, ok := img["path"].(string); ok {
			_ = doc.Set("image", p)
			r.Add(Review, rel, 0, "`image` was a table; its `path` became `image`.")
		}
	}
	if v, ok := doc.Get("sitemap"); ok && isFalse(v) {
		r.Add(Unsupported, rel, 0, "`sitemap: false` is not supported; the page is listed in the sitemap.")
	}
	for _, key := range []string{"render_with_liquid", "excerpt_separator"} {
		doc.Delete(key)
	}
}

func toAny(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

// normalizeJekyllDate rewrites a date Nida cannot read, and attaches the
// site time zone to dates written without an offset.
func (s *jekyllSite) normalizeJekyllDate(d *jekyllDoc, key string) {
	doc := d.doc
	if s.tz != nil && doc.Format == YAML {
		if raw := rawYAMLScalar(doc, key); raw != "" && regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}[T ]\d{1,2}:\d{2}(:\d{2}(\.\d+)?)?$`).MatchString(raw) {
			for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02T15:04", "2006-1-2 15:04:05", "2006-01-02 15:04:05.999999999"} {
				if t, err := time.ParseInLocation(layout, raw, s.tz); err == nil {
					_ = doc.Set(key, t.Format(time.RFC3339))
					s.p.Report.Add(Changed, d.origin, 0, "Gave dates without an offset the site time zone, %s, as Jekyll does.", s.tz)
					return
				}
			}
		}
	}
	NormalizeDate(doc, key, s.p.Report, d.origin)
}

// rawYAMLScalar returns a top-level scalar's source text, before YAML
// turns it into a time in UTC.
func rawYAMLScalar(doc *Document, key string) string {
	if doc.yamlNode == nil {
		return ""
	}
	if i := doc.yamlIndex(key); i >= 0 && doc.yamlNode.Content[i+1].Kind == yaml.ScalarNode {
		return doc.yamlNode.Content[i+1].Value
	}
	return ""
}

var (
	liquidMarker   = regexp.MustCompile(`\{\{|\{%`)
	htmlBlockStart = regexp.MustCompile(`(?i)^<(?:address|article|aside|blockquote|details|div|dl|fieldset|figure|footer|form|h[1-6]|header|hr|main|nav|ol|p|section|table|ul)[\s/>]`)
	preBlock       = regexp.MustCompile(`(?is)<(pre|textarea|script|style)\b.*?</(pre|textarea|script|style)>`)
)

// convertHTML turns an HTML page into Markdown that renders the same
// HTML. It returns false when the page needs Liquid and was kept aside.
func (s *jekyllSite) convertHTML(d *jekyllDoc) bool {
	body := d.doc.Body
	body = liquidComment.ReplaceAllString(body, "")
	if liquidMarker.MatchString(liquidRaw.ReplaceAllString(body, "")) && !d.skipLiquid {
		s.p.Preserve(d.origin)
		if d.isIndex {
			d.doc.SetBody("")
			s.p.Report.Add(Review, d.origin, 0, "This index page is built with Liquid, so its body was dropped; the section lists its pages with Nida's `index` or `list` template instead. The original is in `%s/`.", preservedDir)
			return true
		}
		s.p.Report.Add(Manual, d.origin, 0, "Not migrated: this HTML page is built with Liquid. Rebuild it as a Markdown page or a template; the original is in `%s/`.", preservedDir)
		return false
	}
	body = liquidRaw.ReplaceAllString(body, "$1")
	// Markdown would turn indented or blank-line-separated HTML into code
	// blocks and paragraphs; one unindented block with no blank lines
	// stays raw HTML. Preformatted elements keep their whitespace.
	var pres []string
	body = preBlock.ReplaceAllStringFunc(body, func(m string) string {
		pres = append(pres, m)
		return fmt.Sprintf("\x00PRE%d\x00", len(pres)-1)
	})
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	body = strings.Join(lines, "\n")
	for i, pre := range pres {
		if strings.Contains(pre, "\n\n") {
			s.p.Report.Add(Review, d.origin, 0, "A `<pre>` block has blank lines, which end raw HTML in Markdown; check that it renders whole.")
		}
		body = strings.Replace(body, fmt.Sprintf("\x00PRE%d\x00", i), pre, 1)
	}
	if body != "" {
		// A page that starts with a block tag is one raw HTML block as it
		// is; anything else is wrapped so Markdown leaves it alone.
		if !htmlBlockStart.MatchString(body) {
			body = "<div class=\"jekyll-html\">\n" + body + "\n</div>"
		}
		body += "\n"
		s.p.UsesRawHTML()
	}
	d.doc.SetBody(body)
	s.p.Report.Add(Review, d.origin, 0, "HTML page moved into a Markdown file as raw HTML. Check that it renders as before.")
	return true
}

// findPost resolves a post_url argument such as 2010-07-21-name or
// /subdir/2010-07-21-name to the post's content path.
func (s *jekyllSite) findPost(name string) string {
	name = strings.Trim(name, "/")
	var fallback string
	for _, d := range s.docs {
		if d.kind != "posts" {
			continue
		}
		withoutExt := strings.TrimSuffix(d.origin, path.Ext(d.origin))
		_, inPosts, _ := strings.Cut(withoutExt, "_posts/")
		if inPosts == name {
			return strings.TrimPrefix(d.dest, "content/")
		}
		if d.basename == path.Base(name) && fallback == "" {
			fallback = strings.TrimPrefix(d.dest, "content/")
		}
	}
	return fallback
}

// linkTarget resolves a {% link %} path to an @/ link or a static path.
func (s *jekyllSite) linkTarget(target string) (string, bool) {
	if d, ok := s.byOrigin[target]; ok {
		return "@/" + strings.TrimPrefix(d.dest, "content/"), true
	}
	if s.staticPaths[target] {
		return "/" + target, true
	}
	return "", false
}

// planSass moves SCSS into Nida's pipeline. Entry files keep their
// directory, so /assets/css/main.css stays where it was; the sass_dir
// partials are copied beside them, since Nida's pipeline has no load path.
func (s *jekyllSite) planSass() {
	if len(s.sassEntries) == 0 {
		if s.p.Exists(s.sassDir) {
			_ = s.p.Walk(s.sassDir, func(rel string) error { s.p.Preserve(rel); return nil })
		}
		return
	}
	dirs := map[string]bool{}
	for _, rel := range s.sassEntries {
		dirs[path.Dir(rel)] = true
	}
	entryDir := sortedKeys(dirs)[0]
	for _, rel := range s.sassEntries {
		doc, _, err := s.parse(rel)
		if err != nil || path.Dir(rel) != entryDir {
			s.p.Preserve(rel)
			s.p.Report.Add(Manual, rel, 0, "Nida compiles SCSS entry files from one directory, `static/%s/`; this one was kept in `%s/`. Move it there or import it.", entryDir, preservedDir)
			continue
		}
		if liquidMarker.MatchString(doc.Body) {
			s.p.Report.Add(Manual, rel, 0, "This stylesheet uses Liquid, which Nida's SCSS pipeline does not run. Replace it with plain Sass.")
		}
		s.p.Write(path.Join("static", rel), []byte(doc.Body), rel)
	}
	_ = s.p.Walk(s.sassDir, func(rel string) error {
		inner := strings.TrimPrefix(rel, s.sassDir+"/")
		dir, base := path.Split(inner)
		if dir == "" && !strings.HasPrefix(base, "_") {
			base = "_" + base // a partial, so Nida does not compile it alone
		}
		s.p.Copy(path.Join("static", entryDir, dir, base), rel)
		return nil
	})
	compressed := false
	if sass, ok := s.cfg["sass"].(map[string]any); ok {
		compressed = sass["style"] == "compressed"
	}
	if compressed {
		s.p.Config.Table("pipeline").Set("minify_css", true)
	}
	scss := s.p.Config.Table("pipeline.scss")
	scss.Set("enabled", true)
	scss.Set("entry_dir", entryDir)
	s.p.Report.Add(Review, s.sassDir, 0, "SCSS compiles through Nida's pipeline, which needs the `sass` command. The `%s/` partials were copied into `static/%s/` so imports resolve; they are also copied to the output as static files.", s.sassDir, entryDir)
}
