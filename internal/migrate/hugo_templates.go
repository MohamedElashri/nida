package migrate

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/template/parse"
)

// planHugoTemplates translates Hugo layouts into candidate Nida templates
// with Plan.Template. dirs are source-relative layout directories in
// lookup order: the site's layouts/ first, then each theme's.
//
// The translation is best effort. Hugo templates are Go templates, so the
// syntax carries over, but Hugo gives every template a Page with methods
// for everything, while Nida gives it a context with .Page, .Section,
// .Config, and .Index. The translator parses each layout, tracks what kind
// of value every expression denotes (the template's own page, a page in a
// list, a section, the site, a map of params, ...), and rewrites each
// field and function for that kind. Anything it cannot map renders
// nothing and is reported, so the set still builds; if it does not, the
// core falls back to the starter templates.
//
// Call it after the content has been planned, so section-specific layouts
// can be attached to their sections' _index.md files.
func planHugoTemplates(p *Plan, dirs []string) {
	t := &hugoTr{p: p, files: map[string]hugoFile{}, done: map[string]bool{}, partials: map[string]*strings.Builder{}, assets: map[string]bool{}}
	t.collect(dirs)
	if !t.hasLayouts() {
		if len(t.files) > 0 {
			p.Report.Add(Review, "", 0, "The Hugo layouts hold no page layouts, only partials or shortcodes (the theme may be a Hugo module), so the site uses Nida's starter templates.")
		}
		return
	}
	if err := t.translate(); err != nil {
		p.Report.Add(Review, "", 0, "Hugo layouts were not translated (%v); the site uses Nida's starter templates.", err)
		p.candidates = map[string][]byte{}
		return
	}
	for name, data := range t.out {
		p.Template(name, data)
	}
	p.Report.Add(Changed, "", 0, "Translated %d Hugo layouts into Nida templates. The translation is best effort: compare the pages with the old site.", t.translated)
}

type hugoFile struct {
	origin string // source-relative path
	text   string
}

type hugoTr struct {
	p     *Plan
	files map[string]hugoFile // layout-relative path -> file, first dir wins
	out   map[string][]byte
	// partial variants waiting to be translated, and those done.
	queue    []partialReq
	done     map[string]bool
	partials map[string]*strings.Builder // output file -> defines
	assets   map[string]bool
	// baseBlock is the block in baseof.html that page layouts fill.
	baseBlock  string
	hasBase    bool
	translated int
	helpers    bool
}

type partialReq struct {
	rel      string // layout-relative file
	define   string // template name to define
	out      string // output file
	arg      hval
	rootKind string
}

// reservedNames are template names Nida uses itself; Hugo blocks with
// these names are renamed so they cannot replace Nida's entry points.
var reservedNames = []string{"base", "index", "post", "page", "list", "section", "taxonomy", "taxonomy_list", "taxonomy_single", "404", "nida-seo", "nida-scripts", "redirect", "content"}

func blockName(name string) string {
	if slices.Contains(reservedNames, name) {
		return "hugo-" + name
	}
	return name
}

func (t *hugoTr) collect(dirs []string) {
	for _, dir := range dirs {
		_ = t.p.Walk(dir, func(rel string) error {
			inner := strings.TrimPrefix(rel, strings.TrimSuffix(dir, "/")+"/")
			if _, seen := t.files[inner]; seen {
				return nil
			}
			data, err := t.p.ReadFile(rel)
			if err != nil {
				return nil
			}
			t.files[inner] = hugoFile{origin: rel, text: string(data)}
			return nil
		})
	}
}

// hasLayouts reports whether there is at least one page layout to
// translate. Shortcode, partial, and render-hook templates alone (a site
// whose theme is a Hugo module, for example) would translate into bare
// skeleton pages, so the starter templates are better then.
func (t *hugoTr) hasLayouts() bool {
	for rel := range t.files {
		if !strings.HasSuffix(rel, ".html") {
			continue
		}
		first, _, _ := strings.Cut(rel, "/")
		switch first {
		case "shortcodes", "_shortcodes", "partials", "_partials", "_markup", "_internal":
			continue
		}
		return true
	}
	return false
}

// find returns the first existing layout among candidates.
func (t *hugoTr) find(candidates ...string) (string, bool) {
	for _, c := range candidates {
		if _, ok := t.files[c]; ok {
			return c, true
		}
	}
	return "", false
}

func (t *hugoTr) exists(rel string) bool {
	_, ok := t.files[rel]
	if !ok && strings.HasPrefix(rel, "partials/") {
		_, ok = t.files["_"+rel]
	}
	return ok
}

