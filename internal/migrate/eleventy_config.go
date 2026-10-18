package migrate

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// eleventyConfig is what could be read from an Eleventy config file.
// Eleventy configs are JavaScript, which nida never runs, so the values
// are scraped with patterns that match the common ways of writing them.
type eleventyConfig struct {
	file string // source-relative config file, or "" without one

	// input is relative to the project root; includes, layouts, and data
	// are relative to input, and output is relative to the root, as in
	// Eleventy.
	input, output, includes, layouts, data string

	formats    []string
	mdEngine   string // "liquid", "njk", ..., or "" for none
	pathPrefix string

	passthrough []passRule
	shortcodes  []string // single-tag shortcode names
	paired      []string // paired shortcode names
	filters     []string
	collections []string
	transforms  []string
	globalData  []string
	plugins     []eleventyPlugin
	others      []string // other eleventyConfig calls, not migrated

	excerpt    bool
	excerptSep string

	feedPath, feedType string
	anchors            bool // markdown-it-anchor is used

	// found lists the settings that were read, for the report.
	found []string
}

type passRule struct {
	src  string // project-relative path or glob
	dest string // output-relative destination, or "" to derive it
	set  bool   // dest was given explicitly
}

type eleventyPlugin struct {
	ident, pkg string
}

var eleventyDefaultFormats = []string{"liquid", "ejs", "md", "hbs", "mustache", "haml", "pug", "njk", "html", "11ty.js"}

// eleventyConfigFiles are the config file names Eleventy looks for, in its
// order of precedence.
var eleventyConfigFiles = []string{".eleventy.js", "eleventy.config.js", "eleventy.config.mjs", "eleventy.config.cjs", "eleventy.config.ts", "eleventy.config.mts", "eleventy.config.cts"}

var (
	jsDirBlock     = regexp.MustCompile(`\bdir\s*:\s*\{([^{}]*)\}`)
	jsDirKey       = regexp.MustCompile(`\b(input|output|includes|layouts|data)\s*:\s*(?:"([^"]*)"|'([^']*)'|` + "`([^`$]*)`" + `)`)
	jsDirSetter    = regexp.MustCompile(`\.set(Input|Output|Includes|Layouts|Data)Directory\(\s*(?:"([^"]*)"|'([^']*)'|` + "`([^`$]*)`" + `)`)
	jsFormats      = regexp.MustCompile(`(?:templateFormats\s*:|\.setTemplateFormats\(|\.addTemplateFormats\()\s*(\[[^\]]*\]|"[^"]*"|'[^']*')`)
	jsMdEngine     = regexp.MustCompile(`markdownTemplateEngine\s*:\s*(false|"(\w*)"|'(\w*)')`)
	jsPathPrefix   = regexp.MustCompile(`pathPrefix\s*:\s*(?:"([^"]*)"|'([^']*)'|` + "`([^`$]*)`" + `)`)
	jsPassthrough  = regexp.MustCompile(`\.addPassthroughCopy\(\s*`)
	jsShortcode    = regexp.MustCompile(`\.add(Paired)?(?:Nunjucks|Liquid|Async|NunjucksAsync|Handlebars|JavaScript)?(Shortcode|Function)\(\s*["'` + "`" + `]([\w-]+)`)
	jsFilter       = regexp.MustCompile(`\.add(?:Nunjucks|Liquid|Async|NunjucksAsync|Handlebars|JavaScript)?Filter\(\s*["'` + "`" + `]([\w-]+)`)
	jsCollection   = regexp.MustCompile(`\.addCollection\(\s*["'` + "`" + `]([\w-]+)`)
	jsTransform    = regexp.MustCompile(`\.addTransform\(\s*["'` + "`" + `]([\w-]+)`)
	jsGlobalData   = regexp.MustCompile(`\.addGlobalData\(\s*["'` + "`" + `]([\w.-]+)`)
	jsPlugin       = regexp.MustCompile(`\.addPlugin\(\s*([\w$.]+)`)
	jsImport       = regexp.MustCompile(`import\s+(?:\*\s+as\s+)?(\{[^}]*\}|[\w$]+)(?:\s*,\s*(\{[^}]*\}))?\s+from\s+["']([^"']+)["']`)
	jsRequire      = regexp.MustCompile(`(?:const|let|var)\s+(\{[^}]*\}|[\w$]+)\s*=\s*require\(\s*["']([^"']+)["']\s*\)`)
	jsExcerpt      = regexp.MustCompile(`\.setFrontMatterParsingOptions\(\s*\{([^{}]*)\}`)
	jsExcerptOn    = regexp.MustCompile(`\bexcerpt\s*:\s*true`)
	jsExcerptSep   = regexp.MustCompile(`excerpt_separator\s*:\s*(?:"([^"]*)"|'([^']*)')`)
	jsOutputPath   = regexp.MustCompile(`outputPath\s*:\s*(?:"([^"]*)"|'([^']*)')`)
	jsFeedType     = regexp.MustCompile(`\btype\s*:\s*["'](atom|rss|json)["']`)
	jsConfigCall   = regexp.MustCompile(`eleventyConfig\.(\w+)\(`)
	jsPrefixScript = regexp.MustCompile(`--pathprefix[= ]["']?([^\s"']+)`)
)

