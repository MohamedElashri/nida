package feeds

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
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

func formatPubDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC1123Z)
}