type entrySpec struct {
	name   string
	flavor string
	file   string // layout-relative, or "" to generate one
}

func (t *hugoTr) translate() error {
	t.out = map[string][]byte{}

	// Report layouts that have no Nida counterpart.
	for _, rel := range sortedKeys(t.files) {
		f := t.files[rel]
		switch {
		case strings.Contains(rel, "_markup/"):
			t.p.Report.Add(Unsupported, f.origin, 0, "Render hooks have no Nida equivalent; links, images, and headings use Nida's Markdown output.")
		case !strings.HasSuffix(rel, ".html"):
			t.p.Report.Add(Unsupported, f.origin, 0, "Hugo output formats other than HTML are not translated; Nida writes its own feeds, sitemap, robots.txt, and search index.")
		}
	}

	base, hasBase := t.find("_default/baseof.html", "baseof.html")
	t.hasBase = hasBase
	if hasBase {
		if err := t.translateBase(base); err != nil {
			return err
		}
	} else {
		t.out["templates/base.html"] = []byte("{{ define \"base\" }}{{ end }}\n")
	}

	single, hasSingle := t.find("_default/single.html", "single.html", "page.html")
	list, hasList := t.find("_default/list.html", "list.html", "section.html", "_default/section.html")
	home, hasHome := t.find("index.html", "home.html", "_default/home.html")
	if !hasHome && hasList {
		home, hasHome = list, true
	}

	entries := []entrySpec{
		{"post", "single", pick(single, hasSingle)},
		{"page", "single", pick(single, hasSingle)},
		{"index", "home", pick(home, hasHome)},
		{"list", "list", pick(list, hasList)},
	}
	if section, ok := t.find("_default/section.html", "section.html"); ok && section != list {
		entries = append(entries, entrySpec{"section", "list", section})
	}

	// Taxonomy layouts: Hugo 0.73+ uses taxonomy.html for the term list and
	// term.html for one term; older themes used terms.html and taxonomy.html.
	termsFile, termFile := "", ""
	if term, ok := t.find("_default/term.html", "term.html"); ok {
		termFile = term
		termsFile, _ = t.find("_default/taxonomy.html", "taxonomy.html", "_default/terms.html", "terms.html")
	} else {
		termsFile, _ = t.find("_default/terms.html", "terms.html")
		termFile, _ = t.find("_default/taxonomy.html", "taxonomy.html")
	}
	if termFile == "" && hasList {
		termFile = list
	}
	entries = append(entries, entrySpec{"taxonomy_single", "term", termFile}, entrySpec{"taxonomy_list", "terms", termsFile})
	if nf, ok := t.find("404.html", "_default/404.html"); ok {
		entries = append(entries, entrySpec{"404", "404", nf})
	}

	// Other layouts: custom ones chosen with `layout` front matter, and
	// section-specific ones.
	known := map[string]bool{}
	for _, e := range entries {
		known[e.file] = true
	}
	known[base] = true
	for _, rel := range sortedKeys(t.files) {
		if known[rel] || !strings.HasSuffix(rel, ".html") || isSpecialLayoutDir(rel) {
			continue
		}
		dir, file := path.Split(rel)
		dir = strings.Trim(dir, "/")
		stem := strings.TrimSuffix(file, ".html")
		if dir == "" || strings.Contains(dir, "/") || stem == "baseof" {
			continue
		}
		switch {
		case dir == "_default":
			if slices.Contains([]string{"summary", "li", "card"}, stem) || strings.HasPrefix(stem, "summary") {
				continue // views, translated when .Render uses them
			}
			entries = append(entries, entrySpec{stem, "single", rel})
		case stem == "single":
			entries = append(entries, entrySpec{dir + "-single", "single", rel})
			t.attachSection(dir, "page_template", dir+"-single", rel)
		case stem == "list" || stem == "section":
			entries = append(entries, entrySpec{dir + "-list", "list", rel})
			t.attachSection(dir, "template", dir+"-list", rel)
		default:
			flavor := "single"
			if slices.Contains([]string{"terms", "taxonomy", "term"}, stem) {
				continue
			}
			entries = append(entries, entrySpec{dir + "-" + stem, flavor, rel})
			if _, clash := t.files["_default/"+stem+".html"]; !clash {
				entries = append(entries, entrySpec{stem, flavor, rel})
			}
		}
	}

	seen := map[string]bool{}
	for _, e := range entries {
		if seen[e.name] {
			continue
		}
		seen[e.name] = true
		if err := t.translateEntry(e); err != nil {
			return err
		}
	}

	// Shortcodes.
	for _, rel := range sortedKeys(t.files) {
		for _, prefix := range []string{"shortcodes/", "_shortcodes/"} {
			if name, ok := strings.CutPrefix(rel, prefix); ok && strings.HasSuffix(name, ".html") && !strings.Contains(name, "/") {
				if err := t.translateShortcode(rel, strings.TrimSuffix(name, ".html")); err != nil {
					return err
				}
			}
		}
	}

	if err := t.drainPartials(); err != nil {
		return err
	}
	for file, b := range t.partials {
		t.out[file] = []byte(b.String())
	}
	if t.helpers {
		t.out["templates/partials/nida-migrate-helpers.html"] = []byte(hugoHelpers)
	}
	return nil
}

