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

	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/frontmatter"
)

// hugoContentFile is a content file found in the first pass, before
// conversion, so references between pages can be resolved.
type hugoContentFile struct {
	src  string // source-relative path
	rel  string // path relative to the content dir, as Hugo sees it
	dest string // destination path under content/
}

var (
	markdownExts = []string{".md", ".markdown", ".mdown", ".mkd", ".mkdn", ".mdwn"}
	htmlExts     = []string{".html", ".htm"}
	otherFormats = []string{".org", ".adoc", ".asciidoc", ".ad", ".rst", ".pandoc", ".pdc", ".ipynb"}
)

func (h *hugoSite) planContent() error {
	p := h.p
	dir := strings.Trim(h.contentDir, "/")
	var files []hugoContentFile
	bundles := map[string]bool{}  // leaf bundle dirs, content-relative
	branches := map[string]bool{} // dirs with _index.md, content-relative
	var resources []string

	err := p.Walk(dir, func(src string) error {
		rel := strings.TrimPrefix(src, dir+"/")
		for _, langDir := range h.langDirs {
			if src == langDir || strings.HasPrefix(src, strings.Trim(langDir, "/")+"/") {
				return nil // another language's content tree
			}
		}
		ext := strings.ToLower(path.Ext(rel))
		stem := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		// post.fr.md belongs to language fr; post.en.md to the default.
		if base, lang, ok := cutLang(stem); ok && (slices.Contains(markdownExts, ext) || slices.Contains(htmlExts, ext)) {
			if slices.Contains(h.otherLangs, lang) {
				p.Preserve(src)
				p.Report.Add(Unsupported, src, 0, "Translation for language `%s` was not migrated (Nida builds one language per site); it is kept in `%s/`.", lang, preservedDir)
				return nil
			}
			if lang == h.defaultLang {
				stem = base
			}
		}
		switch {
		case slices.Contains(markdownExts, ext), slices.Contains(htmlExts, ext):
			name := stem + ".md"
			dest := path.Join("content", path.Dir(rel), name)
			files = append(files, hugoContentFile{src: src, rel: rel, dest: path.Clean(dest)})
			switch stem {
			case "index":
				if path.Dir(rel) != "." {
					bundles[path.Dir(rel)] = true
				}
			case "_index":
				branches[path.Dir(rel)] = true
			}
		case slices.Contains(otherFormats, ext):
			p.Preserve(src)
			p.Report.Add(Manual, src, 0, "Content in `%s` format is not supported (Nida reads Markdown only); convert it to Markdown. The file is kept in `%s/`.", ext, preservedDir)
		default:
			resources = append(resources, src)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Resources: leaf bundle files stay beside index.md; Hugo publishes
	// branch bundle and loose files at the section's URL, so they go to
	// static/ to keep it.
	for _, src := range resources {
		rel := strings.TrimPrefix(src, dir+"/")
		if hugoInBundle(rel, bundles) {
			p.Copy(path.Join("content", rel), src)
			continue
		}
		dest := path.Join("static", h.makePath(path.Dir(rel)), path.Base(rel))
		if p.Planned(dest) {
			p.Report.Add(Manual, src, 0, "Not migrated: `%s` is already used by a static file.", dest)
			continue
		}
		p.Copy(dest, src)
	}

	index := map[string]string{} // content-relative path → dest
	for _, f := range files {
		index[strings.TrimPrefix(f.dest, "content/")] = f.dest
	}
	for _, f := range files {
		index[f.rel] = f.dest // source names win over renamed ones
	}

	// Hugo makes every top-level directory a section, with its own RSS
	// feed. Give those without an _index.md one, so Nida writes the feed.
	if h.p.Config.HasTable("rss") {
		tops := map[string]bool{}
		for _, f := range files {
			base := path.Base(f.rel)
			if strings.HasPrefix(base, "_index.") || (!strings.HasPrefix(base, "index.") && hugoInBundle(f.rel, bundles)) {
				continue
			}
			if top, _, _ := strings.Cut(sectionDir(f), "/"); top != "" {
				tops[top] = true
			}
		}
		for _, top := range sortedKeys(tops) {
			if branches[top] {
				continue
			}
			doc := NewDocument(TOML, "")
			_ = doc.Set("title", titleFromSlug(top))
			_ = doc.Set("generate_feeds", true)
			p.Page(path.Join("content", top, "_index.md"), doc, "", "")
		}
	}
	h.planPermalinks(files, bundles)

	templatesUsed := map[string]string{}
	for _, f := range files {
		// Markdown files inside a leaf bundle (other than index.md) are page
		// resources in Hugo, not pages.
		if base := path.Base(f.rel); !strings.HasPrefix(base, "index.") && !strings.HasPrefix(base, "_index.") && hugoInBundle(f.rel, bundles) {
			p.Copy(f.dest, f.src)
			continue
		}
		if err := h.convertFile(f, index, templatesUsed); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(templatesUsed) {
		p.Report.Add(Review, templatesUsed[name], 0, "Content asks for layout `%s` through `layout` or `type`; it renders with the default template unless a template with that name exists.", name)
	}
	if p.unsafeHTML && !h.unsafeHTML {
		p.Report.Add(Review, "", 0, "Content has raw HTML, but the Hugo site had `markup.goldmark.renderer.unsafe = false`, so Hugo left it out of pages. `markdown.unsafe_html` stays off to match; turn it on if you want the HTML rendered.")
	}
	// Hugo lowercases section URLs; Nida's follow the directory names.
	sectionDirs := map[string]bool{}
	for _, f := range files {
		if d := sectionDir(f); d != "" {
			sectionDirs[d] = true
		}
	}
	for _, d := range sortedKeys(sectionDirs) {
		if h.makePath(d) != d {
			p.Report.Add(Review, path.Join(dir, d), 0, "Hugo served this section at `/%s/`, but Nida serves it at `/%s/`. Rename the directory to keep the old URL.", h.makePath(d), d)
		}
	}
	return nil
}

// cutLang splits "post.fr" into "post" and "fr".
func cutLang(stem string) (string, string, bool) {
	i := strings.LastIndexByte(stem, '.')
	if i <= 0 {
		return stem, "", false
	}
	lang := strings.ToLower(stem[i+1:])
	if len(lang) < 2 || len(lang) > 7 {
		return stem, "", false
	}
	for _, r := range lang {
		if (r < 'a' || r > 'z') && r != '-' && r != '_' {
			return stem, "", false
		}
	}
	return stem[:i], lang, true
}

func hugoInBundle(rel string, bundles map[string]bool) bool {
	for dir := path.Dir(rel); dir != "."; dir = path.Dir(dir) {
		if bundles[dir] {
			return true
		}
	}
	return false
}

func (h *hugoSite) convertFile(f hugoContentFile, index map[string]string, templatesUsed map[string]string) error {
	p := h.p
	data, err := p.ReadFile(f.src)
	if err != nil {
		return err
	}
	doc, err := ParseDocument(data)
	if err != nil {
		p.Report.Add(Manual, f.src, 1, "Front matter could not be parsed (%v); the file was copied unchanged and will not build.", err)
		p.Copy(f.dest, f.src)
		return nil
	}
	MoveCustomPath(doc, p.Report, f.src)
	if f.dest != path.Join("content", f.rel) {
		p.Report.Add(Changed, f.src, 0, "Renamed to `%s`: Nida reads Markdown from `.md` files and has one language.", f.dest)
	}
	if slices.Contains(htmlExts, strings.ToLower(path.Ext(f.rel))) {
		p.Report.Add(Review, f.src, 0, "HTML content file became Markdown with raw HTML; check that it renders as before.")
	}
	isSection := strings.HasPrefix(path.Base(f.dest), "_index.")
	stem := strings.TrimSuffix(path.Base(f.dest), ".md")
	if stem == "index" {
		stem = path.Base(path.Dir(f.dest))
	}
	if doc.Format == NoFrontMatter {
		_ = doc.Set("title", titleFromSlug(stem))
		p.Report.Add(Review, f.src, 0, "The file had no front matter, which Nida requires; its title was set from the file name.")
	}
	h.convertFrontMatter(doc, f, isSection, templatesUsed)
	if isSection {
		dir := path.Dir(f.rel)
		if dir != "." && !strings.Contains(dir, "/") {
			if _, ok := doc.Get("generate_feeds"); !ok && h.p.Config.HasTable("rss") {
				_ = doc.Set("generate_feeds", true)
				p.Report.Add(Changed, f.src, 0, "Set `generate_feeds` so the section keeps its Hugo RSS feed at `index.xml`.")
			}
		}
	}
	h.convertBody(doc, f, index)
	oldURL := ""
	if !isSection {
		oldURL = h.pageURL(doc, f)
		h.alignSlug(doc, f)
	}
	p.Page(f.dest, doc, f.src, oldURL)
	return nil
}

// convertFrontMatter maps Hugo front matter onto Nida's fields.
func (h *hugoSite) convertFrontMatter(doc *Document, f hugoContentFile, isSection bool, templatesUsed map[string]string) {
	r := h.p.Report
	file := f.src

	// Hugo front matter keys are case insensitive; Nida's are lowercase.
	for _, key := range doc.Keys() {
		lower := strings.ToLower(key)
		if lower != key && hugoKnownFields[lower] {
			if doc.Rename(key, lower) {
				r.Add(Changed, file, 0, "Front matter keys are lowercase, as Nida reads them.")
			}
		}
	}
	// Hugo 0.123+ keeps custom fields under params:.
	if params, ok := doc.Get("params"); ok {
		if m, ok := params.(map[string]any); ok {
			for _, k := range sortedKeys(m) {
				if !doc.Has(k) {
					_ = doc.Set(k, m[k])
				}
			}
			doc.Delete("params")
			r.Add(Changed, file, 0, "Moved `params` fields to the top level, where Nida reads custom fields into `.Page.Extra`.")
		}
	}

	for _, from := range []string{"lastmod", "modified"} {
		if doc.Has(from) {
			if doc.Rename(from, "updated") {
				r.Add(Changed, file, 0, "Renamed `%s` to `updated`.", from)
			} else {
				doc.Delete(from)
			}
		}
	}
	for _, from := range []string{"publishdate", "pubdate", "published"} {
		if !doc.Has(from) {
			continue
		}
		if !doc.Has("date") {
			doc.Rename(from, "date")
			r.Add(Changed, file, 0, "Renamed `%s` to `date`.", from)
		} else {
			doc.Delete(from)
			r.Add(Review, file, 0, "Both `date` and `%s` were set; Nida has one publication date, so `date` was kept. Pages are hidden while their `date` is in the future.", from)
		}
	}
	for _, key := range []string{"date", "updated"} {
		NormalizeDate(doc, key, r, file)
	}
	for _, key := range []string{"expirydate", "unpublishdate"} {
		if doc.Has(key) {
			r.Add(Unsupported, file, 0, "`%s` has no equivalent; the page stays published.", key)
		}
	}
	if u := doc.String("url"); u != "" && !isSection {
		doc.Delete("url")
		_ = doc.Set("path", "/"+strings.TrimPrefix(u, "/"))
		r.Add(Changed, file, 0, "`url` became `path`, which sets the page URL exactly.")
	} else if doc.Has("url") && isSection {
		r.Add(Unsupported, file, 0, "Section `url` is not supported; Nida section URLs follow the content directories.")
	}
	if aliases := frontmatter.StringList(doc.Values()["aliases"]); len(aliases) > 0 {
		fixed := make([]string, 0, len(aliases))
		changed := false
		for _, a := range aliases {
			abs := h.absoluteAlias(a, f)
			fixed = append(fixed, abs)
			changed = changed || abs != a
		}
		if isSection {
			r.Add(Unsupported, file, 0, "Section `aliases` are not supported; add redirects for them on your web host.")
		} else if changed {
			_ = doc.Set("aliases", fixed)
			r.Add(Changed, file, 0, "Relative `aliases` were made absolute, as Hugo resolves them against the page's section.")
		}
	}
	if layout := doc.String("layout"); layout != "" {
		doc.Delete("layout")
		if !doc.Has("template") {
			_ = doc.Set("template", layout)
		}
		if _, ok := templatesUsed[layout]; !ok {
			templatesUsed[layout] = file
		}
	}
	if t := doc.String("type"); t != "" {
		if _, ok := templatesUsed[t]; !ok {
			templatesUsed[t] = file
		}
	}
	if s := doc.String("summary"); s != "" && !doc.Has("description") {
		doc.Rename("summary", "description")
		r.Add(Changed, file, 0, "`summary` became `description`, Nida's hand-written summary.")
	}
	if images := frontmatter.StringList(doc.Values()["images"]); len(images) > 0 && !doc.Has("image") {
		_ = doc.Set("image", images[0])
		r.Add(Changed, file, 0, "The first of `images` became `image`, the social preview image of the `nida-seo` partial.")
	}
	if doc.Values()["headless"] == true || hugoBuildNever(doc.Values()) {
		_ = doc.Set("draft", true)
		r.Add(Review, file, 0, "Headless bundle (or `build.render: never`) became a draft, since Nida renders every published page. Templates that read its content need rewriting.")
	} else if b, ok := hugoBuild(doc.Values()); ok && (b["list"] == "never" || b["list"] == false) {
		r.Add(Unsupported, file, 0, "`build.list: never` is not supported; the page is listed in its section.")
	}
	for _, key := range []string{"cascade", "resources", "outputs", "markup", "sitemap"} {
		if doc.Has(key) {
			r.Add(Unsupported, file, 0, "Front matter `%s` has no equivalent and is ignored.", key)
		}
	}
	if doc.Has("menu") || doc.Has("menus") {
		r.Add(Review, file, 0, "Page `menu` entries have no equivalent; they are kept in `.Page.Extra` for templates.")
	}
	// Hugo slugs are used as written; a slug that is not a single URL path
	// segment cannot be a Nida slug.
	if slug := doc.String("slug"); strings.ContainsAny(slug, "/\\") {
		doc.Delete("slug")
		r.Add(Review, file, 0, "`slug` `%s` contains a slash, which Nida slugs cannot; the URL is kept with `path` instead.", slug)
	}
}

var hugoKnownFields = map[string]bool{
	"title": true, "description": true, "date": true, "lastmod": true, "draft": true, "weight": true,
	"slug": true, "url": true, "aliases": true, "layout": true, "type": true, "summary": true,
	"publishdate": true, "expirydate": true, "linktitle": true, "keywords": true, "tags": true,
	"categories": true, "images": true, "headless": true, "params": true, "build": true, "_build": true,
}

func hugoBuild(values map[string]any) (map[string]any, bool) {
	for _, key := range []string{"build", "_build"} {
		if b, ok := values[key].(map[string]any); ok {
			return b, true
		}
	}
	return nil, false
}

func hugoBuildNever(values map[string]any) bool {
	b, ok := hugoBuild(values)
	return ok && (b["render"] == "never" || b["render"] == false)
}

// absoluteAlias resolves a Hugo alias. An alias without a leading slash is
// relative to the page's section directory.
func (h *hugoSite) absoluteAlias(alias string, f hugoContentFile) string {
	if strings.HasPrefix(alias, "/") || strings.Contains(alias, "://") {
		return alias
	}
	dir := path.Dir(f.rel)
	if strings.HasPrefix(path.Base(f.rel), "index.") {
		dir = path.Dir(dir)
	}
	if dir == "." {
		return "/" + alias
	}
	trailing := strings.HasSuffix(alias, "/")
	out := "/" + path.Join(h.makePath(dir), alias)
	if trailing {
		out += "/"
	}
	return out
}

// makePath sanitizes a URL path the way Hugo does: whitespace becomes "-",
// characters other than letters, digits, marks, and ./_-#+~ are dropped,
// and the result is lowercased unless disablePathToLower is set.
func (h *hugoSite) makePath(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), unicode.IsMark(r), strings.ContainsRune("./\\_-#+~", r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte('-')
		}
	}
	if h.pathToLower {
		return strings.ToLower(b.String())
	}
	return b.String()
}

// hugoURLize is Hugo's urlize, used for :title and title-derived slugs.
func (h *hugoSite) urlize(s string) string {
	return strings.ToLower(h.makePath(strings.TrimSpace(s)))
}

// fileSlug is the name Hugo uses for a page: its file name, or for a leaf
// bundle its directory name.
func fileSlug(f hugoContentFile) string {
	stem := strings.TrimSuffix(path.Base(f.dest), ".md")
	if stem == "index" {
		return path.Base(path.Dir(f.dest))
	}
	return stem
}

// sectionDirs returns the page's section directory, content-relative, ""
// for the root.
func sectionDir(f hugoContentFile) string {
	dir := path.Dir(strings.TrimPrefix(f.dest, "content/"))
	if strings.HasSuffix(f.dest, "/index.md") {
		dir = path.Dir(dir)
	}
	if dir == "." {
		return ""
	}
	return dir
}

// hugoTokens lists the permalink tokens this migration understands.
var hugoToken = regexp.MustCompile(`:(slugorcontentbasename|slugorfilename|contentbasename|filename|monthname|weekdayname|weekday|yearday|sections|section|title|slug|year|month|day)`)

// pageURL predicts a page's URL on the Hugo site.
func (h *hugoSite) pageURL(doc *Document, f hugoContentFile) string {
	if p := doc.String("path"); p != "" {
		return p
	}
	section := sectionDir(f)
	top, _, _ := strings.Cut(section, "/")
	if pattern, ok := h.pagePermalinks[top]; ok && top != "" {
		if u, ok := h.expandPermalink(pattern, doc, f); ok {
			return u
		}
		h.p.Report.Add(Manual, f.src, 0, "Permalink pattern `%s` could not be expanded for this page, so its URL was not checked.", pattern)
		return ""
	}
	slug := doc.String("slug")
	if slug == "" {
		slug = fileSlug(f)
	}
	parts := []string{}
	if section != "" {
		parts = append(parts, section)
	}
	if h.uglyURLs {
		return h.makePath("/" + path.Join(append(parts, slug)...) + ".html")
	}
	return h.makePath("/" + path.Join(append(parts, slug)...) + "/")
}

// expandPermalink expands a Hugo permalink pattern for a page.
func (h *hugoSite) expandPermalink(pattern string, doc *Document, f hugoContentFile) (string, bool) {
	date := pageDate(doc)
	slug := doc.String("slug")
	title := doc.String("title")
	section := sectionDir(f)
	top, _, _ := strings.Cut(section, "/")
	ok := true
	out := hugoToken.ReplaceAllStringFunc(pattern, func(tok string) string {
		switch tok[1:] {
		case "year":
			return date.Format("2006")
		case "month":
			return date.Format("01")
		case "monthname":
			return strings.ToLower(date.Format("January"))
		case "day":
			return date.Format("02")
		case "weekday":
			return strconv.Itoa(int(date.Weekday()))
		case "weekdayname":
			return strings.ToLower(date.Format("Monday"))
		case "yearday":
			return strconv.Itoa(date.YearDay())
		case "section":
			return top
		case "sections":
			return section
		case "title":
			return h.urlize(title)
		case "slug":
			if slug != "" {
				return h.urlize(slug)
			}
			return h.urlize(title)
		case "filename", "contentbasename":
			return fileSlug(f)
		case "slugorfilename", "slugorcontentbasename":
			if slug != "" {
				return slug
			}
			return fileSlug(f)
		}
		ok = false
		return tok
	})
	if !ok || strings.Contains(out, ":") {
		return "", false
	}
	if date.IsZero() && regexp.MustCompile(`:(year|month|day|monthname|weekday|weekdayname|yearday)`).MatchString(pattern) {
		h.p.Report.Add(Review, f.src, 0, "The permalink uses the date, but the page has none; Hugo would use the year 0001.")
	}
	out = h.makePath(out)
	if !strings.HasPrefix(out, "/") {
		out = "/" + out
	}
	if h.uglyURLs && strings.HasSuffix(out, "/") && out != "/" {
		out = strings.TrimSuffix(out, "/") + ".html"
	}
	return out, true
}

func pageDate(doc *Document) time.Time {
	if v, ok := doc.Get("date"); ok {
		if t, _, err := ParseDate(v); err == nil {
			return t
		}
	}
	return time.Time{}
}

// planPermalinks writes Nida permalink patterns that reproduce Hugo's,
// so most pages keep their URL without a per-page path. Nida permalinks
// apply to one section directory each, so a top-level Hugo pattern is
// written for that section and every directory under it. Patterns using
// tokens Nida lacks are left out; the URL check then gives those pages a
// `path`.
func (h *hugoSite) planPermalinks(files []hugoContentFile, bundles map[string]bool) {
	dirs := map[string]bool{}
	for _, f := range files {
		base := path.Base(f.rel)
		if strings.HasPrefix(base, "_index.") || (!strings.HasPrefix(base, "index.") && hugoInBundle(f.rel, bundles)) {
			continue
		}
		if d := sectionDir(f); d != "" {
			dirs[d] = true
		}
	}
	perm := h.p.Config.Table("permalinks")
	for _, top := range sortedKeys(h.pagePermalinks) {
		pattern := h.pagePermalinks[top]
		translated, ok := h.translatePermalink(pattern, top)
		if !ok {
			h.p.Report.Add(Review, "config.toml", 0, "Permalink `%s = %q` uses tokens Nida cannot express, so its pages keep their URLs through `path`.", top, pattern)
			continue
		}
		for _, d := range sortedKeys(dirs) {
			if d == top || strings.HasPrefix(d, top+"/") {
				perm.Set(d, strings.ReplaceAll(translated, "{sections}", h.makePath(d)))
			}
		}
		h.p.Report.Add(Changed, "config.toml", 0, "Hugo permalinks became `[permalinks]` patterns for each section directory.")
	}
	if h.uglyURLs {
		for _, d := range sortedKeys(dirs) {
			if _, ok := perm.Get(d); !ok {
				perm.Set(d, "/"+h.makePath(d)+"/{slug}.html")
			}
		}
		h.p.Report.Add(Changed, "config.toml", 0, "`uglyURLs` became `.html` permalink patterns. Section and home pages are still written as `index.html` in their directory.")
	}
}

// translatePermalink turns a Hugo pattern into a Nida one for the section
// top. Slug-like tokens become {slug}; alignSlug then sets each page's
// slug to the value Hugo used.
func (h *hugoSite) translatePermalink(pattern, top string) (string, bool) {
	ok := true
	out := hugoToken.ReplaceAllStringFunc(pattern, func(tok string) string {
		switch tok[1:] {
		case "year":
			return "{year}"
		case "month":
			return "{month}"
		case "day":
			return "{day}"
		case "section":
			return h.makePath(top)
		case "sections":
			return "{sections}"
		case "title", "slug", "filename", "contentbasename", "slugorfilename", "slugorcontentbasename":
			return "{slug}"
		}
		ok = false
		return tok
	})
	if !ok || strings.Contains(out, ":") || strings.Count(out, "{slug}") > 1 {
		return "", false
	}
	if !strings.HasPrefix(out, "/") {
		out = "/" + out
	}
	if h.uglyURLs && strings.HasSuffix(out, "/") {
		out = strings.TrimSuffix(out, "/") + ".html"
	}
	return out, true
}

// alignSlug sets a page's slug to the one Hugo used in its URL when Nida
// would derive a different one, so {slug} permalinks and the default
// /section/slug/ scheme produce the old URL.
func (h *hugoSite) alignSlug(doc *Document, f hugoContentFile) {
	if doc.Has("path") {
		return
	}
	section := sectionDir(f)
	top, _, _ := strings.Cut(section, "/")
	want := ""
	if pattern, ok := h.pagePermalinks[top]; ok && top != "" {
		m := hugoToken.FindAllString(pattern, -1)
		var slugTok string
		for _, tok := range m {
			switch tok[1:] {
			case "title", "slug", "filename", "contentbasename", "slugorfilename", "slugorcontentbasename":
				slugTok = tok[1:]
			}
		}
		if slugTok == "" {
			return
		}
		expanded, ok := h.expandPermalink(":"+slugTok, doc, f)
		if !ok {
			return
		}
		want = strings.Trim(expanded, "/")
		want = strings.TrimSuffix(want, ".html")
	} else {
		slug := doc.String("slug")
		if slug == "" {
			slug = fileSlug(f)
		}
		want = h.makePath(slug)
	}
	have := doc.String("slug")
	if have == "" {
		have = content.DeriveSlug(fileSlug(f))
	}
	if want != "" && want != have && !strings.ContainsAny(want, "/\\") {
		_ = doc.Set("slug", want)
		h.p.Report.Add(Changed, f.src, 0, "Set `slug` to the one Hugo used in the URL, which Nida would derive differently.")
	}
}

// hugoShortcode is one shortcode tag found in a body.
type hugoShortcode struct {
	start, end int // byte range of the whole tag
	name       string
	args       []Arg
	closing    bool
	selfClose  bool
	markdown   bool // {{% %}} rather than {{< >}}
	escaped    bool // {{</* */>}}
	inner      string
}

// scanHugoShortcodes finds every shortcode tag in s, including escaped
// ones. Hugo processes shortcodes everywhere, code blocks included.
// reportUnreadableShortcodes reports {{< and {{% openings outside code
// that are not part of a shortcode the migration could read, such as one
// with an invalid name. They stay in the page as text.
func (h *hugoSite) reportUnreadableShortcodes(body string, tags []hugoShortcode, code [][2]int, file string, line func(int) int) {
	for i := 0; i < len(body); {
		j := strings.Index(body[i:], "{{")
		if j < 0 {
			return
		}
		at := i + j
		i = at + 2
		if at+2 >= len(body) || (body[at+2] != '<' && body[at+2] != '%') {
			continue
		}
		inTag := false
		for _, t := range tags {
			if at >= t.start && at < t.end {
				inTag = true
				break
			}
		}
		inCode := false
		for _, c := range code {
			if at >= c[0] && at < c[1] {
				inCode = true
				break
			}
		}
		if !inTag && !inCode {
			end := min(len(body), at+40)
			h.p.Report.Add(Review, file, line(at), "`%s` looks like a shortcode but could not be read, so it is shown as text.", truncate(body[at:end], 40))
		}
	}
}

func scanHugoShortcodes(s string) []hugoShortcode {
	var out []hugoShortcode
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], "{{")
		if j < 0 {
			break
		}
		start := i + j
		if start+2 >= len(s) || (s[start+2] != '<' && s[start+2] != '%') {
			i = start + 2
			continue
		}
		delim := s[start+2]
		tag, ok := parseHugoTag(s, start, delim)
		if !ok {
			i = start + 3
			continue
		}
		out = append(out, tag)
		i = tag.end
	}
	return out
}

