package site

import (
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/content"
)

// NavEntry is one item in a section's navigation: a page, or a child
// section with its own entries.
type NavEntry struct {
	Title   string
	URL     string
	Weight  int
	Page    *content.Page
	Section *content.Section
	// Children holds a section's entries; it is empty for pages.
	Children []NavEntry
}

// IsSection reports whether the entry is a section.
func (e NavEntry) IsSection() bool { return e.Section != nil }

// Nav returns the pages and child sections of a section as one list ordered
// by weight, then title, with child sections expanded recursively. It is
// meant for documentation sidebars, where authors interleave pages and
// subsections with weights. Use "" for the root section.
func (index SiteIndex) Nav(sectionPath string) []NavEntry {
	section, ok := index.SectionLookup[strings.Trim(sectionPath, "/")]
	if !ok {
		return nil
	}
	return index.navEntries(section)
}

func (index SiteIndex) navEntries(section content.Section) []NavEntry {
	var entries []NavEntry
	for i := range section.Pages {
		page := section.Pages[i]
		entries = append(entries, NavEntry{Title: page.Title, URL: page.URL, Weight: page.Weight, Page: &page})
	}
	for _, child := range section.Sections {
		// Use the lookup entry: it has the child's URL and sorted pages.
		full, ok := index.SectionLookup[child.SectionPath]
		if !ok {
			full = child
		}
		entries = append(entries, NavEntry{
			Title:    full.Title,
			URL:      full.URL,
			Weight:   full.Weight,
			Section:  &full,
			Children: index.navEntries(full),
		})
	}
	slices.SortStableFunc(entries, func(a, b NavEntry) int {
		if a.Weight != b.Weight {
			return a.Weight - b.Weight
		}
		return strings.Compare(a.Title, b.Title)
	})
	return entries
}

// Neighbors returns the entries before and after currentURL when the
// navigation tree under sectionPath is read top to bottom, with each section
// placed before its children. Either result is nil at the ends.
func (index SiteIndex) Neighbors(sectionPath, currentURL string) Neighbors {
	var flat []NavEntry
	var walk func([]NavEntry)
	walk = func(entries []NavEntry) {
		for _, entry := range entries {
			flat = append(flat, entry)
			walk(entry.Children)
		}
	}
	walk(index.Nav(sectionPath))

	for i := range flat {
		if flat[i].URL != currentURL {
			continue
		}
		var result Neighbors
		if i > 0 {
			result.Prev = &flat[i-1]
		}
		if i+1 < len(flat) {
			result.Next = &flat[i+1]
		}
		return result
	}
	return Neighbors{}
}

// Neighbors holds the previous and next entries for a page.
type Neighbors struct {
	Prev *NavEntry
	Next *NavEntry
}
