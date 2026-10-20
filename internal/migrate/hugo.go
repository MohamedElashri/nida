package migrate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
	"text/template"

	"github.com/alecthomas/chroma/v2/styles"
	toml "github.com/pelletier/go-toml/v2"
	yaml "go.yaml.in/yaml/v3"
)

// hugoSource migrates Hugo sites. Hugo shares Go templates, TOML and YAML
// front matter, page bundles, and the {{< >}} shortcode syntax with Nida,
// so content carries over closely; config, permalinks, and a handful of
// built-in shortcodes need translating.
type hugoSource struct{}

func (hugoSource) Name() string  { return "hugo" }
func (hugoSource) Title() string { return "Hugo" }

func (hugoSource) Detect(root string) int {
	p := &Plan{Root: root}
	for _, name := range []string{"hugo.toml", "hugo.yaml", "hugo.yml", "hugo.json"} {
		if p.Exists(name) {
			return 10
		}
	}
	if p.Exists("config/_default") {
		return 8
	}
	score := 0
	for _, name := range []string{"config.toml", "config.yaml", "config.yml", "config.json"} {
		data, err := p.ReadFile(name)
		if err != nil {
			continue
		}
		cfg, err := decodeConfig(name, data)
		if err != nil {
			continue
		}
		if _, ok := lowerKeys(cfg)["baseurl"]; ok {
			score = 3
		}
		if _, ok := cfg["base_url"]; ok {
			return 0 // Zola
		}
	}
	if score == 0 {
		return 0
	}
	for _, dir := range []string{"archetypes", "layouts", "content"} {
		if p.Exists(dir) {
			score++
		}
	}
	return score
}

// hugoSite is what the Hugo source learns from the config, shared by the
// config, content, and URL steps.
type hugoSite struct {
	p   *Plan
	cfg map[string]any // merged config, keys lowercased

	contentDir  string
	staticDirs  []string
	dataDir     string
	i18nDir     string
	layoutDir   string
	archetypes  string
	assetDir    string
	themes      []string // source-relative theme directories
	defaultLang string
	// otherLangs are configured languages other than the default; files
	// such as post.fr.md belong to them.
	otherLangs []string
	langDirs   []string // content directories of other languages

	pathToLower bool
	uglyURLs    bool
	unsafeHTML  bool
	// pagePermalinks maps a top-level section to its Hugo pattern.
	pagePermalinks map[string]string
	params         map[string]any
	// taxonomies maps singular names to plural ones, as Hugo's config does.
	taxonomies map[string]string
}

func (hugoSource) Plan(p *Plan) error {
	h := &hugoSite{p: p}
	if err := h.loadConfig(); err != nil {
		return err
	}
	h.planConfig()
	if err := h.planContent(); err != nil {
		return err
	}
	h.planStatic()
	h.planData()
	h.planI18n()
	h.planArchetypes()
	h.planLayouts()
	return nil
}

// decodeConfig reads a TOML, YAML, or JSON config file.
func decodeConfig(name string, data []byte) (map[string]any, error) {
	out := map[string]any{}
	var err error
	switch path.Ext(name) {
	case ".toml":
		err = toml.Unmarshal(data, &out)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &out)
	case ".json":
		err = json.Unmarshal(data, &out)
	default:
		return nil, fmt.Errorf("unsupported config format %q", name)
	}
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	return out, nil
}

// lowerKeys lowercases map keys recursively. Hugo config keys are case
// insensitive, including params, which Hugo lowercases too.
func lowerKeys(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[strings.ToLower(k)] = lowerValue(v)
	}
	return out
}

func lowerValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return lowerKeys(x)
	case []any:
		items := make([]any, len(x))
		for i, item := range x {
			items[i] = lowerValue(item)
		}
		return items
	case []map[string]any:
		items := make([]any, len(x))
		for i, item := range x {
			items[i] = lowerKeys(item)
		}
		return items
	}
	return v
}

// mergeMissing copies keys from src that dst lacks, merging nested maps.
func mergeMissing(dst, src map[string]any) {
	for k, v := range src {
		existing, ok := dst[k]
		if !ok {
			dst[k] = v
			continue
		}
		if em, ok := existing.(map[string]any); ok {
			if sm, ok := v.(map[string]any); ok {
				mergeMissing(em, sm)
			}
		}
	}
}