func parseHugoTag(s string, start int, delim byte) (hugoShortcode, bool) {
	closeDelim := ">}}"
	if delim == '%' {
		closeDelim = "%}}"
	}
	pos := start + 3
	escaped := strings.HasPrefix(s[pos:], "/*")
	if escaped {
		end := strings.Index(s[pos:], "*/"+closeDelim)
		if end < 0 {
			return hugoShortcode{}, false
		}
		inner := strings.TrimSpace(s[pos+2 : pos+end])
		return hugoShortcode{start: start, end: pos + end + 2 + len(closeDelim), escaped: true, markdown: delim == '%', inner: inner}, true
	}
	// Find the closing delimiter outside quotes and raw strings.
	var quote byte
	end := -1
	for k := pos; k < len(s); k++ {
		c := s[k]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				k++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '`':
			quote = c
		case strings.HasPrefix(s[k:], closeDelim):
			end = k
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return hugoShortcode{}, false
	}
	body := strings.TrimSpace(s[pos:end])
	tag := hugoShortcode{start: start, end: end + len(closeDelim), markdown: delim == '%'}
	if strings.HasPrefix(body, "/") {
		tag.closing = true
		body = strings.TrimSpace(body[1:])
	}
	if strings.HasSuffix(body, "/") {
		tag.selfClose = true
		body = strings.TrimSpace(strings.TrimSuffix(body, "/"))
	}
	tokens, ok := hugoTokens(body)
	if !ok || len(tokens) == 0 || !validHugoName(tokens[0]) {
		return hugoShortcode{}, false
	}
	tag.name = tokens[0]
	for _, tok := range tokens[1:] {
		if key, value, isNamed := cutNamedArg(tok); isNamed {
			tag.args = append(tag.args, Arg{Key: key, Value: value})
		} else {
			tag.args = append(tag.args, Arg{Value: unquoteHugo(tok)})
		}
	}
	return tag, true
}

