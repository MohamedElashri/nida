package feeds

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/buildinfo"
	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/frontmatter"
	"github.com/MohamedElashri/nida/internal/route"
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

	for _, feed := range scopedSpecs(cfg, index) {
		if cfg.Feed.Enabled {
			out, err := renderAtom(cfg, withFilename(feed, cfg.Feed))
			if err != nil {
				return nil, err
			}
			outputs = append(outputs, *out)
		}
		if cfg.RSS.Enabled {
			out, err := renderRSS(cfg, withFilename(feed, cfg.RSS))
			if err != nil {
				return nil, err
			}
			outputs = append(outputs, *out)
		}
	}
	return outputs, nil
}

// Filenames lists every feed file GenerateAll writes, so the build can
// check them for route conflicts and clean them up when they go away.
func Filenames(cfg config.SiteConfig, index site.SiteIndex) []string {
	var names []string
	if cfg.Feed.Enabled {
		names = append(names, cfg.Feed.Filename)
	}
	if cfg.RSS.Enabled {
		names = append(names, cfg.RSS.Filename)
	}
	for _, feed := range scopedSpecs(cfg, index) {
		if cfg.Feed.Enabled {
			names = append(names, withFilename(feed, cfg.Feed).Filename)
		}
		if cfg.RSS.Enabled {
			names = append(names, withFilename(feed, cfg.RSS).Filename)
		}
	}
	return names
}

// scopedSpecs returns section and taxonomy term feeds. Filename holds only
// the directory prefix until withFilename adds the file name.
func scopedSpecs(cfg config.SiteConfig, index site.SiteIndex) []spec {
	var specs []spec

	paths := make([]string, 0, len(index.SectionLookup))
	for path, section := range index.SectionLookup {
		if section.GenerateFeeds && path != "" {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	for _, path := range paths {
		section := index.SectionLookup[path]
		var items []content.Page
		for _, page := range index.AllPages {
			if page.SectionPath == path || strings.HasPrefix(page.SectionPath, path+"/") {
				items = append(items, page)
			}
		}
		specs = append(specs, spec{
			Title:    scopedTitle(cfg.Title, section.Title),
			Filename: path + "/",
			AltURL:   route.CanonicalURL(cfg.BaseURL, "/"+path+"/"),
			Items:    items,
		})
	}

	for _, collection := range index.Taxonomies {
		if !collection.Feed {
			continue
		}
		for _, term := range collection.Terms {
			items := append([]content.Page(nil), term.Items...)
			specs = append(specs, spec{
				Title:    scopedTitle(cfg.Title, term.Name),
				Filename: strings.Trim(term.URL, "/") + "/",
				AltURL:   term.CanonicalURL,
				Items:    items,
			})
		}
	}
	return specs
}

func withFilename(feed spec, feedCfg config.FeedConfig) spec {
	feed.Filename += feedCfg.Filename
	if feedCfg.Limit > 0 && len(feed.Items) > feedCfg.Limit {
		feed.Items = feed.Items[:feedCfg.Limit]
	}
	return feed
}

func scopedTitle(siteTitle, scope string) string {
	if strings.TrimSpace(scope) == "" {
		return siteTitle
	}
	if strings.TrimSpace(siteTitle) == "" {
		return scope
	}
	return siteTitle + " - " + scope
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
		Title: content.PlainText(feed.Title),
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
		link := route.CanonicalURL(cfg.BaseURL, item.URL)

		doc.Entries = append(doc.Entries, atomEntry{
			Title:      content.PlainText(item.Title),
			Link:       atomLink{Href: link, Rel: "alternate", Type: "text/html"},
			ID:         link,
			Authors:    atomEntryAuthors(item, cfg),
			Categories: atomEntryCategories(item, cfg),
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
		Title:       content.PlainText(feed.Title),
		Link:        feed.AltURL,
		Description: content.PlainText(cfg.Description),
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
		link := route.CanonicalURL(cfg.BaseURL, item.URL)
		entry := rssItem{
			Title:       content.PlainText(item.Title),
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
		for _, category := range atomEntryCategories(item, cfg) {
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
	if summary := content.PlainText(strings.TrimSpace(item.Description)); summary != "" {
		return summary
	}
	return content.PlainText(item.Summary)
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
	filtered := content.FilterByRootSections(items, sections)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

// atomEntryCategories lists the page's terms for every configured taxonomy,
// or for tags and categories when none are configured.
func atomEntryCategories(item content.Page, cfg config.SiteConfig) []atomCategory {
	names := []string{"tags", "categories"}
	if len(cfg.Taxonomies) > 0 {
		names = names[:0]
		for _, tc := range cfg.Taxonomies {
			names = append(names, tc.Name)
		}
	}
	var cats []atomCategory
	for _, name := range names {
		for _, term := range frontmatter.StringList(item.Extra[name]) {
			cats = append(cats, atomCategory{Term: term})
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
	for _, name := range frontmatter.StringList(raw) {
		out = append(out, atomAuthor{Name: name})
	}
	list, _ := raw.([]any)
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		author := atomAuthor{}
		author.Name, _ = m["name"].(string)
		author.Email, _ = m["email"].(string)
		author.URI, _ = m["uri"].(string)
		author.Name, author.Email, author.URI = strings.TrimSpace(author.Name), strings.TrimSpace(author.Email), strings.TrimSpace(author.URI)
		if author.Name != "" {
			out = append(out, author)
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

var htmlURLRegex = regexp.MustCompile(`(?i)(href|src)=["']([^"']+)["']`)

// resolveURL makes a page link absolute the way a browser resolves it.
func resolveURL(baseURL, itemURL, rawURL string) string {
	if strings.HasPrefix(rawURL, "data:") || strings.HasPrefix(rawURL, "#") || strings.HasPrefix(rawURL, "mailto:") {
		return rawURL
	}
	rel, err := url.Parse(rawURL)
	if err != nil || rel.Scheme != "" || rel.Host != "" {
		return rawURL
	}
	page, err := url.Parse(route.CanonicalURL(baseURL, itemURL))
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
