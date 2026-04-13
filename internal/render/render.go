package render

import (
	"fmt"
	"html/template"
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

	notFoundPage, err := renderNotFoundPage(set, cfg, state.Index)
	if err != nil {
		return nil, err
	}
	pages = append(pages, notFoundPage)

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

func renderNotFoundPage(set templates.Set, cfg config.SiteConfig, index site.SiteIndex) (Page, error) {
	title := "Page not found"
	canonical := canonicalURL(cfg.BaseURL, "/404.html")

	if set.Has("404") {
		out, err := renderTemplate(set, "404", templateContext{
			Title:        title,
			HomeURL:      "/",
			CanonicalURL: canonical,
			Config:       cfg,
			Index:        index,
		})
		if err != nil {
			return Page{}, err
		}
		return Page{
			URL:          "/404.html",
			CanonicalURL: canonical,
			TemplateName: "404",
			Title:        title,
			Content:      out,
		}, nil
	}

	return Page{
		URL:          "/404.html",
		CanonicalURL: canonical,
		TemplateName: "builtin-404",
		Title:        title,
		Content:      defaultNotFoundHTML(cfg, canonical, title),
	}, nil
}

func defaultNotFoundHTML(cfg config.SiteConfig, canonicalURL, title string) string {
	siteTitle := template.HTMLEscapeString(cfg.Title)
	if strings.TrimSpace(siteTitle) == "" {
		siteTitle = "Site"
	}
	pageTitle := template.HTMLEscapeString(title)
	description := template.HTMLEscapeString(cfg.Description)
	language := template.HTMLEscapeString(defaultLanguage(cfg.Language))
	direction := template.HTMLEscapeString(config.DocumentDirection(cfg.Language))

	var b strings.Builder
	b.WriteString("<!doctype html>\n")
	b.WriteString("<html lang=\"")
	b.WriteString(language)
	b.WriteString("\" dir=\"")
	b.WriteString(direction)
	b.WriteString("\">\n")
	b.WriteString("<head>\n")
	b.WriteString("  <meta charset=\"utf-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("  <title>")
	b.WriteString(pageTitle)
	b.WriteString(" | ")
	b.WriteString(siteTitle)
	b.WriteString("</title>\n")
	b.WriteString("  <meta name=\"robots\" content=\"noindex\">\n")
	if canonicalURL != "" {
		b.WriteString("  <link rel=\"canonical\" href=\"")
		b.WriteString(template.HTMLEscapeString(canonicalURL))
		b.WriteString("\">\n")
	}
	if description != "" {
		b.WriteString("  <meta name=\"description\" content=\"")
		b.WriteString(description)
		b.WriteString("\">\n")
	}
	b.WriteString("</head>\n")
	b.WriteString("<body>\n")
	b.WriteString("  <main>\n")
	b.WriteString("    <h1>")
	b.WriteString(pageTitle)
	b.WriteString("</h1>\n")
	b.WriteString("    <p>The page you requested could not be found.</p>\n")
	b.WriteString("    <p><a href=\"/\">Return to the homepage</a></p>\n")
	b.WriteString("  </main>\n")
	b.WriteString("</body>\n")
	b.WriteString("</html>\n")
	return b.String()
}

func defaultLanguage(value string) string {
	if strings.TrimSpace(value) == "" {
		return "en"
	}
	return value
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