// knownConfigCalls are eleventyConfig methods that are read above, or that
// only affect the old build process and need no migration.
var knownConfigCalls = map[string]bool{
	"addPassthroughCopy": true, "addCollection": true, "addTransform": true, "addPlugin": true,
	"addGlobalData": true, "setFrontMatterParsingOptions": true, "setTemplateFormats": true,
	"addTemplateFormats": true, "setInputDirectory": true, "setOutputDirectory": true,
	"setIncludesDirectory": true, "setLayoutsDirectory": true, "setDataDirectory": true,
	"addWatchTarget": true, "setWatchThrottleWaitTime": true, "setServerOptions": true,
	"setBrowserSyncConfig": true, "setQuietMode": true, "setWatchJavaScriptDependencies": true,
	"setUseGitIgnore": true, "on": true, "ignores": true, "setServerPassthroughCopyBehavior": true,
	"addLayoutAlias": true, "setDataDeepMerge": true, "setLiquidOptions": true, "setNunjucksEnvironmentOptions": true,
}

// readEleventyConfig scrapes the config file, if there is one.
func readEleventyConfig(p *Plan) *eleventyConfig {
	cfg := &eleventyConfig{input: ".", output: "_site", includes: "_includes", data: "_data", mdEngine: "liquid", formats: eleventyDefaultFormats}
	for _, name := range eleventyConfigFiles {
		if p.Exists(name) {
			cfg.file = name
			break
		}
	}
	if pkg, err := p.ReadFile("package.json"); err == nil {
		if m := jsPrefixScript.FindStringSubmatch(string(pkg)); m != nil {
			cfg.pathPrefix = m[1]
			cfg.found = append(cfg.found, "pathPrefix (from package.json scripts)")
		}
	}
	if cfg.file == "" {
		cfg.layouts = cfg.includes
		return cfg
	}
	data, err := p.ReadFile(cfg.file)
	if err != nil {
		cfg.layouts = cfg.includes
		return cfg
	}
	src := stripJSComments(string(data))
	cfg.scrape(src)
	if cfg.layouts == "" {
		cfg.layouts = cfg.includes
	}
	return cfg
}

func firstGroup(m []string) string {
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return ""
}

