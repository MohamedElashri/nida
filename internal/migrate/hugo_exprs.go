package migrate

import (
	"slices"
	"strconv"
	"strings"
	"text/template/parse"
)

// Hugo template values are translated together with a kind, which says
// what Nida value the expression denotes. Hugo hands templates a Page with
// methods for everything; Nida hands them a context struct with .Page,
// .Section, .Config, and .Index, so the same Hugo field maps to different
// Nida paths depending on what it is called on.
type hkind int

const (
	kAny      hkind = iota // unknown; fields pass through unchanged
	kFallback              // could not be translated; prints nothing
	kString
	kHTML  // a string holding HTML, printed through unsafeHTML
	kSafe  // template.HTML, printed as is
	kInner // a shortcode's inner content
	kTime
	kBool
	kNumber
	kCtx   // Nida's template context, standing in for Hugo's page
	kShort // a shortcode call
	kPage
	kSection
	kSite
	kPageList
	kSectionList
	kTermList  // []taxonomies.Term
	kTerm      // taxonomies.Term
	kTermNames // a page's terms for one taxonomy, as strings
	kTermName
	kTaxMap   // .Index.TaxonomyMap: taxonomy -> term -> pages
	kTermMap  // term -> pages
	kWeighted // the pages of one term from a kTermMap
	kPaginator
	kPager
	kPagerList
	kPagerLink
	kMap // a map[string]any reached through Params or Extra
	kDict
	kMenus
	kMenuList
	kMenuEntry
	kFile
	kResources
	kBundleRes
	kAsset
	kOutputFormats
	kLanguage
	kHugo
	kScratch
	kGroupList
	kGroup
	kData
	kDirect // a dict whose keys already use Hugo's field names
)

// hval is a translated Hugo value: a Nida expression and its kind.
type hval struct {
	expr string
	k    hkind
	// flavor is the template kind for kCtx: single, list, home, term,
	// terms, 404, or any.
	flavor string
	// base and path build kMap expressions; base is also the page
	// expression for kFile, kResources, and kBundleRes.
	base string
	path []string
	// fields are the values of a kDict.
	fields map[string]hval
	// elem is the element of a list kind.
	elem *hval
	// name is the taxonomy of kTermNames and kTermName, the asset path of
	// kAsset, the group layout of kGroupList, and the resource name
	// expression of kBundleRes.
	name string
	// mapForm marks a kTermList that Hugo would range over as a map.
	mapForm bool
}

func fallbackVal() hval { return hval{expr: `""`, k: kFallback} }

func lit(s string) hval { return hval{expr: strconv.Quote(s), k: kString} }

func listOf(expr string, k hkind, elem hval) hval {
	return hval{expr: expr, k: k, elem: &elem}
}

func pageVal(expr string) hval { return hval{expr: expr, k: kPage} }

func mapVal(base string) hval { return hval{expr: base, k: kMap, base: base} }

// arg returns the expression to pass as a function argument.
func (v hval) arg() string {
	if v.k == kFallback || v.expr == "" {
		return `""`
	}
	return paren(v.expr)
}

// isSimpleRef reports whether e can take a .Field suffix without parens.
func isSimpleRef(e string) bool {
	if e == "" || (e[0] != '$' && e[0] != '.') {
		return false
	}
	return !strings.ContainsAny(e, " ()\"")
}

func paren(e string) string {
	if _, err := strconv.Unquote(e); err == nil {
		return e
	}
	if isSimpleRef(e) || !strings.ContainsAny(e, " ") || wrapped(e) {
		return e
	}
	return "(" + e + ")"
}

// wrapped reports whether e is one parenthesized expression.
func wrapped(e string) bool {
	if !strings.HasPrefix(e, "(") || !strings.HasSuffix(e, ")") {
		return false
	}
	depth, inStr := 0, false
	for i := 0; i < len(e); i++ {
		c := e[i]
		switch {
		case inStr && c == '\\':
			i++
		case c == '"':
			inStr = !inStr
		case inStr:
		case c == '(':
			depth++
		case c == ')':
			depth--
			if depth == 0 && i != len(e)-1 {
				return false
			}
		}
	}
	return true
}

func unwrap(e string) string {
	if wrapped(e) {
		return e[1 : len(e)-1]
	}
	return e
}

func sel(e, field string) string {
	switch {
	case e == ".":
		return "." + field
	case isSimpleRef(e):
		return e + "." + field
	}
	if !wrapped(e) {
		e = "(" + e + ")"
	}
	return e + "." + field
}

func call(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return "(" + strings.Join(parts, " ") + ")"
}

func (v hval) withExpr(e string) hval {
	v.expr = e
	if v.k == kMap {
		v.base, v.path = e, nil
	}
	return v
}

func (v hval) mapExpr() string {
	switch len(v.path) {
	case 0:
		return v.base
	case 1:
		if validIdent(v.path[0]) {
			return sel(v.base, v.path[0])
		}
	}
	return call("dig", paren(v.base), strconv.Quote(strings.Join(v.path, ".")))
}

func validIdent(s string) bool {
	for i, r := range s {
		ok := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (i > 0 && r >= '0' && r <= '9')
		if !ok {
			return false
		}
	}
	return s != ""
}

// sig is a short, stable description of a kind, used to name the variant
// of a partial translated for arguments of that kind.
func (v hval) sig() string {
	switch v.k {
	case kCtx:
		return "ctx-" + v.flavor
	case kDict:
		keys := make([]string, 0, len(v.fields))
		for key := range v.fields {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+"="+v.fields[key].sig())
		}
		return "dict(" + strings.Join(parts, ",") + ")"
	}
	return kindNames[v.k]
}

var kindNames = map[hkind]string{
	kAny: "any", kFallback: "none", kString: "string", kHTML: "html", kSafe: "html", kInner: "inner",
	kTime: "time", kBool: "bool", kNumber: "number", kShort: "shortcode", kPage: "page", kSection: "section",
	kSite: "site", kPageList: "pages", kSectionList: "sections", kTermList: "terms", kTerm: "term",
	kTermNames: "termnames", kTermName: "termname", kTaxMap: "taxonomies", kTermMap: "termmap",
	kWeighted: "weighted", kPaginator: "paginator", kPager: "pager", kPagerList: "pagers", kPagerLink: "pagerlink",
	kMap: "map", kDict: "dict", kMenus: "menus", kMenuList: "menu", kMenuEntry: "menuentry", kFile: "file",
	kResources: "resources", kBundleRes: "resource", kAsset: "asset", kOutputFormats: "outputformats",
	kLanguage: "language", kHugo: "hugo", kScratch: "scratch", kGroupList: "groups", kGroup: "group",
	kData: "data", kDirect: "direct",
}

