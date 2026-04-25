package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
	"github.com/MohamedElashri/nida/internal/taxonomies"
	"github.com/MohamedElashri/nida/internal/templates"
)

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

	theme, err := buildTheme(siteRoot, cfg)
	if err != nil {
		return nil, err
	}

	return renderAll(set, cfg, theme, state)
}

func RenderIncremental(siteRoot string, cfg config.SiteConfig, state site.State, prevPages []Page, changedItems []content.Item, removedPaths []string) ([]Page, error) {
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

	theme, err := buildTheme(siteRoot, cfg)
	if err != nil {
		return nil, err
	}

	affected := computeAffected(state.Index, changedItems, removedPaths)

	prevByURL := make(map[string]Page, len(prevPages))
	for _, p := range prevPages {
		prevByURL[p.URL] = p
	}

	pages := make([]Page, 0, len(prevPages)+4)

	// Homepage
	if affected["/"] || prevByURL["/"].Content == "" {
		home, err := renderHomePage(set, cfg, theme, state)
		if err != nil {
			return nil, err
		}
		pages = append(pages, home)
		affected["/"] = false
	}

	// Posts
	for _, item := range state.Index.Posts {
		if affected[item.URL] {
			templateName := templateForItem(set, state.Index, item, "post")
			out, err := renderTemplate(set, templateName, templateContext{
				Title:        item.Title,
				Description:  firstNonEmpty(item.Description, cfg.Description),
				HomeURL:      "/",
				CanonicalURL: state.Index.CanonicalLookup[item.URL],
				Config:       cfg,
				Theme:        theme,
				Index:        state.Index,
				Page:         item,
				Section:      state.Index.SectionLookup[item.SectionPath],
				Robots:       "noai, noimageai",
			})
			if err != nil {
				return nil, err
			}
			pages = append(pages, Page{
				URL:          item.URL,
				CanonicalURL: state.Index.CanonicalLookup[item.URL],
				TemplateName: templateName,
				Title:        item.Title,
				Content:      out,
			})
		} else if prev, ok := prevByURL[item.URL]; ok {
			pages = append(pages, prev)
		}
	}

	// Pages
	for _, item := range state.Index.Pages {
		if affected[item.URL] {
			templateName := templateForItem(set, state.Index, item, "page")
			out, err := renderTemplate(set, templateName, templateContext{
				Title:        item.Title,
				Description:  firstNonEmpty(item.Description, cfg.Description),
				HomeURL:      "/",
				CanonicalURL: state.Index.CanonicalLookup[item.URL],
				Config:       cfg,
				Theme:        theme,
				Index:        state.Index,
				Page:         item,
				Section:      state.Index.SectionLookup[item.SectionPath],
				Robots:       "noai, noimageai",
			})
			if err != nil {
				return nil, err
			}
			pages = append(pages, Page{
				URL:          item.URL,
				CanonicalURL: state.Index.CanonicalLookup[item.URL],
				TemplateName: templateName,
				Title:        item.Title,
				Content:      out,
			})
		} else if prev, ok := prevByURL[item.URL]; ok {
			pages = append(pages, prev)
		}
	}

	// Sections (skip root section, handled by homepage)
	for _, section := range state.Index.Sections {
		if section.SectionPath == "" {
			continue
		}
		perPage := section.PaginateBy
		if perPage <= 0 {
			perPage = cfg.Paginate
		}
		if perPage <= 0 {
			perPage = 10
		}
		totalPages := max(1, (len(section.Pages)+perPage-1)/perPage)
		sectionAffected := affected[section.URL]
		for pageNum := 1; pageNum <= totalPages; pageNum++ {
			route := section.URL
			if pageNum > 1 {
				route = strings.TrimSuffix(section.URL, "/") + "/page/" + strconv.Itoa(pageNum) + "/"
			}
			if !sectionAffected && !affected[route] {
				if prev, ok := prevByURL[route]; ok {
					pages = append(pages, prev)
					continue
				}
			}
			start := (pageNum - 1) * perPage
			end := min(start+perPage, len(section.Pages))
			paginator := buildPaginator(section.URL, pageNum, totalPages, section.Pages[start:end])
			templateName := templateForSection(set, section)
			out, err := renderTemplate(set, templateName, templateContext{
				Title:        section.Title,
				Description:  firstNonEmpty(section.Description, cfg.Description),
				HomeURL:      "/",
				CanonicalURL: canonicalURL(cfg.BaseURL, route),
				Config:       cfg,
				Theme:        theme,
				Index:        state.Index,
				Section:      section,
				Pages:        section.Pages[start:end],
				Paginator:    paginator,
				Robots:       "noai, noimageai",
			})
			if err != nil {
				return nil, err
			}
			pages = append(pages, Page{
				URL:          route,
				CanonicalURL: canonicalURL(cfg.BaseURL, route),
				TemplateName: templateName,
				Title:        section.Title,
				Content:      out,
			})
			if pageNum == 1 && section.PaginateBy > 0 {
				aliasRoute := strings.TrimSuffix(section.URL, "/") + "/page/1/"
				pages = append(pages, Page{
					URL:          aliasRoute,
					CanonicalURL: canonicalURL(cfg.BaseURL, section.URL),
					TemplateName: templateName,
					Title:        section.Title,
					Content:      out,
				})
			}
		}
	}

	// Taxonomy pages
	taxPages, err := renderAffectedTaxonomyPages(set, cfg, theme, state.Index, affected, prevByURL)
	if err != nil {
		return nil, err
	}
	pages = append(pages, taxPages...)

	// 404 page
	if prev, ok := prevByURL["/404.html"]; ok {
		pages = append(pages, prev)
	} else {
		notFoundPage, err := renderNotFoundPage(set, cfg, theme, state.Index)
		if err != nil {
			return nil, err
		}
		pages = append(pages, notFoundPage)
	}

	if cfg.MinifyHTML {
		for i := range pages {
			pages[i].Content = minifyHTML(pages[i].Content)
		}
	}

	return pages, nil
}

