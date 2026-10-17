package searchindex

import (
	"encoding/json"
	"fmt"
	"html"
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
		if root := rootSection(section); root != "" {
			allowed[root] = true
		}
	}
	include := func(sectionPath string) bool {
		return len(allowed) == 0 || allowed[rootSection(sectionPath)]
	}

	for _, page := range state.Index.AllPages {
		if page.Draft && !cfg.Drafts {
			continue
		}
		if page.URL == "" || !include(page.SectionPath) {
			continue
		}
		docs[page.URL] = Doc{
			Title:       page.Title,
			Description: page.Description,
			Body:        limitBody(stripHTML(page.BodyHTML), cfg.Search.BodyLimit),
		}
	}

	for _, section := range state.Index.SectionLookup {
		if section.URL == "" || !include(section.SectionPath) {
			continue
		}
		docs[section.URL] = Doc{
			Title:       section.Title,
			Description: section.Description,
			Body:        limitBody(stripHTML(section.BodyHTML), cfg.Search.BodyLimit),
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

func stripHTML(markup string) string {
	// Very basic HTML stripping for search indexing
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

func rootSection(sectionPath string) string {
	sectionPath = strings.Trim(sectionPath, "/")
	root, _, _ := strings.Cut(sectionPath, "/")
	return root
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
