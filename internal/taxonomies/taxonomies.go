package taxonomies

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

type Term struct {
	Name         string
	Slug         string
	URL          string
	CanonicalURL string
	Items        []content.Page
}

type Collection struct {
	Name         string
	URL          string
	CanonicalURL string
	PaginateBy   int
	PaginatePath string
	Feed         bool
	Render       bool
	Terms        []Term
}

type TaxonomyMap map[string]map[string][]content.Page

func BuildAll(cfg config.SiteConfig, pages []content.Page) ([]Collection, TaxonomyMap, error) {
	taxMap := make(TaxonomyMap)
	for _, page := range pages {
		if page.Draft && !cfg.Drafts {
			continue
		}
		for key, value := range page.Extra {
			strList, ok := tryStringList(value)
			if !ok {
				continue
			}
			for _, termName := range strList {
				termName = strings.TrimSpace(termName)
				if termName == "" {
					continue
				}
				if _, ok := taxMap[key]; !ok {
					taxMap[key] = map[string][]content.Page{}
				}
				taxMap[key][termName] = append(taxMap[key][termName], page)
			}
		}
	}

	var collections []Collection
	for _, tc := range cfg.Taxonomies {
		termMap, ok := taxMap[tc.Name]
		if !ok {
			termMap = map[string][]content.Page{}
		}
		collection, err := buildCollection(tc, termMap, cfg)
		if err != nil {
			return nil, nil, err
		}
		collections = append(collections, collection)
	}

	return collections, taxMap, nil
}

func tryStringList(value any) ([]string, bool) {
	switch v := value.(type) {
	case []string:
		return v, true
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out, len(out) > 0
	}
	return nil, false
}

func buildCollection(tc config.TaxonomyConfig, termMap map[string][]content.Page, cfg config.SiteConfig) (Collection, error) {
	rootURL := "/" + strings.ToLower(strings.TrimSpace(tc.Name)) + "/"
	pattern := tc.Name

	if p, ok := cfg.Permalinks[tc.Name]; ok && p != "" {
		pattern = p
	}

	rootURL, err := normalizeRoute(pattern)
	if err != nil {
		return Collection{}, fmt.Errorf("build %s taxonomy root: %w", tc.Name, err)
	}

	collection := Collection{
		Name:         tc.Name,
		URL:          rootURL,
		CanonicalURL: canonicalURL(cfg.BaseURL, rootURL),
		PaginateBy:   tc.PaginateBy,
		PaginatePath: strings.TrimSpace(defaultString(tc.PaginatePath, "page")),
		Feed:         tc.Feed,
		Render:       tc.Render,
		Terms:        make([]Term, 0, len(termMap)),
	}

	for termName, termItems := range termMap {
		slug := content.DeriveSlug(termName)
		if slug == "" {
			return Collection{}, fmt.Errorf("build %s taxonomy: empty slug for term %q", tc.Name, termName)
		}

		route, err := expandPattern(rootURL, slug)
		if err != nil {
			return Collection{}, fmt.Errorf("build %s taxonomy term %q: %w", tc.Name, termName, err)
		}

		termCopy := append([]content.Page(nil), termItems...)
		sortPages(termCopy)

		collection.Terms = append(collection.Terms, Term{
			Name:         termName,
			Slug:         slug,
			URL:          route,
			CanonicalURL: canonicalURL(cfg.BaseURL, route),
			Items:        termCopy,
		})
	}

	slices.SortFunc(collection.Terms, func(a, b Term) int {
		if cmp := strings.Compare(a.Slug, b.Slug); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Name, b.Name)
	})

	return collection, nil
}

func normalizeRoute(pattern string) (string, error) {
	cleaned := strings.ReplaceAll(pattern, "{slug}", "")
	if strings.Contains(cleaned, "{") || strings.Contains(cleaned, "}") {
		return "", fmt.Errorf("unsupported placeholder in %q", pattern)
	}
	cleaned = strings.Trim(cleaned, "/")
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if !strings.HasSuffix(cleaned, "/") {
		cleaned += "/"
	}
	return cleaned, nil
}

func expandPattern(pattern, slug string) (string, error) {
	var route string
	if strings.Contains(pattern, "{slug}") {
		route = strings.ReplaceAll(pattern, "{slug}", slug)
	} else {
		route = strings.TrimSuffix(pattern, "/") + "/" + slug
	}
	return normalizeRoute(route)
}

func canonicalURL(baseURL, route string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return route
	}
	base.Path = path.Join(base.Path, route)
	if strings.HasSuffix(route, "/") && !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	return base.String()
}

func sortPages(items []content.Page) {
	slices.SortFunc(items, func(a, b content.Page) int {
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

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