func renderAll(set templates.Set, cfg config.SiteConfig, theme Theme, state site.State) ([]Page, error) {
	pages := make([]Page, 0, len(state.Index.Posts)+len(state.Index.Pages)+len(state.Index.Sections)+4)

	home, err := renderHomePage(set, cfg, theme, state)
	if err != nil {
		return nil, err
	}
	pages = append(pages, home)

	contentPages, err := renderContentPages(set, cfg, theme, state)
	if err != nil {
		return nil, err
	}
	pages = append(pages, contentPages...)

	sectionPages, err := renderSectionPages(set, cfg, theme, state)
	if err != nil {
		return nil, err
	}
	pages = append(pages, sectionPages...)

	taxonomyPages, err := renderTaxonomyPages(set, cfg, theme, state.Index)
	if err != nil {
		return nil, err
	}
	pages = append(pages, taxonomyPages...)

	notFoundPage, err := renderNotFoundPage(set, cfg, theme, state.Index)
	if err != nil {
		return nil, err
	}
	pages = append(pages, notFoundPage)

	if cfg.MinifyHTML {
		for i := range pages {
			pages[i].Content = minifyHTML(pages[i].Content)
		}
	}

	return pages, nil
}

func renderHomePage(set templates.Set, cfg config.SiteConfig, theme Theme, state site.State) (Page, error) {
	title := cfg.Title
	description := cfg.Description
	if state.Index.RootSection != nil {
		if strings.TrimSpace(state.Index.RootSection.Title) != "" {
			title = state.Index.RootSection.Title
		}
		if strings.TrimSpace(state.Index.RootSection.Description) != "" {
			description = state.Index.RootSection.Description
		}
	}

	out, err := renderTemplate(set, "index", templateContext{
		Title:        title,
		Description:  description,
		HomeURL:      "/",
		CanonicalURL: canonicalURL(cfg.BaseURL, "/"),
		Config:       cfg,
		Theme:        theme,
		Index:        state.Index,
		Pages:        latestItems(state.Index.Posts, config.MainSections(cfg), 5),
		Section:      derefSection(state.Index.RootSection),
		Robots:       "noai, noimageai",
	})
	if err != nil {
		return Page{}, err
	}

	return Page{
		URL:          "/",
		CanonicalURL: canonicalURL(cfg.BaseURL, "/"),
		TemplateName: "index",
		Title:        title,
		Content:      out,
	}, nil
}

