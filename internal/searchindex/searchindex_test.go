package searchindex

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
)

func TestGenerateIncludesDescriptionsAndPlainText(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.Search.Enabled = true

	state := site.State{
		Index: site.SiteIndex{
			AllPages: []content.Page{
				{
					Title:       "About",
					Description: "A short summary.",
					URL:         "/about/",
					BodyHTML:    "<p>Tom &amp; Jerry <strong>notes</strong></p>",
				},
			},
			Sections: []content.Section{
				{
					Title:       "Posts",
					Description: "Recent writing.",
					URL:         "/posts/",
					BodyHTML:    "<p>Archive text</p>",
				},
			},
		},
	}

	out, err := Generate(cfg, state)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if out == nil {
		t.Fatal("expected search output")
	}

	raw := strings.TrimSuffix(strings.TrimPrefix(string(out.Content), "window.searchIndex = "), ";\n")
	var index SearchIndex
	if err := json.Unmarshal([]byte(raw), &index); err != nil {
		t.Fatalf("unmarshal search index: %v", err)
	}

	page := index.DocumentStore.Docs["/about/"]
	if page.Description != "A short summary." {
		t.Fatalf("expected page description, got %q", page.Description)
	}
	if page.Body != "Tom & Jerry notes" {
		t.Fatalf("expected stripped and unescaped body, got %q", page.Body)
	}

	section := index.DocumentStore.Docs["/posts/"]
	if section.Description != "Recent writing." {
		t.Fatalf("expected section description, got %q", section.Description)
	}
}