func (cfg *eleventyConfig) scrape(src string) {
	setDir := func(key, value string) {
		value = cleanRel(value)
		switch strings.ToLower(key) {
		case "input":
			cfg.input = value
		case "output":
			cfg.output = value
		case "includes":
			cfg.includes = value
		case "layouts":
			cfg.layouts = value
		case "data":
			cfg.data = value
		}
		cfg.found = append(cfg.found, "dir."+strings.ToLower(key))
	}
	for _, block := range jsDirBlock.FindAllStringSubmatch(src, -1) {
		for _, m := range jsDirKey.FindAllStringSubmatch(block[1], -1) {
			setDir(m[1], firstGroup(m[1:]))
		}
	}
	for _, m := range jsDirSetter.FindAllStringSubmatch(src, -1) {
		setDir(m[1], firstGroup(m[1:]))
	}

	for _, m := range jsFormats.FindAllStringSubmatch(src, -1) {
		var formats []string
		if v, err := parseJSLiteral(m[1]); err == nil {
			switch x := v.(type) {
			case string:
				formats = strings.Split(x, ",")
			case []any:
				for _, item := range x {
					if s, ok := item.(string); ok {
						formats = append(formats, s)
					}
				}
			}
		}
		for i := range formats {
			formats[i] = strings.TrimSpace(formats[i])
		}
		if strings.Contains(m[0], "addTemplateFormats") {
			cfg.formats = append(slices.Clone(cfg.formats), formats...)
		} else {
			cfg.formats = formats
		}
		cfg.found = append(cfg.found, "templateFormats")
	}
	if m := jsMdEngine.FindStringSubmatch(src); m != nil {
		if m[1] == "false" {
			cfg.mdEngine = ""
		} else {
			cfg.mdEngine = firstGroup(m[1:])
			if m[2] == "" && m[3] == "" {
				cfg.mdEngine = ""
			}
		}
		cfg.found = append(cfg.found, "markdownTemplateEngine")
	}
	if m := jsPathPrefix.FindStringSubmatch(src); m != nil {
		cfg.pathPrefix = firstGroup(m)
		cfg.found = append(cfg.found, "pathPrefix")
	}

	for _, loc := range jsPassthrough.FindAllStringIndex(src, -1) {
		arg, ok := jsBalancedArg(src[loc[1]:])
		if !ok {
			cfg.others = append(cfg.others, "addPassthroughCopy(…)")
			continue
		}
		v, err := parseJSLiteral(arg)
		if err != nil {
			cfg.others = append(cfg.others, "addPassthroughCopy("+truncate(arg, 40)+")")
			continue
		}
		switch x := v.(type) {
		case string:
			cfg.passthrough = append(cfg.passthrough, passRule{src: cleanRel(x)})
		case map[string]any:
			for _, from := range sortedKeys(x) {
				to, _ := x[from].(string)
				cfg.passthrough = append(cfg.passthrough, passRule{src: cleanRel(from), dest: cleanRel(to), set: true})
			}
		}
		cfg.found = append(cfg.found, "addPassthroughCopy")
	}

	for _, m := range jsShortcode.FindAllStringSubmatch(src, -1) {
		if m[1] == "Paired" {
			cfg.paired = appendUnique(cfg.paired, m[3])
		} else {
			cfg.shortcodes = appendUnique(cfg.shortcodes, m[3])
		}
	}
	for _, m := range jsFilter.FindAllStringSubmatch(src, -1) {
		cfg.filters = appendUnique(cfg.filters, m[1])
	}
	for _, m := range jsCollection.FindAllStringSubmatch(src, -1) {
		cfg.collections = appendUnique(cfg.collections, m[1])
	}
	for _, m := range jsTransform.FindAllStringSubmatch(src, -1) {
		cfg.transforms = appendUnique(cfg.transforms, m[1])
	}
	for _, m := range jsGlobalData.FindAllStringSubmatch(src, -1) {
		cfg.globalData = appendUnique(cfg.globalData, m[1])
	}

	imports := map[string]string{}
	addBinding := func(binding, pkg string) {
		binding = strings.TrimSpace(binding)
		if strings.HasPrefix(binding, "{") {
			for _, name := range strings.Split(strings.Trim(binding, "{}"), ",") {
				name = strings.TrimSpace(name)
				if _, alias, ok := strings.Cut(name, " as "); ok {
					name = strings.TrimSpace(alias)
				}
				if _, alias, ok := strings.Cut(name, ":"); ok {
					name = strings.TrimSpace(alias)
				}
				if name != "" {
					imports[name] = pkg
				}
			}
			return
		}
		imports[binding] = pkg
	}
	for _, m := range jsImport.FindAllStringSubmatch(src, -1) {
		addBinding(m[1], m[3])
		if m[2] != "" {
			addBinding(m[2], m[3])
		}
	}
	for _, m := range jsRequire.FindAllStringSubmatch(src, -1) {
		addBinding(m[1], m[2])
	}
	for _, pkg := range imports {
		if strings.Contains(pkg, "markdown-it-anchor") {
			cfg.anchors = true
		}
	}
	for _, m := range jsPlugin.FindAllStringSubmatch(src, -1) {
		ident := m[1]
		root, _, _ := strings.Cut(ident, ".")
		pkg := imports[root]
		cfg.plugins = append(cfg.plugins, eleventyPlugin{ident: ident, pkg: pkg})
		if strings.Contains(pkg, "plugin-rss") && (strings.Contains(ident, "feedPlugin") || strings.HasSuffix(ident, "feedPlugin")) {
			if out := jsOutputPath.FindStringSubmatch(src); out != nil {
				cfg.feedPath = firstGroup(out)
				cfg.feedType = "atom"
				if t := jsFeedType.FindStringSubmatch(src); t != nil {
					cfg.feedType = t[1]
				}
			}
		}
	}

	if m := jsExcerpt.FindStringSubmatch(src); m != nil {
		cfg.excerpt = jsExcerptOn.MatchString(m[1])
		if sep := jsExcerptSep.FindStringSubmatch(m[1]); sep != nil {
			cfg.excerptSep = firstGroup(sep)
		}
		cfg.found = append(cfg.found, "excerpt options")
	}

	for _, m := range jsConfigCall.FindAllStringSubmatch(src, -1) {
		name := m[1]
		if knownConfigCalls[name] || jsShortcode.MatchString("."+name+`("x`) || jsFilter.MatchString("."+name+`("x`) {
			continue
		}
		cfg.others = appendUnique(cfg.others, name)
	}
	slices.Sort(cfg.others)
	cfg.found = uniqueSorted(cfg.found)
}