func renderContentPages(set templates.Set, cfg config.SiteConfig, theme Theme, state site.State) ([]Page, error) {
	pages := make([]Page, 0, len(state.Index.Posts)+len(state.Index.Pages))

	for _, item := range state.Index.Posts {
		templateName := templateForItem(set, state.Index, item, "post")
		out, err := renderTemplate(set, templateName, templateContext{
			Title:        item.Title,
			Description:  firstNonEmpty(item.Description, cfg.Description),
			HomeURL:      "/",
			CanonicalURL: state.Index.CanonicalLookup[item.URL],
			Config:       cfg,
			Theme:        theme,
			Index:        state.Index,
			Page:         item,
			Section:      state.Index.SectionLookup[item.SectionPath],
			Robots:       "noai, noimageai",
		})
		if err != nil {
			return nil, err
		}
		pages = append(pages, Page{
			URL:          item.URL,
			CanonicalURL: state.Index.CanonicalLookup[item.URL],
			TemplateName: templateName,
			Title:        item.Title,
			Content:      out,
		})
	}

	for _, item := range state.Index.Pages {
		templateName := templateForItem(set, state.Index, item, "page")
		out, err := renderTemplate(set, templateName, templateContext{
			Title:        item.Title,
			Description:  firstNonEmpty(item.Description, cfg.Description),
			HomeURL:      "/",
			CanonicalURL: state.Index.CanonicalLookup[item.URL],
			Config:       cfg,
			Theme:        theme,
			Index:        state.Index,
			Page:         item,
			Section:      state.Index.SectionLookup[item.SectionPath],
			Robots:       "noai, noimageai",
		})
		if err != nil {
			return nil, err
		}
		pages = append(pages, Page{
			URL:          item.URL,
			CanonicalURL: state.Index.CanonicalLookup[item.URL],
			TemplateName: templateName,
			Title:        item.Title,
			Content:      out,
		})
	}

	return pages, nil
}

func renderSectionPages(set templates.Set, cfg config.SiteConfig, theme Theme, state site.State) ([]Page, error) {
	pages := make([]Page, 0, len(state.Index.Sections))

	for _, section := range state.Index.Sections {
		if section.SectionPath == "" {
			continue
		}

		templateName := templateForSection(set, section)

		perPage := section.PaginateBy
		if perPage <= 0 {
			perPage = cfg.Paginate
		}
		if perPage <= 0 {
			perPage = 10
		}

		totalPages := max(1, (len(section.Pages)+perPage-1)/perPage)
		for pageNum := 1; pageNum <= totalPages; pageNum++ {
			start := (pageNum - 1) * perPage
			end := min(start+perPage, len(section.Pages))
			route := section.URL
			if pageNum > 1 {
				route = strings.TrimSuffix(section.URL, "/") + "/page/" + strconv.Itoa(pageNum) + "/"
			}
			paginator := buildPaginator(section.URL, pageNum, totalPages, section.Pages[start:end])
			out, err := renderTemplate(set, templateName, templateContext{
				Title:        section.Title,
				Description:  firstNonEmpty(section.Description, cfg.Description),
				HomeURL:      "/",
				CanonicalURL: canonicalURL(cfg.BaseURL, route),
				Config:       cfg,
				Theme:        theme,
				Index:        state.Index,
				Section:      section,
				Pages:        section.Pages[start:end],
				Paginator:    paginator,
				Robots:       "noai, noimageai",
			})
			if err != nil {
				return nil, err
			}
			pages = append(pages, Page{
				URL:          route,
				CanonicalURL: canonicalURL(cfg.BaseURL, route),
				TemplateName: templateName,
				Title:        section.Title,
				Content:      out,
			})
			if pageNum == 1 && section.PaginateBy > 0 {
				aliasRoute := strings.TrimSuffix(section.URL, "/") + "/page/1/"
				pages = append(pages, Page{
					URL:          aliasRoute,
					CanonicalURL: canonicalURL(cfg.BaseURL, section.URL),
					TemplateName: templateName,
					Title:        section.Title,
					Content:      out,
				})
			}
		}
	}

	return pages, nil
}

func renderTaxonomyPages(set templates.Set, cfg config.SiteConfig, theme Theme, index site.SiteIndex) ([]Page, error) {
	pages := make([]Page, 0)
	listTemplate := pickExistingTemplate(set, "taxonomy_list", "taxonomy")
	singleTemplate := pickExistingTemplate(set, "taxonomy_single", "taxonomy")

	for _, collection := range []taxonomies.Collection{index.Tags, index.Categories} {
		if collection.Name == "" {
			continue
		}

		if listTemplate != "" {
			landing, err := renderTemplate(set, listTemplate, templateContext{
				Title:        strings.Title(collection.Name),
				Description:  strings.Title(collection.Name),
				HomeURL:      "/",
				CanonicalURL: collection.CanonicalURL,
				Config:       cfg,
				Theme:        theme,
				Index:        index,
				Terms:        collection.Terms,
				Taxonomy:     collection,
				Robots:       "noai, noimageai",
			})
			if err != nil {
				return nil, err
			}
			pages = append(pages, Page{
				URL:          collection.URL,
				CanonicalURL: collection.CanonicalURL,
				TemplateName: listTemplate,
				Title:        strings.Title(collection.Name),
				Content:      landing,
			})
		}

		if singleTemplate == "" {
			continue
		}
		for _, term := range collection.Terms {
			out, err := renderTemplate(set, singleTemplate, templateContext{
				Title:        term.Name,
				Description:  term.Name,
				HomeURL:      "/",
				CanonicalURL: term.CanonicalURL,
				Config:       cfg,
				Theme:        theme,
				Index:        index,
				Pages:        term.Items,
				Taxonomy:     collection,
				Term:         term,
				Robots:       "noai, noimageai",
			})
			if err != nil {
				return nil, err
			}
			pages = append(pages, Page{
				URL:          term.URL,
				CanonicalURL: term.CanonicalURL,
				TemplateName: singleTemplate,
				Title:        term.Name,
				Content:      out,
			})
		}
	}
	return pages, nil
}

