package feeds

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/url"
	"path"
	"regexp"
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
	XMLName xml.Name     `xml:"http://www.w3.org/2005/Atom feed"`
	Lang    string       `xml:"xml:lang,attr,omitempty"`
	Title   string       `xml:"title"`
	Link    []atomLink   `xml:"link"`
	Updated string       `xml:"updated"`
	ID      string       `xml:"id"`
	Authors []atomAuthor `xml:"author,omitempty"`
	Entries []atomEntry  `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name  string `xml:"name"`
	URI   string `xml:"uri,omitempty"`
	Email string `xml:"email,omitempty"`
}

type atomEntry struct {
	Title     string       `xml:"title"`
	Link      atomLink     `xml:"link"`
	ID        string       `xml:"id"`
	Authors   []atomAuthor `xml:"author,omitempty"`
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
		Title: stripHTML(cfg.Title),
		Link: []atomLink{
			{Href: feedURL, Rel: "self", Type: "application/atom+xml"},
			{Href: strings.TrimSpace(cfg.BaseURL), Rel: "alternate", Type: "text/html"},
		},
		Updated: formatAtomDate(updated),
		ID:      feedURL,
		Entries: make([]atomEntry, 0, len(items)),
	}
	if strings.TrimSpace(cfg.Author) != "" {
		doc.Authors = []atomAuthor{{Name: strings.TrimSpace(cfg.Author)}}
	}

	for _, item := range items {
		link := canonicalURL(cfg.BaseURL, item.URL)

		summary := stripHTML(strings.TrimSpace(item.Description))

		doc.Entries = append(doc.Entries, atomEntry{
			Title:     stripHTML(item.Title),
			Link:      atomLink{Href: link, Rel: "alternate", Type: "text/html"},
			ID:        link,
			Authors:   atomEntryAuthors(item, cfg),
			Published: formatAtomDate(item.Date),
			Updated:   formatAtomDate(item.Date),
			Summary:   summary,
			Content: &atomContent{
				Type:  "html",
				Value: absolutizeHTML(item.BodyHTML, cfg.BaseURL, item.URL),
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

func atomEntryAuthors(item content.Page, cfg config.SiteConfig) []atomAuthor {
	if raw, ok := item.Extra["authors"]; ok {
		if authors := parseAuthorsExtra(raw); len(authors) > 0 {
			return authors
		}
	}
	if strings.TrimSpace(cfg.Author) != "" {
		return []atomAuthor{{Name: strings.TrimSpace(cfg.Author)}}
	}
	return nil
}

func parseAuthorsExtra(raw any) []atomAuthor {
	var out []atomAuthor
	switch v := raw.(type) {
	case []string:
		for _, s := range v {
			if strings.TrimSpace(s) != "" {
				out = append(out, atomAuthor{Name: strings.TrimSpace(s)})
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, atomAuthor{Name: strings.TrimSpace(s)})
			} else if m, ok := item.(map[string]any); ok {
				author := atomAuthor{}
				if name, ok := m["name"].(string); ok {
					author.Name = strings.TrimSpace(name)
				}
				if email, ok := m["email"].(string); ok {
					author.Email = strings.TrimSpace(email)
				}
				if uri, ok := m["uri"].(string); ok {
					author.URI = strings.TrimSpace(uri)
				}
				if author.Name != "" {
					out = append(out, author)
				}
			}
		}
	}
	return out
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

func stripHTML(markup string) string {
	var result strings.Builder
	inTag := false
	for _, r := range markup {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(html.UnescapeString(result.String())), " ")
}

var htmlURLRegex = regexp.MustCompile(`(?i)(href|src)=["']([^"']+)["']`)

func resolveURL(baseURL, itemURL, rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && (u.Scheme != "" || u.Host != "") {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "data:") || strings.HasPrefix(rawURL, "#") || strings.HasPrefix(rawURL, "mailto:") {
		return rawURL
	}

	rel, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	var route string
	if strings.HasPrefix(rel.Path, "/") {
		route = rel.Path
	} else {
		route = path.Join(itemURL, rel.Path)
	}

	resolved := canonicalURL(baseURL, route)

	if resURL, err := url.Parse(resolved); err == nil {
		resURL.RawQuery = rel.RawQuery
		resURL.Fragment = rel.Fragment
		return resURL.String()
	}

	return resolved
}

func absolutizeHTML(htmlStr string, baseURL string, itemURL string) string {
	return htmlURLRegex.ReplaceAllStringFunc(htmlStr, func(match string) string {
		parts := htmlURLRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		attr := parts[1]
		rawURL := parts[2]

		resolved := resolveURL(baseURL, itemURL, rawURL)
		return fmt.Sprintf(`%s="%s"`, attr, resolved)
	})
}