func pick(file string, ok bool) string {
	if ok {
		return file
	}
	return ""
}

func isSpecialLayoutDir(rel string) bool {
	for _, dir := range []string{"partials/", "_partials/", "shortcodes/", "_shortcodes/", "_markup/"} {
		if strings.HasPrefix(rel, dir) || strings.Contains(rel, "/"+dir) {
			return true
		}
	}
	return false
}

// attachSection points a section at a section-specific layout.
func (t *hugoTr) attachSection(section, key, name, origin string) {
	dest := "content/" + section + "/_index.md"
	doc := t.p.Doc(dest)
	if doc == nil {
		t.p.Report.Add(Review, t.files[origin].origin, 0, "This layout became the `%s` template; set `%s = %q` in the section's `_index.md` to use it.", name, key, name)
		return
	}
	if !doc.Has(key) {
		_ = doc.Set(key, name)
	}
}

// parseHugo parses a layout without checking function names, since most
// Hugo functions do not exist in Nida.
func parseHugo(name, text string) (*parse.Tree, map[string]*parse.Tree, error) {
	tree := parse.New(name)
	tree.Mode = parse.ParseComments | parse.SkipFuncCheck
	set := map[string]*parse.Tree{}
	main, err := tree.Parse(text, "", "", set)
	if err != nil {
		return nil, nil, err
	}
	return main, set, nil
}

func (t *hugoTr) parseFile(rel string) (*parse.Tree, map[string]*parse.Tree, bool) {
	f := t.files[rel]
	main, set, err := parseHugo(rel, f.text)
	if err != nil {
		t.p.Report.Add(Manual, f.origin, 0, "This layout does not parse as a Go template (%v), so it was not translated.", err)
		return nil, nil, false
	}
	t.translated++
	return main, set, true
}

func ctxVal(flavor string) hval { return hval{expr: ".", k: kCtx, flavor: flavor} }

func (t *hugoTr) entryScope(file string, tree *parse.Tree, flavor string) *scope {
	dot := ctxVal(flavor)
	dollar := dot
	dollar.expr = "$"
	return &scope{dot: dot, dollar: dollar, vars: map[string]hval{}, root: "$", rootKind: "entry", file: t.files[file].origin, tree: tree}
}

func (t *hugoTr) translateBase(rel string) error {
	main, set, ok := t.parseFile(rel)
	if !ok {
		return fmt.Errorf("baseof.html does not parse")
	}
	var b strings.Builder
	sc := t.entryScope(rel, main, "any")
	body := t.list(sc, main.Root)
	// Nida's bundled math stylesheet loads through nida-scripts.
	if i := strings.LastIndex(body, "</body>"); i >= 0 {
		body = body[:i] + "{{ template \"nida-scripts\" . }}\n" + body[i:]
	}
	b.WriteString("{{ define \"base\" }}" + body + "{{ end }}\n")
	for _, name := range sortedKeys(set) {
		if name == rel {
			continue
		}
		if t.baseBlock == "" || name == "main" {
			t.baseBlock = name
		}
		tsc := t.entryScope(rel, set[name], "any")
		fmt.Fprintf(&b, "{{ define %q }}%s{{ end }}\n", blockName(name), t.list(tsc, set[name].Root))
	}
	if t.baseBlock == "" {
		t.baseBlock = "main"
	}
	t.out["templates/base.html"] = []byte(b.String())
	return nil
}

