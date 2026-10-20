package site

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/diagnostics"
	"github.com/MohamedElashri/nida/internal/frontmatter"
	"github.com/MohamedElashri/nida/internal/i18n"
	"github.com/MohamedElashri/nida/internal/markdown"
	routes "github.com/MohamedElashri/nida/internal/route"
	"github.com/MohamedElashri/nida/internal/shortcodes"
	"github.com/MohamedElashri/nida/internal/taxonomies"
)

type State struct {
	Pages    []content.Page
	Sections []content.Section
	Index    SiteIndex
}

type SiteIndex struct {
	Sections      []content.Section
	SectionLookup map[string]content.Section
	RootSection   *content.Section
	AllPages      []content.Page
	Taxonomies    []taxonomies.Collection
	TaxonomyMap   taxonomies.TaxonomyMap
	RouteRegistry map[string]string
}

func Load(siteRoot string, cfg config.SiteConfig) (State, error) {
	pages, sections, err := content.Discover(siteRoot, cfg)
	if err != nil {
		return State{}, err
	}
	if cfg.UpdatedFromGit {
		if contentRoot, err := filepath.Abs(filepath.Join(siteRoot, cfg.ContentDir)); err == nil {
			content.ApplyGitDates(contentRoot, pages)
		}
	}

	pathLookup := BuildPathLookup(pages, sections, cfg)
	if cfg.Diagnostics.Enabled {
		if err := diagnostics.Check(siteRoot, cfg, pages, sections, pathLookup); err != nil {
			return State{}, err
		}
	}

	codes, err := shortcodes.Load(siteRoot, cfg)
	if err != nil {
		return State{}, err
	}
	tr, err := i18n.Load(siteRoot, cfg)
	if err != nil {
		return State{}, err
	}
	env := markdown.Env{Lookup: pathLookup, Shortcodes: codes, Images: NewImages(siteRoot, cfg), Translate: tr.Lookup}

	renderedPages, err := markdown.RenderPages(pages, cfg, env)
	if err != nil {
		return State{}, err
	}

	renderedSections, err := markdown.RenderSections(sections, cfg, env)
	if err != nil {
		return State{}, err
	}

	index, sortedPages, err := BuildIndex(renderedPages, renderedSections, cfg)
	if err != nil {
		return State{}, err
	}

	return State{
		Pages:    sortedPages,
		Sections: renderedSections,
		Index:    index,
	}, nil
}