// loadConfig reads the root config file and config/_default/, and the
// theme configs for params defaults.
func (h *hugoSite) loadConfig() error {
	p := h.p
	merged := map[string]any{}
	found := false
	for _, name := range []string{"hugo.toml", "hugo.yaml", "hugo.yml", "hugo.json", "config.toml", "config.yaml", "config.yml", "config.json"} {
		data, err := p.ReadFile(name)
		if err != nil {
			continue
		}
		cfg, err := decodeConfig(name, data)
		if err != nil {
			return err
		}
		merged = lowerKeys(cfg)
		found = true
		break // Hugo reads the first one it finds
	}

	// config/_default/<key>.<ext> holds the value of <key>; hugo.* and
	// config.* hold root keys. Files named key.<lang>.toml are language
	// specific and belong to other languages unless they match the default.
	dirFiles := map[string]bool{}
	_ = p.Walk("config/_default", func(rel string) error {
		dirFiles[rel] = true
		return nil
	})
	for _, rel := range sortedKeys(dirFiles) {
		data, err := p.ReadFile(rel)
		if err != nil {
			return err
		}
		cfg, err := decodeConfig(rel, data)
		if err != nil {
			p.Report.Add(Manual, rel, 0, "Config file could not be read (%v) and was skipped.", err)
			continue
		}
		cfg = lowerKeys(cfg)
		found = true
		base := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		key, lang, hasLang := strings.Cut(strings.ToLower(base), ".")
		switch {
		case key == "hugo" || key == "config":
			mergeMissing(merged, cfg)
		case hasLang:
			// menus.fr.toml and similar: keep only the default language,
			// decided once languages are known.
			if existing, ok := merged["_langfiles"].(map[string]any); ok {
				existing[key+"."+lang] = cfg
			} else {
				merged["_langfiles"] = map[string]any{key + "." + lang: cfg}
			}
		default:
			if existing, ok := merged[key].(map[string]any); ok {
				mergeMissing(existing, cfg)
			} else if _, ok := merged[key]; !ok {
				merged[key] = cfg
			}
		}
	}
	if !found {
		return fmt.Errorf("no hugo.toml, config.toml, or config/_default/ found")
	}
	for _, env := range []string{"production", "staging", "development"} {
		if p.Exists("config/" + env) {
			p.Report.Add(Review, "config/"+env, 0, "Environment config in `config/%s/` was not merged; only `config/_default/` and the root config were read. Copy any settings you need into config.toml.", env)
		}
	}
	h.cfg = merged

	h.contentDir = h.str("contentdir", "content")
	h.dataDir = h.str("datadir", "data")
	h.i18nDir = h.str("i18ndir", "i18n")
	h.layoutDir = h.str("layoutdir", "layouts")
	h.archetypes = h.str("archetypedir", "archetypes")
	h.assetDir = h.str("assetdir", "assets")
	h.staticDirs = stringList(merged["staticdir"])
	if len(h.staticDirs) == 0 {
		h.staticDirs = []string{"static"}
	}
	themesDir := h.str("themesdir", "themes")
	for _, theme := range stringList(merged["theme"]) {
		dir := path.Join(themesDir, theme)
		if !p.Exists(dir) {
			p.Report.Add(Manual, "", 0, "Theme `%s` is not in `%s/` (it may be a Hugo module or a missing git submodule), so its files were not migrated.", theme, dir)
			continue
		}
		h.themes = append(h.themes, dir)
	}
	h.pathToLower = merged["disablepathtolower"] != true
	h.uglyURLs = merged["uglyurls"] == true
	h.unsafeHTML = h.bool("markup.goldmark.renderer.unsafe")

	// Languages.
	h.defaultLang = strings.ToLower(h.str("defaultcontentlanguage", "en"))
	if langs, ok := merged["languages"].(map[string]any); ok {
		for _, lang := range sortedKeys(langs) {
			if lang == h.defaultLang {
				if lc, ok := langs[lang].(map[string]any); ok {
					if dir, ok := lc["contentdir"].(string); ok && dir != "" {
						h.contentDir = dir
					}
				}
				continue
			}
			h.otherLangs = append(h.otherLangs, lang)
			if lc, ok := langs[lang].(map[string]any); ok {
				if dir, ok := lc["contentdir"].(string); ok && dir != "" {
					h.langDirs = append(h.langDirs, dir)
				}
			}
		}
	}

	// Params: site params override the default language's and the themes'.
	h.params, _ = merged["params"].(map[string]any)
	if h.params == nil {
		h.params = map[string]any{}
	}
	if langs, ok := merged["languages"].(map[string]any); ok {
		if lc, ok := langs[h.defaultLang].(map[string]any); ok {
			if lp, ok := lc["params"].(map[string]any); ok {
				mergeMissing(h.params, lp)
			}
		}
	}
	for _, dir := range h.themes {
		for _, name := range []string{"hugo.toml", "hugo.yaml", "hugo.json", "config.toml", "config.yaml", "config.json", "theme.toml"} {
			data, err := p.ReadFile(path.Join(dir, name))
			if err != nil {
				continue
			}
			cfg, err := decodeConfig(name, data)
			if err != nil {
				continue
			}
			if tp, ok := lowerKeys(cfg)["params"].(map[string]any); ok {
				mergeMissing(h.params, tp)
			}
		}
	}

	// Taxonomies. Hugo defaults to tags and categories.
	h.taxonomies = map[string]string{"tag": "tags", "category": "categories"}
	if tax, ok := merged["taxonomies"].(map[string]any); ok {
		h.taxonomies = map[string]string{}
		for singular, plural := range tax {
			if s, ok := plural.(string); ok && s != "" {
				h.taxonomies[singular] = strings.ToLower(s)
			}
		}
	}

	// Page permalinks: [permalinks] <section> = pattern (old form) or
	// [permalinks.page] <section> = pattern.
	h.pagePermalinks = map[string]string{}
	if perm, ok := merged["permalinks"].(map[string]any); ok {
		for key, value := range perm {
			if s, ok := value.(string); ok && !slices.Contains(h.taxonomyNames(), key) {
				h.pagePermalinks[key] = s
			}
		}
		if page, ok := perm["page"].(map[string]any); ok {
			for key, value := range page {
				if s, ok := value.(string); ok {
					h.pagePermalinks[key] = s
				}
			}
		}
	}
	return nil
}

