package feeds

import (
	"encoding/xml"
	"fmt"
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

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description,omitempty"`
	Language    string    `xml:"language,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate,omitempty"`
	Description string  `xml:"description,omitempty"`
}

type rssGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
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
	outputs := make([]Output, 0, 2)

	rssOutput, err := Generate(cfg, index)
	if err != nil {
		return nil, err
	}
	if rssOutput != nil {
		outputs = append(outputs, *rssOutput)
	}

	atomOutput, err := GenerateAtom(cfg, index)
	if err != nil {
		return nil, err
	}
	if atomOutput != nil {
		outputs = append(outputs, *atomOutput)
	}

	return outputs, nil
}

func Generate(cfg config.SiteConfig, index site.SiteIndex) (*Output, error) {
	if !cfg.RSS.Enabled {
		return nil, nil
	}

	items := index.Posts
	if cfg.RSS.Limit > 0 && len(items) > cfg.RSS.Limit {
		items = items[:cfg.RSS.Limit]
	}

	doc := rssDocument{
		Version: "2.0",
		Channel: rssChannel{
			Title:       cfg.Title,
			Link:        strings.TrimSpace(cfg.BaseURL),
			Description: cfg.Description,
			Language:    cfg.Language,
			Items:       make([]rssItem, 0, len(items)),
		},
	}

	for _, item := range items {
		link, ok := index.CanonicalLookup[item.URL]
		if !ok {
			return nil, fmt.Errorf("generate RSS: missing canonical URL for %q", item.URL)
		}

		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = item.Title
		}

		doc.Channel.Items = append(doc.Channel.Items, rssItem{
			Title: item.Title,
			Link:  link,
			GUID: rssGUID{
				IsPermaLink: true,
				Value:       link,
			},
			PubDate:     formatPubDate(item.Date),
			Description: description,
		})
	}

	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("generate RSS XML: %w", err)
	}

	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')

	return &Output{
		Filename: cfg.RSS.Filename,
		Content:  data,
	}, nil
}

func GenerateAtom(cfg config.SiteConfig, index site.SiteIndex) (*Output, error) {
	if !cfg.Atom.Enabled {
		return nil, nil
	}

	items := index.Posts
	if cfg.Atom.Limit > 0 && len(items) > cfg.Atom.Limit {
		items = items[:cfg.Atom.Limit]
	}

	feedURL, err := feedURL(cfg.BaseURL, cfg.Atom.Filename)
	if err != nil {
		return nil, fmt.Errorf("generate Atom: %w", err)
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
		link, ok := index.CanonicalLookup[item.URL]
		if !ok {
			return nil, fmt.Errorf("generate Atom: missing canonical URL for %q", item.URL)
		}

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
		return nil, fmt.Errorf("generate Atom XML: %w", err)
	}

	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')

	return &Output{
		Filename: cfg.Atom.Filename,
		Content:  data,
	}, nil
}

func formatPubDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC1123Z)
}

func formatAtomDate(value time.Time) string {
	if value.IsZero() {
		return time.Unix(0, 0).UTC().Format(time.RFC3339)
	}
	return value.UTC().Format(time.RFC3339)
}

func latestUpdated(items []content.Item) time.Time {
	var latest time.Time
	for _, item := range items {
		if item.Date.After(latest) {
			latest = item.Date
		}
	}
	return latest
}

func atomEntryAuthor(item content.Item, cfg config.SiteConfig) *atomAuthor {
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