func BuildIndex(pages []content.Page, sections []content.Section, cfg config.SiteConfig) (SiteIndex, []content.Page, error) {
	index := SiteIndex{
		SectionLookup: map[string]content.Section{},
		RouteRegistry: map[string]string{},
	}

	sortedPages := make([]content.Page, len(pages))
	copy(sortedPages, pages)
	slices.SortFunc(sortedPages, content.CompareByDateDesc)

	now := time.Now()
	filtered := make([]content.Page, 0, len(sortedPages))
	for _, p := range sortedPages {
		if published(p, cfg, now) {
			filtered = append(filtered, p)
		}
	}
	sortedPages = filtered

	for i := range sortedPages {
		routed, err := routePage(sortedPages[i], cfg)
		if err != nil {
			return SiteIndex{}, nil, err
		}
		sortedPages[i].URL = routed

		if existing, exists := index.RouteRegistry[routed]; exists {
			return SiteIndex{}, nil, fmt.Errorf("route conflict for %q between %q and %q", routed, existing, sortedPages[i].RelativePath)
		}
		index.RouteRegistry[routed] = sortedPages[i].RelativePath
	}

	sections = slices.Clone(sections)
	for i := range sections {
		if url, err := ResolveSectionURL(sections[i]); err == nil {
			sections[i].URL = url
		}
	}
	builtSections, sectionMap := buildSectionTree(sections, sortedPages)

	for path, s := range sectionMap {
		index.SectionLookup[path] = s
	}

	slices.SortFunc(builtSections, func(a, b content.Section) int {
		if a.SectionPath == "" && b.SectionPath != "" {
			return -1
		}
		if a.SectionPath != "" && b.SectionPath == "" {
			return 1
		}
		return strings.Compare(a.SectionPath, b.SectionPath)
	})
	index.Sections = builtSections

	for i := range index.Sections {
		s := &index.Sections[i]
		if s.SectionPath == "" {
			index.RootSection = s
			break
		}
	}

	index.AllPages = sortedPages

	// Prev/next follow each section's page order. Section page slices share
	// backing arrays with their copies, so setting them once updates all.
	type links struct{ prevURL, prevTitle, nextURL, nextTitle string }
	neighbors := map[string]links{}
	for _, section := range index.SectionLookup {
		pages := section.Pages
		for j := range pages {
			var l links
			if j > 0 {
				l.nextURL, l.nextTitle = pages[j-1].URL, pages[j-1].Title
			}
			if j < len(pages)-1 {
				l.prevURL, l.prevTitle = pages[j+1].URL, pages[j+1].Title
			}
			pages[j].PrevURL, pages[j].PrevTitle, pages[j].NextURL, pages[j].NextTitle = l.prevURL, l.prevTitle, l.nextURL, l.nextTitle
			neighbors[pages[j].RelativePath] = l
		}
	}
	for i := range sortedPages {
		l := neighbors[sortedPages[i].RelativePath]
		sortedPages[i].PrevURL, sortedPages[i].PrevTitle, sortedPages[i].NextURL, sortedPages[i].NextTitle = l.prevURL, l.prevTitle, l.nextURL, l.nextTitle
	}

	var err error
	index.Taxonomies, index.TaxonomyMap, err = taxonomies.BuildAll(cfg, sortedPages)
	if err != nil {
		return SiteIndex{}, nil, err
	}

	return index, sortedPages, nil
}

// ResolveSectionURL returns /<section>/; permalinks apply only to pages.
func ResolveSectionURL(section content.Section) (string, error) {
	route := "/" + section.SectionPath + "/"
	if section.SectionPath == "" {
		route = "/"
	}
	return routes.Normalize("section route", route)
}

// BuildPathLookup maps content paths to routes for @/ links.
func BuildPathLookup(pages []content.Page, sections []content.Section, cfg config.SiteConfig) markdown.PathLookup {
	lookup := make(markdown.PathLookup)

	now := time.Now()
	for _, page := range pages {
		// A link to a page the build leaves out stays unresolved, so it
		// is reported instead of pointing at a missing page.
		if !published(page, cfg, now) {
			continue
		}
		url, err := routePage(page, cfg)
		if err != nil {
			continue
		}
		lookup[page.RelativePath] = url
		withoutExt := strings.TrimSuffix(page.RelativePath, ".md")
		if withoutExt != page.RelativePath {
			lookup[withoutExt] = url
		}
	}

	for _, section := range sections {
		url, err := ResolveSectionURL(section)
		if err != nil {
			continue
		}
		if section.RelativePath != "" {
			lookup[section.RelativePath] = url
			withoutExt := strings.TrimSuffix(section.RelativePath, ".md")
			if withoutExt != section.RelativePath {
				lookup[withoutExt] = url
			}
		}
	}

	return lookup
}

// published reports whether the build includes a page: drafts need
// --drafts and future-dated pages need --future.
func published(page content.Page, cfg config.SiteConfig, now time.Time) bool {
	return (!page.Draft || cfg.Drafts) && (cfg.Future || !page.Date.After(now))
}