func (h *hugoSite) taxonomyNames() []string {
	names := make([]string, 0, len(h.taxonomies))
	for _, plural := range h.taxonomies {
		names = append(names, plural)
	}
	slices.Sort(names)
	return names
}

// get looks up a dotted key in the merged config.
func (h *hugoSite) get(key string) (any, bool) {
	var cur any = h.cfg
	for _, part := range strings.Split(key, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = m[part]; !ok {
			return nil, false
		}
	}
	return cur, true
}

func (h *hugoSite) str(key, fallback string) string {
	if v, ok := h.get(key); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return fallback
}

func (h *hugoSite) bool(key string) bool {
	v, _ := h.get(key)
	return v == true
}

// hugoConfigKeys are root keys planConfig handles, so the rest can be
// reported as dropped.
var hugoConfigKeys = map[string]bool{
	"baseurl": true, "title": true, "languagecode": true, "defaultcontentlanguage": true,
	"defaultcontentlanguageinsubdir": true, "params": true, "copyright": true, "author": true,
	"taxonomies": true, "permalinks": true, "uglyurls": true, "paginate": true, "paginatepath": true,
	"pagination": true, "summarylength": true, "markup": true, "menus": true, "menu": true,
	"outputs": true, "outputformats": true, "mediatypes": true, "services": true,
	"googleanalytics": true, "disqusshortname": true, "related": true, "enableemoji": true,
	"enablegitinfo": true, "contentdir": true, "staticdir": true, "assetdir": true, "datadir": true,
	"i18ndir": true, "layoutdir": true, "archetypedir": true, "themesdir": true, "theme": true,
	"module": true, "languages": true, "publishdir": true, "builddrafts": true, "buildfuture": true,
	"buildexpired": true, "disablekinds": true, "enablerobotstxt": true, "sitemap": true,
	"minify": true, "disablepathtolower": true, "_langfiles": true, "rsslimit": true,
	"pluralizelisttitles": true, "capitalizelisttitles": true, "canonifyurls": true,
	"relativeurls": true, "timezone": true, "frontmatter": true, "imaging": true, "security": true,
	"caches": true, "build": true, "deployment": true, "server": true, "privacy": true,
	"cascade": true, "segments": true, "removepathaccents": true, "enableinlineshortcodes": true,
	"hasCJKLanguage": true, "hascjklanguage": true, "titlecasestyle": true, "markdown": true,
	"newcontenteditor": true, "ignorefiles": true, "refLinksErrorLevel": true, "reflinkserrorlevel": true,
	"reflinksnotfoundurl": true, "enablemissingtranslationplaceholders": true, "disablehugogeneratorinject": true,
	"disablealiases": true, "workingdir": true, "cachedir": true, "resourcedir": true,
	"ignorecache": true, "ignorelogs": true, "watch": true, "noTimes": true, "notimes": true,
	"defaultcontentlanguagesubdir": true, "disablelanguages": true, "mainsections": true,
}

