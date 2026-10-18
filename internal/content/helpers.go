package content

import (
	"html"
	"strings"
	"unicode"
)

// RootSection returns the first segment of a section path.
func RootSection(sectionPath string) string {
	root, _, _ := strings.Cut(strings.Trim(sectionPath, "/"), "/")
	return root
}

// FilterByRootSections keeps pages whose root section is listed. An empty
// list keeps every page.
func FilterByRootSections(pages []Page, sections []string) []Page {
	allowed := map[string]bool{}
	for _, section := range sections {
		if root := RootSection(section); root != "" {
			allowed[root] = true
		}
	}
	if len(allowed) == 0 {
		return pages
	}
	filtered := make([]Page, 0, len(pages))
	for _, page := range pages {
		if allowed[RootSection(page.SectionPath)] {
			filtered = append(filtered, page)
		}
	}
	return filtered
}

// CompareByDateDesc orders pages newest first, then by slug.
func CompareByDateDesc(a, b Page) int {
	if c := b.Date.Compare(a.Date); c != 0 {
		return c
	}
	return strings.Compare(a.Slug, b.Slug)
}

// PlainText strips HTML tags, unescapes entities, and collapses whitespace.
func PlainText(markup string) string {
	var b strings.Builder
	inTag := false
	for _, r := range markup {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(html.UnescapeString(b.String())), " ")
}

// TitleCase capitalizes the first letter of each space-separated word.
func TitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