// rangeElem returns the value the dot takes inside a range over v.
func (v hval) rangeElem() hval {
	if v.elem != nil {
		return *v.elem
	}
	switch v.k {
	case kTaxMap:
		return hval{k: kTermMap}
	case kTermMap:
		return hval{k: kWeighted}
	case kMenus:
		return hval{k: kMenuList, elem: &hval{k: kMenuEntry}}
	}
	return hval{k: kAny}
}

// scope is what a template expression is translated against.
type scope struct {
	dot, dollar hval
	vars        map[string]hval
	// root is an expression for Nida's template context (with .Config,
	// .Index, .Data, .CurrentURL, .BasePath), or for a shortcode call
	// (with .Config only), or "" when neither is reachable.
	root     string
	rootKind string // "entry", "shortcode", or ""
	file     string
	tree     *parse.Tree
}

func (sc *scope) child() *scope {
	c := *sc
	c.vars = make(map[string]hval, len(sc.vars))
	for k, v := range sc.vars {
		c.vars[k] = v
	}
	return &c
}

func (sc *scope) cfg() string {
	if sc.root == "" {
		return ""
	}
	return sel(sc.root, "Config")
}

func (sc *scope) index() string {
	if sc.rootKind != "entry" {
		return ""
	}
	return sel(sc.root, "Index")
}

// step translates one field access, with call arguments for a method.
func (t *hugoTr) step(sc *scope, n parse.Node, v hval, f string, args []hval) hval {
	miss := func() hval {
		t.review(sc, n, "`.%s` has no Nida equivalent here, so it renders nothing.", f)
		return fallbackVal()
	}
	switch v.k {
	case kFallback:
		return v
	case kAny, kTime, kString, kNumber, kBool, kDirect:
		if v.k == kString {
			return miss()
		}
		e := sel(v.expr, f)
		if len(args) > 0 {
			e = call(e, argList(args)...)
		}
		if v.k == kTime {
			switch f {
			case "Format":
				return hval{expr: e, k: kString}
			case "Year", "YearDay", "Day", "Hour", "Minute", "Second", "Unix":
				return hval{expr: e, k: kNumber}
			case "IsZero", "Before", "After", "Equal":
				return hval{expr: e, k: kBool}
			}
			return hval{expr: e, k: kAny}
		}
		return hval{expr: e, k: kAny}
	case kMap:
		if len(args) > 0 {
			return miss()
		}
		out := hval{k: kMap, base: v.base, path: append(slices.Clone(v.path), f)}
		out.expr = out.mapExpr()
		return out
	case kDict:
		if fv, ok := v.fields[f]; ok {
			return fv.withExpr(sel(v.expr, f))
		}
		return hval{expr: sel(v.expr, f), k: kAny}
	case kCtx:
		return t.ctxField(sc, n, v, f, args, miss)
	case kShort:
		return t.shortField(v, f, args, miss)
	case kPage:
		return t.pageField(sc, v.expr, f, args, miss)
	case kSection:
		return t.sectionField(v.expr, f, miss)
	case kSite:
		return t.siteField(sc, n, f, args, miss)
	case kPageList, kSectionList, kTermList, kWeighted:
		return t.listField(sc, n, v, f, args, miss)
	case kTerm:
		switch f {
		case "Name", "Term", "Title", "LinkTitle":
			return hval{expr: sel(v.expr, "Name"), k: kString}
		case "Count", "Len":
			return hval{expr: call("len", sel(v.expr, "Items")), k: kNumber}
		case "Pages", "WeightedPages":
			return listOf(sel(v.expr, "Items"), kPageList, pageVal(""))
		case "Page":
			return hval{expr: v.expr, k: kTerm}
		case "RelPermalink":
			return hval{expr: call("relURL", sel(v.expr, "URL")), k: kString}
		case "Permalink":
			return hval{expr: sel(v.expr, "CanonicalURL"), k: kString}
		}
		return miss()
	case kTermName:
		switch f {
		case "Name", "Title", "LinkTitle", "Term", "Page":
			return hval{expr: v.expr, k: kTermName, name: v.name}
		case "RelPermalink", "Permalink":
			fn := "relURL"
			if f == "Permalink" {
				fn = "absURL"
			}
			return hval{expr: call(fn, call("print", strconv.Quote("/"+v.name+"/"), call("urlize", v.arg()), `"/"`)), k: kString}
		}
		return miss()
	case kTaxMap:
		return hval{expr: call("index", paren(v.expr), strconv.Quote(f)), k: kTermMap}
	case kPaginator:
		return t.paginatorField(v, f, miss)
	case kPager:
		switch f {
		case "URL", "RelPermalink":
			return hval{expr: v.expr, k: kString}
		}
		return miss()
	case kPagerLink:
		switch f {
		case "PageNumber":
			return hval{expr: sel(v.expr, "Number"), k: kNumber}
		case "URL", "RelPermalink":
			return hval{expr: call("relURL", sel(v.expr, "URL")), k: kString}
		}
		return miss()
	case kMenus:
		return listOf(call("dig", paren(v.expr), strconv.Quote("menu."+f)), kMenuList, hval{k: kMenuEntry})
	case kMenuEntry:
		switch f {
		case "Name", "Title", "Weight", "Identifier", "Pre", "Post", "Params", "Parent":
			if f == "Params" {
				return mapVal(call("index", paren(v.expr), `"params"`))
			}
			return hval{expr: call("index", paren(v.expr), strconv.Quote(strings.ToLower(f))), k: kAny}
		case "URL", "RelPermalink", "Permalink":
			return hval{expr: call("relURL", call("print", call("or", call("index", paren(v.expr), `"url"`), call("index", paren(v.expr), `"pageRef"`), `""`))), k: kString}
		case "HasChildren":
			return hval{expr: "false", k: kBool}
		case "Children":
			return listOf("(slice)", kMenuList, hval{k: kMenuEntry})
		}
		return miss()
	case kFile:
		switch f {
		case "Path", "Filename", "UniqueID", "LogicalName":
			return hval{expr: sel(v.base, "RelativePath"), k: kString}
		case "BaseFileName", "ContentBaseName", "TranslationBaseName":
			return hval{expr: sel(v.base, "Slug"), k: kString}
		case "Dir", "Section":
			return hval{expr: sel(v.base, "SectionPath"), k: kString}
		case "Ext":
			return lit("md")
		case "Lang":
			if cfg := sc.cfg(); cfg != "" {
				return hval{expr: sel(cfg, "Language"), k: kString}
			}
		}
		return miss()
	case kResources:
		switch f {
		case "GetMatch", "Get":
			if len(args) == 1 {
				t.review(sc, n, "Page resource lookups become links next to the page; image processing and `.Width`/`.Height` are not available.")
				return hval{expr: args[0].arg(), k: kBundleRes, base: v.base, name: args[0].arg()}
			}
		}
		t.review(sc, n, "`.Resources.%s` has no Nida equivalent; bundle files are listed in `.Page.Resources` as names.", f)
		return listOf("(slice)", kPageList, hval{k: kAny})
	case kBundleRes:
		switch f {
		case "RelPermalink", "Permalink":
			if sc.rootKind == "shortcode" {
				// Shortcodes render before pages get their URLs, so link the
				// file relative to the page, where bundle files are copied.
				return hval{expr: call("print", v.name), k: kString}
			}
			fn := "relURL"
			if f == "Permalink" {
				fn = "absURL"
			}
			return hval{expr: call(fn, call("print", sel(v.base, "URL"), v.name)), k: kString}
		case "Name", "Title":
			return hval{expr: v.name, k: kString}
		case "Resize", "Fit", "Fill", "Crop", "Process", "Filter":
			return v
		case "Width", "Height":
			return hval{expr: "0", k: kNumber}
		}
		return miss()
	case kAsset:
		switch f {
		case "RelPermalink":
			return hval{expr: call("asset", strconv.Quote(v.name)), k: kString}
		case "Permalink":
			return hval{expr: call("absURL", call("asset", strconv.Quote(v.name))), k: kString}
		case "Data":
			// .Data.Integrity and friends: no fingerprinting data.
			return fallbackVal()
		case "Name":
			return lit(v.name)
		}
		return miss()
	case kOutputFormats:
		if f == "Get" && len(args) == 1 && sc.cfg() != "" {
			return t.feedFormat(sc)
		}
		return miss()
	case kLanguage:
		cfg := sc.cfg()
		if cfg == "" {
			return miss()
		}
		switch f {
		case "Lang", "LanguageCode", "LanguageName":
			return hval{expr: sel(cfg, "Language"), k: kString}
		case "LanguageDirection":
			return hval{expr: call("documentDirection", sel(cfg, "Language")), k: kString}
		}
		return miss()
	case kHugo:
		switch f {
		case "Generator":
			return hval{expr: call("unsafeHTML", strconv.Quote(`<meta name="generator" content="Nida">`)), k: kSafe}
		case "Version":
			return lit("nida")
		case "Environment":
			return lit("production")
		case "IsProduction":
			return hval{expr: "true", k: kBool}
		case "IsDevelopment", "IsServer", "IsExtended":
			return hval{expr: "false", k: kBool}
		}
		return miss()
	case kScratch:
		t.manual(sc, n, "`.Scratch` and `.Store` have no Nida equivalent; use template variables (`$x = ...`) instead.")
		return fallbackVal()
	case kGroup:
		switch f {
		case "Key":
			if v.name == "month" {
				return hval{expr: call(sel(sel(v.expr, "Month"), "Format"), strconv.Quote(v.base)), k: kString}
			}
			return hval{expr: sel(v.expr, "Year"), k: kString}
		case "Pages":
			return listOf(sel(v.expr, "Pages"), kPageList, pageVal(""))
		}
		return miss()
	case kData:
		switch f {
		case "Pages":
			return listOf(sel(v.expr, "Pages"), kPageList, pageVal(""))
		case "Terms":
			out := listOf(sel(v.expr, "Terms"), kTermList, hval{k: kTerm})
			out.mapForm = true
			return out
		case "Plural", "Singular":
			return hval{expr: sel(sel(v.expr, "Taxonomy"), "Name"), k: kString}
		case "Term":
			return hval{expr: sel(sel(v.expr, "Term"), "Name"), k: kString}
		}
		return miss()
	}
	return miss()
}