func (h *hugoSite) planConfig() {
	c := h.p.Config
	r := h.p.Report
	const file = "config.toml"

	if v, ok := h.cfg["baseurl"].(string); ok && v != "" && v != "/" {
		c.Set("base_url", v)
	}
	if v, ok := h.cfg["title"].(string); ok {
		c.Set("title", v)
	}
	if v, ok := h.params["description"].(string); ok && v != "" {
		c.Set("description", v)
	}
	lang := h.str("languagecode", h.defaultLang)
	if lang != "" {
		c.Set("language", lang)
	}
	switch author := h.cfg["author"].(type) {
	case string:
		c.Set("author", author)
	case map[string]any:
		if name, ok := author["name"].(string); ok {
			c.Set("author", name)
		}
	}
	if _, ok := c.Get("author"); !ok {
		switch author := h.params["author"].(type) {
		case string:
			c.Set("author", author)
		case map[string]any:
			if name, ok := author["name"].(string); ok {
				c.Set("author", name)
			}
		}
	}
	if v, ok := h.cfg["publishdir"].(string); ok && v != "public" {
		c.Set("output_dir", v)
	}
	if h.cfg["builddrafts"] == true {
		c.Set("drafts", true)
	}
	if h.cfg["buildfuture"] == true {
		c.Set("future", true)
	}
	if h.cfg["enablegitinfo"] == true {
		c.Set("updated_from_git", true)
		r.Add(Changed, file, 0, "`enableGitInfo` became `updated_from_git`, which sets each page's `updated` date from git.")
	}

	// Pagination: paginate (before 0.128) or pagination.pagerSize.
	if v, ok := h.get("pagination.pagersize"); ok {
		c.Set("paginate", v)
	} else if v, ok := h.cfg["paginate"]; ok {
		c.Set("paginate", v)
	}
	if v, ok := h.get("pagination.path"); ok {
		c.Table("sections").Set("paginate_path", v)
	} else if v, ok := h.cfg["paginatepath"]; ok {
		c.Table("sections").Set("paginate_path", v)
	}

	disabled := map[string]bool{}
	for _, kind := range stringList(h.cfg["disablekinds"]) {
		disabled[strings.ToLower(kind)] = true
	}

	// Taxonomies render term and list pages unless disabled.
	for _, singular := range sortedKeys(h.taxonomies) {
		t := c.Taxonomy(h.taxonomies[singular])
		t.Set("render", !disabled["taxonomy"] && !disabled["term"])
		if !disabled["rss"] {
			t.Set("feed", true)
		}
	}
	if perm, ok := h.cfg["permalinks"].(map[string]any); ok {
		h.planTermPermalinks(perm)
	}

	// Hugo writes an RSS feed at /index.xml for the home page, each section,
	// and each term, and no Atom feed.
	c.Table("feed").Set("enabled", false)
	if !disabled["rss"] && !h.outputsDisableRSS() {
		rss := c.Table("rss")
		rss.Set("enabled", true)
		rss.Set("filename", "index.xml")
		limit, ok := h.get("services.rss.limit")
		if !ok {
			limit, ok = h.cfg["rsslimit"]
		}
		if ok {
			if n, isInt := toInt(limit); isInt && n > 0 {
				rss.Set("limit", n)
			} else if isInt && n < 0 {
				rss.Set("limit", 100000)
			}
		}
		r.Add(Changed, file, 0, "Hugo's RSS feeds map to `[rss]` with the file name `index.xml`, so `/index.xml`, section feeds, and term feeds keep their URLs. Nida's Atom feed is off, as Hugo has none.")
	}
	if disabled["sitemap"] {
		c.Table("sitemap").Set("enabled", false)
	} else if v, ok := h.get("sitemap.filename"); ok {
		c.Table("sitemap").Set("filename", v)
	}
	// Hugo writes robots.txt only with enableRobotsTXT; a static robots.txt
	// must not collide with a generated one.
	robots := h.cfg["enablerobotstxt"] == true && !disabled["robotstxt"]
	for _, dir := range h.staticDirs {
		if h.p.Exists(path.Join(dir, "robots.txt")) {
			robots = false
		}
	}
	if !robots {
		c.Table("robots").Set("enabled", false)
	} else if h.p.Exists(path.Join(h.layoutDir, "robots.txt")) {
		r.Add(Review, path.Join(h.layoutDir, "robots.txt"), 0, "robots.txt was a Hugo template; Nida writes a default one. Set `[robots] content` to match.")
	}

	// Markup.
	md := c.Table("markdown")
	md.Set("unsafe_html", h.unsafeHTML)
	if h.bool("markup.goldmark.renderer.hardwraps") {
		md.Set("hard_wraps", true)
	}
	if v, ok := h.get("markup.goldmark.extensions.passthrough.enable"); ok && v == true {
		md.Set("math", true)
		r.Add(Changed, file, 0, "Goldmark passthrough became `markdown.math`, which renders `$...$` and `$$...$$` to MathML at build time. Remove any KaTeX or MathJax scripts from your templates.")
	}
	if v, ok := h.get("markup.goldmark.extensions.typographer"); !ok || v != false {
		if m, isMap := v.(map[string]any); !isMap || m["disable"] != true {
			r.Add(Review, file, 0, "Hugo turns straight quotes and dashes into typographic ones by default; Nida leaves them as written.")
		}
	}
	if v, ok := h.get("markup.goldmark.parser.attribute.block"); ok && v == true {
		r.Add(Unsupported, file, 0, "Markdown attributes such as `{.class}` after blocks are not supported; Nida shows them as text.")
	}
	style := h.str("markup.highlight.style", "monokai")
	if styles.Registry[strings.ToLower(style)] != nil {
		c.Set("syntax_theme", strings.ToLower(style))
	} else {
		r.Add(Review, file, 0, "Highlight style `%s` is not a Chroma style Nida knows, so Nida's default is used.", style)
	}
	if markup, ok := h.cfg["markup"].(map[string]any); ok {
		if hl, ok := markup["highlight"].(map[string]any); ok {
			for _, key := range sortedKeys(hl) {
				if key != "style" && key != "nocclasses" {
					r.Add(Unsupported, file, 0, "`markup.highlight.%s` has no Nida equivalent; set code block options per block instead.", key)
				}
			}
		}
		if _, ok := markup["tableofcontents"]; ok {
			r.Add(Unsupported, file, 0, "`markup.tableOfContents` has no equivalent; templates build tables of contents from `.Page.TOC` and choose the levels there.")
		}
		for _, key := range sortedKeys(markup) {
			switch key {
			case "goldmark", "highlight", "tableofcontents":
			default:
				r.Add(Unsupported, file, 0, "`markup.%s` has no Nida equivalent and was dropped.", key)
			}
		}
	}
	if h.cfg["enableemoji"] == true {
		r.Add(Unsupported, file, 0, "`enableEmoji` has no equivalent; emoji shortcodes such as `:smile:` show as text.")
	}
	if v, ok := h.cfg["summarylength"]; ok {
		r.Add(Unsupported, file, 0, "`summaryLength = %v` has no equivalent; Nida summaries are the text before `<!-- more -->` or the first paragraph.", v)
	}
	if h.cfg["minify"] != nil {
		if v, ok := h.get("minify.minifyoutput"); ok && v == true {
			c.Set("minify_html", true)
		}
	}

	// Params go to [extra], with Hugo's lowercased keys.
	extra := c.Table("extra")
	for _, key := range sortedKeys(h.params) {
		extra.Set(key, h.params[key])
	}
	if v, ok := h.cfg["copyright"]; ok {
		extra.Set("copyright", v)
	}
	if v, ok := h.cfg["mainsections"]; ok {
		extra.Set("mainsections", v)
	}
	if len(h.params) > 0 {
		r.Add(Changed, file, 0, "`params` became `[extra]`. Hugo lowercases param keys, and so does the migration: `.Site.Params.fooBar` is `.Config.Extra.foobar`.")
	}
	menus, hasMenus := h.cfg["menus"]
	if !hasMenus {
		menus, hasMenus = h.cfg["menu"]
	}
	if langFiles, ok := h.cfg["_langfiles"].(map[string]any); ok {
		for _, key := range sortedKeys(langFiles) {
			name, lang, _ := strings.Cut(key, ".")
			if lang != h.defaultLang {
				r.Add(Unsupported, "config/_default", 0, "Config file `%s.%s` is for language `%s`; Nida builds one language per site, so it was not migrated.", name, lang, lang)
				continue
			}
			if name == "menus" || name == "menu" {
				menus, hasMenus = langFiles[key], true
			}
		}
	}
	if hasMenus {
		extra.Set("menu", normalizeHugoMenus(menus))
		r.Add(Review, file, 0, "Nida has no menus; Hugo's menus were copied to `[extra.menu]`, one list of `name`, `url`, and `weight` entries per menu, for templates to read with `dig .Config.Extra \"menu.main\"`. Menu entries in page front matter stay in each page's `extra`.")
	}
	if ga, ok := h.get("services.googleanalytics.id"); ok {
		extra.Set("google_analytics", ga)
		r.Add(Review, file, 0, "The Google Analytics ID is in `[extra] google_analytics`; add the tracking snippet to your base template.")
	} else if ga, ok := h.cfg["googleanalytics"]; ok {
		extra.Set("google_analytics", ga)
		r.Add(Review, file, 0, "The Google Analytics ID is in `[extra] google_analytics`; add the tracking snippet to your base template.")
	}
	if dq, ok := h.get("services.disqus.shortname"); ok {
		extra.Set("disqus_shortname", dq)
		r.Add(Review, file, 0, "The Disqus shortname is in `[extra] disqus_shortname`; add the embed code to your templates.")
	} else if dq, ok := h.cfg["disqusshortname"]; ok {
		extra.Set("disqus_shortname", dq)
		r.Add(Review, file, 0, "The Disqus shortname is in `[extra] disqus_shortname`; add the embed code to your templates.")
	}
	sortTableKeys(extra)

	if len(h.otherLangs) > 0 {
		r.Add(Unsupported, file, 0, "Languages %s are not migrated: Nida builds one language per site. Only `%s` content was migrated; translated files are kept in `%s/`.", strings.Join(h.otherLangs, ", "), h.defaultLang, preservedDir)
	}
	for _, key := range []string{"outputs", "outputformats", "mediatypes"} {
		if _, ok := h.cfg[key]; ok {
			r.Add(Unsupported, file, 0, "`%s` has no equivalent; Nida writes HTML pages, feeds, the sitemap, and the search index only.", key)
		}
	}
	if _, ok := h.cfg["related"]; ok {
		r.Add(Unsupported, file, 0, "`related` settings have no equivalent; `.Index.Related` ranks pages by shared taxonomy terms.")
	}
	if _, ok := h.cfg["module"]; ok {
		r.Add(Unsupported, file, 0, "Hugo modules are not supported; themes and components imported as modules were not downloaded. Vendor them into `themes/` and migrate again if you need their files.")
	}
	if h.cfg["canonifyurls"] == true || h.cfg["relativeurls"] == true {
		r.Add(Unsupported, file, 0, "`canonifyURLs`/`relativeURLs` have no equivalent; Nida links are root-relative with the `base_url` path.")
	}
	if v, ok := h.cfg["timezone"]; ok {
		r.Add(Review, file, 0, "`timeZone = %v` has no equivalent; dates without an offset are read as UTC.", v)
	}
	if _, ok := h.cfg["frontmatter"]; ok {
		r.Add(Review, file, 0, "`[frontmatter]` date settings have no equivalent; `date` and `updated` are read from front matter only (use `updated_from_git` for git dates).")
	}
	if _, ok := h.cfg["cascade"]; ok {
		r.Add(Unsupported, file, 0, "Site `cascade` has no equivalent; set those values in each page's front matter.")
	}
	if section, ok := h.get("permalinks.section"); ok {
		r.Add(Unsupported, file, 0, "`permalinks.section` (%v) is not supported; Nida section URLs follow the content directories.", section)
	}
	if tax, ok := h.get("permalinks.taxonomy"); ok {
		r.Add(Unsupported, file, 0, "`permalinks.taxonomy` (%v) is not supported; taxonomy list pages are at `/<taxonomy>/`.", tax)
	}

	for _, key := range sortedKeys(h.cfg) {
		if !hugoConfigKeys[key] {
			r.Add(Unsupported, file, 0, "Config key `%s` has no Nida equivalent and was dropped.", key)
		}
	}
}