func routePage(page content.Page, cfg config.SiteConfig) (string, error) {
	if page.Path != "" {
		route, err := routes.Page("path", page.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path for %q: %w", page.RelativePath, err)
		}
		return route, nil
	}
	sectionPath := page.SectionPath
	if err := routes.ValidateComponent("slug", page.Slug); err != nil {
		return "", fmt.Errorf("invalid slug for %q: %w", page.RelativePath, err)
	}

	var pattern string
	if p, ok := cfg.Permalinks[sectionPath]; ok && p != "" {
		pattern = p
	} else {
		pattern = "/{section}/{slug}/"
	}

	route := strings.ReplaceAll(pattern, "{slug}", page.Slug)
	if sectionPath == "" {
		route = strings.ReplaceAll(route, "{section}/", "")
		route = strings.ReplaceAll(route, "{section}", "")
	} else {
		route = strings.ReplaceAll(route, "{section}", sectionPath)
	}
	route = strings.ReplaceAll(route, "{year}", page.Date.Format("2006"))
	route = strings.ReplaceAll(route, "{month}", page.Date.Format("01"))
	route = strings.ReplaceAll(route, "{day}", page.Date.Format("02"))

	for _, tc := range cfg.Taxonomies {
		placeholder := "{" + tc.Name + "}"
		if strings.Contains(route, placeholder) {
			taxValue := extractTaxonomyValue(page.Extra, tc.Name)
			route = strings.ReplaceAll(route, placeholder, taxValue)
		}
	}

	if strings.Contains(route, "{") || strings.Contains(route, "}") {
		return "", fmt.Errorf("unsupported placeholder in permalink pattern %q for %q", pattern, page.RelativePath)
	}
	return routes.Page("page route", route)
}

func extractTaxonomyValue(extra map[string]any, name string) string {
	if values := frontmatter.StringList(extra[name]); len(values) > 0 {
		return content.DeriveSlug(values[0])
	}
	return ""
}

func buildSectionTree(sections []content.Section, allPages []content.Page) ([]content.Section, map[string]content.Section) {
	ptrMap := make(map[string]*content.Section, len(sections))

	for i := range sections {
		s := &sections[i]
		pagesInSection := filterPagesForSection(allPages, s.SectionPath)
		s.Pages = sortPagesBySection(pagesInSection, *s)
		ptrMap[s.SectionPath] = s
	}

	sortedPaths := make([]string, 0, len(ptrMap))
	for path := range ptrMap {
		sortedPaths = append(sortedPaths, path)
	}
	slices.SortFunc(sortedPaths, func(a, b string) int {
		depthA := strings.Count(a, "/")
		depthB := strings.Count(b, "/")
		if depthA != depthB {
			return depthB - depthA
		}
		return strings.Compare(a, b)
	})

	for _, path := range sortedPaths {
		if path == "" {
			continue
		}
		s := ptrMap[path]
		dir := filepath.ToSlash(filepath.Dir(path))
		if dir == "." {
			dir = ""
		}
		if parent, ok := ptrMap[dir]; ok {
			sortSubsections(s.Sections)
			parent.Sections = append(parent.Sections, *s)
		}
	}
	for _, s := range ptrMap {
		sortSubsections(s.Sections)
	}

	sectionMap := map[string]content.Section{}
	for path, s := range ptrMap {
		sectionMap[path] = *s
	}

	var roots []content.Section
	rootSection := content.Section{}
	rootPages := filterPagesForSection(allPages, "")

	if existingRoot, ok := ptrMap[""]; ok {
		rootSection = *existingRoot
		roots = append(roots, rootSection)
	} else if len(rootPages) > 0 {
		var topLevelSections []content.Section
		for _, path := range sortedPaths {
			dir := filepath.ToSlash(filepath.Dir(path))
			if path != "" && (dir == "." || dir == "") {
				topLevelSections = append(topLevelSections, *ptrMap[path])
			}
		}
		sortSubsections(topLevelSections)
		rootSection = content.Section{
			SectionPath:   "",
			Title:         "Home",
			Slug:          "",
			URL:           "/",
			PaginateBy:    0,
			PaginatePath:  "page",
			SortBy:        "date",
			Transparent:   false,
			GenerateFeeds: false,
			Sections:      topLevelSections,
			Pages:         sortPagesBySection(rootPages, rootSection),
			Extra:         map[string]any{},
		}
		roots = append(roots, rootSection)
		sectionMap[""] = rootSection
	}

	for _, path := range sortedPaths {
		s := ptrMap[path]
		if s.SectionPath != "" && s.Transparent {
			roots = append(roots, *s)
		}
	}

	return roots, sectionMap
}

