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

	"github.com/MohamedElashri/nida/internal/buildinfo"
	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
)

type Output struct {
	Filename string
	Content  []byte
}

type atomDocument struct {
	XMLName   xml.Name       `xml:"http://www.w3.org/2005/Atom feed"`
	Lang      string         `xml:"xml:lang,attr,omitempty"`
	Title     string         `xml:"title"`
	Generator *atomGenerator `xml:"generator,omitempty"`
	Link      []atomLink     `xml:"link"`
	Updated   string         `xml:"updated"`
	ID        string         `xml:"id"`
	Authors   []atomAuthor   `xml:"author,omitempty"`
	Entries   []atomEntry    `xml:"entry"`
}

type atomGenerator struct {
	URI     string `xml:"uri,attr,omitempty"`
	Version string `xml:"version,attr,omitempty"`
	Value   string `xml:",chardata"`
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

type atomCategory struct {
	Term string `xml:"term,attr"`
}

type atomEntry struct {
	Title      string         `xml:"title"`
	Link       atomLink       `xml:"link"`
	ID         string         `xml:"id"`
	Authors    []atomAuthor   `xml:"author,omitempty"`
	Categories []atomCategory `xml:"category,omitempty"`
	Published  string         `xml:"published,omitempty"`
	Updated    string         `xml:"updated"`
	Summary    string         `xml:"summary,omitempty"`
	Content    *atomContent   `xml:"content,omitempty"`
}

type atomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// spec describes one feed document. Site feeds, section feeds, and taxonomy
// term feeds all share the same item selection and rendering code.
type spec struct {
	Title    string
	Filename string
	AltURL   string
	Items    []content.Page
}

func GenerateAll(cfg config.SiteConfig, index site.SiteIndex) ([]Output, error) {
	outputs := make([]Output, 0, 2)
	for _, generate := range []func(config.SiteConfig, site.SiteIndex) (*Output, error){Generate, GenerateRSS} {
		out, err := generate(cfg, index)
		if err != nil {
			return nil, err
		}
		if out != nil {
			outputs = append(outputs, *out)
		}
	}
	return outputs, nil
}

// GenerateRSS renders the optional site-wide RSS 2.0 feed configured by [rss].
func GenerateRSS(cfg config.SiteConfig, index site.SiteIndex) (*Output, error) {
	if !cfg.RSS.Enabled {
		return nil, nil
	}
	return renderRSS(cfg, siteSpec(cfg, cfg.RSS, index))
}

// Generate renders the site-wide Atom feed configured by [feed].
func Generate(cfg config.SiteConfig, index site.SiteIndex) (*Output, error) {
	if !cfg.Feed.Enabled {
		return nil, nil
	}
	return renderAtom(cfg, siteSpec(cfg, cfg.Feed, index))
}

func siteSpec(cfg config.SiteConfig, feed config.FeedConfig, index site.SiteIndex) spec {
	return spec{
		Title:    cfg.Title,
		Filename: feed.Filename,
		AltURL:   strings.TrimSpace(cfg.BaseURL),
		Items:    feedItems(index.AllPages, feed.Sections, feed.Limit),
	}
}

func renderAtom(cfg config.SiteConfig, feed spec) (*Output, error) {
	items := feed.Items

	feedURL, err := feedURL(cfg.BaseURL, feed.Filename)
	if err != nil {
		return nil, fmt.Errorf("generate Atom feed: %w", err)
	}

	updated := latestUpdated(items)
	doc := atomDocument{
		Lang:  cfg.Language,
		Title: stripHTML(feed.Title),
		Generator: &atomGenerator{
			URI:     "https://github.com/MohamedElashri/nida",
			Version: buildinfo.Version,
			Value:   "Nida",
		},
		Link: []atomLink{
			{Href: feedURL, Rel: "self", Type: "application/atom+xml"},
			{Href: feed.AltURL, Rel: "alternate", Type: "text/html"},
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

		doc.Entries = append(doc.Entries, atomEntry{
			Title:      stripHTML(item.Title),
			Link:       atomLink{Href: link, Rel: "alternate", Type: "text/html"},
			ID:         link,
			Authors:    atomEntryAuthors(item, cfg),
			Categories: atomEntryCategories(item),
			Published:  formatAtomDate(item.Date),
			Updated:    formatAtomDate(itemUpdated(item)),
			Summary:    itemSummary(item),
			Content: &atomContent{
				Type:  "html",
				Value: absolutizeHTML(item.BodyHTML, cfg.BaseURL, item.URL),
			},
		})
	}

	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("generate Atom feed XML: %w", err)
	}

	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')

	return &Output{
		Filename: feed.Filename,
		Content:  data,
	}, nil
}

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Content string     `xml:"xmlns:content,attr"`
	DC      string     `xml:"xmlns:dc,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string      `xml:"title"`
	Link          string      `xml:"link"`
	Description   string      `xml:"description"`
	Language      string      `xml:"language,omitempty"`
	Generator     string      `xml:"generator"`
	LastBuildDate string      `xml:"lastBuildDate,omitempty"`
	AtomLink      rssAtomLink `xml:"atom:link"`
	Items         []rssItem   `xml:"item"`
}

type rssAtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	GUID        rssGUID   `xml:"guid"`
	PubDate     string    `xml:"pubDate,omitempty"`
	Creators    []string  `xml:"dc:creator,omitempty"`
	Categories  []string  `xml:"category,omitempty"`
	Description string    `xml:"description,omitempty"`
	Content     *rssCDATA `xml:"content:encoded,omitempty"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type rssCDATA struct {
	Value string `xml:",cdata"`
}

func renderRSS(cfg config.SiteConfig, feed spec) (*Output, error) {
	selfURL, err := feedURL(cfg.BaseURL, feed.Filename)
	if err != nil {
		return nil, fmt.Errorf("generate RSS feed: %w", err)
	}

	channel := rssChannel{
		Title:       stripHTML(feed.Title),
		Link:        feed.AltURL,
		Description: stripHTML(cfg.Description),
		Language:    cfg.Language,
		Generator:   "Nida " + buildinfo.Version,
		AtomLink:    rssAtomLink{Href: selfURL, Rel: "self", Type: "application/rss+xml"},
		Items:       make([]rssItem, 0, len(feed.Items)),
	}
	if channel.Description == "" {
		channel.Description = channel.Title
	}
	if latest := latestUpdated(feed.Items); !latest.IsZero() {
		channel.LastBuildDate = latest.UTC().Format(time.RFC1123Z)
	}

	for _, item := range feed.Items {
		link := canonicalURL(cfg.BaseURL, item.URL)
		entry := rssItem{
			Title:       stripHTML(item.Title),
			Link:        link,
			GUID:        rssGUID{IsPermaLink: "true", Value: link},
			Description: itemSummary(item),
			Content:     &rssCDATA{Value: absolutizeHTML(item.BodyHTML, cfg.BaseURL, item.URL)},
		}
		if !item.Date.IsZero() {
			entry.PubDate = item.Date.UTC().Format(time.RFC1123Z)
		}
		for _, author := range atomEntryAuthors(item, cfg) {
			entry.Creators = append(entry.Creators, author.Name)
		}
		for _, category := range atomEntryCategories(item) {
			entry.Categories = append(entry.Categories, category.Term)
		}
		channel.Items = append(channel.Items, entry)
	}

	doc := rssDocument{
		Version: "2.0",
		Content: "http://purl.org/rss/1.0/modules/content/",
		DC:      "http://purl.org/dc/elements/1.1/",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: channel,
	}
	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("generate RSS feed XML: %w", err)
	}
	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')

	return &Output{Filename: feed.Filename, Content: data}, nil
}