func argList(args []hval) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = a.arg()
	}
	return out
}

// ctxField maps a field of Hugo's page when the page is the template's
// own: Nida's context, whose kind depends on the template flavor.
func (t *hugoTr) ctxField(sc *scope, n parse.Node, v hval, f string, args []hval, miss func() hval) hval {
	e := v.expr
	page := sel(e, "Page")
	section := sel(e, "Section")
	listy := v.flavor == "list" || v.flavor == "home"
	switch f {
	case "Title", "LinkTitle", "Name":
		return hval{expr: sel(e, "Title"), k: kString}
	case "Description":
		return hval{expr: sel(e, "Description"), k: kString}
	case "Permalink":
		return hval{expr: sel(e, "CanonicalURL"), k: kString}
	case "RelPermalink":
		return hval{expr: call("relURL", sel(e, "CurrentURL")), k: kString}
	case "Page":
		return v
	case "Site":
		return hval{k: kSite}
	case "Data":
		return hval{expr: e, k: kData}
	case "Pages", "RegularPages", "RegularPagesRecursive":
		return listOf(sel(e, "Pages"), kPageList, pageVal(""))
	case "Paginator", "Paginate":
		return hval{expr: e, k: kPaginator}
	case "Content":
		switch {
		case listy:
			return hval{expr: sel(section, "BodyHTML"), k: kHTML}
		case v.flavor == "term" || v.flavor == "terms" || v.flavor == "404":
			return hval{expr: `""`, k: kString}
		}
		return hval{expr: sel(page, "BodyHTML"), k: kHTML}
	case "Params":
		if listy {
			return mapVal(sel(section, "Extra"))
		}
		return mapVal(sel(page, "Extra"))
	case "Param":
		if len(args) == 1 && sc.cfg() != "" {
			target := page
			if listy {
				target = section
			}
			return hval{expr: call("or", call("dig", sel(target, "Extra"), args[0].arg()), call("dig", sel(sc.cfg(), "Extra"), args[0].arg())), k: kAny}
		}
		return miss()
	case "IsHome":
		return hval{expr: call("eq", sel(e, "CurrentURL"), `"/"`), k: kBool}
	case "IsPage":
		switch v.flavor {
		case "single":
			return hval{expr: "true", k: kBool}
		case "any":
			return hval{expr: call("ne", sel(page, "RelativePath"), `""`), k: kBool}
		}
		return hval{expr: "false", k: kBool}
	case "IsNode":
		switch v.flavor {
		case "single":
			return hval{expr: "false", k: kBool}
		case "any":
			return hval{expr: call("eq", sel(page, "RelativePath"), `""`), k: kBool}
		}
		return hval{expr: "true", k: kBool}
	case "IsSection":
		switch v.flavor {
		case "list":
			return hval{expr: "true", k: kBool}
		case "any":
			return hval{expr: call("and", call("eq", sel(page, "RelativePath"), `""`), call("ne", sel(e, "CurrentURL"), `"/"`), call("eq", sel(sel(e, "Term"), "Name"), `""`), call("eq", sel(sel(e, "Taxonomy"), "Name"), `""`)), k: kBool}
		}
		return hval{expr: "false", k: kBool}
	case "Kind":
		switch v.flavor {
		case "single":
			return lit("page")
		case "list":
			return lit("section")
		case "home":
			return lit("home")
		case "term":
			return lit("term")
		case "terms":
			return lit("taxonomy")
		}
		return hval{expr: call("or", call("and", sel(page, "RelativePath"), `"page"`), call("and", call("eq", sel(e, "CurrentURL"), `"/"`), `"home"`), call("and", sel(sel(e, "Term"), "Name"), `"term"`), call("and", sel(sel(e, "Taxonomy"), "Name"), `"taxonomy"`), `"section"`), k: kString}
	case "Section", "Type":
		target := page
		if listy {
			target = section
		}
		return hval{expr: call("index", call("split", sel(target, "SectionPath"), `"/"`), "0"), k: kString}
	case "Sections":
		return listOf(sel(section, "Sections"), kSectionList, hval{k: kSection})
	case "Parent", "CurrentSection", "FirstSection":
		return hval{expr: section, k: kSection}
	case "Ancestors":
		if idx := sc.index(); idx != "" {
			return listOf(call(sel(idx, "Ancestors"), sel(section, "SectionPath")), kSectionList, hval{k: kSection})
		}
		return miss()
	case "Terms":
		return listOf(sel(e, "Terms"), kTermList, hval{k: kTerm})
	case "Term":
		return hval{expr: sel(sel(e, "Term"), "Name"), k: kString}
	case "Language":
		return hval{k: kLanguage}
	case "Lang":
		if cfg := sc.cfg(); cfg != "" {
			return hval{expr: sel(cfg, "Language"), k: kString}
		}
		return miss()
	case "OutputFormats":
		return hval{k: kOutputFormats}
	case "RSSLink":
		if sc.cfg() != "" {
			return hval{expr: call("relURL", sel(sel(sc.cfg(), "Feed"), "Filename")), k: kString}
		}
		return miss()
	case "AlternativeOutputFormats", "Translations", "AllTranslations":
		return listOf("(slice)", kPageList, hval{k: kAny})
	case "IsTranslated", "HasShortcode", "Draft":
		if f == "Draft" && !listy {
			break
		}
		return hval{expr: "false", k: kBool}
	case "Scratch", "Store":
		return hval{k: kScratch}
	case "IsMenuCurrent", "HasMenuCurrent":
		if len(args) == 2 && args[1].k == kMenuEntry {
			url := t.step(sc, n, args[1], "URL", nil)
			return hval{expr: call("eq", call("relURL", sel(e, "CurrentURL")), url.arg()), k: kBool}
		}
		return hval{expr: "false", k: kBool}
	}
	if listy {
		return t.sectionField(section, f, miss)
	}
	if v.flavor == "term" || v.flavor == "terms" {
		return miss()
	}
	return t.pageField(sc, page, f, args, miss)
}