func validHugoName(name string) bool {
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '/' {
			return false
		}
	}
	return name != ""
}

func cutNamedArg(tok string) (string, string, bool) {
	eq := strings.IndexByte(tok, '=')
	if eq <= 0 || strings.ContainsAny(tok[:eq], "\"` ") {
		return "", "", false
	}
	return tok[:eq], unquoteHugo(tok[eq+1:]), true
}

// hugoTokens splits shortcode arguments on whitespace, keeping "quoted"
// and `raw` strings (which may span lines) together.
func hugoTokens(body string) ([]string, bool) {
	var tokens []string
	var b strings.Builder
	var quote byte
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case quote != 0:
			b.WriteByte(c)
			if c == '\\' && quote == '"' && i+1 < len(body) {
				i++
				b.WriteByte(body[i])
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '`':
			quote = c
			b.WriteByte(c)
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			if b.Len() > 0 {
				tokens = append(tokens, b.String())
				b.Reset()
			}
		default:
			b.WriteByte(c)
		}
	}
	if quote != 0 {
		return nil, false
	}
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
	}
	return tokens, true
}

func unquoteHugo(s string) string {
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1]
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
		return s[1 : len(s)-1]
	}
	return s
}

func argValue(args []Arg, key string, pos int) (string, bool) {
	for _, a := range args {
		if a.Key == key {
			return a.Value, true
		}
	}
	n := 0
	for _, a := range args {
		if a.Key == "" {
			if n == pos {
				return a.Value, true
			}
			n++
		}
	}
	return "", false
}

