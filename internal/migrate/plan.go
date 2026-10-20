package migrate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/safepath"
)

// Source converts one kind of site. Each generator lives in its own file.
type Source interface {
	// Name is the --from value, such as "hugo".
	Name() string
	// Title is the generator's display name, such as "Hugo".
	Title() string
	// Detect reports how confident the source is that root is its kind of
	// site: 0 for no, higher for stronger evidence. It must be cheap and
	// must not fail on unexpected input.
	Detect(root string) int
	// Plan reads the site at p.Root and fills in p. It must not write
	// anything; the plan is applied only after every source file has been
	// read, so a failure leaves nothing half-written.
	Plan(p *Plan) error
}

// Plan is everything a migration will write, built before anything is.
type Plan struct {
	// Root is the absolute source site directory.
	Root string
	// realRoot is Root with symlinks resolved, for containment checks.
	realRoot string
	Config   *Config
	Report   *Report

	files map[string]*planned
	order []string
	// skip holds absolute directories that walks must not enter, such as
	// the destination when it is inside the source.
	skip []string
	// candidates are translated templates to try before falling back to
	// the starter set; see Run.
	candidates map[string][]byte
	// placeholders are stand-in shortcode templates; see Placeholder.
	placeholders map[string][]byte
	// unsafeHTML is set when migrated content uses raw HTML.
	unsafeHTML bool
}

type planned struct {
	dest   string
	origin string // source-relative path, for the report
	data   []byte
	from   string // absolute source path to copy, when data is nil
	doc    *Document
	// oldURL is the page's URL on the old site, for content documents.
	oldURL string
}

func newPlan(root string, report *Report) *Plan {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		realRoot = root
	}
	return &Plan{
		Root:       root,
		realRoot:   realRoot,
		Config:     &Config{},
		Report:     report,
		files:      map[string]*planned{},
		candidates: map[string][]byte{},
	}
}

// Write plans a generated file. dest is relative to the new site root, in
// forward-slash form. origin is the source-relative file it came from, or
// "" for generated files.
func (p *Plan) Write(dest string, data []byte, origin string) {
	p.add(&planned{dest: dest, origin: origin, data: data})
}

// Copy plans copying a source file unchanged.
func (p *Plan) Copy(dest, origin string) {
	from, ok := p.sourcePath(origin)
	if !ok {
		p.Report.Add(Manual, origin, 0, "Not migrated: `%s` is outside the source site.", origin)
		return
	}
	if info, err := os.Lstat(from); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		p.Report.Add(Manual, origin, 0, "Symlink not followed. Copy what it points to into the migrated site if it is needed.")
		return
	}
	p.add(&planned{dest: dest, origin: origin, from: from})
}

// sourcePath resolves a source-relative path, refusing paths that leave
// the source site, so a config value such as "../../secret" cannot pull
// outside files into the migrated site.
func (p *Plan) sourcePath(rel string) (string, bool) {
	clean := path.Clean(strings.TrimPrefix(filepath.ToSlash(rel), "/"))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	full := filepath.Join(p.Root, filepath.FromSlash(clean))
	return full, p.insideSource(full)
}

// insideSource reports whether full, with every symlink in it resolved,
// stays inside the source site. A symlinked parent directory such as
// themes -> ../elsewhere would otherwise pull outside files in. Paths that
// do not exist pass; reading them fails on its own.
func (p *Plan) insideSource(full string) bool {
	if p.realRoot == "" {
		// Plans built only for detection skip newPlan.
		p.realRoot = realPath(p.Root)
	}
	real, err := filepath.EvalSymlinks(full)
	if err != nil {
		return true
	}
	return real == p.realRoot || within(p.realRoot, real)
}

// Page plans a content document. dest must be under content/. oldURL is
// the page's URL on the old site, or "" if it is not known; after the
// plan is written, pages whose Nida URL differs get a path or alias.
func (p *Plan) Page(dest string, doc *Document, origin, oldURL string) {
	if HasRawHTML(doc.Body) {
		p.unsafeHTML = true
	}
	p.add(&planned{dest: dest, origin: origin, doc: doc, oldURL: oldURL})
}

// Preserve plans copying a source file into _migration/original/ for
// reference, for files such as templates that have no Nida equivalent.
func (p *Plan) Preserve(origin string) {
	p.Copy(path.Join(preservedDir, origin), origin)
}

// Template plans a translated template, such as templates/post.html. The
// migrated site uses translated templates only if it builds with them;
// otherwise it gets the starter templates and these are kept in
// _migration/translated-templates/ for reference.
func (p *Plan) Template(name string, data []byte) {
	p.candidates[name] = data
}

// UsesRawHTML records that content needs raw HTML to render as before.
func (p *Plan) UsesRawHTML() { p.unsafeHTML = true }

// Planned reports whether dest has already been planned.
func (p *Plan) Planned(dest string) bool {
	_, ok := p.files[dest]
	return ok
}

