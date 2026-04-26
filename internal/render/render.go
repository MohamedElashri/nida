package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
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

func renderAll(set templates.Set, cfg config.SiteConfig, theme Theme, state site.State) ([]Page, error) {
	pages := make([]Page, 0, len(state.Index.Sections)*3+len(state.Index.AllPages)+5)

	for _, section := range state.Index.Sections {
		sectionPages, err := renderSectionPages(set, cfg, theme, state.Index, section)
		if err != nil {
			return nil, err
		}
		pages = append(pages, sectionPages...)
	}

	pagePages, err := renderPages(set, cfg, theme, state.Index, state.Pages)
	if err != nil {
		return nil, err
	}
	pages = append(pages, pagePages...)

	taxPages, err := renderTaxonomyPages(set, cfg, theme, state.Index)
	if err != nil {
		return nil, err
	}
	pages = append(pages, taxPages...)

	notFound, err := renderNotFoundPage(set, cfg, theme)
	if err != nil {
		return nil, err
	}
	pages = append(pages, notFound)

	if cfg.MinifyHTML {
		for i := range pages {
			pages[i].Content = minifyHTML(pages[i].Content)
		}
	}

	return pages, nil
}

func renderSectionPages(set templates.Set, cfg config.SiteConfig, theme Theme, index site.SiteIndex, section content.Section) ([]Page, error) {
	var pages []Page

	templateName := sectionTemplateName(set, section)

	sectionURL := "/" + section.SectionPath
	if section.SectionPath == "" {
		sectionURL = "/"
	}

	var ctxPages []content.Page
	if section.SectionPath == "" {
		ctxPages = latestItems(index.AllPages, nil, 5)
	} else {
		ctxPages = section.Pages
	}

	var out string
	var err error
	if set.Has(templateName) {
		canonical := canonicalURL(cfg.BaseURL, "/"+section.SectionPath)

		ctx := templateContext{
			Title:        section.Title,
			Description:  section.Description,
			HomeURL:      "/",
			CanonicalURL: canonical,
			Config:       cfg,
			Theme:        theme,
			Index:        index,
			Section:      section,
			Pages:        ctxPages,
			Robots:       "noai, noimageai",
		}

		out, err = renderTemplate(set, templateName, ctx)
		if err != nil {
			return nil, fmt.Errorf("render section %q: %w", section.SectionPath, err)
		}
	}

	if out != "" {
		pages = append(pages, Page{
			URL:          sectionURL,
			CanonicalURL: canonicalURL(cfg.BaseURL, sectionURL),
			TemplateName: templateName,
			Title:        section.Title,
			Content:      out,
		})
	}

	perPage := section.PaginateBy
	if perPage <= 0 {
		perPage = cfg.Sections.PaginateBy
	}
	if perPage <= 0 {
		perPage = cfg.Paginate
	}

	if perPage > 0 && len(section.Pages) > perPage {
		totalPages := max(1, (len(section.Pages)+perPage-1)/perPage)
		paginatePath := section.PaginatePath
		if paginatePath == "" {
			paginatePath = "page"
		}

		for pageNum := 1; pageNum <= totalPages; pageNum++ {
			start := (pageNum - 1) * perPage
			end := min(start+perPage, len(section.Pages))
			pageURL := sectionURL
			if pageNum > 1 {
				pageURL = sectionURL + paginatePath + "/" + strconv.Itoa(pageNum) + "/"
			}

			paginator := buildPaginator(sectionURL, pageNum, totalPages, section.Pages[start:end])

			ctx := templateContext{
				Title:        section.Title,
				Description:  section.Description,
				HomeURL:      "/",
				CanonicalURL: canonicalURL(cfg.BaseURL, pageURL),
				Config:       cfg,
				Theme:        theme,
				Index:        index,
				Section:      section,
				Pages:        section.Pages[start:end],
				Paginator:    paginator,
				Robots:       "noai, noimageai",
			}

			out, err = renderTemplate(set, templateName, ctx)
			if err != nil {
				return nil, fmt.Errorf("render section %q page %d: %w", section.SectionPath, pageNum, err)
			}

			pages = append(pages, Page{
				URL:          pageURL,
				CanonicalURL: canonicalURL(cfg.BaseURL, pageURL),
				TemplateName: templateName,
				Title:        section.Title,
				Content:      out,
			})
		}
	}

	return pages, nil
}