func (t *hugoTr) translateEntry(e entrySpec) error {
	var b strings.Builder
	if e.file == "" {
		t.generateEntry(&b, e)
		t.out["templates/"+e.name+".html"] = []byte(b.String())
		return nil
	}
	main, set, ok := t.parseFile(e.file)
	if !ok {
		t.generateEntry(&b, e)
		t.out["templates/"+e.name+".html"] = []byte(b.String())
		return nil
	}
	defines := 0
	for name := range set {
		if name != e.file {
			defines++
		}
	}
	if defines > 0 && t.hasBase {
		fmt.Fprintf(&b, "{{ define %q }}{{ template \"base\" . }}{{ end }}\n", e.name)
		for _, name := range sortedKeys(set) {
			if name == e.file {
				continue
			}
			sc := t.entryScope(e.file, set[name], e.flavor)
			fmt.Fprintf(&b, "{{ define %q }}%s{{ end }}\n", blockName(name), t.list(sc, set[name].Root))
		}
	} else {
		sc := t.entryScope(e.file, main, e.flavor)
		fmt.Fprintf(&b, "{{ define %q }}%s{{ end }}\n", e.name, t.list(sc, main.Root))
		for _, name := range sortedKeys(set) {
			if name == e.file {
				continue
			}
			dsc := t.entryScope(e.file, set[name], e.flavor)
			fmt.Fprintf(&b, "{{ define %q }}%s{{ end }}\n", blockName(name), t.list(dsc, set[name].Root))
		}
	}
	t.out["templates/"+e.name+".html"] = []byte(b.String())
	return nil
}

// generateEntry writes a simple template for a kind the Hugo layouts do
// not cover, filling the base layout's main block when there is one.
func (t *hugoTr) generateEntry(b *strings.Builder, e entrySpec) {
	var body string
	switch e.flavor {
	case "single", "404":
		body = `<article><h1>{{ .Title }}</h1>{{ unsafeHTML .Page.BodyHTML }}</article>`
		if e.flavor == "404" {
			body = `<h1>Page not found</h1><p><a href="{{ relURL "/" }}">Home</a></p>`
		}
	case "terms":
		body = `<h1>{{ .Taxonomy.Name }}</h1><ul>{{ range .Terms }}<li><a href="{{ relURL .URL }}">{{ .Name }}</a> ({{ len .Items }})</li>{{ end }}</ul>`
	default:
		body = `<h1>{{ .Title }}</h1>{{ unsafeHTML .Section.BodyHTML }}<ul>{{ range .Pages }}<li><a href="{{ relURL .URL }}">{{ .Title }}</a></li>{{ end }}</ul>`
	}
	if t.hasBase {
		fmt.Fprintf(b, "{{ define %q }}{{ template \"base\" . }}{{ end }}\n{{ define %q }}%s{{ end }}\n", e.name, blockName(t.baseBlock), body)
		return
	}
	fmt.Fprintf(b, "{{ define %q }}<!doctype html>\n<html><head><meta charset=\"utf-8\"><title>{{ .Title }}</title></head><body>%s</body></html>{{ end }}\n", e.name, body)
}

func (t *hugoTr) translateShortcode(rel, name string) error {
	main, set, ok := t.parseFile(rel)
	if !ok {
		return nil
	}
	dot := hval{expr: ".", k: kShort}
	dollar := hval{expr: "$", k: kShort}
	sc := &scope{dot: dot, dollar: dollar, vars: map[string]hval{}, root: "$", rootKind: "shortcode", file: t.files[rel].origin, tree: main}
	var b strings.Builder
	b.WriteString(t.list(sc, main.Root))
	for _, dname := range sortedKeys(set) {
		if dname != rel {
			fmt.Fprintf(&b, "{{ define %q }}%s{{ end }}", blockName(dname), t.list(sc, set[dname].Root))
		}
	}
	t.out["templates/shortcodes/"+name+".html"] = []byte(b.String())
	return nil
}

// partialCall translates {{ partial "name" arg }} into a call of the
// partial's variant for the kind of arg. Each variant receives a dict with
// the argument as .dot and Nida's context as .ctx.
func (t *hugoTr) partialCall(sc *scope, n parse.Node, name string, arg hval, view bool) string {
	rel := name
	if !strings.HasSuffix(rel, ".html") && path.Ext(rel) == "" {
		rel += ".html"
	}
	var file string
	var ok bool
	if view {
		file, ok = t.find("_default/"+rel, rel)
	} else {
		file, ok = t.find("partials/"+rel, "_partials/"+rel)
	}
	if !ok {
		t.manual(sc, n, "Partial `%s` was not found in the layouts, so it renders nothing.", name)
		return ""
	}
	kind := sc.rootKind
	define := strings.TrimSuffix(file, ".html") + "#" + arg.sig() + "#" + kind
	out := "templates/partials/" + strings.ReplaceAll(strings.TrimSuffix(file, ".html"), "/", "__") + ".html"
	if !t.done[define] {
		t.done[define] = true
		t.queue = append(t.queue, partialReq{rel: file, define: define, out: out, arg: arg, rootKind: kind})
	}
	d := call("dict", `"dot"`, arg.arg())
	if sc.root != "" {
		d = call("dict", `"ctx"`, sc.root, `"dot"`, arg.arg())
	}
	return fmt.Sprintf("{{ template %q %s }}", define, d)
}

