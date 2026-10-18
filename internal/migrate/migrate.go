// Package migrate converts sites from other static site generators into
// Nida sites. Each generator is a Source that reads the old site and fills
// in a Plan; Run then writes the plan to a new directory, checks every
// page's URL against Nida's router, picks templates that build, and
// writes a MIGRATION.md report of everything that changed or needs work.
package migrate

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/MohamedElashri/nida/internal/safepath"

	// Jekyll and Pelican sites name a timezone for naive dates. Embedding
	// the zone database keeps migrations the same on systems without one.
	_ "time/tzdata"
)

//go:embed starter
var starterFS embed.FS

const (
	preservedDir  = "_migration/original"
	translatedDir = "_migration/translated-templates"
	reportName    = "MIGRATION.md"
)

// URL modes for Options.URLs.
const (
	// KeepURLs gives pages whose Nida URL differs a `path` so they stay at
	// their old URL.
	KeepURLs = "keep"
	// RedirectURLs leaves Nida's URLs and adds the old URL as an alias,
	// which writes a redirect page.
	RedirectURLs = "redirect"
)

// Sources lists the supported generators in detection order.
var Sources = []Source{hugoSource{}, jekyllSource{}, zolaSource{}, eleventySource{}, pelicanSource{}}

// Options configures a migration.
type Options struct {
	// Source is the old site's root directory.
	Source string
	// Dest is the new site's directory. It must not exist or be empty.
	Dest string
	// From names the source generator. Empty or "auto" detects it.
	From string
	// URLs is KeepURLs (the default) or RedirectURLs.
	URLs string
	// DryRun migrates into a temporary directory, so the report is
	// complete, and removes it afterwards. Nothing is written to Dest.
	DryRun bool
	// Check builds the migrated site the way `nida check` does and
	// returns its error. It is optional; the CLI provides it, since the
	// build lives outside this package.
	Check func(siteRoot string) error
}

// Result describes a finished migration.
type Result struct {
	Report *Report
	// ReportPath is where MIGRATION.md was written, or "" for a dry run.
	ReportPath string
	// Markdown is the report text.
	Markdown string
}

