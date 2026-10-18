package migrate

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	yaml "go.yaml.in/yaml/v3"

	"github.com/MohamedElashri/nida/internal/content"
)

// eleventySource migrates Eleventy (11ty) sites. Eleventy is configured in
// JavaScript and renders content through Liquid or Nunjucks first, so this
// source reads the config without running it, applies Eleventy's data
// cascade to each page's front matter, and converts template syntax in
// Markdown to Nida shortcodes or plain text where it can.
type eleventySource struct{}

func (eleventySource) Name() string  { return "eleventy" }
func (eleventySource) Title() string { return "Eleventy" }

func (eleventySource) Detect(root string) int {
	p := &Plan{Root: root}
	score := 0
	for _, name := range eleventyConfigFiles {
		if p.Exists(name) {
			score += 3
			break
		}
	}
	if data, err := p.ReadFile("package.json"); err == nil && strings.Contains(string(data), `"@11ty/eleventy"`) {
		score += 2
	}
	if score > 0 && p.Exists(".eleventyignore") {
		score++
	}
	return score
}

// eleventyCollectionTags are tags that Eleventy sites use to build
// collections rather than to describe content. "all" is reserved by
// Eleventy itself.
var eleventyCollectionTags = []string{"all", "nav", "post", "posts", "page", "pages"}

// eleventyTemplateExts are template languages other than Markdown and
// HTML, which Nida cannot render.
var eleventyTemplateExts = []string{"njk", "liquid", "hbs", "mustache", "haml", "pug", "ejs", "11ty.js", "webc", "jsx", "tsx", "mdx", "11ty.ts"}

type eleventyRun struct {
	p      *Plan
	cfg    *eleventyConfig
	ignore []string
	global map[string]any
	// globalFiles maps a global data key to its source file.
	globalFiles map[string]string
	// dirData maps an input-relative directory to its directory data
	// file's values; fileData maps an input-relative template path to its
	// template data file's values.
	dirData  map[string]map[string]any
	fileData map[string]map[string]any
	dataFile map[string]string // input-relative data file -> source-relative path
	// dataSources maps a directory or template key to its data files.
	dataSources map[string][]string
	// templateCount counts content templates under each input-relative
	// directory, recursively.
	templateCount map[string]int

	pages       []*eleventyPlanned
	listings    []string // template pages that Nida sections replace
	unpublished []string
	tagsUsed    bool
	sectionDirs map[string]bool // input-relative dirs with content pages
}

type eleventyPlanned struct {
	origin, dest string
	page         eleventyPage
	doc          *Document
	oldURL       string
	defaultURL   bool
	permalink    string // the permalink template, when it came from directory data
	permalinkDir string
}

func (s eleventySource) Plan(p *Plan) error {
	cfg := readEleventyConfig(p)
	r := &eleventyRun{p: p, cfg: cfg, dirData: map[string]map[string]any{}, fileData: map[string]map[string]any{}, dataFile: map[string]string{}, dataSources: map[string][]string{}, templateCount: map[string]int{}, sectionDirs: map[string]bool{}}
	r.reportConfig()
	r.ignore = r.readIgnores()
	r.global, r.globalFiles = r.readGlobalData()
	r.planSiteConfig()

	files, err := r.files()
	if err != nil {
		return err
	}
	r.readCascade(files)
	for _, rel := range files {
		if err := r.planFile(rel); err != nil {
			return err
		}
	}
	r.planPermalinks()
	for _, pp := range r.pages {
		p.Page(pp.dest, pp.doc, pp.origin, pp.oldURL)
	}
	r.planListings()
	if r.cfg.feedPath != "" {
		r.planFeed(r.cfg.file, "/"+strings.TrimPrefix(r.cfg.feedPath, "/"), r.cfg.feedType == "rss")
	}
	if r.tagsUsed {
		p.Config.Taxonomy("tags").Set("render", true)
	}
	if len(r.unpublished) > 0 {
		slices.Sort(r.unpublished)
		p.Report.Add(Review, "", 0, "%d files were neither templates nor passthrough copies, so Eleventy did not publish them, and they were not migrated: %s.", len(r.unpublished), strings.Join(quoteAll(r.unpublished, 12), ", "))
	}
	return nil
}

func quoteAll(list []string, max int) []string {
	out := make([]string, 0, len(list))
	for i, s := range list {
		if i == max {
			out = append(out, fmt.Sprintf("and %d more", len(list)-max))
			break
		}
		out = append(out, "`"+s+"`")
	}
	return out
}