// codeRegions returns the byte ranges of body that are code.
func codeRegions(body string) [][2]int {
	var prose [][2]int
	RewriteProse(body, func(text string, offset int) string {
		prose = append(prose, [2]int{offset, offset + len(text)})
		return text
	})
	var code [][2]int
	pos := 0
	for _, r := range prose {
		if r[0] > pos {
			code = append(code, [2]int{pos, r[0]})
		}
		pos = r[1]
	}
	if pos < len(body) {
		code = append(code, [2]int{pos, len(body)})
	}
	return code
}

func inRegions(regions [][2]int, at int) bool {
	for _, r := range regions {
		if at >= r[0] && at < r[1] {
			return true
		}
	}
	return false
}

// convertBody rewrites Hugo shortcodes for Nida.
func (h *hugoSite) convertBody(doc *Document, f hugoContentFile, index map[string]string) {
	body := doc.Body
	tags := scanHugoShortcodes(body)
	code := codeRegions(body)
	r := h.p.Report
	line := func(at int) int { return doc.BodyLine + LineOf(body, at) - 1 }
	h.reportUnreadableShortcodes(body, tags, code, f.src, line)
	if len(tags) == 0 {
		return
	}

	var out strings.Builder
	last := 0
	for i := 0; i < len(tags); i++ {
		t := tags[i]
		inCode := inRegions(code, t.start)
		out.WriteString(body[last:t.start])
		last = t.end
		if t.escaped {
			literal := "{{< " + t.inner + " >}}"
			if t.markdown {
				literal = "{{% " + t.inner + " %}}"
			}
			if inCode {
				out.WriteString(literal) // Nida leaves code alone
			} else {
				out.WriteString("&#123;" + literal[1:])
			}
			continue
		}
		if inCode {
			r.Add(Review, f.src, line(t.start), "Hugo ran the `%s` shortcode inside a code block; Nida shows it as text.", t.name)
			out.WriteString(body[t.start:t.end])
			continue
		}
		if t.markdown {
			r.Add(Changed, f.src, 0, "`{{%% %%}}` shortcodes became `{{< >}}`; Nida renders their inner text as Markdown.")
		}
		if t.closing {
			switch t.name {
			case "details":
				out.WriteString("{% end %}")
			case "highlight", "param", "ref", "relref":
				out.WriteString(body[t.start:t.end])
			default:
				out.WriteString(ClosingShortcode(t.name))
			}
			continue
		}
		switch t.name {
		case "ref", "relref":
			out.WriteString(h.resolveRef(t, f, index, line(t.start)))
		case "param":
			out.WriteString(h.paramValue(t, doc, f, line(t.start)))
		case "highlight":
			closeIdx := findClosing(tags, i, "highlight")
			if t.selfClose || closeIdx < 0 {
				r.Add(Manual, f.src, line(t.start), "`highlight` shortcode without a closing tag was left as it was.")
				out.WriteString(body[t.start:t.end])
				continue
			}
			out.WriteString(hugoHighlight(t, body[t.end:tags[closeIdx].start], r, f.src, line(t.start)))
			last = tags[closeIdx].end
			i = closeIdx
		case "details":
			summary, _ := argValue(t.args, "summary", 0)
			if summary == "" {
				summary = "Details"
			}
			out.WriteString(`{% details(summary="` + strings.ReplaceAll(summary, `"`, `'`) + `") %}`)
			for _, a := range t.args {
				if a.Key != "summary" && a.Key != "" {
					r.Add(Review, f.src, line(t.start), "`details` option `%s` was dropped.", a.Key)
				}
			}
			if t.selfClose {
				out.WriteString("{% end %}")
			}
		case "figure":
			out.WriteString(hugoFigure(t, r, f.src, line(t.start)))
		case "youtube":
			out.WriteString(hugoYouTube(t, r, f.src, line(t.start)))
		default:
			tag := Shortcode(t.name, t.args)
			if t.selfClose {
				tag = strings.TrimSuffix(tag, " >}}") + " />}}"
			}
			out.WriteString(tag)
			if !validShortcode(t.name) {
				r.Add(Manual, f.src, line(t.start), "Shortcode name `%s` is not valid in Nida (letters, digits, `-`, and `_` only); rename its template and calls.", t.name)
				continue
			}
			h.p.Placeholder(t.name, f.src, line(t.start))
		}
	}
	out.WriteString(body[last:])
	doc.SetBody(out.String())
}