func sectionTemplateName(set templates.Set, section content.Section) string {
	if section.Template != "" && set.Has(section.Template) {
		return section.Template
	}
	if section.SectionPath == "" && set.Has("index") {
		return "index"
	}
	if set.Has("section") {
		return "section"
	}
	return "list"
}

func renderPages(set templates.Set, cfg config.SiteConfig, theme Theme, index site.SiteIndex, pages []content.Page) ([]Page, error) {
	var out []Page

	for _, page := range pages {
		templateName := pageTemplateName(set, index, page)
		canonical := canonicalURL(cfg.BaseURL, page.URL)

		ctx := templateContext{
			Title:        page.Title,
			Description:  page.Description,
			HomeURL:      "/",
			CanonicalURL: canonical,
			Config:       cfg,
			Theme:        theme,
			Index:        index,
			Page:         page,
			Section:      index.SectionLookup[page.SectionPath],
			Robots:       "noai, noimageai",
		}

		rendered, err := renderTemplate(set, templateName, ctx)
		if err != nil {
			return nil, fmt.Errorf("render page %q: %w", page.RelativePath, err)
		}

		out = append(out, Page{
			URL:          page.URL,
			CanonicalURL: canonical,
			TemplateName: templateName,
			Title:        page.Title,
			Content:      rendered,
		})
	}

	return out, nil
}

func pageTemplateName(set templates.Set, index site.SiteIndex, page content.Page) string {
	if page.Template != "" && set.Has(page.Template) {
		return page.Template
	}
	if section, ok := index.SectionLookup[page.SectionPath]; ok {
		if section.PageTemplate != "" && set.Has(section.PageTemplate) {
			return section.PageTemplate
		}
	}
	if set.Has("post") {
		return "post"
	}
	return "page"
}

func renderTaxonomyPages(set templates.Set, cfg config.SiteConfig, theme Theme, index site.SiteIndex) ([]Page, error) {
	var pages []Page

	for _, collection := range index.Taxonomies {
		if !collection.Render {
			continue
		}

		listTemplate := pickExistingTemplate(set, "taxonomy_list", "taxonomy")
		singleTemplate := pickExistingTemplate(set, "taxonomy_single", "taxonomy")

		if listTemplate != "" {
			ctx := templateContext{
				Title:        collection.Name,
				Description:  collection.Name,
				HomeURL:      "/",
				CanonicalURL: collection.CanonicalURL,
				Config:       cfg,
				Theme:        theme,
				Index:        index,
				Taxonomy:     collection,
				Terms:        collection.Terms,
				Robots:       "noai, noimageai",
			}

			rendered, err := renderTemplate(set, listTemplate, ctx)
			if err != nil {
				return nil, fmt.Errorf("render taxonomy list %q: %w", collection.Name, err)
			}

			pages = append(pages, Page{
				URL:          collection.URL,
				CanonicalURL: collection.CanonicalURL,
				TemplateName: listTemplate,
				Title:        collection.Name,
				Content:      rendered,
			})
		}

		if singleTemplate == "" {
			continue
		}

		for _, term := range collection.Terms {
			ctx := templateContext{
				Title:        term.Name,
				Description:  term.Name,
				HomeURL:      "/",
				CanonicalURL: term.CanonicalURL,
				Config:       cfg,
				Theme:        theme,
				Index:        index,
				Taxonomy:     collection,
				Term:         term,
				Pages:        term.Items,
				Robots:       "noai, noimageai",
			}

			rendered, err := renderTemplate(set, singleTemplate, ctx)
			if err != nil {
				return nil, fmt.Errorf("render taxonomy term %q: %w", term.Name, err)
			}

			pages = append(pages, Page{
				URL:          term.URL,
				CanonicalURL: term.CanonicalURL,
				TemplateName: singleTemplate,
				Title:        term.Name,
				Content:      rendered,
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

func renderNotFoundPage(set templates.Set, cfg config.SiteConfig, theme Theme) (Page, error) {
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