// reportConfig records what the config file held that Nida cannot use.
func (r *eleventyRun) reportConfig() {
	cfg, rep := r.cfg, r.p.Report
	if cfg.file == "" {
		rep.Add(Review, "", 0, "No Eleventy config file was found, so Eleventy's defaults were assumed (input `.`, includes `_includes`, data `_data`).")
		return
	}
	found := "nothing it recognized"
	if len(cfg.found) > 0 {
		found = strings.Join(cfg.found, ", ")
	}
	rep.Add(Review, cfg.file, 0, "The config file is JavaScript and was read without running it. These settings were picked up: %s. Settings computed in code were not seen.", found)
	r.p.Preserve(cfg.file)
	if len(cfg.collections) > 0 {
		rep.Add(Unsupported, cfg.file, 0, "Custom collections (%s) have no Nida equivalent. Nida lists a section's pages with `.Pages` and filters them with `where .Index.AllPages`.", strings.Join(cfg.collections, ", "))
	}
	if len(cfg.filters) > 0 {
		rep.Add(Unsupported, cfg.file, 0, "Filters (%s) are JavaScript and were not migrated. Nida's template functions cover many of them; see the template function reference.", strings.Join(cfg.filters, ", "))
	}
	if len(cfg.transforms) > 0 {
		rep.Add(Unsupported, cfg.file, 0, "Transforms (%s) post-processed Eleventy's output and were not migrated. For HTML minification, set `minify_html = true`.", strings.Join(cfg.transforms, ", "))
	}
	if len(cfg.globalData) > 0 {
		rep.Add(Unsupported, cfg.file, 0, "Global data added in the config (%s) was not migrated. Put it in a file under `data/`.", strings.Join(cfg.globalData, ", "))
	}
	if len(cfg.shortcodes)+len(cfg.paired) > 0 {
		rep.Add(Review, cfg.file, 0, "Shortcodes defined in the config (%s) are JavaScript. Uses in content became Nida shortcodes with placeholder templates.", strings.Join(append(slices.Clone(cfg.shortcodes), cfg.paired...), ", "))
	}
	for _, plugin := range cfg.plugins {
		name := plugin.pkg
		if name == "" {
			name = plugin.ident
		}
		switch {
		case strings.Contains(name, "syntaxhighlight"):
			rep.Add(Changed, cfg.file, 0, "Plugin `%s`: Nida highlights fenced code blocks natively; set `syntax_theme` to choose a style.", name)
		case strings.Contains(name, "plugin-rss"):
			rep.Add(Changed, cfg.file, 0, "Plugin `%s`: Nida writes feeds itself through `[feed]` and `[rss]`.", name)
		case strings.Contains(name, "navigation"):
			rep.Add(Review, cfg.file, 0, "Plugin `%s`: Nida builds navigation from sections with `.Index.Nav`; `eleventyNavigation` values are kept in each page's extra data.", name)
		case strings.Contains(name, "img"):
			rep.Add(Review, cfg.file, 0, "Plugin `%s`: Nida's image pipeline (`[pipeline.images]`) writes responsive variants for Markdown images and `figure`.", name)
		case strings.Contains(name, "HtmlBase") || strings.Contains(name, "EleventyHtmlBasePlugin"):
			rep.Add(Changed, cfg.file, 0, "Plugin `%s`: Nida applies the base path from `base_url` to links.", name)
		case strings.Contains(name, "id-attribute") || strings.Contains(name, "IdAttribute"):
			rep.Add(Changed, cfg.file, 0, "Plugin `%s`: Nida gives headings `id` attributes itself.", name)
		default:
			rep.Add(Unsupported, cfg.file, 0, "Plugin `%s` has no Nida equivalent and was not migrated.", name)
		}
	}
	others := slices.DeleteFunc(slices.Clone(cfg.others), func(name string) bool { return name == "setLibrary" || name == "amendLibrary" })
	if len(others) < len(cfg.others) {
		rep.Add(Unsupported, cfg.file, 0, "The Markdown library setup (`setLibrary`/`amendLibrary`) was not migrated: Nida renders Markdown with goldmark and its `[markdown]` options, so markdown-it plugins have no effect.")
	}
	if len(others) > 0 {
		rep.Add(Unsupported, cfg.file, 0, "Other config calls were not migrated: %s.", strings.Join(others, ", "))
	}
	if cfg.anchors {
		r.p.Config.Table("markdown").Set("heading_anchors", true)
		rep.Add(Review, cfg.file, 0, "markdown-it-anchor became `markdown.heading_anchors`, which adds a `#` link after each heading.")
	}
	if cfg.mdEngine == "" {
		rep.Add(Changed, cfg.file, 0, "Markdown was not run through a template engine, so content was migrated as written.")
	} else if cfg.mdEngine != "liquid" && cfg.mdEngine != "njk" {
		rep.Add(Review, cfg.file, 0, "Markdown was preprocessed with `%s`; template syntax in content was converted as if it were Liquid.", cfg.mdEngine)
	}
}

// readIgnores reads .eleventyignore and .gitignore at the project root and
// in the input directory.
func (r *eleventyRun) readIgnores() []string {
	var patterns []string
	for _, dir := range slices.Compact([]string{".", r.cfg.input}) {
		for _, name := range []string{".eleventyignore", ".gitignore"} {
			data, err := r.p.ReadFile(path.Join(dir, name))
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
					continue
				}
				anchored := strings.HasPrefix(line, "/") || strings.HasPrefix(line, "./") || strings.Contains(strings.TrimSuffix(line, "/"), "/")
				line = strings.TrimPrefix(strings.TrimPrefix(line, "./"), "/")
				if dir != "." {
					if anchored {
						line = path.Join(dir, line)
					} else {
						line = path.Join(dir, "**", line)
					}
				} else if !anchored {
					line = "**/" + line
				}
				patterns = append(patterns, line)
			}
		}
	}
	return patterns
}

func (r *eleventyRun) ignored(rel string) bool {
	for _, pattern := range r.ignore {
		dirPattern := strings.HasSuffix(pattern, "/")
		pattern = strings.TrimSuffix(pattern, "/")
		if globMatch(pattern, rel) && !dirPattern {
			return true
		}
		// A pattern matching a parent directory ignores everything inside.
		for dir := path.Dir(rel); dir != "."; dir = path.Dir(dir) {
			if globMatch(pattern, dir) {
				return true
			}
		}
	}
	return false
}

// globMatch matches a slash path against a glob with *, ?, and ** (any
// number of directories).
func globMatch(pattern, name string) bool {
	for _, expanded := range expandBraces(pattern) {
		if globParts(strings.Split(expanded, "/"), strings.Split(name, "/")) {
			return true
		}
	}
	return false
}

// expandBraces expands {a,b} alternatives, as in *.{png,jpg}.
func expandBraces(pattern string) []string {
	open := strings.IndexByte(pattern, '{')
	if open < 0 {
		return []string{pattern}
	}
	end := strings.IndexByte(pattern[open:], '}')
	if end < 0 {
		return []string{pattern}
	}
	end += open
	var out []string
	for _, alt := range strings.Split(pattern[open+1:end], ",") {
		out = append(out, expandBraces(pattern[:open]+alt+pattern[end+1:])...)
	}
	return out
}