func (t *hugoTr) drainPartials() error {
	for len(t.queue) > 0 {
		req := t.queue[0]
		t.queue = t.queue[1:]
		main, set, ok := t.parseFile(req.rel)
		if !ok {
			fmt.Fprintf(t.partialBuilder(req.out), "{{ define %q }}{{ end }}\n", req.define)
			continue
		}
		root := ""
		if req.rootKind != "" {
			root = "$.ctx"
		}
		sc := &scope{dot: req.arg.withExpr(".dot"), dollar: req.arg.withExpr("$.dot"), vars: map[string]hval{}, root: root, rootKind: req.rootKind, file: t.files[req.rel].origin, tree: main}
		b := t.partialBuilder(req.out)
		fmt.Fprintf(b, "{{ define %q }}%s{{ end }}\n", req.define, t.list(sc, main.Root))
		for _, name := range sortedKeys(set) {
			if name != req.rel {
				fmt.Fprintf(b, "{{ define %q }}%s{{ end }}\n", blockName(name), t.list(sc, set[name].Root))
			}
		}
	}
	return nil
}

func (t *hugoTr) partialBuilder(file string) *strings.Builder {
	b, ok := t.partials[file]
	if !ok {
		b = &strings.Builder{}
		t.partials[file] = b
	}
	return b
}

// list prints a node list.
func (t *hugoTr) list(sc *scope, l *parse.ListNode) string {
	if l == nil {
		return ""
	}
	var b strings.Builder
	for _, n := range l.Nodes {
		b.WriteString(t.node(sc, n))
	}
	return b.String()
}

func (t *hugoTr) node(sc *scope, n parse.Node) string {
	switch n := n.(type) {
	case *parse.TextNode:
		return string(n.Text)
	case *parse.CommentNode:
		return n.String()
	case *parse.ActionNode:
		return t.action(sc, n)
	case *parse.IfNode:
		c := sc.child()
		cond := t.pipe(c, n.Pipe)
		decl := t.declPrefix(c, n.Pipe)
		t.declare(c, n.Pipe, cond)
		out := "{{ if " + decl + unwrap(cond.condExpr()) + " }}" + t.list(c, n.List)
		if n.ElseList != nil {
			out += "{{ else }}" + t.list(c, n.ElseList)
		}
		return out + "{{ end }}"
	case *parse.WithNode:
		c := sc.child()
		v := t.pipe(c, n.Pipe)
		decl := t.declPrefix(c, n.Pipe)
		t.declare(c, n.Pipe, v)
		inner := c.child()
		inner.dot = v.withExpr(".")
		out := "{{ with " + decl + unwrap(v.condExpr()) + " }}" + t.list(inner, n.List)
		if n.ElseList != nil {
			out += "{{ else }}" + t.list(c, n.ElseList)
		}
		return out + "{{ end }}"
	case *parse.RangeNode:
		return t.rangeNode(sc, n)
	case *parse.TemplateNode:
		return t.templateNode(sc, n)
	case *parse.BreakNode, *parse.ContinueNode:
		return n.String()
	}
	t.review(sc, n, "This template construct was copied without translation: `%s`.", truncate(n.String(), 60))
	return n.String()
}

// condExpr is the expression used in if, with, and range.
func (v hval) condExpr() string {
	switch {
	case v.k == kFallback:
		return `""`
	case v.expr == "":
		// A value known only at translation time, such as .Site or a Hugo
		// Pipes resource, always exists.
		return "true"
	}
	return v.expr
}

// declPrefix renders "$x := " for a pipeline with declarations, recording
// the variables' kinds in sc.
func (t *hugoTr) declPrefix(sc *scope, p *parse.PipeNode) string {
	if len(p.Decl) == 0 {
		return ""
	}
	names := make([]string, len(p.Decl))
	for i, d := range p.Decl {
		names[i] = d.Ident[0]
	}
	op := " := "
	if p.IsAssign {
		op = " = "
	}
	return strings.Join(names, ", ") + op
}

// declare records the kinds of a pipeline's variables.
func (t *hugoTr) declare(sc *scope, p *parse.PipeNode, v hval) {
	for _, d := range p.Decl {
		if v.expr == "" {
			sc.vars[d.Ident[0]] = v
		} else {
			sc.vars[d.Ident[0]] = v.withExpr(d.Ident[0])
		}
	}
}