// pageField maps a field of a Hugo page that is a content.Page in Nida,
// such as each page in a range over .Pages.
func (t *hugoTr) pageField(sc *scope, p, f string, args []hval, miss func() hval) hval {
	switch f {
	case "Title", "LinkTitle", "Name":
		return hval{expr: sel(p, "Title"), k: kString}
	case "Description", "Slug", "Aliases":
		return hval{expr: sel(p, f), k: kAny}
	case "Content":
		return hval{expr: sel(p, "BodyHTML"), k: kHTML}
	case "Summary":
		return hval{expr: sel(p, "Summary"), k: kHTML}
	case "Plain":
		return hval{expr: call("plainify", sel(p, "BodyHTML")), k: kString}
	case "PlainWords":
		return hval{expr: call("split", call("plainify", sel(p, "BodyHTML")), `" "`), k: kAny}
	case "Truncated":
		return hval{expr: call("ne", sel(p, "Summary"), sel(p, "BodyHTML")), k: kBool}
	case "Date", "PublishDate":
		return hval{expr: sel(p, "Date"), k: kTime}
	case "Lastmod":
		return hval{expr: sel(p, "Updated"), k: kTime}
	case "ExpiryDate":
		return hval{expr: sel(p, "Updated"), k: kTime}
	case "Params":
		return mapVal(sel(p, "Extra"))
	case "Param":
		if len(args) == 1 {
			e := call("dig", sel(p, "Extra"), args[0].arg())
			if cfg := sc.cfg(); cfg != "" {
				e = call("or", e, call("dig", sel(cfg, "Extra"), args[0].arg()))
			}
			return hval{expr: e, k: kAny}
		}
		return miss()
	case "Keywords":
		return hval{expr: call("dig", sel(p, "Extra"), `"keywords"`), k: kAny}
	case "Permalink":
		return hval{expr: call("absURL", sel(p, "URL")), k: kString}
	case "RelPermalink":
		return hval{expr: call("relURL", sel(p, "URL")), k: kString}
	case "WordCount", "FuzzyWordCount":
		return hval{expr: sel(p, "WordCount"), k: kNumber}
	case "ReadingTime", "Weight":
		return hval{expr: sel(p, f), k: kNumber}
	case "Draft":
		return hval{expr: sel(p, "Draft"), k: kBool}
	case "Section", "Type":
		return hval{expr: call("index", call("split", sel(p, "SectionPath"), `"/"`), "0"), k: kString}
	case "Layout":
		return hval{expr: sel(p, "Template"), k: kString}
	case "Kind":
		return lit("page")
	case "IsPage":
		return hval{expr: "true", k: kBool}
	case "IsHome", "IsNode", "IsSection", "IsTranslated", "HasShortcode":
		return hval{expr: "false", k: kBool}
	case "Pages", "RegularPages", "Translations", "AllTranslations", "AlternativeOutputFormats":
		return listOf("(slice)", kPageList, pageVal(""))
	case "Page":
		return pageVal(p)
	case "Site":
		return hval{k: kSite}
	case "File":
		return hval{k: kFile, base: p}
	case "Resources":
		return hval{k: kResources, base: p}
	case "GetTerms":
		if len(args) == 1 {
			tax := strings.Trim(args[0].expr, `"`)
			return listOf(call("dig", sel(p, "Extra"), args[0].arg()), kTermNames, hval{k: kTermName, name: tax})
		}
		return miss()
	case "PrevInSection", "Prev", "PrevPage":
		return neighbor(p, "Prev")
	case "NextInSection", "Next", "NextPage":
		return neighbor(p, "Next")
	case "Parent", "CurrentSection", "FirstSection":
		if idx := sc.index(); idx != "" {
			return hval{expr: call(sel(idx, "GetSection"), sel(p, "SectionPath")), k: kSection}
		}
		return miss()
	case "Language":
		return hval{k: kLanguage}
	case "Lang":
		if cfg := sc.cfg(); cfg != "" {
			return hval{expr: sel(cfg, "Language"), k: kString}
		}
		return miss()
	case "Scratch", "Store":
		return hval{k: kScratch}
	case "RenderString":
		if len(args) >= 1 {
			return hval{expr: call("markdownify", call("print", args[len(args)-1].arg())), k: kSafe}
		}
	}
	return miss()
}

