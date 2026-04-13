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
	Items        []content.Item
}

type Collection struct {
	Name         string
	URL          string
	CanonicalURL string
	Terms        []Term
}

func Build(name string, enabled bool, pattern string, rootPattern string, items map[string][]content.Item, cfg config.SiteConfig) (Collection, error) {
	if !enabled {
		return Collection{}, nil
	}

	rootURL, err := normalizeRoute(rootPattern)
	if err != nil {
		return Collection{}, fmt.Errorf("build %s taxonomy root: %w", name, err)
	}

	collection := Collection{
		Name:         name,
		URL:          rootURL,
		CanonicalURL: canonicalURL(cfg.BaseURL, rootURL),
		Terms:        make([]Term, 0, len(items)),
	}

	for termName, termItems := range items {
		slug := content.DeriveSlug(termName)
		if slug == "" {
			return Collection{}, fmt.Errorf("build %s taxonomy: empty slug for term %q", name, termName)
		}

		route, err := expandPattern(pattern, slug)
		if err != nil {
			return Collection{}, fmt.Errorf("build %s taxonomy term %q: %w", name, termName, err)
		}

		termCopy := append([]content.Item(nil), termItems...)
		sortItems(termCopy)

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
	if strings.Contains(pattern, "{") || strings.Contains(pattern, "}") {
		return "", fmt.Errorf("unsupported placeholder in %q", pattern)
	}
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	return pattern, nil
}

func expandPattern(pattern, slug string) (string, error) {
	route := strings.ReplaceAll(pattern, "{slug}", slug)
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