func (t *hugoTr) action(sc *scope, n *parse.ActionNode) string {
	p := n.Pipe
	if len(p.Decl) == 0 && len(p.Cmds) == 1 {
		if out, ok := t.specialAction(sc, n, p.Cmds[0]); ok {
			return out
		}
	}
	v := t.pipe(sc, p)
	if len(p.Decl) > 0 {
		decl := t.declPrefix(sc, p)
		expr := v.expr
		if v.k == kFallback {
			expr = `""`
		}
		t.declare(sc, p, v)
		if expr == "" {
			// The value exists only at translation time, such as a Hugo
			// Pipes resource; uses of the variable are translated from it.
			return ""
		}
		return "{{ " + decl + unwrap(expr) + " }}"
	}
	if v.expr == "" && v.k != kFallback {
		t.review(sc, n, "`%s` has no printable Nida value; it renders nothing.", truncate(n.String(), 60))
		return ""
	}
	switch v.k {
	case kFallback:
		return ""
	case kHTML:
		return "{{ unsafeHTML " + paren(v.expr) + " }}"
	case kInner:
		return "{{ " + v.base + " }}"
	case kMap:
		return "{{ " + unwrap(v.expr) + " }}"
	}
	return "{{ " + unwrap(v.expr) + " }}"
}

// specialAction handles actions that become template calls: partials,
// .Render views, .TableOfContents, and scratch writes.
func (t *hugoTr) specialAction(sc *scope, n *parse.ActionNode, cmd *parse.CommandNode) (string, bool) {
	name := funcName(cmd.Args[0])
	switch name {
	case "partial", "partialCached", "partials.Include", "partials.IncludeCached":
		if len(cmd.Args) < 2 {
			return "", false
		}
		s, ok := cmd.Args[1].(*parse.StringNode)
		if !ok {
			t.manual(sc, n, "A partial with a computed name cannot be translated; it renders nothing.")
			return "", true
		}
		arg := hval{expr: "nil", k: kAny}
		if len(cmd.Args) > 2 {
			arg = t.arg(sc, cmd.Args[2], nil)
		}
		return t.partialCall(sc, n, s.Text, arg, false), true
	case "template":
		return "", false
	}
	// Field chains ending in a method that becomes a template call.
	fields, base, ok := splitChain(cmd.Args[0])
	if !ok || len(fields) == 0 {
		return "", false
	}
	last := fields[len(fields)-1]
	switch last {
	case "TableOfContents":
		owner := t.chain(sc, cmd.Args[0], base, fields[:len(fields)-1], nil)
		toc := ""
		switch owner.k {
		case kCtx:
			toc = sel(sel(owner.expr, "Page"), "TOC")
		case kPage:
			toc = sel(owner.expr, "TOC")
		}
		if toc == "" {
			t.review(sc, n, "`.TableOfContents` is only available for pages here; it renders nothing.")
			return "", true
		}
		t.helpers = true
		return fmt.Sprintf("{{ template \"nida-migrate-toc\" %s }}", toc), true
	case "Render":
		if len(cmd.Args) == 2 {
			if s, ok := cmd.Args[1].(*parse.StringNode); ok {
				owner := t.chain(sc, cmd.Args[0], base, fields[:len(fields)-1], nil)
				return t.partialCall(sc, n, s.Text, owner, true), true
			}
		}
	case "Set", "Add", "SetInMap", "DeleteInMap", "Delete":
		owner := t.chain(sc, cmd.Args[0], base, fields[:len(fields)-1], nil)
		if owner.k == kScratch || owner.k == kFallback {
			t.manual(sc, n, "`.Scratch` and `.Store` writes have no Nida equivalent; use template variables (`$x = ...`) instead.")
			return "", true
		}
	}
	return "", false
}

func funcName(n parse.Node) string {
	switch n := n.(type) {
	case *parse.IdentifierNode:
		return n.Ident
	case *parse.ChainNode:
		if id, ok := n.Node.(*parse.IdentifierNode); ok && hugoNamespaces[id.Ident] {
			return id.Ident + "." + strings.Join(n.Field, ".")
		}
	}
	return ""
}

// hugoNamespaces are Hugo's function namespaces, such as strings.ToLower.
var hugoNamespaces = map[string]bool{
	"collections": true, "compare": true, "crypto": true, "css": true, "data": true, "debug": true,
	"diagrams": true, "encoding": true, "fmt": true, "hash": true, "images": true, "inflect": true,
	"js": true, "lang": true, "math": true, "openapi3": true, "os": true, "partials": true, "path": true,
	"reflect": true, "resources": true, "safe": true, "strings": true, "templates": true, "time": true,
	"transform": true, "urls": true, "cast": true,
}