// outputsDisableRSS reports whether outputs.home leaves out RSS.
func (h *hugoSite) outputsDisableRSS() bool {
	home, ok := h.get("outputs.home")
	if !ok {
		return false
	}
	for _, f := range stringList(home) {
		if strings.EqualFold(f, "rss") {
			return false
		}
	}
	return true
}

// planTermPermalinks maps taxonomy term permalinks, which Nida configures
// in [permalinks] by taxonomy name with a {slug} placeholder.
func (h *hugoSite) planTermPermalinks(perm map[string]any) {
	terms := map[string]string{}
	for key, value := range perm {
		if s, ok := value.(string); ok && slices.Contains(h.taxonomyNames(), key) {
			terms[key] = s
		}
	}
	if term, ok := perm["term"].(map[string]any); ok {
		for key, value := range term {
			if s, ok := value.(string); ok {
				terms[key] = s
			}
		}
	}
	for _, name := range sortedKeys(terms) {
		pattern := terms[name]
		translated := strings.NewReplacer(":slug", "{slug}", ":title", "{slug}").Replace(pattern)
		if strings.Contains(translated, ":") {
			h.p.Report.Add(Unsupported, "config.toml", 0, "Term permalink `%s = %q` uses tokens Nida cannot express; terms are at `/%s/{slug}/`.", name, pattern, name)
			continue
		}
		if !strings.HasSuffix(translated, "/") && path.Ext(translated) == "" {
			translated += "/"
		}
		h.p.Config.Table("permalinks").Set(name, translated)
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case uint64:
		return int(n), true
	}
	return 0, false
}

