package render

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
	"github.com/MohamedElashri/nida/internal/taxonomies"
	"github.com/MohamedElashri/nida/internal/templates"
)

type Page struct {
	URL          string
	CanonicalURL string
	TemplateName string
	Title        string
	Content      string
}

type templateContext struct {
	Title        string
	HomeURL      string
	CanonicalURL string
	Config       config.SiteConfig
	Index        site.SiteIndex
	Page         content.Item
	Pages        []content.Item
	Terms        []taxonomies.Term
	Taxonomy     taxonomies.Collection
	Term         taxonomies.Term
}

func RenderSite(siteRoot string, cfg config.SiteConfig, state site.State) ([]Page, error) {
	set, err := templates.Load(siteRoot, cfg)
	if err != nil {
		return nil, err
	}

	required := []string{"index", "post", "page"}
	for _, name := range required {
		if !set.Has(name) {
			return nil, fmt.Errorf("render site: missing template %q", name)
		}
	}

	pages := make([]Page, 0, len(state.Index.Posts)+len(state.Index.Pages)+2)

	homePage, err := renderTemplate(set, "index", templateContext{
		Title:        cfg.Title,
		HomeURL:      "/",
		CanonicalURL: canonicalURL(cfg.BaseURL, "/"),
		Config:       cfg,
		Index:        state.Index,
		Pages:        state.Index.RecentPosts,
	})
	if err != nil {
		return nil, err
	}
	pages = append(pages, Page{
		URL:          "/",
		CanonicalURL: canonicalURL(cfg.BaseURL, "/"),
		TemplateName: "index",
		Title:        cfg.Title,
		Content:      homePage,
	})

	for _, item := range state.Index.Posts {
		out, err := renderTemplate(set, "post", templateContext{
			Title:        item.Title,
			HomeURL:      "/",
			CanonicalURL: state.Index.CanonicalLookup[item.URL],
			Config:       cfg,
			Index:        state.Index,
			Page:         item,
		})
		if err != nil {
			return nil, err
		}
		pages = append(pages, Page{
			URL:          item.URL,
			CanonicalURL: state.Index.CanonicalLookup[item.URL],
			TemplateName: "post",
			Title:        item.Title,
			Content:      out,
		})
	}

	for _, item := range state.Index.Pages {
		out, err := renderTemplate(set, "page", templateContext{
			Title:        item.Title,
			HomeURL:      "/",
			CanonicalURL: state.Index.CanonicalLookup[item.URL],
			Config:       cfg,
			Index:        state.Index,
			Page:         item,
		})
		if err != nil {
			return nil, err
		}
		pages = append(pages, Page{
			URL:          item.URL,
			CanonicalURL: state.Index.CanonicalLookup[item.URL],
			TemplateName: "page",
			Title:        item.Title,
			Content:      out,
		})
	}

	if set.Has("list") {
		out, err := renderTemplate(set, "list", templateContext{
			Title:        "Posts",
			HomeURL:      "/",
			CanonicalURL: canonicalURL(cfg.BaseURL, "/posts/"),
			Config:       cfg,
			Index:        state.Index,
			Pages:        state.Index.Posts,
		})
		if err != nil {
			return nil, err
		}
		pages = append(pages, Page{
			URL:          "/posts/",
			CanonicalURL: canonicalURL(cfg.BaseURL, "/posts/"),
			TemplateName: "list",
			Title:        "Posts",
			Content:      out,
		})
	}

	if set.Has("taxonomy") {
		taxonomyPages, err := renderTaxonomyPages(set, cfg, state.Index)
		if err != nil {
			return nil, err
		}
		pages = append(pages, taxonomyPages...)
	}

	return pages, nil
}

func renderTaxonomyPages(set templates.Set, cfg config.SiteConfig, index site.SiteIndex) ([]Page, error) {
	pages := make([]Page, 0)
	for _, collection := range []taxonomies.Collection{index.Tags, index.Categories} {
		if collection.Name == "" {
			continue
		}

		landing, err := renderTemplate(set, "taxonomy", templateContext{
			Title:        strings.Title(collection.Name),
			HomeURL:      "/",
			CanonicalURL: collection.CanonicalURL,
			Config:       cfg,
			Index:        index,
			Terms:        collection.Terms,
			Taxonomy:     collection,
		})
		if err != nil {
			return nil, err
		}
		pages = append(pages, Page{
			URL:          collection.URL,
			CanonicalURL: collection.CanonicalURL,
			TemplateName: "taxonomy",
			Title:        strings.Title(collection.Name),
			Content:      landing,
		})

		for _, term := range collection.Terms {
			out, err := renderTemplate(set, "taxonomy", templateContext{
				Title:        term.Name,
				HomeURL:      "/",
				CanonicalURL: term.CanonicalURL,
				Config:       cfg,
				Index:        index,
				Pages:        term.Items,
				Taxonomy:     collection,
				Term:         term,
			})
			if err != nil {
				return nil, err
			}
			pages = append(pages, Page{
				URL:          term.URL,
				CanonicalURL: term.CanonicalURL,
				TemplateName: "taxonomy",
				Title:        term.Name,
				Content:      out,
			})
		}
	}
	return pages, nil
}

func renderTemplate(set templates.Set, name string, data templateContext) (string, error) {
	out, err := set.Execute(name, data)
	if err != nil {
		return "", fmt.Errorf("render %s page: %w", name, err)
	}
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
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
