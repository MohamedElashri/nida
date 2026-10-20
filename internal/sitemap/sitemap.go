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
	lastMods := lastModified(state)

	for _, page := range pages {
		if page.CanonicalURL == "" || page.URL == render.NotFoundURL {
			continue
		}
		if _, ok := seen[page.CanonicalURL]; ok {
			continue
		}
		seen[page.CanonicalURL] = struct{}{}

		entry := urlEntry{Loc: page.CanonicalURL}
		entry.LastMod = lastMods[page.URL]
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

// lastModified maps page routes to the date each page last changed: its
// updated date when later than its date, as in the feeds.
func lastModified(state site.State) map[string]string {
	out := make(map[string]string, len(state.Index.AllPages))
	for _, item := range state.Index.AllPages {
		changed := item.Date
		if item.Updated.After(changed) {
			changed = item.Updated
		}
		if !changed.IsZero() {
			out[item.URL] = changed.UTC().Format("2006-01-02")
		}
	}
	return out
}