// planStatic copies the site's and themes' static directories into
// static/, the site's files winning.
func (h *hugoSite) planStatic() {
	dirs := slices.Clone(h.staticDirs)
	for _, theme := range h.themes {
		dirs = append(dirs, path.Join(theme, "static"))
	}
	for _, dir := range dirs {
		_ = h.p.Walk(dir, func(rel string) error {
			dest := path.Join("static", strings.TrimPrefix(rel, dir+"/"))
			if !h.p.Planned(dest) {
				h.p.Copy(dest, rel)
			}
			return nil
		})
	}
	assetDirs := []string{h.assetDir}
	for _, theme := range h.themes {
		assetDirs = append(assetDirs, path.Join(theme, "assets"))
	}
	// Translated templates link Pipes resources with `asset "<path>"`, so
	// assets are copied to the same paths under static/. SCSS entry files
	// compile through Nida's pipeline, which writes static/<dir>/x.scss
	// to /<dir>/x.css, the URL the translation expects.
	scssDirs := map[string]bool{}
	for _, dir := range assetDirs {
		found := false
		_ = h.p.Walk(dir, func(rel string) error {
			found = true
			inner := strings.TrimPrefix(rel, dir+"/")
			dest := path.Join("static", inner)
			if h.p.Planned(dest) {
				return nil // the site's own asset wins over a theme's
			}
			h.p.Copy(dest, rel)
			base := path.Base(inner)
			switch ext := strings.ToLower(path.Ext(base)); {
			case (ext == ".scss" || ext == ".sass") && !strings.HasPrefix(base, "_"):
				scssDirs[path.Dir(inner)] = true
			case ext == ".ts" || ext == ".tsx" || ext == ".jsx":
				h.p.Report.Add(Manual, rel, 0, "Hugo built this with `js.Build`; Nida does not compile %s. Build it to JavaScript and put the result in `static/`.", ext)
			}
			return nil
		})
		if found {
			h.p.Report.Add(Review, dir, 0, "Hugo Pipes assets in `%s/` were copied to `static/` at the same paths. Nida does not fingerprint, minify, or bundle them the way Pipes did; see `[pipeline]` for the options it has.", dir)
		}
	}
	if scssDirs["."] {
		delete(scssDirs, ".")
		h.p.Report.Add(Manual, h.assetDir, 0, "SCSS entry files at the top of `%s/` were copied to `static/`, but Nida compiles SCSS only from a directory such as `static/css/`. Move them there and update the links.", h.assetDir)
	}
	if len(scssDirs) > 0 {
		dirs := sortedKeys(scssDirs)
		scss := h.p.Config.Table("pipeline.scss")
		scss.Set("enabled", true)
		scss.Set("entry_dir", dirs[0])
		h.p.Report.Add(Review, h.assetDir, 0, "SCSS compiles through Nida's pipeline from `static/%s/`, which needs the `sass` command installed (Hugo uses its own Dart Sass or LibSass).", dirs[0])
		if len(dirs) > 1 {
			h.p.Report.Add(Manual, h.assetDir, 0, "Nida compiles SCSS entry files from one directory below `static/`; entry files in %s were copied but are not compiled. Move them to `static/%s/` or import them from there.", strings.Join(dirs, ", "), dirs[0])
		}
	}
}

