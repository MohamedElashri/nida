package content

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/frontmatter"
)

func Discover(siteRoot string, cfg config.SiteConfig) ([]Page, []Section, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	contentRoot := filepath.Join(absSiteRoot, cfg.ContentDir)
	return discoverAll(contentRoot, cfg)
}

func discoverAll(contentRoot string, cfg config.SiteConfig) ([]Page, []Section, error) {
	type fileEntry struct {
		path    string
		isIndex bool
	}

	dirSections := map[string]bool{}
	potentialPages := []fileEntry{}

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

		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}

		name := filepath.Base(path)
		if name == "_index.md" {
			dir := filepath.ToSlash(filepath.Dir(path))
			if dir != contentRoot {
				rel, _ := filepath.Rel(contentRoot, dir)
				dirSections[filepath.ToSlash(rel)] = true
			}
			potentialPages = append(potentialPages, fileEntry{path: path, isIndex: true})
		} else {
			potentialPages = append(potentialPages, fileEntry{path: path, isIndex: false})
		}

		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("discover content under %q: %w", contentRoot, err)
	}

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
		} else {
			page, err := loadPage(contentRoot, entry.path, cfg)
			if err != nil {
				return nil, nil, err
			}
			pages = append(pages, page)
		}
	}

	sections = synthesizeImplicitSections(contentRoot, pages, sections)

	slices.SortFunc(pages, func(a, b Page) int {
		return strings.Compare(a.RelativePath, b.RelativePath)
	})
	slices.SortFunc(sections, func(a, b Section) int {
		return strings.Compare(a.SectionPath, b.SectionPath)
	})

	return pages, sections, nil
}

func synthesizeImplicitSections(contentRoot string, pages []Page, sections []Section) []Section {
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
			SourcePath:        "",
			RelativePath:      filepath.Join(contentRoot, dirPath, "_index.md"),
			SectionPath:       dirPath,
			Title:             strings.Title(strings.ReplaceAll(base, "-", " ")),
			Slug:              DeriveSlug(base),
			URL:               "/" + dirPath + "/",
			PaginateBy:        0,
			PaginatePath:      "page",
			PaginateReversed:  false,
			SortBy:            "date",
			Transparent:       false,
			GenerateFeeds:     false,
			Sections:          nil,
			Pages:             nil,
			Extra:             map[string]any{},
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
		SourcePath:        sourcePath,
		RelativePath:      filepath.ToSlash(relativePath),
		SectionPath:       sectionPath,
		BodyMarkdown:      doc.BodyMarkdown,
		Title:             strings.TrimSpace(meta.Title),
		Description:       strings.TrimSpace(meta.Description),
		Slug:              slug,
		Draft:             meta.Draft,
		Template:          strings.TrimSpace(meta.Template),
		PageTemplate:      strings.TrimSpace(meta.PageTemplate),
		PaginateBy:        meta.PaginateBy,
		PaginatePath:      strings.TrimSpace(defaultString(meta.PaginatePath, "page")),
		PaginateReversed:  false,
		SortBy:            strings.TrimSpace(defaultString(meta.SortBy, "date")),
		Transparent:       meta.Transparent,
		GenerateFeeds:     meta.GenerateFeeds,
		Extra:             meta.Extra,
	}, nil
}

func loadPage(contentRoot, sourcePath string, cfg config.SiteConfig) (Page, error) {
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

	meta := doc.Metadata
	slug := meta.Slug
	if slug == "" {
		slug = DeriveSlug(filepath.Base(sourcePath))
	}

	sectionPath := filepath.ToSlash(filepath.Dir(filepath.ToSlash(relativePath)))
	if sectionPath == "." {
		sectionPath = ""
	}

	return Page{
		SourcePath:     sourcePath,
		RelativePath:   filepath.ToSlash(relativePath),
		SectionPath:    sectionPath,
		RawFrontMatter: doc.RawFrontMatter,
		BodyMarkdown:   doc.BodyMarkdown,
		Title:          strings.TrimSpace(meta.Title),
		Slug:           slug,
		Description:    strings.TrimSpace(meta.Description),
		Date:           meta.Date,
		Updated:        meta.Updated,
		Draft:          meta.Draft,
		Weight:         meta.Weight,
		Template:       strings.TrimSpace(meta.Template),
		Extra:          meta.Extra,
	}, nil
}

func LoadPage(contentRoot, sourcePath string, cfg config.SiteConfig) (Page, error) {
	return loadPage(contentRoot, sourcePath, cfg)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func DeriveSlug(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, filepath.Ext(value))
	value = strings.ToLower(value)

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
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
			b.WriteRune(unicode.ToLower(r))
			lastHyphen = false
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
