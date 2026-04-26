package sitemap

import (
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/site"
)

type Output struct {
	Filename string
	Content  []byte
}

type urlSet struct {
	XMLName xml.Name   `xml:"urlset"`
	Xmlns   string     `xml:"xmlns,attr"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func Generate(cfg config.SiteConfig, state site.State, pages []render.Page) (*Output, error) {
	if !cfg.Sitemap.Enabled {
		return nil, nil
	}

	entries := make([]urlEntry, 0, len(pages))
	seen := make(map[string]struct{}, len(pages))

	for _, page := range pages {
		if page.CanonicalURL == "" {
			continue
		}
		if _, ok := seen[page.CanonicalURL]; ok {
			continue
		}
		seen[page.CanonicalURL] = struct{}{}

		entry := urlEntry{Loc: page.CanonicalURL}
		if lastMod, ok := lastModified(page.URL, state); ok {
			entry.LastMod = lastMod
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Loc < entries[j].Loc
	})

	doc := urlSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  entries,
	}

	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("generate sitemap XML: %w", err)
	}
	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')

	return &Output{
		Filename: cfg.Sitemap.Filename,
		Content:  data,
	}, nil
}

func lastModified(route string, state site.State) (string, bool) {
	for _, item := range state.Index.AllPages {
		if item.URL == route && !item.Date.IsZero() {
			return item.Date.UTC().Format("2006-01-02"), true
		}
	}
	return "", false
}