// sortSubsections orders child sections by weight, then title, so navigation
// can follow the order authors set in _index.md.
func sortSubsections(sections []content.Section) {
	slices.SortStableFunc(sections, func(a, b content.Section) int {
		if a.Weight != b.Weight {
			return a.Weight - b.Weight
		}
		if c := strings.Compare(a.Title, b.Title); c != 0 {
			return c
		}
		return strings.Compare(a.SectionPath, b.SectionPath)
	})
}

func filterPagesForSection(pages []content.Page, sectionPath string) []content.Page {
	var filtered []content.Page
	for _, page := range pages {
		if page.SectionPath == sectionPath {
			filtered = append(filtered, page)
		}
	}
	return filtered
}

func sortPagesBySection(pages []content.Page, section content.Section) []content.Page {
	if len(pages) == 0 {
		return pages
	}

	sortBy := section.SortBy
	if sortBy == "" {
		sortBy = "date"
	}

	switch sortBy {
	case "title":
		slices.SortFunc(pages, func(a, b content.Page) int {
			if c := strings.Compare(a.Title, b.Title); c != 0 {
				return c
			}
			return strings.Compare(a.Slug, b.Slug)
		})
	case "weight":
		slices.SortFunc(pages, func(a, b content.Page) int {
			if a.Weight != b.Weight {
				return a.Weight - b.Weight
			}
			return strings.Compare(a.Slug, b.Slug)
		})
	case "none":
	default:
		slices.SortFunc(pages, content.CompareByDateDesc)
	}

	return pages
}

// GetPage finds a published page by content path ("posts/hello.md", with or
// without ".md" or "@/"), or returns nil.
func (index SiteIndex) GetPage(contentPath string) *content.Page {
	contentPath = normalizeContentPath(contentPath)
	for i := range index.AllPages {
		page := &index.AllPages[i]
		rel := page.RelativePath
		if rel == contentPath || strings.TrimSuffix(rel, ".md") == contentPath || strings.TrimSuffix(rel, "/index.md") == contentPath {
			return page
		}
	}
	return nil
}

// GetSection finds a section by its path, such as "posts" or
// "posts/_index.md". An empty path or "_index.md" returns the root section.
func (index SiteIndex) GetSection(sectionPath string) *content.Section {
	sectionPath = normalizeContentPath(sectionPath)
	sectionPath = strings.TrimSuffix(strings.TrimSuffix(sectionPath, "_index.md"), "/")
	section, ok := index.SectionLookup[sectionPath]
	if !ok {
		return nil
	}
	return &section
}

func normalizeContentPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "@/")
	return strings.TrimPrefix(value, "/")
}

// NewImages returns the image resolver Markdown rendering uses for image
// dimensions and responsive variants.
func NewImages(siteRoot string, cfg config.SiteConfig) markdown.Images {
	if r := newImageResolver(siteRoot, cfg); r != nil {
		return r
	}
	return nil
}

// Ancestors returns the sections from the root down to sectionPath,
// inclusive, for breadcrumbs. Sections without an entry are skipped.
func (index SiteIndex) Ancestors(sectionPath string) []content.Section {
	sectionPath = strings.Trim(sectionPath, "/")
	paths := []string{""}
	if sectionPath != "" {
		parts := strings.Split(sectionPath, "/")
		for i := range parts {
			paths = append(paths, strings.Join(parts[:i+1], "/"))
		}
	}
	var out []content.Section
	for _, path := range paths {
		if section, ok := index.SectionLookup[path]; ok {
			out = append(out, section)
		}
	}
	return out
}
