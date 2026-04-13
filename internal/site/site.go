package site

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/markdown"
	"github.com/MohamedElashri/nida/internal/taxonomies"
)

type State struct {
	Items []content.Item
	Index SiteIndex
}

type Section struct {
	content.Item
	Pages []content.Item
}

type SiteIndex struct {
	Posts           []content.Item
	Pages           []content.Item
	RecentPosts     []content.Item
	Sections        []Section
	SectionLookup   map[string]Section
	RootSection     *Section
	TagMap          map[string][]content.Item
	CategoryMap     map[string][]content.Item
	Tags            taxonomies.Collection
	Categories      taxonomies.Collection
	RouteRegistry   map[string]string
	CanonicalLookup map[string]string
}

func Load(siteRoot string, cfg config.SiteConfig) (State, error) {
	items, err := content.Discover(siteRoot, cfg)
	if err != nil {
		return State{}, err
	}

	renderedItems, err := markdown.RenderItems(items, cfg)
	if err != nil {
		return State{}, err
	}

	index, err := BuildIndex(renderedItems, cfg)
	if err != nil {
		return State{}, err
	}

	return State{
		Items: renderedItems,
		Index: index,
	}, nil
}

func BuildIndex(items []content.Item, cfg config.SiteConfig) (SiteIndex, error) {
	index := SiteIndex{
		TagMap:          map[string][]content.Item{},
		CategoryMap:     map[string][]content.Item{},
		SectionLookup:   map[string]Section{},
		RouteRegistry:   map[string]string{},
		CanonicalLookup: map[string]string{},
	}
	var err error
	sectionPages := map[string][]content.Item{}
	sectionItems := map[string]content.Item{}

	for _, item := range items {
		if item.Draft && !cfg.Drafts {
			continue
		}

		routed, canonical, err := routeItem(item, cfg)
		if err != nil {
			return SiteIndex{}, err
		}

		if existing, exists := index.RouteRegistry[routed.URL]; exists {
			return SiteIndex{}, fmt.Errorf("route conflict for %q between %q and %q", routed.URL, existing, routed.RelativePath)
		}

		index.RouteRegistry[routed.URL] = routed.RelativePath
		index.CanonicalLookup[routed.URL] = canonical

		switch routed.Type {
		case content.TypePost:
			index.Posts = append(index.Posts, routed)
			sectionPages[routed.SectionPath] = append(sectionPages[routed.SectionPath], routed)
		case content.TypePage:
			index.Pages = append(index.Pages, routed)
		case content.TypeSection:
			copy := routed
			sectionItems[routed.SectionPath] = copy
		default:
			return SiteIndex{}, fmt.Errorf("unsupported content type %q for %q", routed.Type, routed.RelativePath)
		}

		for _, tag := range routed.Tags {
			index.TagMap[tag] = append(index.TagMap[tag], routed)
		}
		for _, category := range routed.Categories {
			index.CategoryMap[category] = append(index.CategoryMap[category], routed)
		}
	}

	sortItems(index.Posts)
	sortItems(index.Pages)
	for sectionPath := range sectionPages {
		sortItems(sectionPages[sectionPath])
	}
	for key := range index.TagMap {
		sortItems(index.TagMap[key])
	}
	for key := range index.CategoryMap {
		sortItems(index.CategoryMap[key])
	}

	index.RecentPosts = append(index.RecentPosts, index.Posts...)
	index.Sections = buildSections(cfg, sectionItems, sectionPages)
	for _, section := range index.Sections {
		index.SectionLookup[section.SectionPath] = section
		if section.SectionPath == "" {
			copy := section
			index.RootSection = &copy
		}
	}

	index.Tags, err = taxonomies.Build("tags", cfg.Taxonomies.Tags, cfg.Permalinks.Tags, "/tags/", index.TagMap, cfg)
	if err != nil {
		return SiteIndex{}, err
	}
	index.Categories, err = taxonomies.Build("categories", cfg.Taxonomies.Categories, cfg.Permalinks.Categories, "/categories/", index.CategoryMap, cfg)
	if err != nil {
		return SiteIndex{}, err
	}

	return index, nil
}