func validShortcode(name string) bool {
	for _, r := range name {
		if r != '-' && r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return name != ""
}

// findClosing returns the index of the tag closing tags[open], allowing
// nested tags with the same name.
func findClosing(tags []hugoShortcode, open int, name string) int {
	depth := 0
	for j := open + 1; j < len(tags); j++ {
		if tags[j].name != name || tags[j].escaped {
			continue
		}
		if tags[j].closing {
			if depth == 0 {
				return j
			}
			depth--
		} else if !tags[j].selfClose {
			depth++
		}
	}
	return -1
}

// hugoHighlight turns {{< highlight lang "opts" >}} into a fenced code
// block with Nida's info string options.
func hugoHighlight(t hugoShortcode, code string, r *Report, file string, line int) string {
	lang, _ := argValue(t.args, "lang", 0)
	opts, _ := argValue(t.args, "options", 1)
	var info []string
	for _, opt := range splitHighlightOpts(opts) {
		key, value, _ := strings.Cut(opt, "=")
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(value)
		switch key {
		case "":
		case "hl_lines":
			info = append(info, fmt.Sprintf("hl_lines=%q", value))
		case "linenos":
			if value != "false" && value != "" {
				info = append(info, "linenos=true")
			} else if value == "" {
				info = append(info, "linenos=true")
			}
		case "linenostart":
			info = append(info, "linenostart="+value)
		default:
			r.Add(Review, file, line, "`highlight` option `%s` has no Nida equivalent and was dropped.", key)
		}
	}
	code = strings.TrimPrefix(code, "\r\n")
	code = strings.TrimPrefix(code, "\n")
	code = strings.TrimRight(code, " \t\r\n")
	fence := "```"
	for strings.Contains(code, fence) {
		fence += "`"
	}
	head := fence + lang
	if len(info) > 0 {
		head += " {" + strings.Join(info, ", ") + "}"
	}
	r.Add(Changed, file, 0, "`highlight` shortcodes became fenced code blocks.")
	return head + "\n" + code + "\n" + fence
}

// splitHighlightOpts splits "linenos=table,hl_lines=8 15-17" on commas;
// hl_lines values may hold spaces but no commas.
func splitHighlightOpts(opts string) []string {
	var out []string
	for _, part := range strings.Split(opts, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func hugoFigure(t hugoShortcode, r *Report, file string, line int) string {
	var args []Arg
	get := func(key string) string { v, _ := argValue(t.args, key, -1); return v }
	src := get("src")
	if src == "" {
		src, _ = argValue(t.args, "", 0)
	}
	args = append(args, Arg{Key: "src", Value: src})
	caption := get("caption")
	if title := get("title"); title != "" {
		if caption == "" {
			caption = title
		} else {
			caption = title + ": " + caption
		}
		r.Add(Review, file, line, "`figure` title was merged into the caption; Nida's figure has no separate title.")
	}
	for _, key := range []string{"alt", "link", "width", "height", "class"} {
		if v := get(key); v != "" {
			args = append(args, Arg{Key: key, Value: v})
		}
	}
	if caption != "" {
		args = append(args, Arg{Key: "caption", Value: caption})
	}
	for _, a := range t.args {
		switch a.Key {
		case "src", "caption", "title", "alt", "link", "width", "height", "class", "":
		default:
			r.Add(Review, file, line, "`figure` option `%s` has no Nida equivalent and was dropped.", a.Key)
		}
	}
	return Shortcode("figure", args)
}

func hugoYouTube(t hugoShortcode, r *Report, file string, line int) string {
	id, _ := argValue(t.args, "id", 0)
	args := []Arg{{Key: "id", Value: id}}
	for _, a := range t.args {
		switch a.Key {
		case "id", "":
		case "start", "title":
			args = append(args, a)
		default:
			r.Add(Review, file, line, "`youtube` option `%s` has no Nida equivalent and was dropped.", a.Key)
		}
	}
	return Shortcode("youtube", args)
}

// resolveRef turns {{< ref "path" >}} into an @/ content link.
func (h *hugoSite) resolveRef(t hugoShortcode, f hugoContentFile, index map[string]string, line int) string {
	target, _ := argValue(t.args, "path", 0)
	for _, a := range t.args {
		if a.Key == "lang" || a.Key == "outputFormat" {
			h.p.Report.Add(Review, f.src, line, "`%s` option `%s` was dropped.", t.name, a.Key)
		}
	}
	target, anchor, _ := strings.Cut(target, "#")
	if anchor != "" {
		anchor = "#" + anchor
	}
	if target == "" {
		return anchor // a same-page anchor
	}
	dest, ambiguous := h.findContent(target, f, index)
	if dest == "" {
		h.p.Report.Add(Manual, f.src, line, "`%s` to `%s` matches no content file; the link was left as `%s`.", t.name, target, target)
		return target + anchor
	}
	if ambiguous {
		h.p.Report.Add(Review, f.src, line, "`%s` to `%s` matches more than one file; it now links to `%s`.", t.name, target, dest)
	}
	return "@/" + strings.TrimPrefix(dest, "content/") + anchor
}

// findContent resolves a Hugo ref path: relative to the page first, then
// to the content root, then by unique file name.
func (h *hugoSite) findContent(target string, f hugoContentFile, index map[string]string) (string, bool) {
	candidates := func(p string) []string {
		p = strings.Trim(path.Clean(p), "/")
		var out []string
		if ext := path.Ext(p); slices.Contains(markdownExts, ext) || slices.Contains(htmlExts, ext) {
			out = append(out, p)
			p = strings.TrimSuffix(p, ext)
		}
		for _, ext := range []string{".md", ".markdown", ".html"} {
			out = append(out, p+ext, p+"/index"+ext, p+"/_index"+ext)
		}
		if p == "" || p == "." {
			out = append(out, "_index.md")
		}
		return out
	}
	var bases []string
	if !strings.HasPrefix(target, "/") {
		dir := path.Dir(f.rel)
		if strings.HasPrefix(path.Base(f.rel), "index.") {
			dir = path.Dir(dir)
		}
		bases = append(bases, path.Join(dir, target))
	}
	bases = append(bases, target)
	for _, base := range bases {
		for _, c := range candidates(base) {
			if dest, ok := index[c]; ok {
				return dest, false
			}
		}
	}
	// Hugo also finds a page by file name alone.
	name := strings.TrimSuffix(path.Base(target), path.Ext(target))
	var matches []string
	for _, rel := range sortedKeys(index) {
		stem := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		if stem == "index" {
			stem = path.Base(path.Dir(rel))
		}
		if stem == name {
			matches = append(matches, index[rel])
		}
	}
	if len(matches) > 0 {
		return matches[0], len(matches) > 1
	}
	return "", false
}

// paramValue inlines {{< param "key" >}}: the page's front matter value,
// else the site param. Dotted keys reach into maps.
func (h *hugoSite) paramValue(t hugoShortcode, doc *Document, f hugoContentFile, line int) string {
	key, _ := argValue(t.args, "", 0)
	lookup := func(m map[string]any) (any, bool) {
		var cur any = m
		for _, part := range strings.Split(key, ".") {
			mm, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			v, ok := mm[part]
			if !ok {
				v, ok = mm[strings.ToLower(part)]
			}
			if !ok {
				return nil, false
			}
			cur = v
		}
		return cur, true
	}
	v, ok := lookup(doc.Values())
	if !ok {
		if params, isMap := doc.Values()["params"].(map[string]any); isMap {
			v, ok = lookup(params)
		}
	}
	if !ok {
		v, ok = lookup(h.params)
	}
	if !ok {
		h.p.Report.Add(Manual, f.src, line, "`param` `%s` is not set in the page or site params, so it renders a placeholder box.", key)
		h.p.Placeholder("param", f.src, line)
		return Shortcode("param", []Arg{{Value: key}})
	}
	h.p.Report.Add(Changed, f.src, 0, "`param` shortcodes were replaced with their values.")
	return fmt.Sprint(v)
}