// Run migrates a site.
func Run(opts Options) (*Result, error) {
	root, err := filepath.Abs(opts.Source)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(root); err != nil {
		return nil, fmt.Errorf("read source site: %w", err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("source site %q is not a directory", opts.Source)
	}
	dest, err := filepath.Abs(opts.Dest)
	if err != nil {
		return nil, err
	}
	if dest == root {
		return nil, errors.New("destination must differ from the source; migrate writes a new site and never changes the old one")
	}
	if within(dest, root) {
		return nil, errors.New("destination must not contain the source site")
	}
	if !opts.DryRun {
		if err := ensureEmpty(dest); err != nil {
			return nil, err
		}
	}
	switch opts.URLs {
	case "":
		opts.URLs = KeepURLs
	case KeepURLs, RedirectURLs:
	default:
		return nil, fmt.Errorf("unknown --urls mode %q; use %q or %q", opts.URLs, KeepURLs, RedirectURLs)
	}

	src, err := pickSource(root, opts.From)
	if err != nil {
		return nil, err
	}
	report := &Report{Source: opts.Source, From: src.Title(), Dest: opts.Dest}
	plan := newPlan(root, report)
	if within(root, dest) {
		plan.skip = append(plan.skip, dest)
	}
	if err := src.Plan(plan); err != nil {
		return nil, fmt.Errorf("read %s site: %w", src.Title(), err)
	}
	if err := finishPlan(plan); err != nil {
		return nil, err
	}

	target := dest
	if opts.DryRun {
		if target, err = os.MkdirTemp("", "nida-migrate-*"); err != nil {
			return nil, err
		}
		defer func() { _ = os.RemoveAll(target) }()
	} else if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}

	if err := plan.apply(target); err != nil {
		return nil, err
	}
	if err := fixURLs(plan, target, opts.URLs); err != nil {
		return nil, err
	}
	if err := chooseTemplates(plan, target, src, opts.Check); err != nil {
		return nil, err
	}
	countStats(plan)

	result := &Result{Report: report, Markdown: report.Markdown()}
	if !opts.DryRun {
		result.ReportPath = filepath.Join(dest, reportName)
		if err := os.WriteFile(result.ReportPath, []byte(result.Markdown), 0o644); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// pickSource returns the named source, or detects one.
func pickSource(root, name string) (Source, error) {
	if name != "" && name != "auto" {
		for _, s := range Sources {
			if s.Name() == name {
				return s, nil
			}
		}
		return nil, fmt.Errorf("unknown source %q; use one of %s", name, strings.Join(sourceNames(), ", "))
	}
	if isNidaSite(root) {
		return nil, fmt.Errorf("%s is already a Nida site; pass --from to migrate it anyway", root)
	}
	type scored struct {
		src   Source
		score int
	}
	var found []scored
	for _, s := range Sources {
		if score := s.Detect(root); score > 0 {
			found = append(found, scored{s, score})
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].score > found[j].score })
	switch {
	case len(found) == 0:
		return nil, fmt.Errorf("could not tell which generator built %s; pass --from with one of %s", root, strings.Join(sourceNames(), ", "))
	case len(found) > 1 && found[0].score == found[1].score:
		return nil, fmt.Errorf("%s looks like both a %s and a %s site; pass --from to choose", root, found[0].src.Title(), found[1].src.Title())
	}
	return found[0].src, nil
}

// goTemplateAction matches Go template actions that Nida templates use and
// Zola's Tera templates never do.
var goTemplateAction = regexp.MustCompile(`\{\{-?\s*(?:define|template|block)\s+"`)

// isNidaSite reports whether root looks like a Nida site: a config.toml
// with base_url and Go templates under templates/, its own or a theme's.
// Zola sites share the config file but use Tera templates, and Hugo keeps
// its Go templates in layouts/.
func isNidaSite(root string) bool {
	data, err := os.ReadFile(filepath.Join(root, "config.toml"))
	if err != nil || !strings.Contains(string(data), "base_url") {
		return false
	}
	dirs, _ := filepath.Glob(filepath.Join(root, "themes", "*", "templates"))
	dirs = append([]string{filepath.Join(root, "templates")}, dirs...)
	for _, dir := range dirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.html"))
		for _, file := range files {
			if data, err := os.ReadFile(file); err == nil && goTemplateAction.Match(data) {
				return true
			}
		}
	}
	return false
}

func sourceNames() []string {
	names := make([]string, 0, len(Sources))
	for _, s := range Sources {
		names = append(names, s.Name())
	}
	return names
}

// finishPlan adds what every migrated site needs: section index files,
// config.toml, and the starter stylesheet.
func finishPlan(p *Plan) error {
	addMissingSections(p)
	if p.unsafeHTML {
		if v, ok := p.Config.Table("markdown").Get("unsafe_html"); !ok || v != false {
			p.Config.Table("markdown").Set("unsafe_html", true)
			p.Report.Add(Review, "", 0, "Content uses raw HTML, so `markdown.unsafe_html` is on. Nida turns it off by default because it lets Markdown authors inject scripts; keep it only if you trust everyone who writes content.")
		}
	}
	if title, _ := p.Config.Get("title"); title == nil || title == "" {
		p.Config.Set("title", "My Site")
		p.Report.Add(Manual, "", 0, "The old site sets no title, so `title` is \"My Site\". Change it in config.toml.")
	}
	if _, ok := p.Config.Get("base_url"); !ok {
		p.Config.Set("base_url", "http://127.0.0.1:1702")
		p.Report.Add(Manual, "", 0, "The old site sets no base URL, so `base_url` is the local preview address. Set it to the site's real URL before deploying.")
	}
	data, err := p.Config.Bytes()
	if err != nil {
		return err
	}
	p.Write("config.toml", data, "")
	css, err := starterFS.ReadFile("starter/static/nida-migrate.css")
	if err != nil {
		return err
	}
	if !p.Planned("static/nida-migrate.css") {
		p.Write("static/nida-migrate.css", css, "")
	}
	return nil
}