// neighbor builds the previous or next page as a dict with Hugo's field
// names. It is empty, and so false in if and with, when there is none.
func neighbor(p, dir string) hval {
	url := sel(p, dir+"URL")
	d := call("dict", `"Title"`, sel(p, dir+"Title"), `"LinkTitle"`, sel(p, dir+"Title"),
		`"RelPermalink"`, call("relURL", url), `"Permalink"`, call("absURL", url), `"URL"`, url)
	return hval{expr: call("and", url, d), k: kDirect}
}

func (t *hugoTr) sectionField(s, f string, miss func() hval) hval {
	switch f {
	case "Title", "LinkTitle", "Name":
		return hval{expr: sel(s, "Title"), k: kString}
	case "Description":
		return hval{expr: sel(s, "Description"), k: kString}
	case "Content":
		return hval{expr: sel(s, "BodyHTML"), k: kHTML}
	case "Params":
		return mapVal(sel(s, "Extra"))
	case "Pages", "RegularPages", "RegularPagesRecursive":
		return listOf(sel(s, "Pages"), kPageList, pageVal(""))
	case "Sections":
		return listOf(sel(s, "Sections"), kSectionList, hval{k: kSection})
	case "Permalink":
		return hval{expr: call("absURL", sel(s, "URL")), k: kString}
	case "RelPermalink":
		return hval{expr: call("relURL", sel(s, "URL")), k: kString}
	case "Section", "Type":
		return hval{expr: call("index", call("split", sel(s, "SectionPath"), `"/"`), "0"), k: kString}
	case "Weight":
		return hval{expr: sel(s, "Weight"), k: kNumber}
	case "Kind":
		return lit("section")
	case "IsSection", "IsNode":
		return hval{expr: "true", k: kBool}
	case "IsPage", "IsHome":
		return hval{expr: "false", k: kBool}
	case "Page", "Parent", "CurrentSection", "FirstSection":
		return hval{expr: s, k: kSection}
	case "Site":
		return hval{k: kSite}
	case "Summary":
		return hval{expr: sel(s, "Description"), k: kString}
	}
	return miss()
}

func (t *hugoTr) siteField(sc *scope, n parse.Node, f string, args []hval, miss func() hval) hval {
	cfg, idx := sc.cfg(), sc.index()
	if cfg == "" {
		return miss()
	}
	extra := sel(cfg, "Extra")
	switch f {
	case "Title":
		return hval{expr: sel(cfg, "Title"), k: kString}
	case "BaseURL":
		return hval{expr: sel(cfg, "BaseURL"), k: kString}
	case "LanguageCode":
		return hval{expr: sel(cfg, "Language"), k: kString}
	case "Language":
		return hval{k: kLanguage}
	case "Params":
		return mapVal(extra)
	case "Copyright", "DisqusShortname", "GoogleAnalytics":
		key := strings.ToLower(f[:1]) + f[1:]
		return hval{expr: call("dig", extra, strconv.Quote(key)), k: kAny}
	case "Author", "Social":
		return mapVal(extra).withExpr(call("dig", extra, strconv.Quote(strings.ToLower(f)))).asMapAt(extra, strings.ToLower(f))
	case "Menus":
		return hval{expr: extra, k: kMenus}
	case "IsServer", "BuildDrafts":
		return hval{expr: "false", k: kBool}
	case "LastChange", "Lastmod":
		return hval{expr: "now", k: kTime}
	case "Hugo":
		return hval{k: kHugo}
	case "Languages", "Home.Translations":
		return listOf("(slice)", kPageList, hval{k: kAny})
	case "Data":
		if sc.rootKind == "entry" {
			return mapVal(sel(sc.root, "Data"))
		}
	}
	if idx == "" {
		return miss()
	}
	switch f {
	case "RegularPages", "Pages", "AllPages":
		return listOf(sel(idx, "AllPages"), kPageList, pageVal(""))
	case "Sections":
		return listOf(sel(idx, "Sections"), kSectionList, hval{k: kSection})
	case "Taxonomies":
		return hval{expr: sel(idx, "TaxonomyMap"), k: kTaxMap}
	case "Home":
		return hval{expr: sel(idx, "RootSection"), k: kSection}
	case "GetPage":
		if len(args) >= 1 {
			pathArg := args[len(args)-1]
			if s, err := strconv.Unquote(pathArg.expr); err == nil {
				pathArg = lit(strings.Trim(s, "/"))
			} else {
				pathArg = hval{expr: call("trimSpace", pathArg.arg()), k: kString}
			}
			t.review(sc, n, "`.Site.GetPage` became a lookup by content path (page first, then section); check that it finds the right content.")
			// Hugo's GetPage finds pages and sections alike.
			return pageVal(call("or", call(sel(idx, "GetPage"), pathArg.arg()), call(sel(idx, "GetSection"), pathArg.arg())))
		}
	}
	return miss()
}

// asMapAt turns v into a kMap rooted at base.key.
func (v hval) asMapAt(base, key string) hval {
	out := hval{k: kMap, base: base, path: []string{key}}
	out.expr = out.mapExpr()
	return out
}

func (t *hugoTr) listField(sc *scope, n parse.Node, v hval, f string, args []hval, miss func() hval) hval {
	switch f {
	case "ByDate", "ByWeight", "ByTitle", "ByLinkTitle", "ByPublishDate", "ByLastmod", "ByLength", "Reverse", "Alphabetical", "ByCount", "ByParam":
		t.review(sc, n, "List ordering methods such as `.%s` are ignored; Nida lists pages in each section's `sort_by` order.", f)
		return v
	case "Len", "Count":
		return hval{expr: call("len", v.arg()), k: kNumber}
	case "Limit":
		if len(args) == 1 {
			out := v
			out.expr = call("first", args[0].arg(), v.arg())
			return out
		}
	case "GroupByDate", "GroupByPublishDate":
		if len(args) >= 1 {
			return t.groupBy(sc, n, v, args[0])
		}
	case "Pages":
		if v.k == kWeighted {
			return listOf(v.expr, kPageList, pageVal(""))
		}
	}
	return miss()
}

func (t *hugoTr) groupBy(sc *scope, n parse.Node, v hval, layout hval) hval {
	l, err := strconv.Unquote(layout.expr)
	if err != nil {
		t.review(sc, n, "`GroupByDate` with a computed layout groups by month.")
		l = "January 2006"
	}
	if l == "2006" {
		return listOf(call("groupByYear", v.arg()), kGroupList, hval{k: kGroup, name: "year"})
	}
	return listOf(call("groupByMonth", v.arg()), kGroupList, hval{k: kGroup, name: "month", base: l})
}