func renderAffectedTaxonomyPages(set templates.Set, cfg config.SiteConfig, theme Theme, index site.SiteIndex, affected map[string]bool, prevByURL map[string]Page) ([]Page, error) {
	pages := make([]Page, 0)
	listTemplate := pickExistingTemplate(set, "taxonomy_list", "taxonomy")
	singleTemplate := pickExistingTemplate(set, "taxonomy_single", "taxonomy")

	for _, collection := range []taxonomies.Collection{index.Tags, index.Categories} {
		if collection.Name == "" {
			continue
		}

		if listTemplate != "" && affected[collection.URL] {
			landing, err := renderTemplate(set, listTemplate, templateContext{
				Title:        strings.Title(collection.Name),
				Description:  strings.Title(collection.Name),
				HomeURL:      "/",
				CanonicalURL: collection.CanonicalURL,
				Config:       cfg,
				Theme:        theme,
				Index:        index,
				Terms:        collection.Terms,
				Taxonomy:     collection,
				Robots:       "noai, noimageai",
			})
			if err != nil {
				return nil, err
			}
			pages = append(pages, Page{
				URL:          collection.URL,
				CanonicalURL: collection.CanonicalURL,
				TemplateName: listTemplate,
				Title:        strings.Title(collection.Name),
				Content:      landing,
			})
		} else if listTemplate != "" {
			if prev, ok := prevByURL[collection.URL]; ok {
				pages = append(pages, prev)
			}
		}

		if singleTemplate == "" {
			continue
		}
		for _, term := range collection.Terms {
			if affected[term.URL] {
				out, err := renderTemplate(set, singleTemplate, templateContext{
					Title:        term.Name,
					Description:  term.Name,
					HomeURL:      "/",
					CanonicalURL: term.CanonicalURL,
					Config:       cfg,
					Theme:        theme,
					Index:        index,
					Pages:        term.Items,
					Taxonomy:     collection,
					Term:         term,
					Robots:       "noai, noimageai",
				})
				if err != nil {
					return nil, err
				}
				pages = append(pages, Page{
					URL:          term.URL,
					CanonicalURL: term.CanonicalURL,
					TemplateName: singleTemplate,
					Title:        term.Name,
					Content:      out,
				})
			} else if prev, ok := prevByURL[term.URL]; ok {
				pages = append(pages, prev)
			}
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

func renderNotFoundPage(set templates.Set, cfg config.SiteConfig, theme Theme, index site.SiteIndex) (Page, error) {
	title := "Page not found"
	canonical := canonicalURL(cfg.BaseURL, "/404.html")
	if set.Has("404") {
		out, err := renderTemplate(set, "404", templateContext{
			Title:        title,
			Description:  cfg.Description,
			HomeURL:      "/",
			CanonicalURL: canonical,
			Config:       cfg,
			Theme:        theme,
			Index:        index,
			Robots:       "noindex, noai, noimageai",
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
	pageTitle := title
	if strings.TrimSpace(cfg.Title) != "" {
		pageTitle = title + " | " + cfg.Title
	}
	language := defaultLanguage(cfg.Language)
	direction := config.DocumentDirection(cfg.Language)

	var b strings.Builder
	b.WriteString("<!doctype html>\n")
	b.WriteString(`<html lang="` + language + `" dir="` + direction + `">`)
	b.WriteString("<head>")
	b.WriteString(`<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">`)
	b.WriteString(`<title>` + pageTitle + `</title>`)
	b.WriteString(`<meta name="robots" content="noindex">`)
	b.WriteString(`<link rel="canonical" href="` + canonicalURL + `">`)
	b.WriteString("</head><body><main><h1>Page not found</h1><p>The page you requested could not be found.</p><p><a href=\"/\">Return to the homepage</a></p></main></body></html>\n")
	return b.String()
}
