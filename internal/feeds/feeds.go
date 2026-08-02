package feeds

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
)

type Output struct {
	Filename string
	Content  []byte
}

type atomDocument struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Lang    string      `xml:"xml:lang,attr,omitempty"`
	Title   string      `xml:"title"`
	Link    []atomLink  `xml:"link"`
	Updated string      `xml:"updated"`
	ID      string      `xml:"id"`
	Author  *atomAuthor `xml:"author,omitempty"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	Title     string       `xml:"title"`
	Link      atomLink     `xml:"link"`
	ID        string       `xml:"id"`
	Author    *atomAuthor  `xml:"author,omitempty"`
	Published string       `xml:"published,omitempty"`
	Updated   string       `xml:"updated"`
	Summary   string       `xml:"summary,omitempty"`
	Content   *atomContent `xml:"content,omitempty"`
}

type atomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

func GenerateAll(cfg config.SiteConfig, index site.SiteIndex) ([]Output, error) {
	outputs := make([]Output, 0, 1)

	feedOutput, err := Generate(cfg, index)
	if err != nil {
		return nil, err
	}
	if feedOutput != nil {
		outputs = append(outputs, *feedOutput)
	}

	return outputs, nil
}

func Generate(cfg config.SiteConfig, index site.SiteIndex) (*Output, error) {
	if !cfg.Feed.Enabled {
		return nil, nil
	}

	items := feedItems(index.AllPages, cfg.Feed.Sections, cfg.Feed.Limit)

	feedURL, err := feedURL(cfg.BaseURL, cfg.Feed.Filename)
	if err != nil {
		return nil, fmt.Errorf("generate Feed: %w", err)
	}

	updated := latestUpdated(items)
	doc := atomDocument{
		Lang:  cfg.Language,
		Title: cfg.Title,
		Link: []atomLink{
			{Href: feedURL, Rel: "self", Type: "application/atom+xml"},
			{Href: strings.TrimSpace(cfg.BaseURL), Rel: "alternate", Type: "text/html"},
		},
		Updated: formatAtomDate(updated),
		ID:      feedURL,
		Entries: make([]atomEntry, 0, len(items)),
	}
	if strings.TrimSpace(cfg.Author) != "" {
		doc.Author = &atomAuthor{Name: strings.TrimSpace(cfg.Author)}
	}

	for _, item := range items {
		link := canonicalURL(cfg.BaseURL, item.URL)

		summary := strings.TrimSpace(item.Description)
		if summary == "" {
			summary = item.Title
		}

		doc.Entries = append(doc.Entries, atomEntry{
			Title:     item.Title,
			Link:      atomLink{Href: link, Rel: "alternate", Type: "text/html"},
			ID:        link,
			Author:    atomEntryAuthor(item, cfg),
			Published: formatAtomDate(item.Date),
			Updated:   formatAtomDate(item.Date),
			Summary:   summary,
			Content: &atomContent{
				Type:  "html",
				Value: item.BodyHTML,
			},
		})
	}

	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("generate Feed XML: %w", err)
	}

	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')

	return &Output{
		Filename: cfg.Feed.Filename,
		Content:  data,
	}, nil
}

func formatAtomDate(value time.Time) string {
	if value.IsZero() {
		return time.Unix(0, 0).UTC().Format(time.RFC3339)
	}
	return value.UTC().Format(time.RFC3339)
}

func latestUpdated(items []content.Page) time.Time {
	var latest time.Time
	for _, item := range items {
		if item.Date.After(latest) {
			latest = item.Date
		}
	}
	return latest
}

func feedItems(items []content.Page, sections []string, limit int) []content.Page {
	allowed := map[string]struct{}{}
	for _, section := range sections {
		section = rootSectionName(section)
		if section != "" {
			allowed[section] = struct{}{}
		}
	}

	filtered := items
	if len(allowed) > 0 {
		filtered = make([]content.Page, 0, len(items))
		for _, item := range items {
			if _, ok := allowed[rootSectionName(item.SectionPath)]; ok {
				filtered = append(filtered, item)
			}
		}
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
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

func atomEntryAuthor(item content.Page, cfg config.SiteConfig) *atomAuthor {
	if authors := stringListExtra(item.Extra, "authors"); len(authors) > 0 {
		return &atomAuthor{Name: strings.Join(authors, ", ")}
	}
	if strings.TrimSpace(cfg.Author) != "" {
		return &atomAuthor{Name: strings.TrimSpace(cfg.Author)}
	}
	return nil
}

func stringListExtra(values map[string]any, key string) []string {
	raw, ok := values[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func feedURL(baseURL, filename string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	filename = strings.Trim(strings.TrimSpace(filename), "/")
	if filename == "" {
		return "", fmt.Errorf("feed filename is required")
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL + filename, nil
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
