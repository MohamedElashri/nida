package searchindex

import (
	"encoding/json"
	"fmt"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
)

type SearchIndex struct {
	DocumentStore DocumentStore `json:"documentStore"`
}

type DocumentStore struct {
	Docs map[string]Doc `json:"docs"`
}

type Doc struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body"`
}

func Generate(cfg config.SiteConfig, state site.State) (*Output, error) {
	if !cfg.Search.Enabled {
		return nil, nil
	}

	docs := make(map[string]Doc)
	allowed := map[string]bool{}
	for _, section := range cfg.Search.Sections {
		allowed[content.RootSection(section)] = true
	}

	for _, page := range content.FilterByRootSections(state.Index.AllPages, cfg.Search.Sections) {
		if page.URL == "" {
			continue
		}
		docs[page.URL] = Doc{
			Title:       page.Title,
			Description: page.Description,
			Body:        limitBody(content.PlainText(page.BodyHTML), cfg.Search.BodyLimit),
		}
	}

	for _, section := range state.Index.SectionLookup {
		if section.URL == "" || (len(allowed) > 0 && !allowed[content.RootSection(section.SectionPath)]) {
			continue
		}
		docs[section.URL] = Doc{
			Title:       section.Title,
			Description: section.Description,
			Body:        limitBody(content.PlainText(section.BodyHTML), cfg.Search.BodyLimit),
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

	script := []byte(fmt.Sprintf("window.searchIndex = %s;\n", data))
	return &Output{
		Filename: cfg.Search.Filename,
		Content:  script,
	}, nil
}

type Output struct {
	Filename string
	Content  []byte
}

func limitBody(body string, limit int) string {
	if limit <= 0 {
		return body
	}
	runes := []rune(body)
	if len(runes) <= limit {
		return body
	}
	return string(runes[:limit])
}
