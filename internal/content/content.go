package content

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/frontmatter"
	"github.com/MohamedElashri/nida/internal/safepath"
)

func Discover(siteRoot string, cfg config.SiteConfig) ([]Page, []Section, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	contentRoot, err := safepath.Join(absSiteRoot, cfg.ContentDir)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve content dir %q: %w", cfg.ContentDir, err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, contentRoot); err != nil {
		return nil, nil, fmt.Errorf("check content dir %q: %w", cfg.ContentDir, err)
	}
	return discoverAll(contentRoot, cfg)
}

func discoverAll(contentRoot string, cfg config.SiteConfig) ([]Page, []Section, error) {
	type fileEntry struct {
		path     string
		isIndex  bool
		isBundle bool
	}

	var potentialPages []fileEntry
	bundleDirs := map[string]bool{}

	err := filepath.WalkDir(contentRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == contentRoot && os.IsNotExist(walkErr) {
				return fs.SkipDir
			}
			return walkErr
		}

		if d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}

		name := filepath.Base(path)
		if name == "_index.md" {
			potentialPages = append(potentialPages, fileEntry{path: path, isIndex: true})
		} else if name == "index.md" && !isContentRoot(path, contentRoot) {
			bundleDirs[filepath.Dir(path)] = true
			potentialPages = append(potentialPages, fileEntry{path: path, isBundle: true})
		} else {
			potentialPages = append(potentialPages, fileEntry{path: path})
		}

		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("discover content under %q: %w", contentRoot, err)
	}

	filtered := potentialPages[:0]
	for _, entry := range potentialPages {
		if !entry.isIndex && !entry.isBundle && bundleDirs[filepath.Dir(entry.path)] {
			continue
		}
		filtered = append(filtered, entry)
	}
	potentialPages = filtered

	slices.SortFunc(potentialPages, func(a, b fileEntry) int {
		return strings.Compare(a.path, b.path)
	})

	var pages []Page
	var sections []Section

	for _, entry := range potentialPages {
		if entry.isIndex {
			section, err := loadSection(contentRoot, entry.path, cfg)
			if err != nil {
				return nil, nil, err
			}
			sections = append(sections, section)
		} else if entry.isBundle {
			page, err := loadPage(contentRoot, entry.path, true)
			if err != nil {
				return nil, nil, err
			}
			pages = append(pages, page)
		} else {
			page, err := loadPage(contentRoot, entry.path, false)
			if err != nil {
				return nil, nil, err
			}
			pages = append(pages, page)
		}
	}

	sections = synthesizeImplicitSections(pages, sections, cfg)

	slices.SortFunc(pages, func(a, b Page) int {
		return strings.Compare(a.RelativePath, b.RelativePath)
	})
	slices.SortFunc(sections, func(a, b Section) int {
		return strings.Compare(a.SectionPath, b.SectionPath)
	})

	return pages, sections, nil
}

func synthesizeImplicitSections(pages []Page, sections []Section, cfg config.SiteConfig) []Section {
	existingPaths := map[string]bool{}
	for _, s := range sections {
		existingPaths[s.SectionPath] = true
	}

	dirsWithPages := map[string]bool{}
	for _, page := range pages {
		if page.SectionPath != "" {
			dirsWithPages[page.SectionPath] = true
		}
	}

	var implicit []Section
	for dirPath := range dirsWithPages {
		if existingPaths[dirPath] {
			continue
		}

		base := filepath.Base(dirPath)
		implicit = append(implicit, Section{
			SourcePath:    "",
			RelativePath:  filepath.ToSlash(filepath.Join(dirPath, "_index.md")),
			SectionPath:   dirPath,
			Title:         TitleCase(strings.ReplaceAll(base, "-", " ")),
			Slug:          DeriveSlug(base),
			URL:           "/" + dirPath + "/",
			PaginateBy:    0,
			PaginatePath:  cfg.Sections.PaginatePath,
			SortBy:        cfg.Sections.DefaultSortBy,
			Transparent:   false,
			GenerateFeeds: false,
			Sections:      nil,
			Pages:         nil,
			Extra:         map[string]any{},
		})
	}

	return append(sections, implicit...)
}