// Doc returns the planned document at dest, or nil.
func (p *Plan) Doc(dest string) *Document {
	if f, ok := p.files[dest]; ok {
		return f.doc
	}
	return nil
}

func (p *Plan) add(f *planned) {
	f.dest = path.Clean(strings.TrimPrefix(filepath.ToSlash(f.dest), "/"))
	if f.dest == "." || f.dest == ".." || strings.HasPrefix(f.dest, "../") {
		p.Report.Add(Manual, f.origin, 0, "Not migrated: it maps to `%s`, which is outside the new site. Rename it.", f.dest)
		return
	}
	if existing, ok := p.files[f.dest]; ok {
		p.Report.Add(Manual, f.origin, 0, "Not migrated: it maps to `%s`, which `%s` already uses. Rename one of them.", f.dest, existing.origin)
		return
	}
	p.files[f.dest] = f
	p.order = append(p.order, f.dest)
}

// Rel returns the source-relative, forward-slash form of an absolute path.
func (p *Plan) Rel(abs string) string {
	rel, err := filepath.Rel(p.Root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// Exists reports whether a source-relative path exists.
func (p *Plan) Exists(rel string) bool {
	full, ok := p.sourcePath(rel)
	if !ok {
		return false
	}
	_, err := os.Stat(full)
	return err == nil
}

// ReadFile reads a source-relative file.
func (p *Plan) ReadFile(rel string) ([]byte, error) {
	full, ok := p.sourcePath(rel)
	if !ok {
		return nil, fmt.Errorf("%s is outside the source site", rel)
	}
	if info, err := os.Lstat(full); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s is a symlink, which migrate does not follow", rel)
	}
	return os.ReadFile(full)
}

// vcsDirs are never walked and never reported.
var vcsDirs = map[string]bool{".git": true, ".hg": true, ".svn": true}

// skippedDirs are dependency, cache, and build directories of the
// supported generators. They are skipped anywhere, and reported when they
// sit inside a directory being migrated, in case the site needs them.
var skippedDirs = map[string]bool{
	"node_modules": true, ".jekyll-cache": true, ".sass-cache": true,
	"__pycache__": true, ".hugo_build.lock": true,
}

// rootOnlyDirs are skipped only at the top of the source site, where they
// hold dependencies; inside static/, for example, vendor/ is site content.
var rootOnlyDirs = map[string]bool{"vendor": true, ".bundle": true, ".cache": true}

// Walk visits the regular files under the source-relative directory dir,
// passing each file's source-relative path. It skips version control,
// dependency, and cache directories, hidden files other than .well-known
// and .htaccess, and the destination. Symlinks are not followed; each one
// is reported.
func (p *Plan) Walk(dir string, fn func(rel string) error) error {
	start, ok := p.sourcePath(dir)
	if !ok {
		p.Report.Add(Manual, dir, 0, "Not migrated: `%s` leads outside the source site.", dir)
		return nil
	}
	if _, err := os.Stat(start); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(start, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.Type()&fs.ModeSymlink != 0 {
			p.Report.Add(Manual, p.Rel(abs), 0, "Symlink not followed. Copy what it points to into the migrated site if it is needed.")
			return nil
		}
		if d.IsDir() {
			if abs == start {
				return nil
			}
			switch {
			case vcsDirs[name] || slices.Contains(p.skip, abs):
				return fs.SkipDir
			case filepath.Dir(abs) == p.Root && (rootOnlyDirs[name] || skippedDirs[name] || hiddenName(name)):
				// Tooling at the top of the site: .github/, node_modules/.
				return fs.SkipDir
			case skippedDirs[name] || hiddenName(name):
				p.Report.Add(Review, p.Rel(abs), 0, "Skipped `%s/` as a dependency, cache, or hidden directory. Copy it into the migrated site by hand if the site needs it.", p.Rel(abs))
				return fs.SkipDir
			}
			return nil
		}
		if hiddenName(name) || name == ".DS_Store" {
			return nil
		}
		return fn(p.Rel(abs))
	})
}

func hiddenName(name string) bool {
	return strings.HasPrefix(name, ".") && name != ".well-known" && name != ".htaccess" && name != ".nojekyll"
}

// apply writes the plan into dest.
func (p *Plan) apply(dest string) error {
	for _, rel := range p.order {
		f := p.files[rel]
		target, err := safepath.Join(dest, filepath.FromSlash(rel))
		if err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		switch {
		case f.doc != nil:
			err = os.WriteFile(target, f.doc.Bytes(), 0o644)
		case f.data != nil:
			err = os.WriteFile(target, f.data, 0o644)
		default:
			err = copyFile(f.from, target)
		}
		if err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	}
	return nil
}

func copyFile(from, to string) error {
	if info, err := os.Lstat(from); err != nil {
		return err
	} else if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink, which migrate does not follow", from)
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