func appendUnique(list []string, s string) []string {
	if slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}

func uniqueSorted(list []string) []string {
	out := slices.Clone(list)
	slices.Sort(out)
	return slices.Compact(out)
}

// cleanRel normalizes a project-relative path from a config file:
// "./src/" and "src" are the same, and "" or "/" is ".".
func cleanRel(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, `\`, "/"))
	p = strings.Trim(p, "/")
	if p == "" {
		return "."
	}
	return path.Clean(p)
}

// stripJSComments removes // and /* */ comments outside string literals.
func stripJSComments(src string) string {
	var b strings.Builder
	var quote byte
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case quote != 0:
			b.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				i++
				b.WriteByte(src[i])
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'' || c == '`':
			quote = c
			b.WriteByte(c)
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				b.WriteByte('\n')
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return b.String()
			}
			b.WriteString(strings.Repeat("\n", strings.Count(src[i:i+2+end], "\n")))
			i += end + 3
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// jsBalancedArg returns the first argument of a call whose opening
// parenthesis has just been consumed: a string literal or a balanced
// object or array literal.
func jsBalancedArg(s string) (string, bool) {
	if s == "" {
		return "", false
	}
	switch s[0] {
	case '"', '\'', '`':
		for i := 1; i < len(s); i++ {
			if s[i] == '\\' {
				i++
				continue
			}
			if s[i] == s[0] {
				return s[:i+1], true
			}
		}
		return "", false
	case '{', '[':
		depth := 0
		var quote byte
		for i := 0; i < len(s); i++ {
			c := s[i]
			switch {
			case quote != 0:
				switch c {
				case '\\':
					i++
				case quote:
					quote = 0
				}
			case c == '"' || c == '\'' || c == '`':
				quote = c
			case c == '{' || c == '[':
				depth++
			case c == '}' || c == ']':
				depth--
				if depth == 0 {
					return s[:i+1], true
				}
			}
		}
	}
	return "", false
}

var jsExport = regexp.MustCompile(`(?:module\.exports\s*=|export\s+default)\s*`)

// parseJSDataFile reads a JavaScript data file that exports a plain
// object or array literal, such as `export default { title: "x" }`. Files
// that compute their data return an error.
func parseJSDataFile(src string) (any, error) {
	src = stripJSComments(src)
	loc := jsExport.FindStringIndex(src)
	if loc == nil {
		return nil, fmt.Errorf("no module.exports or export default")
	}
	rest := strings.TrimSpace(src[loc[1]:])
	arg, ok := jsBalancedArg(rest)
	if !ok {
		return nil, fmt.Errorf("the export is not a plain object or array")
	}
	tail := strings.TrimSpace(rest[len(arg):])
	tail = strings.TrimSpace(strings.TrimPrefix(tail, ";"))
	if tail != "" {
		return nil, fmt.Errorf("the file runs code after its export")
	}
	return parseJSLiteral(arg)
}

