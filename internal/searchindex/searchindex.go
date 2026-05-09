package searchindex

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/site"
)

type SearchIndex struct {
	DocumentStore DocumentStore `json:"documentStore"`
}

type DocumentStore struct {
	Docs map[string]Doc `json:"docs"`
}

type Doc struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func Generate(cfg config.SiteConfig, state site.State) (*Output, error) {
	if !cfg.Search.Enabled {
		return nil, nil
	}

	docs := make(map[string]Doc)

	// Add all pages
	for _, page := range state.Index.AllPages {
		if page.Draft && !cfg.Drafts {
			continue
		}
		url := page.URL
		if url == "" {
			continue
		}
		body := stripHTML(page.BodyHTML)
		docs[url] = Doc{
			Title: page.Title,
			Body:  body,
		}
	}

	// Add all sections
	for _, section := range state.Index.Sections {
		url := section.URL
		if url == "" {
			continue
		}
		body := stripHTML(section.BodyHTML)
		docs[url] = Doc{
			Title: section.Title,
			Body:  body,
		}
	}

	index := SearchIndex{
		DocumentStore: DocumentStore{
			Docs: docs,
		},
	}

	data, err := json.Marshal(index)
	if err != nil {
		return nil, fmt.Errorf("marshal search index: %w", err)
	}

	content := []byte(fmt.Sprintf("window.searchIndex = %s;\n", data))
	return &Output{
		Filename: cfg.Search.Filename,
		Content:  content,
	}, nil
}

type Output struct {
	Filename string
	Content  []byte
}

func stripHTML(html string) string {
	// Very basic HTML stripping for search indexing
	var result strings.Builder
	inTag := false
	for _, r := range html {
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
	return strings.Join(strings.Fields(result.String()), " ")
}
