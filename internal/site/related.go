package site

import (
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/content"
)

// Related returns up to limit pages ranked by shared taxonomy terms, then
// by date.
func (index SiteIndex) Related(page content.Page, limit int) []content.Page {
	own := map[string]bool{}
	for _, collection := range index.Taxonomies {
		for _, term := range termValues(page.Extra[collection.Name]) {
			own[collection.Name+"\x00"+strings.ToLower(term)] = true
		}
	}
	if len(own) == 0 {
		return nil
	}

	type scored struct {
		page  content.Page
		score int
	}
	var candidates []scored
	for _, other := range index.AllPages {
		if other.RelativePath == page.RelativePath {
			continue
		}
		score := 0
		for _, collection := range index.Taxonomies {
			for _, term := range termValues(other.Extra[collection.Name]) {
				if own[collection.Name+"\x00"+strings.ToLower(term)] {
					score++
				}
			}
		}
		if score > 0 {
			candidates = append(candidates, scored{other, score})
		}
	}

	slices.SortStableFunc(candidates, func(a, b scored) int {
		if a.score != b.score {
			return b.score - a.score
		}
		return b.page.Date.Compare(a.page.Date)
	})
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}
	out := make([]content.Page, len(candidates))
	for i, c := range candidates {
		out[i] = c.page
	}
	return out
}

// Series describes a page's place in a series of posts that share the same
// series front matter value.
type Series struct {
	Name     string
	Pages    []content.Page
	Position int // 1-based position of the current page
	Prev     *content.Page
	Next     *content.Page
}

// Series returns the series the page belongs to, ordered by weight and then
// date (oldest first), or nil if the page has no series value.
func (index SiteIndex) Series(page content.Page) *Series {
	name := seriesName(page)
	if name == "" {
		return nil
	}
	var pages []content.Page
	for _, other := range index.AllPages {
		if strings.EqualFold(seriesName(other), name) {
			pages = append(pages, other)
		}
	}
	slices.SortStableFunc(pages, func(a, b content.Page) int {
		if a.Weight != b.Weight {
			return a.Weight - b.Weight
		}
		if c := a.Date.Compare(b.Date); c != 0 {
			return c
		}
		return strings.Compare(a.RelativePath, b.RelativePath)
	})

	series := &Series{Name: name, Pages: pages}
	for i := range pages {
		if pages[i].RelativePath != page.RelativePath {
			continue
		}
		series.Position = i + 1
		if i > 0 {
			series.Prev = &pages[i-1]
		}
		if i+1 < len(pages) {
			series.Next = &pages[i+1]
		}
	}
	return series
}

// seriesName reads `series`, which may be a string or a single-item list so
// it also works when series is configured as a taxonomy.
func seriesName(page content.Page) string {
	values := termValues(page.Extra["series"])
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func termValues(value any) []string {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{v}
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