// itemSummary prefers the author-written description and falls back to the
// rendered summary so feed readers always get a short teaser.
func itemSummary(item content.Page) string {
	if summary := stripHTML(strings.TrimSpace(item.Description)); summary != "" {
		return summary
	}
	return stripHTML(item.Summary)
}

func itemUpdated(item content.Page) time.Time {
	if item.Updated.After(item.Date) {
		return item.Updated
	}
	return item.Date
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
		if updated := itemUpdated(item); updated.After(latest) {
			latest = updated
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

func atomEntryCategories(item content.Page) []atomCategory {
	var cats []atomCategory

	// Add tags
	if raw, ok := item.Extra["tags"]; ok {
		if tags, ok := raw.([]string); ok {
			for _, t := range tags {
				cats = append(cats, atomCategory{Term: strings.TrimSpace(t)})
			}
		} else if tagList, ok := raw.([]any); ok {
			for _, t := range tagList {
				if s, ok := t.(string); ok {
					cats = append(cats, atomCategory{Term: strings.TrimSpace(s)})
				}
			}
		}
	}

	// Add categories
	if raw, ok := item.Extra["categories"]; ok {
		if categories, ok := raw.([]string); ok {
			for _, c := range categories {
				cats = append(cats, atomCategory{Term: strings.TrimSpace(c)})
			}
		} else if catList, ok := raw.([]any); ok {
			for _, c := range catList {
				if s, ok := c.(string); ok {
					cats = append(cats, atomCategory{Term: strings.TrimSpace(s)})
				}
			}
		}
	}

	return cats
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

// resolveURL makes a link from a page's HTML absolute the way a browser
// would: root-relative paths resolve against the host, other relative paths
// against the page URL.
func resolveURL(baseURL, itemURL, rawURL string) string {
	if strings.HasPrefix(rawURL, "data:") || strings.HasPrefix(rawURL, "#") || strings.HasPrefix(rawURL, "mailto:") {
		return rawURL
	}
	rel, err := url.Parse(rawURL)
	if err != nil || rel.Scheme != "" || rel.Host != "" {
		return rawURL
	}
	page, err := url.Parse(canonicalURL(baseURL, itemURL))
	if err != nil {
		return rawURL
	}
	return page.ResolveReference(rel).String()
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
