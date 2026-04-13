package render

import (
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
	"github.com/MohamedElashri/nida/internal/templates"
)

func buildPaginator(baseURL string, current, total int, pages []content.Item) *Paginator {
	if total <= 1 {
		return nil
	}
	pageLinks := make([]PageLink, 0, total)
	for i := 1; i <= total; i++ {
		pageLinks = append(pageLinks, PageLink{
			Number:  i,
			URL:     pageURL(baseURL, i),
			Current: i == current,
		})
	}
	paginator := &Paginator{
		CurrentIndex: current,
		NumberPagers: total,
		PageLinks:    pageLinks,
		Pages:        pages,
	}
	if current > 1 {
		paginator.Previous = pageURL(baseURL, current-1)
	}
	if current < total {
		paginator.Next = pageURL(baseURL, current+1)
	}
	return paginator
}

func pageURL(baseURL string, pageNum int) string {
	if pageNum <= 1 {
		return baseURL
	}
	return strings.TrimSuffix(baseURL, "/") + "/page/" + strconv.Itoa(pageNum) + "/"
}

func latestItems(items []content.Item, mainSections []string, limit int) []content.Item {
	filtered := make([]content.Item, 0, len(items))
	allowed := map[string]struct{}{}
	for _, section := range mainSections {
		allowed[strings.TrimSpace(section)] = struct{}{}
	}
	for _, item := range items {
		if len(allowed) > 0 {
			root := rootSectionName(item.SectionPath)
			if _, ok := allowed[root]; !ok {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	slices.SortFunc(filtered, func(a, b content.Item) int {
		if !a.Date.Equal(b.Date) {
			if a.Date.After(b.Date) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Title, b.Title)
	})
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

func templateForItem(set templates.Set, index site.SiteIndex, item content.Item, fallback string) string {
	if name := normalizeTemplateName(item.Template); name != "" && set.Has(name) {
		return name
	}
	if section, ok := index.SectionLookup[item.SectionPath]; ok {
		if name := normalizeTemplateName(section.PageTemplate); name != "" && set.Has(name) {
			return name
		}
	}
	if set.Has(fallback) {
		return fallback
	}
	return "page"
}

func templateForSection(set templates.Set, section site.Section) string {
	if name := normalizeTemplateName(section.Template); name != "" && set.Has(name) {
		return name
	}
	if set.Has("section") {
		return "section"
	}
	return "list"
}

func pickExistingTemplate(set templates.Set, names ...string) string {
	for _, name := range names {
		if name != "" && set.Has(name) {
			return name
		}
	}
	return ""
}

func normalizeTemplateName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, filepath.Ext(value))
	value = filepath.Base(value)
	return strings.TrimSpace(value)
}

func derefSection(value *site.Section) site.Section {
	if value == nil {
		return site.Section{}
	}
	return *value
}

func rootSectionName(sectionPath string) string {
	sectionPath = strings.Trim(sectionPath, "/")
	if sectionPath == "" {
		return ""
	}
	if index := strings.Index(sectionPath, "/"); index >= 0 {
		return sectionPath[:index]
	}
	return sectionPath
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func defaultLanguage(value string) string {
	if strings.TrimSpace(value) == "" {
		return "en"
	}
	return value
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
