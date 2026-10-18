package taxonomies

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/frontmatter"
	"github.com/MohamedElashri/nida/internal/route"
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

// BuildAll groups pages by term for each configured taxonomy.
func BuildAll(cfg config.SiteConfig, pages []content.Page) ([]Collection, TaxonomyMap, error) {
	taxMap := make(TaxonomyMap)
	var collections []Collection
	for _, tc := range cfg.Taxonomies {
		termMap := map[string][]content.Page{}
		for _, page := range pages {
			for _, term := range frontmatter.StringList(page.Extra[tc.Name]) {
				termMap[term] = append(termMap[term], page)
			}
		}
		taxMap[tc.Name] = termMap
		collection, err := buildCollection(tc, termMap, cfg)
		if err != nil {
			return nil, nil, err
		}
		collections = append(collections, collection)
	}
	return collections, taxMap, nil
}

func buildCollection(tc config.TaxonomyConfig, termMap map[string][]content.Page, cfg config.SiteConfig) (Collection, error) {
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
		CanonicalURL: route.CanonicalURL(cfg.BaseURL, rootURL),
		PaginateBy:   tc.PaginateBy,
		PaginatePath: cmp.Or(strings.TrimSpace(tc.PaginatePath), "page"),
		Feed:         tc.Feed,
		Render:       tc.Render,
		Terms:        make([]Term, 0, len(termMap)),
	}

	for termName, termItems := range termMap {
		slug := content.DeriveSlug(termName)
		if slug == "" {
			return Collection{}, fmt.Errorf("build %s taxonomy: empty slug for term %q", tc.Name, termName)
		}

		termURL, err := expandPattern(pattern, slug)
		if err != nil {
			return Collection{}, fmt.Errorf("build %s taxonomy term %q: %w", tc.Name, termName, err)
		}

		termCopy := append([]content.Page(nil), termItems...)
		sortPages(termCopy)

		collection.Terms = append(collection.Terms, Term{
			Name:         termName,
			Slug:         slug,
			URL:          termURL,
			CanonicalURL: route.CanonicalURL(cfg.BaseURL, termURL),
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

// normalizeRoute validates a taxonomy URL pattern with {slug} removed.
func normalizeRoute(pattern string) (string, error) {
	cleaned := strings.ReplaceAll(pattern, "{slug}", "")
	if strings.ContainsAny(cleaned, "{}") {
		return "", fmt.Errorf("unsupported placeholder in %q", pattern)
	}
	normalized, err := route.Normalize("taxonomy route", cleaned)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(normalized, "/") {
		normalized += "/"
	}
	return normalized, nil
}

func expandPattern(pattern, slug string) (string, error) {
	if strings.Contains(pattern, "{slug}") {
		return normalizeRoute(strings.ReplaceAll(pattern, "{slug}", slug))
	}
	return normalizeRoute(strings.TrimSuffix(pattern, "/") + "/" + slug)
}

func sortPages(items []content.Page) {
	slices.SortFunc(items, func(a, b content.Page) int {
		if c := content.CompareByDateDesc(a, b); c != 0 {
			return c
		}
		return strings.Compare(a.RelativePath, b.RelativePath)
	})
}