func (t *hugoTr) paginatorField(v hval, f string, miss func() hval) hval {
	// .Paginator is nil when a section fits on one page, so every field is
	// guarded with and/or, which stop at the first falsy value.
	pg := sel(v.expr, "Paginator")
	guard := func(field, zero string) string { return call("or", call("and", pg, sel(pg, field)), zero) }
	switch f {
	case "Pages":
		return listOf(sel(v.expr, "Pages"), kPageList, pageVal(""))
	case "HasPrev":
		return hval{expr: call("ne", guard("Previous", `""`), `""`), k: kBool}
	case "HasNext":
		return hval{expr: call("ne", guard("Next", `""`), `""`), k: kBool}
	case "Prev":
		return hval{expr: call("and", call("and", pg, sel(pg, "Previous")), call("relURL", sel(pg, "Previous"))), k: kPager}
	case "Next":
		return hval{expr: call("and", call("and", pg, sel(pg, "Next")), call("relURL", sel(pg, "Next"))), k: kPager}
	case "First":
		return hval{expr: call("relURL", call("or", call("and", pg, sel(call("index", sel(pg, "PageLinks"), "0"), "URL")), sel(v.expr, "CurrentURL"))), k: kPager}
	case "Last":
		return hval{expr: call("relURL", call("or", call("and", pg, sel(call("index", sel(pg, "PageLinks"), call("sub", call("len", sel(pg, "PageLinks")), "1")), "URL")), sel(v.expr, "CurrentURL"))), k: kPager}
	case "PageNumber":
		return hval{expr: guard("CurrentIndex", "1"), k: kNumber}
	case "TotalPages":
		return hval{expr: guard("NumberPagers", "1"), k: kNumber}
	case "Pagers":
		return listOf(guard("PageLinks", "(slice)"), kPagerList, hval{k: kPagerLink})
	case "PagerSize", "PageSize", "TotalNumberOfElements", "NumberOfElements":
		return hval{expr: call("len", sel(v.expr, "Pages")), k: kNumber}
	}
	return miss()
}

func (t *hugoTr) feedFormat(sc *scope) hval {
	file := sel(sel(sc.cfg(), "Feed"), "Filename")
	return hval{expr: call("dict", `"RelPermalink"`, call("relURL", file), `"Permalink"`, call("absURL", file),
		`"Rel"`, `"alternate"`, `"Name"`, `"atom"`, `"MediaType"`, call("dict", `"Type"`, `"application/atom+xml"`)), k: kDirect}
}

func (t *hugoTr) shortField(v hval, f string, args []hval, miss func() hval) hval {
	e := v.expr
	switch f {
	case "Get":
		if len(args) == 1 {
			return hval{expr: call(sel(e, "Get"), args[0].arg()), k: kString}
		}
	case "Inner", "InnerDeindent":
		return hval{expr: sel(e, "Inner"), k: kInner, base: sel(e, "InnerHTML")}
	case "Params":
		return mapVal(sel(e, "Args"))
	case "IsNamedParams":
		return hval{expr: call("gt", call("len", sel(e, "Args")), "0"), k: kBool}
	case "Page":
		return pageVal(sel(e, "Page"))
	case "Site":
		return hval{k: kSite}
	case "Ordinal":
		return hval{expr: "0", k: kNumber}
	case "Position", "Name":
		return lit("")
	}
	return miss()
}