// splitChain returns the field names of a field chain and its base node.
func splitChain(n parse.Node) ([]string, parse.Node, bool) {
	switch n := n.(type) {
	case *parse.FieldNode:
		return n.Ident, nil, true
	case *parse.VariableNode:
		return n.Ident[1:], n, true
	case *parse.ChainNode:
		if funcName(n) != "" {
			return nil, nil, false
		}
		return n.Field, n.Node, true
	}
	return nil, nil, false
}

func (t *hugoTr) rangeNode(sc *scope, n *parse.RangeNode) string {
	c := sc.child()
	v := t.pipe(c, n.Pipe)
	elem := v.rangeElem()
	expr := v.condExpr()
	if v.k == kFallback || v.expr == "" {
		expr = "(slice)"
	}
	inner := c.child()
	prefix := ""
	var head string
	switch len(n.Pipe.Decl) {
	case 0:
		head = "{{ range " + unwrap(expr) + " }}"
	case 1:
		name := n.Pipe.Decl[0].Ident[0]
		inner.vars[name] = elem.withExpr(name)
		head = "{{ range " + name + " := " + unwrap(expr) + " }}"
	default:
		key, value := n.Pipe.Decl[0].Ident[0], n.Pipe.Decl[1].Ident[0]
		if v.k == kTermList && v.mapForm {
			// Hugo ranges over terms as a map of name to pages; Nida has a
			// list of terms, so the name is read from each one.
			inner.vars[value] = elem.withExpr(value)
			inner.vars[key] = hval{expr: key, k: kString}
			head = "{{ range " + value + " := " + unwrap(expr) + " }}"
			prefix = "{{ " + key + " := " + sel(value, "Name") + " }}"
		} else {
			keyKind := hval{expr: key, k: kAny}
			if v.k == kTaxMap || v.k == kTermMap || v.k == kMap {
				keyKind.k = kString
			} else if v.k != kAny {
				keyKind.k = kNumber
			}
			inner.vars[key] = keyKind
			inner.vars[value] = elem.withExpr(value)
			head = "{{ range " + key + ", " + value + " := " + unwrap(expr) + " }}"
		}
	}
	inner.dot = elem.withExpr(".")
	out := head + prefix + t.list(inner, n.List)
	if n.ElseList != nil {
		out += "{{ else }}" + t.list(c, n.ElseList)
	}
	return out + "{{ end }}"
}

func (t *hugoTr) templateNode(sc *scope, n *parse.TemplateNode) string {
	arg := hval{expr: "nil", k: kAny}
	if n.Pipe != nil {
		arg = t.pipe(sc, n.Pipe)
	}
	switch {
	case strings.HasPrefix(n.Name, "_internal/"):
		switch strings.TrimPrefix(n.Name, "_internal/") {
		case "pagination.html":
			if arg.k == kCtx {
				t.helpers = true
				return fmt.Sprintf("{{ template \"nida-migrate-pagination\" %s }}", arg.arg())
			}
		case "opengraph.html", "twitter_cards.html", "schema.html":
			t.review(sc, n, "Hugo's `%s` was dropped. Nida's `nida-seo` partial writes the same tags, plus `<title>`: replace your `<title>` with `{{ template \"nida-seo\" . }}`.", n.Name)
			return ""
		}
		t.review(sc, n, "Hugo's internal template `%s` has no Nida equivalent and was dropped.", n.Name)
		return ""
	case strings.HasPrefix(n.Name, "partials/") || strings.HasPrefix(n.Name, "_partials/"):
		name := strings.TrimPrefix(strings.TrimPrefix(n.Name, "_partials/"), "partials/")
		return t.partialCall(sc, n, name, arg, false)
	}
	if arg.k == kFallback {
		return fmt.Sprintf("{{ template %q }}", blockName(n.Name))
	}
	return fmt.Sprintf("{{ template %q %s }}", blockName(n.Name), unwrap(arg.expr))
}

// pipe translates a pipeline into one nested expression: Go templates
// pass the piped value as the last argument, which lets functions whose
// argument order differs between Hugo and Nida be reordered.
func (t *hugoTr) pipe(sc *scope, p *parse.PipeNode) hval {
	var v hval
	for i, cmd := range p.Cmds {
		var piped *hval
		if i > 0 {
			piped = &v
		}
		v = t.command(sc, cmd, piped)
	}
	return v
}

