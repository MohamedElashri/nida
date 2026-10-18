package render

import (
	"html"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/templates"
)

func redirectHTML(target string) string {
	jsTarget := strconv.Quote(target)
	escapedTarget := html.EscapeString(target)
	return `<!doctype html>
<meta charset="utf-8">
<title>Redirect</title>
<script>
  const target = ` + jsTarget + `;
  const hash = window.location.hash || "";
  window.location.replace(target + hash);
</script>
<noscript>
  <meta http-equiv="refresh" content="0; url=` + escapedTarget + `">
</noscript>
<p><a href="` + escapedTarget + `">Click here</a> to be redirected.</p>
`
}

func buildPaginator(baseURL, paginatePath string, current, total int, pages []content.Page) *Paginator {
	if total <= 1 {
		return nil
	}
	pageLinks := make([]PageLink, 0, total)
	for i := 1; i <= total; i++ {
		pageLinks = append(pageLinks, PageLink{
			Number:  i,
			URL:     pageURL(baseURL, paginatePath, i),
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
		paginator.Previous = pageURL(baseURL, paginatePath, current-1)
	}
	if current < total {
		paginator.Next = pageURL(baseURL, paginatePath, current+1)
	}
	return paginator
}

func pageURL(baseURL, paginatePath string, pageNum int) string {
	if pageNum <= 1 {
		return baseURL
	}
	if paginatePath == "" {
		paginatePath = "page"
	}
	return strings.TrimSuffix(baseURL, "/") + "/" + paginatePath + "/" + strconv.Itoa(pageNum) + "/"
}

func latestItems(items []content.Page, sections []string, limit int) []content.Page {
	filtered := slices.Clone(content.FilterByRootSections(items, sections))
	slices.SortStableFunc(filtered, content.CompareByDateDesc)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

// joinRobots combines a required robots directive with the site-wide
// robots_meta value, skipping whichever is empty.
func joinRobots(required, configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return required
	}
	return required + ", " + configured
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

func defaultLanguage(value string) string {
	if strings.TrimSpace(value) == "" {
		return "en"
	}
	return value
}

func joinEditURL(prefix, relativePath string) string {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + strings.TrimPrefix(relativePath, "/")
}