// fn translates a call to a Hugo template function. args include the
// piped value as the last argument.
func (t *hugoTr) fn(sc *scope, n parse.Node, name string, args []hval) hval {
	short := name
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		short = name[i+1:]
	}
	a := argList(args)
	need := func(min int) bool {
		if len(args) < min {
			t.review(sc, n, "`%s` is called with too few arguments; it renders nothing.", name)
			return false
		}
		return true
	}
	passthrough := func(nidaName string, k hkind) hval {
		return hval{expr: call(nidaName, a...), k: k}
	}
	dropped := func(sev Severity, why string) hval {
		if sev == Manual {
			t.manual(sc, n, "`%s` %s", name, why)
		} else {
			t.review(sc, n, "`%s` %s", name, why)
		}
		return fallbackVal()
	}

	switch name {
	case "site":
		return hval{k: kSite}
	case "hugo":
		return hval{k: kHugo}
	case "now", "time.Now":
		return hval{expr: "now", k: kTime}
	case "eq", "ne", "lt", "le", "gt", "ge", "not":
		return passthrough(name, kBool)
	case "and", "or":
		out := passthrough(name, kAny)
		if len(args) > 0 {
			out.k, out.flavor, out.elem = args[len(args)-1].k, args[len(args)-1].flavor, args[len(args)-1].elem
			if out.k == kMap || out.k == kDict || out.k == kCtx {
				out.k = kAny
			}
		}
		return out
	case "len", "collections.Len":
		return passthrough("len", kNumber)
	case "index", "collections.Index":
		if len(args) == 2 && (args[0].k == kMap) {
			if s, err := strconv.Unquote(args[1].expr); err == nil {
				out := hval{k: kMap, base: args[0].base, path: append(slices.Clone(args[0].path), s)}
				out.expr = out.mapExpr()
				return out
			}
		}
		return passthrough("index", kAny)
	case "print", "printf", "println", "fmt.Print", "fmt.Printf", "fmt.Println", "string", "cast.ToString":
		fname := strings.TrimPrefix(short, "")
		if fname == "string" || fname == "ToString" {
			fname = "print"
		}
		fname = strings.ToLower(fname[:1]) + fname[1:]
		return passthrough(fname, kString)
	case "int", "float", "cast.ToInt", "cast.ToFloat":
		if len(args) == 1 {
			return args[0]
		}
	case "dict", "collections.Dictionary":
		d := hval{expr: call("dict", a...), k: kDict, fields: map[string]hval{}}
		for i := 0; i+1 < len(args); i += 2 {
			if key, err := strconv.Unquote(args[i].expr); err == nil {
				d.fields[key] = args[i+1]
			}
		}
		return d
	case "slice", "collections.Slice":
		return passthrough("slice", kAny)
	case "first", "last", "collections.First", "collections.Last":
		if need(2) {
			out := args[len(args)-1]
			out.expr = call(strings.ToLower(short), a...)
			return out
		}
	case "after", "collections.After":
		if need(2) {
			out := args[1]
			out.expr = call("last", call("sub", call("len", a[1]), a[0]), a[1])
			return out
		}
	case "where", "collections.Where":
		return t.where(sc, n, args)
	case "default":
		if need(2) {
			return hval{expr: call("or", a[1], a[0]), k: args[1].k, elem: args[1].elem}
		}
	case "cond", "compare.Conditional":
		if need(3) {
			return hval{expr: call("or", call("and", a[0], a[1]), a[2]), k: kAny}
		}
	case "isset", "collections.IsSet":
		if need(2) {
			if args[0].k == kMap {
				return hval{expr: call("dig", a[0], a[1]), k: kAny}
			}
			return hval{expr: call("index", a[0], a[1]), k: kAny}
		}
	case "in", "collections.In":
		if need(2) {
			t.review(sc, n, "`in` became a text search over the joined list, which can match part of an item.")
			return hval{expr: call("contains", call("join", a[0], `"|"`), call("print", a[1])), k: kBool}
		}
	case "delimit", "collections.Delimit":
		if need(2) {
			return hval{expr: call("join", a[0], a[1]), k: kString}
		}
	case "seq", "collections.Seq":
		if len(args) == 1 {
			return passthrough("seq", kAny)
		}
		return dropped(Review, "with more than one argument has no Nida equivalent.")
	case "merge", "collections.Merge":
		if len(args) == 2 {
			return passthrough("merge", kAny)
		}
	case "add", "sub", "math.Add", "math.Sub":
		return passthrough(strings.ToLower(short), kNumber)
	case "safeHTML", "safe.HTML":
		if len(args) == 1 && args[0].k == kInner {
			return hval{expr: args[0].base, k: kSafe}
		}
		return hval{expr: call("unsafeHTML", call("print", a...)), k: kSafe}
	case "safeCSS", "safe.CSS":
		return hval{expr: call("unsafeCSS", call("print", a...)), k: kSafe}
	case "safeJS", "safe.JS", "safeURL", "safe.URL", "safeHTMLAttr", "safe.HTMLAttr", "safeJSStr", "safe.JSStr":
		t.review(sc, n, "`%s` has no Nida equivalent; the value is escaped normally.", name)
		if len(args) == 1 {
			return hval{expr: a[0], k: kString}
		}
	case "markdownify", "transform.Markdownify":
		if len(args) == 1 && args[0].k == kInner {
			return hval{expr: args[0].base, k: kSafe}
		}
		return hval{expr: call("markdownify", call("print", a...)), k: kSafe}
	case "plainify", "transform.Plainify":
		return hval{expr: call("plainify", call("print", a...)), k: kString}
	case "htmlEscape", "transform.HTMLEscape", "htmlUnescape", "transform.HTMLUnescape", "emojify", "transform.Emojify", "chomp", "strings.Chomp":
		if len(args) == 1 {
			if short == "Chomp" || name == "chomp" {
				return hval{expr: call("trimSpace", call("print", a[0])), k: kString}
			}
			return hval{expr: call("print", a[0]), k: kString}
		}
	case "jsonify", "encoding.Jsonify":
		if len(args) >= 1 {
			return hval{expr: call("jsonify", a[len(a)-1]), k: kSafe}
		}
	case "urlize", "anchorize", "urls.Anchorize", "urls.URLize":
		return hval{expr: call("urlize", call("print", a...)), k: kString}
	case "relURL", "relLangURL", "urls.RelURL", "urls.RelLangURL":
		return hval{expr: call("relURL", call("print", a...)), k: kString}
	case "absURL", "absLangURL", "urls.AbsURL", "urls.AbsLangURL":
		return hval{expr: call("absURL", call("print", a...)), k: kString}
	case "lower", "strings.ToLower":
		return hval{expr: call("lower", call("print", a...)), k: kString}
	case "upper", "strings.ToUpper":
		return hval{expr: call("upper", call("print", a...)), k: kString}
	case "title", "strings.Title", "strings.FirstUpper":
		return hval{expr: call("title", call("print", a...)), k: kString}
	case "humanize", "inflect.Humanize":
		return hval{expr: call("title", call("replace", call("print", a...), `"-"`, `" "`)), k: kString}
	case "trim", "strings.Trim", "strings.TrimSpace":
		if len(args) >= 1 {
			if len(args) == 2 && strings.Trim(unquoteOr(args[1].expr), " \t\r\n") != "" {
				t.review(sc, n, "`%s` with a cut set other than whitespace became `trimSpace`.", name)
			}
			return hval{expr: call("trimSpace", call("print", a[0])), k: kString}
		}
	case "replace", "strings.Replace", "strings.ReplaceAll":
		if need(3) {
			return hval{expr: call("replace", call("print", a[0]), a[1], a[2]), k: kString}
		}
	case "split", "strings.Split":
		if need(2) {
			return hval{expr: call("split", call("print", a[0]), a[1]), k: kAny}
		}
	case "strings.Contains", "strings.HasPrefix", "strings.HasSuffix", "hasPrefix", "hasSuffix":
		if need(2) {
			fname := strings.ToLower(short[:1]) + short[1:]
			return hval{expr: call(fname, call("print", a[0]), call("print", a[1])), k: kBool}
		}
	case "truncate", "strings.Truncate":
		if need(2) {
			text := args[len(args)-1]
			textExpr := call("print", text.arg())
			if text.k == kHTML || text.k == kSafe {
				textExpr = call("plainify", textExpr)
			}
			if len(args) == 3 {
				t.review(sc, n, "`truncate` ignores the custom ellipsis; Nida always uses \"…\".")
			}
			return hval{expr: call("truncate", a[0], textExpr), k: kString}
		}
	case "countwords", "strings.CountWords":
		if len(args) == 1 {
			return hval{expr: call("len", call("split", call("plainify", call("print", a[0])), `" "`)), k: kNumber}
		}
	case "T", "i18n", "lang.Translate":
		if need(1) {
			out := []string{a[0]}
			if len(args) > 1 && args[1].k == kDict {
				out = append(out, a[1])
			}
			return hval{expr: call("t", out...), k: kString}
		}
	case "dateFormat", "time.Format":
		if need(2) {
			layout := args[0].expr
			if s, err := strconv.Unquote(layout); err == nil {
				layout = strconv.Quote(hugoDateLayout(s))
			}
			if args[1].k == kTime {
				return hval{expr: call(sel(args[1].expr, "Format"), layout), k: kString}
			}
			t.review(sc, n, "`%s` on a value that is not a date prints the value as it is.", name)
			return hval{expr: call("print", a[1]), k: kString}
		}
	case "time", "time.AsTime":
		if len(args) == 1 {
			t.review(sc, n, "`%s` cannot parse dates in Nida; the value is used as it is.", name)
			return args[0]
		}
	case "templates.Exists":
		if len(args) == 1 {
			if s, err := strconv.Unquote(args[0].expr); err == nil {
				return hval{expr: strconv.FormatBool(t.exists(s)), k: kBool}
			}
		}
		return hval{expr: "false", k: kBool}
	case "resources.Get", "resources.GetMatch":
		if len(args) == 1 {
			if s, err := strconv.Unquote(args[0].expr); err == nil {
				return hval{k: kAsset, name: s}
			}
		}
		return dropped(Manual, "with a computed path has no Nida equivalent.")
	case "resources.ToCSS", "toCSS", "css.Sass", "css.PostCSS", "postCSS", "resources.PostCSS", "css.TailwindCSS",
		"resources.Minify", "minify", "resources.Fingerprint", "fingerprint", "resources.ExecuteAsTemplate",
		"js.Build", "js.Babel", "babel", "resources.Babel", "js.Batch":
		if len(args) >= 1 && args[len(args)-1].k == kAsset {
			asset := args[len(args)-1]
			switch {
			case strings.Contains(name, "CSS") || strings.Contains(name, "Sass") || name == "toCSS" || name == "postCSS":
				asset.name = swapExt(asset.name, ".css", ".scss", ".sass")
			case strings.HasPrefix(name, "js.") || strings.Contains(name, "abel"):
				asset.name = swapExt(asset.name, ".js", ".ts", ".tsx", ".jsx", ".mjs")
			}
			if !t.assets[asset.name] {
				t.assets[asset.name] = true
				t.review(sc, n, "Hugo Pipes asset `%s` is linked as the static file `%s`; Nida's asset pipeline must produce it (see `[pipeline]`), or copy the built file into `static/`.", unquoteOr(args[len(args)-1].name), asset.name)
			}
			return asset
		}
		return dropped(Manual, "has no Nida equivalent.")
	case "partial", "partialCached", "partials.Include", "partials.IncludeCached":
		return dropped(Manual, "is used as a value (not as `{{ partial ... }}` on its own), which Nida cannot do; it renders nothing.")
	case "getJSON", "getCSV", "data.GetJSON", "data.GetCSV", "resources.GetRemote", "transform.Unmarshal", "unmarshal", "os.ReadFile", "os.Getenv", "newScratch":
		return dropped(Manual, "has no Nida equivalent; it renders nothing.")
	case "errorf", "warnf", "erroridf", "warnidf", "fmt.Errorf", "fmt.Warnf", "fmt.Erroridf", "fmt.Warnidf", "debug.Dump", "debug.Timer":
		return hval{expr: `""`, k: kString}
	case "ref", "relref", "absref":
		if len(args) >= 2 {
			if s, err := strconv.Unquote(args[len(args)-1].expr); err == nil {
				target := "/" + strings.TrimSuffix(strings.Trim(strings.TrimPrefix(s, "/"), "/"), ".md") + "/"
				t.review(sc, n, "`%s %q` became the link `%s`; check it.", name, s, target)
				fname := "relURL"
				if name == "absref" || name == "ref" {
					fname = "absURL"
				}
				return hval{expr: call(fname, strconv.Quote(target)), k: kString}
			}
		}
	case "readFile":
		return passthrough("readFile", kString)
	case "sort", "collections.Sort", "uniq", "collections.Uniq", "shuffle", "collections.Shuffle", "collections.Reverse":
		if len(args) >= 1 {
			t.review(sc, n, "`%s` has no Nida equivalent; the list is used in its existing order.", name)
			return args[0]
		}
	case "append", "union", "intersect", "complement", "symdiff", "collections.Append", "collections.Union", "collections.Intersect":
		if len(args) >= 1 {
			t.review(sc, n, "`%s` has no Nida equivalent; only the first list is kept.", name)
			if name == "append" || name == "collections.Append" {
				return args[len(args)-1] // append's list is its last argument
			}
			return args[0]
		}
	case "html", "js", "urlquery":
		return hval{expr: call("print", a...), k: kString}
	}
	return dropped(Review, "has no Nida equivalent; it renders nothing.")
}