// addMissingSections writes an _index.md for each content directory that
// holds pages but has none, titled after the directory, so every page has
// a section and the starter templates can list it.
func addMissingSections(p *Plan) {
	dirs := map[string]bool{}
	for _, dest := range p.order {
		f := p.files[dest]
		if f.doc == nil || !strings.HasPrefix(dest, "content/") {
			continue
		}
		dir := path.Dir(dest)
		if path.Base(dest) == "index.md" {
			dir = path.Dir(dir) // a bundle belongs to its parent section
		}
		for d := dir; d != "." && d != "/"; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	keys := make([]string, 0, len(dirs))
	for d := range dirs {
		keys = append(keys, d)
	}
	slices.Sort(keys)
	for _, dir := range keys {
		index := dir + "/_index.md"
		if p.Planned(index) || p.Planned(dir+"/index.md") {
			continue
		}
		doc := NewDocument(TOML, "")
		title := "Home"
		if dir != "content" {
			title = titleFromSlug(path.Base(dir))
		}
		_ = doc.Set("title", title)
		p.add(&planned{dest: index, doc: doc})
	}
}

func titleFromSlug(slug string) string {
	words := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	for i, w := range words {
		r, size := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToTitle(r)) + w[size:]
	}
	return strings.Join(words, " ")
}

// chooseTemplates writes the translated templates if the site builds with
// them, and the starter templates otherwise.
func chooseTemplates(p *Plan, target string, src Source, check func(string) error) error {
	starter, err := starterTemplates()
	if err != nil {
		return err
	}
	if len(p.candidates) > 0 {
		if err := writeTemplates(target, p.placeholders); err != nil {
			return err
		}
		if err := writeTemplates(target, p.candidates); err != nil {
			return err
		}
		if check == nil {
			p.Report.TemplateSet = "translated from " + src.Title() + " (not built: no check available)"
			p.Report.Templates = len(p.candidates)
			return nil
		}
		checkErr := check(target)
		if checkErr == nil {
			p.Report.TemplateSet = "translated from " + src.Title()
			p.Report.Templates = len(p.candidates)
			p.Report.Check = "passed"
			return nil
		}
		// Fall back to the starter set and keep the translations aside.
		if err := os.RemoveAll(filepath.Join(target, "templates")); err != nil {
			return err
		}
		kept := map[string][]byte{}
		for name, data := range p.candidates {
			kept[path.Join(translatedDir, strings.TrimPrefix(name, "templates/"))] = data
		}
		if err := writeTemplates(target, kept); err != nil {
			return err
		}
		p.Report.Add(Review, "", 0, "The site did not build with the templates translated from %s (%v), so it uses Nida's starter templates. The translations are in `%s/` to finish by hand.", src.Title(), checkErr, translatedDir)
	}
	if err := writeTemplates(target, starter); err != nil {
		return err
	}
	if err := writeTemplates(target, p.placeholders); err != nil {
		return err
	}
	p.Report.Templates = len(starter)
	p.Report.TemplateSet = "Nida starter templates (restyle them, or port your old layouts from `" + preservedDir + "/`)"
	if check != nil {
		if err := check(target); err != nil {
			p.Report.Check = fmt.Sprintf("failed: %v", err)
		} else {
			p.Report.Check = "passed"
		}
	}
	return nil
}

func starterTemplates() (map[string][]byte, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(starterFS, "starter/templates", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := starterFS.ReadFile(name)
		files[strings.TrimPrefix(name, "starter/")] = data
		return err
	})
	return files, err
}

func writeTemplates(target string, files map[string][]byte) error {
	for name, data := range files {
		full, err := safepath.Join(target, filepath.FromSlash(name))
		if err != nil {
			return fmt.Errorf("write template %s: %w", name, err)
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func countStats(p *Plan) {
	for _, dest := range p.order {
		f := p.files[dest]
		switch {
		case f.doc != nil && path.Base(dest) == "_index.md":
			p.Report.Sections++
		case f.doc != nil:
			p.Report.Pages++
		case f.from != "":
			p.Report.Copied++
		}
	}
}

func ensureEmpty(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("destination %q is not empty; migrate writes a new site and will not overwrite files", dir)
	}
	return nil
}

// within reports whether path is inside dir.
func within(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
}
