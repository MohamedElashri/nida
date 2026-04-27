package site

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/markdown"
	"github.com/MohamedElashri/nida/internal/taxonomies"
)

type State struct {
	Pages    []content.Page
	Sections []content.Section
	Index    SiteIndex
}

type SiteIndex struct {
	Sections        []content.Section
	SectionLookup   map[string]content.Section
	RootSection     *content.Section
	AllPages        []content.Page
	Taxonomies      []taxonomies.Collection
	TaxonomyMap     taxonomies.TaxonomyMap
	RouteRegistry   map[string]string
	CanonicalLookup map[string]string
}

func Load(siteRoot string, cfg config.SiteConfig) (State, error) {
	pages, sections, err := content.Discover(siteRoot, cfg)
	if err != nil {
		return State{}, err
	}

	renderedPages, err := markdown.RenderPages(pages, cfg)
	if err != nil {
		return State{}, err
	}

	renderedSections, err := markdown.RenderSections(sections, cfg)
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
		SectionLookup:   map[string]content.Section{},
		RouteRegistry:   map[string]string{},
		CanonicalLookup: map[string]string{},
	}

	sortedPages := make([]content.Page, len(pages))
	copy(sortedPages, pages)
	slices.SortFunc(sortedPages, func(a, b content.Page) int {
		if !a.Date.Equal(b.Date) {
			if a.Date.After(b.Date) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Slug, b.Slug)
	})

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

	builtSections, sectionMap := buildSectionTree(sections, sortedPages, cfg)

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

	var err error
	index.Taxonomies, index.TaxonomyMap, err = taxonomies.BuildAll(cfg, sortedPages)
	if err != nil {
		return SiteIndex{}, nil, err
	}

	return index, sortedPages, nil
}

func ResolveSectionURL(section content.Section, cfg config.SiteConfig) (string, error) {
	route := "/" + section.SectionPath + "/"
	if section.SectionPath == "" {
		route = "/"
	}

	if p, ok := cfg.Permalinks[section.SectionPath]; ok && p != "" {
		route = p
		if !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		if !strings.HasSuffix(route, "/") {
			route += "/"
		}
	}

	return route, nil
}

func routePage(page content.Page, cfg config.SiteConfig) (string, error) {
	sectionPath := page.SectionPath

	var pattern string
	if p, ok := cfg.Permalinks[sectionPath]; ok && p != "" {
		pattern = p
	} else {
		pattern = "/{section}/{slug}/"
	}

	route := strings.ReplaceAll(pattern, "{slug}", page.Slug)
	route = strings.ReplaceAll(route, "{section}", sectionPath)
	route = strings.ReplaceAll(route, "{year}", page.Date.Format("2006"))
	route = strings.ReplaceAll(route, "{month}", page.Date.Format("01"))
	route = strings.ReplaceAll(route, "{day}", page.Date.Format("02"))

	if strings.Contains(route, "{") || strings.Contains(route, "}") {
		return "", fmt.Errorf("unsupported placeholder in permalink pattern %q for %q", pattern, page.RelativePath)
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	if !strings.HasSuffix(route, "/") {
		route += "/"
	}

	return route, nil
}

func buildSectionTree(sections []content.Section, allPages []content.Page, cfg config.SiteConfig) ([]content.Section, map[string]content.Section) {
	sectionMap := map[string]content.Section{}
	childrenMap := map[string][]content.Section{}

	for i := range sections {
		s := sections[i]
		sectionMap[s.SectionPath] = s
	}

	for path, s := range sectionMap {
		if path != "" {
			dir := filepath.ToSlash(filepath.Dir(path))
			if dir == "." {
				dir = ""
			}
			childrenMap[dir] = append(childrenMap[dir], s)
		}
	}

	for i := range sections {
		s := &sections[i]
		if children, ok := childrenMap[s.SectionPath]; ok {
			s.Sections = children
		}
		pagesInSection := filterPagesForSection(allPages, s.SectionPath)
		s.Pages = sortPagesBySection(pagesInSection, *s, cfg)
		sectionMap[s.SectionPath] = *s
	}

	var roots []content.Section
	rootSection := content.Section{}
	rootPages := filterPagesForSection(allPages, "")

	if existingRoot, ok := sectionMap[""]; ok {
		rootSection = existingRoot
		rootSection.Pages = sortPagesBySection(rootPages, rootSection, cfg)
		roots = append(roots, rootSection)
	} else if len(rootPages) > 0 {
		rootSection = content.Section{
			SectionPath:      "",
			Title:            "Home",
			Slug:             "",
			URL:              "/",
			PaginateBy:       0,
			PaginatePath:     "page",
			PaginateReversed: false,
			SortBy:           "date",
			Transparent:      false,
			GenerateFeeds:    false,
			Sections:         nil,
			Pages:           sortPagesBySection(rootPages, rootSection, cfg),
			Extra:            map[string]any{},
		}
		roots = append(roots, rootSection)
	}

	for i := range sections {
		s := sections[i]
		if s.SectionPath != "" && s.Transparent {
			roots = append(roots, s)
		}
	}

	return roots, sectionMap
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

func sortPagesBySection(pages []content.Page, section content.Section, cfg config.SiteConfig) []content.Page {
	if len(pages) == 0 {
		return pages
	}

	sortBy := section.SortBy
	if sortBy == "" {
		sortBy = "date"
	}

	switch sortBy {
	case "date":
		slices.SortFunc(pages, func(a, b content.Page) int {
			if !a.Date.Equal(b.Date) {
				if a.Date.After(b.Date) {
					return -1
				}
				return 1
			}
			return strings.Compare(a.Slug, b.Slug)
		})
	case "title":
		slices.SortFunc(pages, func(a, b content.Page) int {
			if cmp := strings.Compare(a.Title, b.Title); cmp != 0 {
				return cmp
			}
			return strings.Compare(a.Slug, b.Slug)
		})
	case "weight":
		slices.SortFunc(pages, func(a, b content.Page) int {
			if a.Weight != b.Weight {
				if a.Weight < b.Weight {
					return -1
				}
				return 1
			}
			return strings.Compare(a.Slug, b.Slug)
		})
	case "none":
	default:
		slices.SortFunc(pages, func(a, b content.Page) int {
			if !a.Date.Equal(b.Date) {
				if a.Date.After(b.Date) {
					return -1
				}
				return 1
			}
			return strings.Compare(a.Slug, b.Slug)
		})
	}

	if section.PaginateReversed {
		for i, j := 0, len(pages)-1; i < j; i, j = i+1, j-1 {
			pages[i], pages[j] = pages[j], pages[i]
		}
	}

	return pages
}