func unquoteOr(s string) string {
	if u, err := strconv.Unquote(s); err == nil {
		return u
	}
	return s
}

func swapExt(p, to string, from ...string) string {
	for _, ext := range from {
		if strings.HasSuffix(p, ext) {
			return strings.TrimSuffix(p, ext) + to
		}
	}
	return p
}

// hugoDateLayout expands Hugo's named date layouts.
func hugoDateLayout(s string) string {
	switch s {
	case ":date_full":
		return "Monday, January 2, 2006"
	case ":date_long":
		return "January 2, 2006"
	case ":date_medium":
		return "Jan 2, 2006"
	case ":date_short":
		return "1/2/06"
	case ":time_full", ":time_long":
		return "3:04:05 pm MST"
	case ":time_medium":
		return "3:04:05 pm"
	case ":time_short":
		return "3:04 pm"
	}
	return s
}

// hugoWhereKeys maps page field names in where clauses.
var hugoWhereKeys = map[string]string{
	"Section": "SectionPath", "Type": "SectionPath", "Title": "Title", "Draft": "Draft",
	"Date": "Date", "Lastmod": "Updated", "Weight": "Weight", "Slug": "Slug", "Kind": "",
}

func (t *hugoTr) where(sc *scope, n parse.Node, args []hval) hval {
	if len(args) < 3 {
		t.review(sc, n, "`where` with two arguments has no Nida equivalent; the list is used unfiltered.")
		if len(args) > 0 {
			return args[0]
		}
		return fallbackVal()
	}
	coll := args[0]
	key, err := strconv.Unquote(args[1].expr)
	if err != nil {
		t.review(sc, n, "`where` with a computed key is used unfiltered.")
		return coll
	}
	key = strings.TrimPrefix(key, ".")
	switch {
	case strings.HasPrefix(key, "Params."):
		key = "Extra." + strings.TrimPrefix(key, "Params.")
	case strings.Contains(key, "."):
	default:
		if mapped, ok := hugoWhereKeys[key]; ok {
			if mapped == "" {
				return coll // Nida page lists hold only pages
			}
			key = mapped
		}
	}
	rest := args[2:]
	if len(rest) == 2 {
		op := unquoteOr(rest[0].expr)
		if op != "==" && op != "!=" && op != "=" && op != "eq" && op != "ne" {
			t.review(sc, n, "`where` with the `%s` operator has no Nida equivalent; the list is used unfiltered.", op)
			return coll
		}
		switch op {
		case "eq":
			op = "=="
		case "ne":
			op = "!="
		}
		rest = []hval{lit(op), rest[1]}
	}
	out := coll
	out.expr = call("where", append([]string{coll.arg(), strconv.Quote(key)}, argList(rest)...)...)
	return out
}

func (t *hugoTr) review(sc *scope, n parse.Node, format string, args ...any) {
	t.p.Report.Add(Review, sc.file, t.line(sc, n), format, args...)
}

func (t *hugoTr) manual(sc *scope, n parse.Node, format string, args ...any) {
	t.p.Report.Add(Manual, sc.file, t.line(sc, n), format, args...)
}

func (t *hugoTr) line(sc *scope, n parse.Node) int {
	if sc.tree == nil || n == nil {
		return 0
	}
	loc, _ := sc.tree.ErrorContext(n)
	parts := strings.Split(loc, ":")
	if len(parts) >= 3 {
		if line, err := strconv.Atoi(parts[len(parts)-2]); err == nil {
			return line
		}
	}
	return 0
}