// planData copies data files Nida reads (TOML, JSON, YAML).
func (h *hugoSite) planData() {
	dirs := []string{h.dataDir}
	for _, theme := range h.themes {
		dirs = append(dirs, path.Join(theme, "data"))
	}
	for _, dir := range dirs {
		_ = h.p.Walk(dir, func(rel string) error {
			dest := path.Join("data", strings.TrimPrefix(rel, dir+"/"))
			switch strings.ToLower(path.Ext(rel)) {
			case ".toml", ".json", ".yaml", ".yml":
				if !h.p.Planned(dest) {
					h.p.Copy(dest, rel)
				}
			default:
				h.p.Preserve(rel)
				h.p.Report.Add(Unsupported, rel, 0, "Data files in this format are not read by Nida (TOML, JSON, and YAML are); it is kept in `%s/`.", preservedDir)
			}
			return nil
		})
	}
}

// planI18n converts Hugo translation files into Nida's flat TOML. Hugo
// allows `key = "x"`, `[key] other = "x"`, and YAML lists of
// {id, translation}; Nida reads `key = "x"`.
func (h *hugoSite) planI18n() {
	dirs := []string{h.i18nDir}
	for _, theme := range h.themes {
		dirs = append(dirs, path.Join(theme, "i18n"))
	}
	merged := map[string]map[string]string{}
	var order []string
	for _, dir := range dirs {
		_ = h.p.Walk(dir, func(rel string) error {
			lang := strings.ToLower(strings.TrimSuffix(path.Base(rel), path.Ext(rel)))
			data, err := h.p.ReadFile(rel)
			if err != nil {
				return err
			}
			strs, err := hugoTranslations(path.Base(rel), data)
			if err != nil {
				h.p.Report.Add(Manual, rel, 0, "Translation file could not be read (%v) and was not migrated.", err)
				return nil
			}
			if merged[lang] == nil {
				merged[lang] = map[string]string{}
				order = append(order, lang)
			}
			for k, v := range strs {
				if _, ok := merged[lang][k]; !ok {
					merged[lang][k] = v
				}
			}
			return nil
		})
	}
	for _, lang := range order {
		var b strings.Builder
		for _, key := range sortedKeys(merged[lang]) {
			fmt.Fprintf(&b, "%s = %s\n", tomlKey(key), tomlString(merged[lang][key]))
		}
		h.p.Write("i18n/"+lang+".toml", []byte(b.String()), "")
	}
	if len(order) > 0 {
		h.p.Report.Add(Changed, h.i18nDir, 0, "Translation files became Nida's `key = \"value\"` TOML; plural forms keep their `other` text (Nida has no plurals). Templates call `t \"key\"` instead of `i18n \"key\"`.")
	}
}

func hugoTranslations(name string, data []byte) (map[string]string, error) {
	out := map[string]string{}
	pick := func(v any) (string, bool) {
		switch x := v.(type) {
		case string:
			return x, true
		case map[string]any:
			for _, form := range []string{"other", "one", "few", "many", "two", "zero"} {
				if s, ok := x[form].(string); ok {
					return s, true
				}
			}
			if s, ok := x["translation"].(string); ok {
				return s, true
			}
			if m, ok := x["translation"].(map[string]any); ok {
				if s, ok := m["other"].(string); ok {
					return s, true
				}
			}
		}
		return "", false
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".yaml", ".yml", ".json":
		var list []map[string]any
		if yaml.Unmarshal(data, &list) == nil && len(list) > 0 {
			for _, item := range list {
				id, _ := item["id"].(string)
				if s, ok := pick(item); ok && id != "" {
					out[id] = s
				}
			}
			return out, nil
		}
	}
	m, err := decodeConfig(name, data)
	if err != nil {
		return nil, err
	}
	for k, v := range m {
		if s, ok := pick(v); ok {
			out[k] = s
		}
	}
	return out, nil
}