func ResolvePermalink(item content.Item, cfg config.SiteConfig) (string, error) {
	if item.Type != content.TypeSection && item.Slug == "" {
		return "", fmt.Errorf("resolve permalink for %q: slug is required", item.RelativePath)
	}

	pattern, err := patternFor(item, cfg)
	if err != nil {
		return "", err
	}

	route := strings.ReplaceAll(pattern, "{slug}", item.Slug)
	route = strings.ReplaceAll(route, "{section}", item.SectionPath)
	if strings.Contains(route, "{") || strings.Contains(route, "}") {
		return "", fmt.Errorf("resolve permalink for %q: unsupported placeholder in %q", item.RelativePath, pattern)
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	if !strings.HasSuffix(route, "/") {
		route += "/"
	}

	return route, nil
}

func routeItem(item content.Item, cfg config.SiteConfig) (content.Item, string, error) {
	route, err := ResolvePermalink(item, cfg)
	if err != nil {
		return content.Item{}, "", err
	}

	canonical, err := canonicalURL(cfg.BaseURL, route)
	if err != nil {
		return content.Item{}, "", fmt.Errorf("build canonical URL for %q: %w", item.RelativePath, err)
	}

	item.URL = route
	return item, canonical, nil
}

func patternFor(item content.Item, cfg config.SiteConfig) (string, error) {
	switch item.Type {
	case content.TypePost:
		if item.SectionPath != "" && item.SectionPath != cfg.PostsDir {
			return "/{section}/{slug}/", nil
		}
		return cfg.Permalinks.Posts, nil
	case content.TypePage:
		return cfg.Permalinks.Pages, nil
	case content.TypeSection:
		if item.SectionPath == "" {
			return "/", nil
		}
		return "/{section}/", nil
	default:
		return "", fmt.Errorf("resolve permalink: unsupported content type %q", item.Type)
	}
}

func canonicalURL(baseURL, route string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL %q: %w", baseURL, err)
	}

	base.Path = path.Join(base.Path, route)
	if strings.HasSuffix(route, "/") && !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	return base.String(), nil
}

func sortItems(items []content.Item) {
	slices.SortFunc(items, func(a, b content.Item) int {
		if !a.Date.Equal(b.Date) {
			if a.Date.After(b.Date) {
				return -1
			}
			return 1
		}
		if cmp := strings.Compare(a.Slug, b.Slug); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.RelativePath, b.RelativePath)
	})
}

func buildSections(cfg config.SiteConfig, sectionItems map[string]content.Item, sectionPages map[string][]content.Item) []Section {
	keys := map[string]struct{}{}
	for key := range sectionItems {
		keys[key] = struct{}{}
	}
	for key := range sectionPages {
		keys[key] = struct{}{}
	}
	keys[cfg.PostsDir] = struct{}{}

	sections := make([]Section, 0, len(keys))
	for key := range keys {
		item, ok := sectionItems[key]
		if !ok {
			item = syntheticSection(cfg, key)
		}
		item.SectionPath = key
		sections = append(sections, Section{
			Item:  item,
			Pages: append([]content.Item(nil), sectionPages[key]...),
		})
	}

	slices.SortFunc(sections, func(a, b Section) int {
		if a.SectionPath == "" && b.SectionPath != "" {
			return -1
		}
		if a.SectionPath != "" && b.SectionPath == "" {
			return 1
		}
		return strings.Compare(a.SectionPath, b.SectionPath)
	})

	return sections
}

func syntheticSection(cfg config.SiteConfig, sectionPath string) content.Item {
	title := "Home"
	slug := ""
	url := "/"
	if sectionPath != "" {
		base := path.Base(sectionPath)
		slug = content.DeriveSlug(base)
		title = strings.Title(strings.ReplaceAll(base, "-", " "))
		url = "/" + strings.Trim(sectionPath, "/") + "/"
	}

	item := content.Item{
		Type:        content.TypeSection,
		SectionPath: sectionPath,
		Title:       title,
		Slug:        slug,
		PaginateBy:  cfg.Paginate,
		URL:         url,
	}
	return item
}