func (t *hugoTr) command(sc *scope, cmd *parse.CommandNode, piped *hval) hval {
	if len(cmd.Args) == 0 {
		return fallbackVal()
	}
	first := cmd.Args[0]
	var rest []hval
	for _, a := range cmd.Args[1:] {
		rest = append(rest, t.arg(sc, a, nil))
	}
	if piped != nil {
		rest = append(rest, *piped)
	}
	if name := funcName(first); name != "" {
		return t.fn(sc, cmd, name, rest)
	}
	switch f := first.(type) {
	case *parse.FieldNode, *parse.ChainNode, *parse.VariableNode:
		return t.arg(sc, f, rest)
	}
	if len(rest) > 0 {
		t.review(sc, cmd, "`%s` could not be translated; it renders nothing.", truncate(cmd.String(), 60))
		return fallbackVal()
	}
	return t.arg(sc, first, nil)
}

// arg translates one operand. callArgs are arguments for a method at the
// end of a field chain.
func (t *hugoTr) arg(sc *scope, n parse.Node, callArgs []hval) hval {
	switch n := n.(type) {
	case *parse.DotNode:
		return sc.dot
	case *parse.FieldNode:
		return t.walk(sc, n, sc.dot, n.Ident, callArgs)
	case *parse.VariableNode:
		base := sc.dollar
		if n.Ident[0] != "$" {
			v, ok := sc.vars[n.Ident[0]]
			if !ok {
				v = hval{expr: n.Ident[0], k: kAny}
			}
			base = v
		}
		return t.walk(sc, n, base, n.Ident[1:], callArgs)
	case *parse.ChainNode:
		if name := funcName(n); name != "" {
			return t.fn(sc, n, name, callArgs)
		}
		base := t.arg(sc, n.Node, nil)
		return t.walk(sc, n, base, n.Field, callArgs)
	case *parse.IdentifierNode:
		return t.fn(sc, n, n.Ident, callArgs)
	case *parse.PipeNode:
		v := t.pipe(sc, n)
		if v.k != kFallback && !wrapped(v.expr) && strings.Contains(v.expr, " ") {
			v.expr = "(" + v.expr + ")"
		}
		return v
	case *parse.StringNode:
		return hval{expr: n.Quoted, k: kString}
	case *parse.NumberNode:
		return hval{expr: n.Text, k: kNumber}
	case *parse.BoolNode:
		return hval{expr: strconv.FormatBool(n.True), k: kBool}
	case *parse.NilNode:
		return hval{expr: "nil", k: kAny}
	}
	t.review(sc, n, "`%s` could not be translated; it renders nothing.", truncate(n.String(), 60))
	return fallbackVal()
}

func (t *hugoTr) walk(sc *scope, n parse.Node, base hval, fields []string, callArgs []hval) hval {
	if len(fields) == 0 {
		if len(callArgs) > 0 {
			t.review(sc, n, "`%s` could not be translated; it renders nothing.", truncate(n.String(), 60))
			return fallbackVal()
		}
		return base
	}
	v := base
	for i, f := range fields {
		var args []hval
		if i == len(fields)-1 {
			args = callArgs
		}
		v = t.step(sc, n, v, f, args)
	}
	return v
}

// chain translates a field chain whose base is node (nil for the dot).
func (t *hugoTr) chain(sc *scope, n, base parse.Node, fields []string, args []hval) hval {
	start := sc.dot
	switch b := base.(type) {
	case *parse.VariableNode:
		start = sc.dollar
		if b.Ident[0] != "$" {
			if v, ok := sc.vars[b.Ident[0]]; ok {
				start = v
			} else {
				start = hval{expr: b.Ident[0], k: kAny}
			}
		}
	case nil:
	default:
		start = t.arg(sc, b, nil)
	}
	return t.walk(sc, n, start, fields, args)
}

// hugoHelpers are templates the translations call for Hugo features that
// are built from Nida data: the table of contents and pagination.
const hugoHelpers = `{{ define "nida-migrate-toc" }}{{ if . }}<nav id="TableOfContents">{{ template "nida-migrate-toc-list" . }}</nav>{{ end }}{{ end }}
{{ define "nida-migrate-toc-list" }}{{ if . }}<ul>{{ range . }}<li><a href="#{{ .ID }}">{{ .Title }}</a>{{ template "nida-migrate-toc-list" .Children }}</li>{{ end }}</ul>{{ end }}{{ end }}
{{ define "nida-migrate-pagination" }}{{ with .Paginator }}<ul class="pagination">
{{ with .Previous }}<li class="page-item"><a class="page-link" href="{{ relURL . }}" aria-label="Previous">&laquo;</a></li>{{ end }}
{{ range .PageLinks }}<li class="page-item{{ if .Current }} active{{ end }}"><a class="page-link" href="{{ relURL .URL }}">{{ .Number }}</a></li>{{ end }}
{{ with .Next }}<li class="page-item"><a class="page-link" href="{{ relURL . }}" aria-label="Next">&raquo;</a></li>{{ end }}
</ul>{{ end }}{{ end }}
`