func globParts(pattern, name []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			for i := 0; i <= len(name); i++ {
				if globParts(pattern[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, err := path.Match(pattern[0], name[0]); err != nil || !ok {
			return false
		}
		pattern, name = pattern[1:], name[1:]
	}
	return len(name) == 0
}

// inputRel returns a project-relative path relative to the input
// directory, or false when it is outside it.
func (r *eleventyRun) inputRel(rel string) (string, bool) {
	if r.cfg.input == "." {
		return rel, true
	}
	if inner, ok := strings.CutPrefix(rel, r.cfg.input+"/"); ok {
		return inner, true
	}
	return "", false
}

func (r *eleventyRun) projectRel(inputRel string) string {
	return path.Join(r.cfg.input, inputRel)
}

// special reports whether a project file is in the includes, layouts, or
// data directory.
func (r *eleventyRun) special(rel string) bool {
	return underDir(rel, r.projectRel(r.cfg.data)) || underDir(rel, r.projectRel(r.cfg.includes)) || underDir(rel, r.projectRel(r.cfg.layouts))
}

func underDir(rel, dir string) bool {
	return dir != "." && (rel == dir || strings.HasPrefix(rel, dir+"/"))
}

// readGlobalData migrates the global data directory to Nida's data/ and
// returns its values for evaluating permalinks and content variables.
func (r *eleventyRun) readGlobalData() (map[string]any, map[string]string) {
	global := map[string]any{}
	files := map[string]string{}
	dataDir := r.projectRel(r.cfg.data)
	_ = r.p.Walk(dataDir, func(rel string) error {
		inner := strings.TrimPrefix(rel, dataDir+"/")
		ext := path.Ext(inner)
		key := strings.TrimSuffix(inner, ext)
		raw, err := r.p.ReadFile(rel)
		if err != nil {
			return nil
		}
		var value any
		switch ext {
		case ".json", ".yaml", ".yml":
			if yaml.Unmarshal(raw, &value) != nil {
				r.p.Report.Add(Manual, rel, 0, "Global data file could not be parsed; it was copied unchanged.")
			}
			r.p.Copy(path.Join("data", inner), rel)
		case ".js", ".cjs", ".mjs":
			value, err = parseJSDataFile(string(raw))
			if err != nil {
				r.p.Preserve(rel)
				r.p.Report.Add(Unsupported, rel, 0, "JavaScript data file computes its data (%v), so it was not migrated. Write its output to `data/%s.json`.", err, key)
				return nil
			}
			out, _ := json.MarshalIndent(value, "", "  ")
			r.p.Write(path.Join("data", key+".json"), append(out, '\n'), rel)
			r.p.Report.Add(Changed, rel, 0, "JavaScript data file exports a plain value, so it became `data/%s.json`.", key)
		default:
			r.p.Preserve(rel)
			r.p.Report.Add(Unsupported, rel, 0, "Global data in `%s` files is not supported; Nida reads TOML, JSON, and YAML from `data/`.", ext)
			return nil
		}
		if value != nil {
			setNested(global, strings.Split(key, "/"), value)
			files[key] = rel
		}
		return nil
	})
	return global, files
}

func setNested(m map[string]any, keys []string, value any) {
	for _, k := range keys[:len(keys)-1] {
		next, ok := m[k].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[k] = next
		}
		m = next
	}
	m[keys[len(keys)-1]] = value
}

// planSiteConfig fills config.toml from the site metadata data file.
func (r *eleventyRun) planSiteConfig() {
	c, rep := r.p.Config, r.p.Report
	var meta map[string]any
	metaName := ""
	for _, name := range []string{"metadata", "site", "siteData", "meta", "siteMetadata"} {
		if m, ok := r.global[name].(map[string]any); ok {
			meta, metaName = m, name
			break
		}
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if s, ok := meta[k].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
		return ""
	}
	file := r.globalFiles[metaName]
	if title := str("title", "name", "siteName"); title != "" {
		c.Set("title", title)
	} else if pkg, err := r.p.ReadFile("package.json"); err == nil {
		var info map[string]any
		if json.Unmarshal(pkg, &info) == nil {
			if name, ok := info["name"].(string); ok && name != "" {
				c.Set("title", name)
				rep.Add(Review, "package.json", 0, "No site title was found in the data files, so `title` is the package name `%s`.", name)
			}
		}
	}
	if desc := str("description", "subtitle", "tagline"); desc != "" {
		c.Set("description", desc)
	}
	if lang := str("language", "lang", "locale"); lang != "" {
		c.Set("language", lang)
	}
	switch a := meta["author"].(type) {
	case string:
		c.Set("author", a)
	case map[string]any:
		if name, ok := a["name"].(string); ok {
			c.Set("author", name)
		}
	}
	base := str("url", "baseUrl", "baseURL", "base_url", "siteUrl", "siteURL")
	prefix := strings.Trim(r.cfg.pathPrefix, "/")
	switch {
	case base != "":
		if prefix != "" && !strings.HasSuffix(strings.TrimSuffix(base, "/"), "/"+prefix) {
			base = strings.TrimSuffix(base, "/") + "/" + prefix + "/"
			rep.Add(Changed, file, 0, "`base_url` includes the path prefix `/%s/`, so Nida serves the site under it.", prefix)
		}
		c.Set("base_url", base)
	case prefix != "":
		c.Set("base_url", "http://127.0.0.1:1702/"+prefix+"/")
		rep.Add(Manual, "", 0, "The site's URL was not found, so `base_url` is the local preview address with the path prefix `/%s/`. Set it to the real URL before deploying.", prefix)
	}
	if meta != nil {
		rep.Add(Changed, file, 0, "Site title, description, language, author, and URL came from this data file.")
	}
}

// files lists the project files Eleventy would consider, sorted.
func (r *eleventyRun) files() ([]string, error) {
	var files []string
	output := r.cfg.output
	err := r.p.Walk(".", func(rel string) error {
		if underDir(rel, output) || r.ignored(rel) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	slices.Sort(files)
	return files, err
}

func (r *eleventyRun) isTemplate(name string) (string, bool) {
	for _, ext := range append([]string{"md", "html"}, eleventyTemplateExts...) {
		if strings.HasSuffix(name, "."+ext) {
			return ext, slices.Contains(r.cfg.formats, ext)
		}
	}
	return "", false
}

// readCascade reads directory and template data files.
func (r *eleventyRun) readCascade(files []string) {
	templates := map[string]bool{}
	for _, rel := range files {
		if inner, ok := r.inputRel(rel); ok {
			if _, isTmpl := r.isTemplate(inner); isTmpl && !r.special(rel) {
				templates[path.Join(path.Dir(inner), eleventyStem(inner))] = true
				for dir := path.Dir(inner); ; dir = path.Dir(dir) {
					r.templateCount[dir]++
					if dir == "." {
						break
					}
				}
			}
		}
	}
	for _, rel := range files {
		inner, ok := r.inputRel(rel)
		if !ok || r.special(rel) {
			continue
		}
		dir, base := path.Dir(inner), path.Base(inner)
		var stem string
		switch {
		case strings.HasSuffix(base, ".11tydata.json"):
			stem = strings.TrimSuffix(base, ".11tydata.json")
		case strings.HasSuffix(base, ".11tydata.js") || strings.HasSuffix(base, ".11tydata.cjs") || strings.HasSuffix(base, ".11tydata.mjs"):
			r.p.Preserve(rel)
			r.p.Report.Add(Manual, rel, 0, "JavaScript directory data was not applied. Copy the values it sets into the front matter of the pages it covers.")
			r.dataFile[inner] = rel
			continue
		case strings.HasSuffix(base, ".json"):
			stem = strings.TrimSuffix(base, ".json")
		default:
			continue
		}
		var target string
		var store map[string]map[string]any
		switch {
		case dir != "." && stem == path.Base(dir):
			target, store = dir, r.dirData
		case templates[path.Join(dir, stem)]:
			target, store = path.Join(dir, stem), r.fileData
		default:
			continue
		}
		raw, err := r.p.ReadFile(rel)
		if err != nil {
			continue
		}
		var values map[string]any
		if err := yaml.Unmarshal(raw, &values); err != nil {
			r.p.Report.Add(Manual, rel, 0, "Data file could not be parsed (%v); its values were not applied.", err)
			continue
		}
		// .11tydata.json wins over the plain .json file.
		if existing, ok := store[target]; ok && strings.HasSuffix(base, ".11tydata.json") {
			values = mergeData(existing, values)
		} else if ok {
			values = mergeData(values, existing)
		}
		store[target] = values
		r.dataFile[inner] = rel
		r.dataSources[target] = append(r.dataSources[target], rel)
	}
}

// mergeData merges b over a the way Eleventy's data cascade does: arrays
// are concatenated, objects merged, and other values replaced.
func mergeData(a, b map[string]any) map[string]any {
	out := maps.Clone(a)
	if out == nil {
		out = map[string]any{}
	}
	for k, v := range b {
		switch nv := v.(type) {
		case []any:
			if old, ok := out[k].([]any); ok {
				out[k] = append(slices.Clone(old), nv...)
				continue
			}
		case map[string]any:
			if old, ok := out[k].(map[string]any); ok {
				out[k] = mergeData(old, nv)
				continue
			}
		}
		out[k] = v
	}
	return out
}

// cascade returns the directory and template data for an input-relative
// template, outermost first, and the data files it came from.
func (r *eleventyRun) cascade(inner string) (map[string]any, []string) {
	var dirs []string
	for dir := path.Dir(inner); dir != "."; dir = path.Dir(dir) {
		dirs = append([]string{dir}, dirs...)
	}
	data := map[string]any{}
	var sources []string
	for _, dir := range dirs {
		if values, ok := r.dirData[dir]; ok {
			data = mergeData(data, values)
			sources = append(sources, dir)
		}
	}
	key := path.Join(path.Dir(inner), eleventyStem(inner))
	if values, ok := r.fileData[key]; ok {
		data = mergeData(data, values)
		sources = append(sources, key)
	}
	return data, sources
}

// passthroughDest returns where a passthrough copy puts a file, if any
// rule covers it. Paths are relative to the project root; a copy without
// an explicit destination keeps its path minus the input directory.
func (r *eleventyRun) passthroughDest(rel string) (string, bool) {
	for _, rule := range r.cfg.passthrough {
		src := rule.src
		var rest string
		switch {
		case src == ".":
			continue
		case rel == src:
			rest = ""
		case strings.HasPrefix(rel, src+"/"):
			rest = strings.TrimPrefix(rel, src+"/")
		case strings.ContainsAny(src, "*?[") && globMatch(src, rel):
			rest = path.Base(rel)
		default:
			continue
		}
		if rule.set {
			if rest == "" {
				return rule.dest, true
			}
			return path.Join(rule.dest, rest), true
		}
		if inner, ok := r.inputRel(rel); ok {
			return inner, true
		}
		return rel, true
	}
	return "", false
}

// planFile decides what one project file becomes.
func (r *eleventyRun) planFile(rel string) error {
	p := r.p
	if dest, ok := r.passthroughDest(rel); ok {
		if dest == "" || dest == "." || p.Planned(path.Join("static", dest)) {
			return nil // a shortcode's image may have been copied already
		}
		p.Copy(path.Join("static", dest), rel)
		return nil
	}
	switch {
	case underDir(rel, r.projectRel(r.cfg.data)):
		return nil // migrated with the global data
	case underDir(rel, r.projectRel(r.cfg.includes)) || underDir(rel, r.projectRel(r.cfg.layouts)):
		p.Preserve(rel)
		return nil
	}
	inner, ok := r.inputRel(rel)
	if !ok || slices.Contains(eleventyConfigFiles, rel) || rel == "package.json" || rel == "package-lock.json" {
		return nil
	}
	if r.dataFile[inner] != "" {
		return nil
	}
	ext, isTmpl := r.isTemplate(inner)
	switch {
	case ext == "":
		r.unpublished = append(r.unpublished, rel)
		return nil
	case !isTmpl:
		r.unpublished = append(r.unpublished, rel)
		return nil
	case ext == "md" || ext == "html":
		return r.planPage(rel, inner, ext)
	default:
		p.Preserve(rel)
		r.listings = append(r.listings, rel)
		return nil
	}
}

var eleventyFenceVariant = regexp.MustCompile(`^---(json|js|javascript|yaml|toml)[ \t]*\r?\n`)

// readTemplate parses a template's front matter, including Eleventy's
// ---json and ---toml variants.
func readTemplate(data []byte) (*Document, string, error) {
	s := string(data)
	s = strings.TrimPrefix(s, "\xef\xbb\xbf")
	if m := eleventyFenceVariant.FindStringSubmatch(s); m != nil {
		rest := s[len(m[0]):]
		raw, body, _, ok := cutClosing(strings.ReplaceAll(rest, "\r\n", "\n"), "---")
		if !ok {
			return nil, m[1], fmt.Errorf("missing closing --- front matter delimiter")
		}
		switch m[1] {
		case "json":
			doc, err := ParseDocument([]byte(strings.TrimSpace(raw) + "\n" + body))
			return doc, "json", err
		case "toml":
			doc, err := ParseDocument([]byte("+++\n" + raw + "+++\n" + body))
			return doc, "toml", err
		case "yaml":
			doc, err := ParseDocument([]byte("---\n" + raw + "---\n" + body))
			return doc, "yaml", err
		default:
			return nil, m[1], fmt.Errorf("JavaScript front matter")
		}
	}
	doc, err := ParseDocument(data)
	return doc, "", err
}

func (r *eleventyRun) planPage(rel, inner, ext string) error {
	p, rep := r.p, r.p.Report
	data, err := p.ReadFile(rel)
	if err != nil {
		return err
	}
	doc, variant, err := readTemplate(data)
	if err != nil {
		p.Preserve(rel)
		sev := Manual
		if variant == "js" || variant == "javascript" {
			sev = Unsupported
		}
		rep.Add(sev, rel, 1, "Front matter could not be read (%v), so the page was not migrated. The original is in `%s/`.", err, preservedDir)
		return nil
	}
	MoveCustomPath(doc, rep, rel)
	if variant == "json" || variant == "toml" {
		rep.Add(Changed, rel, 1, "`---%s` front matter was rewritten in a form Nida reads.", variant)
	}
	if ext == "html" {
		if liquidAny.MatchString(doc.Body) {
			p.Preserve(rel)
			r.listings = append(r.listings, rel)
			return nil
		}
		doc.SetBody(htmlAsMarkdown(doc.Body))
		rep.Add(Review, rel, 0, "HTML page became Markdown with raw HTML; blank lines outside `<pre>` were removed so it stays one HTML block. Check how it renders.")
	}
	stem := eleventyStem(inner)
	if stem == "404" {
		p.Preserve(rel)
		rep.Add(Review, rel, 0, "Nida writes the 404 page from `templates/404.html`; move this page's text there.")
		return nil
	}

	cascade, sources := r.cascade(inner)
	own := maps.Clone(doc.Values())
	if tags, ok := own["tags"].(string); ok {
		own["tags"] = []any{tags}
	}
	if tags, ok := cascade["tags"].(string); ok {
		cascade["tags"] = []any{tags}
	}
	merged := mergeData(cascade, own)
	if len(sources) > 0 {
		var applied []string
		for k := range cascade {
			if !doc.Has(k) && k != "permalink" {
				applied = append(applied, k)
			}
		}
		slices.Sort(applied)
		for _, src := range sources {
			for _, file := range r.dataSources[src] {
				rep.Add(Changed, file, 0, "Data file values were copied into the front matter of the pages they apply to, since Nida has no data cascade.")
			}
		}
		for _, k := range applied {
			if k == "tags" {
				continue // merged below
			}
			_ = doc.Set(k, cascade[k])
		}
	}

	pg := eleventyPage{inputRel: inner, data: merged}
	r.convertDate(doc, rel, &pg)

	// Destination. Nida treats dir/index.md as a page bundle that hides its
	// siblings, so an index page that shares its directory becomes the
	// section page instead.
	destInner := strings.TrimSuffix(inner, "."+ext) + ".md"
	isSection := false
	if stem == "index" {
		dir := path.Dir(inner)
		if dir == "." || r.hasSiblings(inner) {
			destInner = path.Join(dir, "_index.md")
			isSection = true
		}
	}
	dest := path.Join("content", destInner)
	if !isSection {
		r.sectionDirs[path.Dir(inner)] = true
	}

	pp := &eleventyPlanned{origin: rel, dest: dest, page: pg, doc: doc}
	r.convertPermalink(pp, merged, isSection)
	r.convertTags(doc, rel, merged, inner)
	r.convertKeys(doc, rel)

	body := &eleventyBody{p: p, cfg: r.cfg, file: rel, doc: doc, page: pg, global: r.global, asset: func(src string) (string, bool) { return r.asset(src, rel) }}
	body.convert()

	if doc.Format == NoFrontMatter || !doc.Has("title") {
		title := firstHeading(doc.Body)
		if title == "" {
			title = titleFromSlug(pg.fileSlug())
		}
		if title == "" {
			title = "Home"
		}
		if !doc.Has("title") {
			_ = doc.Set("title", title)
			rep.Add(Changed, rel, 0, "Added a `title`, taken from the first heading or the file name.")
		}
	}
	if isSection {
		if dest != "content/_index.md" {
			rep.Add(Review, rel, 0, "`%s` became the section page `%s`, since Nida would otherwise treat it as a page bundle and hide the other pages in its directory.", inner, dest)
		}
		p.Page(dest, doc, rel, "")
		return nil
	}
	r.pages = append(r.pages, pp)
	return nil
}

var markdownHeading = regexp.MustCompile(`(?m)^#\s+(.+?)\s*#*\s*$`)

func firstHeading(body string) string {
	if m := markdownHeading.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

// hasSiblings reports whether an index template's directory holds other
// content templates, directly or in subdirectories.
func (r *eleventyRun) hasSiblings(inner string) bool {
	return r.templateCount[path.Dir(inner)] > 1
}

// convertDate handles Eleventy's special date values and dates in file
// names, then normalizes the format.
func (r *eleventyRun) convertDate(doc *Document, rel string, pg *eleventyPage) {
	rep := r.p.Report
	if v, ok := doc.Get("date"); ok {
		if s, isStr := v.(string); isStr {
			switch strings.ToLower(strings.TrimSpace(s)) {
			case "created", "last modified", "git created", "git last modified":
				t, how := r.specialDate(rel, strings.ToLower(strings.TrimSpace(s)))
				if t.IsZero() {
					doc.Delete("date")
					rep.Add(Manual, rel, 0, "`date: %s` could not be worked out; set the date by hand.", s)
				} else {
					_ = doc.Set("date", DateValue(doc, t, false))
					rep.Add(Review, rel, 0, "`date: %s` was fixed to %s, from %s, because Nida reads dates only from front matter.", s, t.Format("2006-01-02"), how)
				}
			}
		}
	}
	if !doc.Has("date") {
		if m := eleventyDateInName.FindString(path.Base(rel)); m != "" {
			if t, _, err := ParseDate(m); err == nil {
				_ = doc.Set("date", DateValue(doc, t, true))
				rep.Add(Changed, rel, 0, "Set `date` from the file name, which Eleventy reads and Nida does not.")
			}
		}
	}
	NormalizeDate(doc, "date", rep, rel)
	if v, ok := doc.Get("date"); ok {
		if t, _, err := ParseDate(v); err == nil {
			pg.date, pg.hasDate = t, true
		}
	}
}

// specialDate resolves "Created", "Last Modified", and their git forms.
func (r *eleventyRun) specialDate(rel, kind string) (time.Time, string) {
	if strings.HasPrefix(kind, "git") {
		args := []string{"-C", r.p.Root, "log", "-1", "--format=%cI", "--", rel}
		if kind == "git created" {
			args = []string{"-C", r.p.Root, "log", "--diff-filter=A", "--follow", "--format=%cI", "--", rel}
		}
		if out, err := exec.Command("git", args...).Output(); err == nil {
			lines := strings.Fields(string(out))
			if len(lines) > 0 {
				if t, err := time.Parse(time.RFC3339, lines[len(lines)-1]); err == nil {
					return t, "git history"
				}
			}
		}
	}
	info, err := os.Stat(filepath.Join(r.p.Root, filepath.FromSlash(rel)))
	if err != nil {
		return time.Time{}, ""
	}
	return info.ModTime().UTC().Truncate(time.Second), "the file's modification time"
}

// convertPermalink works out the page's old URL and drops the permalink.
func (r *eleventyRun) convertPermalink(pp *eleventyPlanned, merged map[string]any, isSection bool) {
	doc, rel, rep := pp.doc, pp.origin, r.p.Report
	value, has := merged["permalink"]
	fromCascade := has && !doc.Has("permalink")
	doc.Delete("permalink")
	if !has {
		pp.oldURL = pp.page.defaultURL()
		pp.defaultURL = true
		return
	}
	if m, ok := value.(map[string]any); ok {
		if build, ok := m["build"]; ok {
			value = build
		} else {
			rep.Add(Unsupported, rel, 0, "`permalink` has no `build` entry (serverless or custom permalinks are not supported); the page uses Nida's URL.")
			return
		}
	}
	switch v := value.(type) {
	case bool:
		if !v {
			_ = doc.Set("draft", true)
			rep.Add(Review, rel, 0, "`permalink: false` meant Eleventy did not write this page, so it is a draft, which Nida does not build either.")
		} else {
			pp.oldURL = pp.page.defaultURL()
		}
		return
	case string:
		url, err := evalPermalink(v, pp.page)
		if err != nil {
			rep.Add(Manual, rel, 0, "`permalink: %s` could not be worked out (%v), so the page uses Nida's URL. Set `path` to keep the old URL.", v, err)
			return
		}
		if isSection && strings.Trim(url, "/") != strings.Trim(path.Dir(pp.page.inputRel), "/.") {
			rep.Add(Manual, rel, 0, "The section page had the URL `%s`, which Nida section pages cannot keep. Add a redirect for it on your web host.", url)
			return
		}
		if ext := path.Ext(strings.TrimSuffix(url, "/")); ext != "" && ext != ".html" && ext != ".htm" {
			rep.Add(Review, rel, 0, "The page was written as `%s`, a non-HTML file; Nida writes it as an HTML page at that path.", url)
		}
		pp.oldURL = url
		if fromCascade {
			pp.permalink = v
			pp.permalinkDir = r.permalinkSourceDir(pp.page.inputRel)
		} else {
			rep.Add(Changed, rel, 0, "Removed `permalink`, which Nida does not read; the URL check below keeps the page at its old URL.")
		}
	default:
		rep.Add(Unsupported, rel, 0, "`permalink` of type %T is not supported; the page uses Nida's URL.", v)
	}
}

// permalinkSourceDir returns the directory whose data file sets a page's
// permalink.
func (r *eleventyRun) permalinkSourceDir(inner string) string {
	found := ""
	for dir := path.Dir(inner); dir != "."; dir = path.Dir(dir) {
		if _, ok := r.dirData[dir]["permalink"]; ok && found == "" {
			found = dir
		}
	}
	return found
}

// convertTags splits Eleventy tags into taxonomy terms and collection
// markers, which are kept as extra data for porting templates.
func (r *eleventyRun) convertTags(doc *Document, rel string, merged map[string]any, inner string) {
	raw, ok := merged["tags"]
	if !ok {
		return
	}
	var tags []string
	switch v := raw.(type) {
	case string:
		tags = []string{v}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				tags = append(tags, s)
			}
		}
	}
	dirName := path.Base(path.Dir(inner))
	var terms, collections []string
	for _, tag := range tags {
		lower := strings.ToLower(tag)
		if slices.Contains(eleventyCollectionTags, lower) || lower == strings.ToLower(dirName) || slices.Contains(r.cfg.collections, tag) {
			collections = appendUnique(collections, tag)
		} else {
			terms = appendUnique(terms, tag)
		}
	}
	doc.Delete("tags")
	if len(terms) > 0 {
		_ = doc.Set("tags", terms)
		r.tagsUsed = true
	}
	if len(collections) > 0 {
		_ = doc.Set("collections", collections)
		r.p.Report.Add(Review, rel, 0, "Tags used to build Eleventy collections (%s) moved to `collections` in extra data; the rest are `tags` terms.", strings.Join(collections, ", "))
	}
}

// convertKeys handles the remaining Eleventy-specific front matter keys.
func (r *eleventyRun) convertKeys(doc *Document, rel string) {
	rep := r.p.Report
	if layout := doc.String("layout"); layout != "" {
		name := eleventyStem(layout)
		if !doc.Rename("layout", "template") {
			doc.Delete("layout")
		}
		_ = doc.Set("template", name)
		if !slices.Contains([]string{"post", "page"}, name) {
			rep.Add(Review, rel, 0, "Content asks for template `%s`, which the starter templates do not have, so it renders with the default template until you port it.", name)
		}
	} else if doc.Has("layout") {
		doc.Delete("layout") // layout: false or null
	}
	if nav, ok := doc.Get("eleventyNavigation"); ok {
		if m, ok := nav.(map[string]any); ok {
			if order, ok := m["order"]; ok && !doc.Has("weight") {
				if n, isNum := order.(int); isNum {
					_ = doc.Set("weight", n)
				}
			}
		}
		rep.Add(Review, rel, 0, "`eleventyNavigation` is kept in extra data, and its `order` became `weight`. Nida's navigation comes from sections.")
	}
	if doc.Has("eleventyComputed") {
		rep.Add(Unsupported, rel, 0, "`eleventyComputed` values are computed when Eleventy builds and were not migrated; they stay in extra data unevaluated.")
	}
	if v, ok := doc.Get("eleventyExcludeFromCollections"); ok && v != false {
		rep.Add(Unsupported, rel, 0, "`eleventyExcludeFromCollections` has no Nida equivalent; the page is listed in its section.")
	}
	if doc.Has("pagination") {
		rep.Add(Unsupported, rel, 0, "`pagination` over data is not supported; Nida paginates section lists with `paginate_by`.")
	}
	for _, key := range []string{"eleventyImport", "eleventyDataSchema", "templateEngineOverride", "dynamicPermalink", "permalinkBypassOutputDir", "renderData"} {
		if doc.Delete(key) {
			rep.Add(Unsupported, rel, 0, "`%s` controlled Eleventy's build and was dropped.", key)
		}
	}
}

// asset resolves an image path from a shortcode, copying the file into
// static/ and returning its URL.
func (r *eleventyRun) asset(src, rel string) (string, bool) {
	if strings.Contains(src, "://") || strings.HasPrefix(src, "data:") {
		return src, true
	}
	clean := strings.TrimPrefix(src, "./")
	var candidates []string
	if strings.HasPrefix(src, "/") {
		candidates = []string{r.projectRel(strings.TrimPrefix(src, "/")), strings.TrimPrefix(src, "/")}
	} else {
		candidates = []string{path.Join(path.Dir(rel), clean), clean, r.projectRel(clean)}
	}
	for _, cand := range candidates {
		cand = path.Clean(cand)
		if strings.HasPrefix(cand, "..") || !r.p.Exists(cand) {
			continue
		}
		destRel := cand
		if inner, ok := r.inputRel(cand); ok {
			destRel = inner
		}
		dest := path.Join("static", destRel)
		if !r.p.Planned(dest) {
			r.p.Copy(dest, cand)
		}
		return "/" + destRel, true
	}
	return src, false
}

// planPermalinks turns permalink templates shared through directory data
// into Nida [permalinks] patterns where they translate, so pages do not
// each need a path, and keeps default Eleventy URLs by setting slug where
// Nida would derive a different one.
func (r *eleventyRun) planPermalinks() {
	type group struct {
		tmpl  string
		pages []*eleventyPlanned
	}
	groups := map[string]*group{}
	for _, pp := range r.pages {
		section := strings.TrimPrefix(path.Dir(pp.dest), "content")
		section = strings.TrimPrefix(section, "/")
		if pp.defaultURL {
			// Eleventy keeps the file name as it is; Nida lowercases and
			// hyphenates it unless slug is set.
			stem := eleventyStem(pp.page.inputRel)
			if path.Base(pp.dest) == "index.md" {
				stem = path.Base(path.Dir(pp.page.inputRel))
			}
			if !pp.doc.Has("slug") && content.DeriveSlug(stem) != stem {
				_ = pp.doc.Set("slug", stem)
				r.p.Report.Add(Changed, pp.origin, 0, "Set `slug` to the file name, which Eleventy uses in the URL as it is.")
			}
			continue
		}
		if pp.permalink == "" || section == "" || pp.permalinkDir == "" {
			continue
		}
		g, ok := groups[section]
		if !ok {
			g = &group{tmpl: pp.permalink}
			groups[section] = g
		}
		if g.tmpl != pp.permalink {
			g.tmpl = "\x00mixed"
		}
		g.pages = append(g.pages, pp)
	}
	for _, section := range sortedKeys(groups) {
		g := groups[section]
		pattern, slugFor, ok := permalinkPattern(g.tmpl)
		if !ok {
			if g.tmpl != "\x00mixed" {
				r.p.Report.Add(Review, strings.Join(r.dataSources[g.pages[0].permalinkDir], ", "), 0, "The permalink `%s` has no Nida permalink pattern, so each page keeps its URL with `path`.", g.tmpl)
			}
			continue
		}
		consistent := true
		slugs := map[*eleventyPlanned]string{}
		for _, pp := range g.pages {
			if slugFor == nil {
				continue
			}
			slug, ok := slugFor(pp.page)
			if !ok || slug == "" {
				consistent = false
				break
			}
			slugs[pp] = slug
		}
		if !consistent {
			continue
		}
		r.p.Config.Table("permalinks").Set(section, pattern)
		for _, pp := range g.pages {
			stem := eleventyStem(pp.page.inputRel)
			if slug, ok := slugs[pp]; ok && slug != content.DeriveSlug(stem) && !pp.doc.Has("slug") {
				_ = pp.doc.Set("slug", slug)
			}
		}
		r.p.Report.Add(Changed, "", 0, "The `%s` permalink `%s` became the Nida permalink pattern `%s`.", section, g.tmpl, pattern)
	}
}

// planListings maps template pages, usually listings and feeds, onto the
// Nida features that replace them.
func (r *eleventyRun) planListings() {
	p, rep := r.p, r.p.Report
	slices.Sort(r.listings)
	for _, rel := range r.listings {
		inner, _ := r.inputRel(rel)
		data, err := p.ReadFile(rel)
		if err != nil {
			continue
		}
		doc, _, err := readTemplate(data)
		var values map[string]any
		body := string(data)
		if err == nil {
			values = doc.Values()
			body = doc.Body
		}
		pg := eleventyPage{inputRel: inner, data: values}
		url := pg.defaultURL()
		if pl, ok := values["permalink"].(string); ok {
			if u, err := evalPermalink(pl, pg); err == nil {
				url = u
			} else {
				url = pl
			}
		} else if values["permalink"] == false {
			continue // a template that writes no page
		}
		title, _ := values["title"].(string)
		switch {
		case strings.HasSuffix(url, ".xml") && (strings.Contains(body, "<feed") || strings.Contains(body, "<rss")):
			r.planFeed(rel, url, strings.Contains(body, "<rss"))
		case strings.Contains(url, "sitemap"):
			rep.Add(Changed, rel, 0, "Nida writes `sitemap.xml` itself (`[sitemap]`).")
		case strings.HasSuffix(url, "robots.txt"):
			rep.Add(Changed, rel, 0, "Nida writes `robots.txt` itself (`[robots]`).")
		case values["pagination"] != nil && strings.Contains(url, "tag"):
			rep.Add(Review, rel, 0, "Tag pages are replaced by Nida's taxonomy pages at `/tags/<slug>/`.")
			r.tagsUsed = true
		case url == "/" || strings.Trim(url, "/") != "" && r.sectionDirs[strings.Trim(url, "/")]:
			dir := strings.Trim(url, "/")
			dest := path.Join("content", dir, "_index.md")
			if !p.Planned(dest) {
				section := NewDocument(TOML, "")
				if title == "" {
					title = titleFromSlug(path.Base(dir))
					if dir == "" {
						title = "Home"
					}
				}
				_ = section.Set("title", title)
				if pb, ok := values["pagination"].(map[string]any); ok {
					if size, ok := pb["size"].(int); ok {
						_ = section.Set("paginate_by", size)
					}
				}
				p.Page(dest, section, rel, "")
			}
			rep.Add(Review, rel, 0, "This template page listed content at `%s`; the Nida section page there replaces it, rendered by `templates/%s.html`.", url, map[bool]string{true: "index", false: "list"}[url == "/"])
		default:
			rep.Add(Manual, rel, 0, "Template page (%s) at `%s` was not migrated; recreate it as a Nida template or a Markdown page. The original is in `%s/`.", path.Ext(rel), url, preservedDir)
		}
	}
}

func (r *eleventyRun) planFeed(rel, url string, rss bool) {
	c, rep := r.p.Config, r.p.Report
	name := strings.TrimPrefix(url, "/")
	if rss {
		t := c.Table("rss")
		t.Set("enabled", true)
		t.Set("filename", name)
		if v, ok := c.Table("feed").Get("enabled"); !ok || v != true {
			c.Table("feed").Set("enabled", false)
		}
	} else {
		t := c.Table("feed")
		t.Set("enabled", true)
		t.Set("filename", name)
	}
	rep.Add(Changed, rel, 0, "The feed at `%s` is written by Nida's `[%s]`, at the same path.", url, map[bool]string{true: "rss", false: "feed"}[rss])
}