// parseJSLiteral parses a JavaScript literal made of objects, arrays,
// strings, numbers, booleans, and null. Unquoted object keys and trailing
// commas are allowed; anything computed is an error.
func parseJSLiteral(src string) (any, error) {
	l := &jsLexer{src: src}
	v, err := l.value()
	if err != nil {
		return nil, err
	}
	l.space()
	if l.pos != len(l.src) {
		return nil, fmt.Errorf("unexpected %q", truncate(l.src[l.pos:], 20))
	}
	return v, nil
}

type jsLexer struct {
	src string
	pos int
}

func (l *jsLexer) space() {
	for l.pos < len(l.src) && strings.ContainsRune(" \t\r\n", rune(l.src[l.pos])) {
		l.pos++
	}
}

func (l *jsLexer) value() (any, error) {
	l.space()
	if l.pos >= len(l.src) {
		return nil, fmt.Errorf("unexpected end of literal")
	}
	switch c := l.src[l.pos]; {
	case c == '{':
		return l.object()
	case c == '[':
		return l.array()
	case c == '"' || c == '\'' || c == '`':
		return l.str()
	case c == '-' || (c >= '0' && c <= '9'):
		start := l.pos
		l.pos++
		for l.pos < len(l.src) && strings.ContainsRune("0123456789.eE+-_", rune(l.src[l.pos])) {
			l.pos++
		}
		text := strings.ReplaceAll(l.src[start:l.pos], "_", "")
		if n, err := strconv.ParseInt(text, 10, 64); err == nil {
			return int(n), nil
		}
		f, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, fmt.Errorf("bad number %q", text)
		}
		return f, nil
	default:
		word := l.ident()
		switch word {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null", "undefined":
			return nil, nil
		}
		return nil, fmt.Errorf("computed value %q", truncate(word+l.src[l.pos:], 20))
	}
}

func (l *jsLexer) ident() string {
	start := l.pos
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '_' || c == '$' || c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			l.pos++
			continue
		}
		break
	}
	return l.src[start:l.pos]
}

func (l *jsLexer) str() (string, error) {
	quote := l.src[l.pos]
	l.pos++
	var b strings.Builder
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == '\\' && l.pos+1 < len(l.src):
			l.pos++
			switch e := l.src[l.pos]; e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(e)
			}
		case c == quote:
			l.pos++
			return b.String(), nil
		case quote == '`' && c == '$' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '{':
			return "", fmt.Errorf("template literal with ${}")
		default:
			b.WriteByte(c)
		}
		l.pos++
	}
	return "", fmt.Errorf("unterminated string")
}

func (l *jsLexer) object() (map[string]any, error) {
	l.pos++ // {
	out := map[string]any{}
	for {
		l.space()
		if l.pos < len(l.src) && l.src[l.pos] == '}' {
			l.pos++
			return out, nil
		}
		var key string
		if l.pos < len(l.src) && strings.ContainsRune(`"'`+"`", rune(l.src[l.pos])) {
			k, err := l.str()
			if err != nil {
				return nil, err
			}
			key = k
		} else {
			key = l.ident()
		}
		if key == "" {
			return nil, fmt.Errorf("unsupported object key at %q", truncate(l.src[l.pos:], 20))
		}
		l.space()
		if l.pos >= len(l.src) || l.src[l.pos] != ':' {
			return nil, fmt.Errorf("shorthand or computed property %q", key)
		}
		l.pos++
		v, err := l.value()
		if err != nil {
			return nil, err
		}
		out[key] = v
		l.space()
		if l.pos < len(l.src) && l.src[l.pos] == ',' {
			l.pos++
		}
	}
}

func (l *jsLexer) array() ([]any, error) {
	l.pos++ // [
	out := []any{}
	for {
		l.space()
		if l.pos < len(l.src) && l.src[l.pos] == ']' {
			l.pos++
			return out, nil
		}
		v, err := l.value()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		l.space()
		if l.pos < len(l.src) && l.src[l.pos] == ',' {
			l.pos++
		}
	}
}