// hugoArchetypeExprs are the archetype expressions Nida's archetypes can
// express, mapped to Nida's archetype fields.
var hugoArchetypeExprs = []struct{ from, to string }{
	{`{{ replace .File.ContentBaseName "-" " " | title }}`, `{{ .Title }}`},
	{`{{ replace .Name "-" " " | title }}`, `{{ .Title }}`},
	{`{{ replace .TranslationBaseName "-" " " | title }}`, `{{ .Title }}`},
	{`{{ .Name | humanize | title }}`, `{{ .Title }}`},
	{`{{ .File.ContentBaseName | humanize | title }}`, `{{ .Title }}`},
	{`{{ .Date }}`, `{{ .Date }}`},
	{`{{ .File.ContentBaseName }}`, `{{ .Slug }}`},
	{`{{ .Name }}`, `{{ .Slug }}`},
	{`{{ .Section }}`, `{{ .Section }}`},
	{`{{ .Type }}`, `{{ .Section }}`},
}

// planArchetypes translates archetypes that use only the common Hugo
// expressions; others are kept for reference.
func (h *hugoSite) planArchetypes() {
	dirs := []string{h.archetypes}
	for _, theme := range h.themes {
		dirs = append(dirs, path.Join(theme, "archetypes"))
	}
	for _, dir := range dirs {
		_ = h.p.Walk(dir, func(rel string) error {
			inner := strings.TrimPrefix(rel, dir+"/")
			if strings.Contains(inner, "/") || !strings.EqualFold(path.Ext(inner), ".md") {
				h.p.Preserve(rel)
				h.p.Report.Add(Unsupported, rel, 0, "Directory archetypes and non-Markdown archetypes are not supported; Nida archetypes are `archetypes/<section>.md`. Kept in `%s/`.", preservedDir)
				return nil
			}
			dest := path.Join("archetypes", inner)
			if h.p.Planned(dest) {
				return nil
			}
			data, err := h.p.ReadFile(rel)
			if err != nil {
				return err
			}
			text := string(data)
			for _, e := range hugoArchetypeExprs {
				text = strings.ReplaceAll(text, e.from, e.to)
				text = strings.ReplaceAll(text, strings.ReplaceAll(e.from, "{{ ", "{{"), e.to)
			}
			if !archetypeRenders(text) {
				h.p.Preserve(rel)
				h.p.Report.Add(Manual, rel, 0, "Archetype uses Hugo template features Nida's archetypes lack (they get `.Title`, `.Date`, `.Slug`, `.Section`, and `.Path`); it is kept in `%s/` to rewrite.", preservedDir)
				return nil
			}
			h.p.Write(dest, []byte(text), rel)
			if text != string(data) {
				h.p.Report.Add(Changed, rel, 0, "Archetype expressions now use Nida's `.Title`, `.Date`, and `.Slug`.")
			}
			return nil
		})
	}
}

// archetypeRenders reports whether text runs as a Nida archetype.
func archetypeRenders(text string) bool {
	tmpl, err := template.New("a").Parse(text)
	if err != nil {
		return false
	}
	data := struct{ Title, Date, Slug, Section, Path string }{"T", "2024-01-01T00:00:00Z", "t", "posts", "posts/t.md"}
	return tmpl.Execute(&bytes.Buffer{}, data) == nil
}

// planLayouts keeps every layout file for reference and hands the layout
// directories to the template translator.
func (h *hugoSite) planLayouts() {
	dirs := []string{h.layoutDir}
	for _, theme := range h.themes {
		dirs = append(dirs, path.Join(theme, "layouts"))
	}
	var present []string
	for _, dir := range dirs {
		found := false
		_ = h.p.Walk(dir, func(rel string) error {
			found = true
			h.p.Preserve(rel)
			return nil
		})
		if found {
			present = append(present, dir)
		}
	}
	if len(present) > 0 {
		h.p.Report.Add(Review, "", 0, "Hugo layouts are kept in `%s/` for reference.", preservedDir)
	}
	planHugoTemplates(h.p, present)
}

// normalizeHugoMenus turns Hugo's menus into the shape the translated
// templates read: a map from menu name to a list of entries with
// lowercase keys, where pageRef fills in a missing url.
func normalizeHugoMenus(menus any) map[string]any {
	out := map[string]any{}
	byName, ok := menus.(map[string]any)
	if !ok {
		return out
	}
	for name, raw := range byName {
		var entries []any
		for _, item := range anyList(raw) {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			clean := map[string]any{}
			for key, value := range entry {
				clean[strings.ToLower(key)] = value
			}
			if _, ok := clean["url"]; !ok {
				if ref, ok := clean["pageref"].(string); ok {
					clean["url"] = "/" + strings.Trim(ref, "/") + "/"
					if ref == "/" || ref == "" {
						clean["url"] = "/"
					}
				}
			}
			delete(clean, "pageref")
			entries = append(entries, clean)
		}
		out[strings.ToLower(name)] = entries
	}
	return out
}