func loadSection(contentRoot, sourcePath string, cfg config.SiteConfig) (Section, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return Section{}, fmt.Errorf("read section file %q: %w", sourcePath, err)
	}

	doc, err := frontmatter.Parse(data)
	if err != nil {
		return Section{}, fmt.Errorf("parse section file %q: %w", sourcePath, err)
	}

	relativePath, err := filepath.Rel(contentRoot, sourcePath)
	if err != nil {
		return Section{}, fmt.Errorf("compute relative path for %q: %w", sourcePath, err)
	}

	meta := doc.Metadata
	sectionPath := filepath.ToSlash(filepath.Dir(filepath.ToSlash(relativePath)))
	if sectionPath == "." {
		sectionPath = ""
	}

	slug := DeriveSlug(filepath.Base(filepath.Dir(sourcePath)))

	return Section{
		SourcePath:    sourcePath,
		RelativePath:  filepath.ToSlash(relativePath),
		SectionPath:   sectionPath,
		BodyMarkdown:  doc.BodyMarkdown,
		Title:         strings.TrimSpace(meta.Title),
		Description:   strings.TrimSpace(meta.Description),
		Slug:          slug,
		Draft:         meta.Draft,
		Weight:        meta.Weight,
		Template:      strings.TrimSpace(meta.Template),
		PageTemplate:  strings.TrimSpace(meta.PageTemplate),
		PaginateBy:    meta.PaginateBy,
		PaginatePath:  cmp.Or(strings.TrimSpace(meta.PaginatePath), cfg.Sections.PaginatePath),
		SortBy:        cmp.Or(strings.TrimSpace(meta.SortBy), cfg.Sections.DefaultSortBy),
		Transparent:   meta.Transparent,
		GenerateFeeds: meta.GenerateFeeds,
		Extra:         meta.Extra,
	}, nil
}

// loadPage loads a regular page or, when bundle is true, an index.md page
// bundle whose sibling files become resources.
func loadPage(contentRoot, sourcePath string, bundle bool) (Page, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return Page{}, fmt.Errorf("read page file %q: %w", sourcePath, err)
	}
	doc, err := frontmatter.Parse(data)
	if err != nil {
		return Page{}, fmt.Errorf("parse page file %q: %w", sourcePath, err)
	}
	relativePath, err := filepath.Rel(contentRoot, sourcePath)
	if err != nil {
		return Page{}, fmt.Errorf("compute relative path for %q: %w", sourcePath, err)
	}
	relativePath = filepath.ToSlash(relativePath)
	meta := doc.Metadata

	page := Page{
		SourcePath:     sourcePath,
		RelativePath:   relativePath,
		RawFrontMatter: doc.RawFrontMatter,
		BodyMarkdown:   doc.BodyMarkdown,
		Title:          strings.TrimSpace(meta.Title),
		Slug:           meta.Slug,
		Path:           strings.TrimSpace(meta.Path),
		Description:    strings.TrimSpace(meta.Description),
		Date:           meta.Date,
		Updated:        meta.Updated,
		Draft:          meta.Draft,
		Weight:         meta.Weight,
		Template:       strings.TrimSpace(meta.Template),
		Extra:          meta.Extra,
		Aliases:        meta.Aliases,
	}

	dir := path.Dir(relativePath)
	if bundle {
		page.IsBundle = true
		page.BundleDir = dir
		dir = path.Dir(dir)
		if page.Slug == "" {
			page.Slug = DeriveSlug(filepath.Base(filepath.Dir(sourcePath)))
		}
		if page.Resources, err = collectBundleResources(filepath.Dir(sourcePath)); err != nil {
			return Page{}, fmt.Errorf("collect resources for bundle %q: %w", sourcePath, err)
		}
	} else if page.Slug == "" {
		base := filepath.Base(sourcePath)
		page.Slug = DeriveSlug(strings.TrimSuffix(base, filepath.Ext(base)))
	}
	if dir != "." {
		page.SectionPath = dir
	}
	return page, nil
}

func collectBundleResources(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var resources []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		name := entry.Name()
		if strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		resources = append(resources, name)
	}
	slices.Sort(resources)
	return resources, nil
}

func LoadPage(contentRoot, sourcePath string, cfg config.SiteConfig) (Page, error) {
	name := filepath.Base(sourcePath)
	if name == "index.md" {
		return loadPage(contentRoot, sourcePath, true)
	}
	if name == "_index.md" {
		return Page{}, fmt.Errorf("cannot load _index.md as a page (it is a section index)")
	}
	return loadPage(contentRoot, sourcePath, false)
}

func isContentRoot(path, contentRoot string) bool {
	return filepath.Dir(path) == contentRoot
}

var slugTransliteration = map[rune]string{
	'ϕ': "ph",
	'φ': "ph",
	'ℓ': "l",
}

func transliterateSlug(value string) string {
	var b strings.Builder
	for _, r := range value {
		if replacement, ok := slugTransliteration[r]; ok {
			b.WriteString(replacement)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func DeriveSlug(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	value = transliterateSlug(value)

	var b strings.Builder
	lastHyphen := false

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastHyphen = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(unicode.ToLower(r))
			lastHyphen = false
		case unicode.Is(unicode.Mn, r):
			// Combining marks, such as Arabic harakat, belong to the
			// letter before them, so they are dropped rather than
			// splitting the word.
		case r == '-' || r == '_' || r == ' ':
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		default:
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}
