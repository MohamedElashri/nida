package render

import (
	"fmt"
	"html"
	"slices"
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
	pages := make([]Page, 0, len(state.Index.SectionLookup)*3+len(state.Index.AllPages)+5)

	// Render all sections from SectionLookup to ensure child sections are included
	sectionPaths := make([]string, 0, len(state.Index.SectionLookup))
	for path := range state.Index.SectionLookup {
		sectionPaths = append(sectionPaths, path)
	}
	slices.Sort(sectionPaths)

	for _, path := range sectionPaths {
		section := state.Index.SectionLookup[path]
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

	sectionURL := "/" + section.SectionPath + "/"
	if section.SectionPath == "" {
		sectionURL = "/"
	}

	var ctxPages []content.Page
	if section.SectionPath == "" {
		ctxPages = latestItems(index.AllPages, nil, 5)
	} else {
		ctxPages = section.Pages
	}

	perPage := section.PaginateBy
	if perPage <= 0 {
		perPage = cfg.Sections.PaginateBy
	}
	if perPage <= 0 {
		perPage = cfg.Paginate
	}

	var out string
	var err error

	// When pagination is disabled, render the section once without a paginator.
	// For sections with pagination, paginated page 1 occupies sectionURL, so 
	// rendering a separate non-paginated page would produce a URL conflict.
	paginateSection := perPage > 0
	if !paginateSection && set.Has(templateName) {
		canonical := canonicalURL(cfg.BaseURL, "/"+section.SectionPath)

		ctx := templateContext{
			Title:        section.Title,
			Description:  section.Description,
			HomeURL:      "/",
			CurrentURL:   sectionURL,
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

	if paginateSection {
		var paginatePages []content.Page
		if section.SectionPath == "" {
			paginatePages = index.AllPages
		} else {
			paginatePages = section.Pages
		}

		totalPages := max(1, (len(paginatePages)+perPage-1)/perPage)
		paginatePath := section.PaginatePath
		if paginatePath == "" {
			paginatePath = "page"
		}

		for pageNum := 1; pageNum <= totalPages; pageNum++ {
			start := (pageNum - 1) * perPage
			end := min(start+perPage, len(paginatePages))
			pageURL := sectionURL
			if pageNum > 1 {
				pageURL = sectionURL + paginatePath + "/" + strconv.Itoa(pageNum) + "/"
			}

			paginator := buildPaginator(sectionURL, pageNum, totalPages, paginatePages[start:end])

			ctx := templateContext{
				Title:        section.Title,
				Description:  section.Description,
				HomeURL:      "/",
				CurrentURL:   pageURL,
				CanonicalURL: canonicalURL(cfg.BaseURL, pageURL),
				Config:       cfg,
				Theme:        theme,
				Index:        index,
				Section:      section,
				Pages:        paginatePages[start:end],
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

		// Generate page/1/ redirect to canonical section URL
		pageOneURL := sectionURL + paginatePath + "/1/"
		canonicalSectionURL := canonicalURL(cfg.BaseURL, sectionURL)
		pages = append(pages, Page{
			URL:          pageOneURL,
			CanonicalURL: canonicalSectionURL,
			TemplateName: "redirect",
			Title:        "Redirect",
			Content:      redirectHTML(canonicalSectionURL),
		})
	}

	return pages, nil
}

func sectionTemplateName(set templates.Set, section content.Section) string {
	tmpl := normalizeTemplateName(section.Template)
	if tmpl != "" && set.Has(tmpl) {
		return tmpl
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
			CurrentURL:   page.URL,
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

		// Generate alias redirect pages
		for _, alias := range page.Aliases {
			aliasURL := alias
			if !strings.HasPrefix(aliasURL, "/") {
				aliasURL = "/" + aliasURL
			}
			if !strings.HasSuffix(aliasURL, "/") {
				aliasURL += "/"
			}
			aliasURL, err = site.NormalizeRoute("alias", aliasURL)
			if err != nil {
				return nil, fmt.Errorf("invalid alias %q for page %q: %w", alias, page.RelativePath, err)
			}
			targetURL := canonicalURL(cfg.BaseURL, page.URL)
			out = append(out, Page{
				URL:          aliasURL,
				CanonicalURL: targetURL,
				TemplateName: "redirect",
				Title:        "Redirect",
				Content:      redirectHTML(targetURL),
			})
		}
	}

	return out, nil
}

func pageTemplateName(set templates.Set, index site.SiteIndex, page content.Page) string {
	tmpl := normalizeTemplateName(page.Template)
	if tmpl != "" && set.Has(tmpl) {
		return tmpl
	}
	if section, ok := index.SectionLookup[page.SectionPath]; ok {
		ptmpl := normalizeTemplateName(section.PageTemplate)
		if ptmpl != "" && set.Has(ptmpl) {
			return ptmpl
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
				CurrentURL:   collection.URL,
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

		perPage := collection.PaginateBy
		paginatePath := collection.PaginatePath
		if paginatePath == "" {
			paginatePath = "page"
		}

		for _, term := range collection.Terms {
			if perPage <= 0 {
				ctx := templateContext{
					Title:        term.Name,
					Description:  term.Name,
					HomeURL:      "/",
					CurrentURL:   term.URL,
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
				continue
			}

			totalPages := max(1, (len(term.Items)+perPage-1)/perPage)
			for pageNum := 1; pageNum <= totalPages; pageNum++ {
				start := (pageNum - 1) * perPage
				end := min(start+perPage, len(term.Items))
				pageURL := term.URL
				if pageNum > 1 {
					pageURL = term.URL + paginatePath + "/" + strconv.Itoa(pageNum) + "/"
				}

				paginator := buildPaginator(term.URL, pageNum, totalPages, term.Items[start:end])

				ctx := templateContext{
					Title:        term.Name,
					Description:  term.Name,
					HomeURL:      "/",
					CurrentURL:   pageURL,
					CanonicalURL: canonicalURL(cfg.BaseURL, pageURL),
					Config:       cfg,
					Theme:        theme,
					Index:        index,
					Taxonomy:     collection,
					Term:         term,
					Pages:        term.Items[start:end],
					Paginator:    paginator,
					Robots:       "noai, noimageai",
				}

				rendered, err := renderTemplate(set, singleTemplate, ctx)
				if err != nil {
					return nil, fmt.Errorf("render taxonomy term %q page %d: %w", term.Name, pageNum, err)
				}

				pages = append(pages, Page{
					URL:          pageURL,
					CanonicalURL: canonicalURL(cfg.BaseURL, pageURL),
					TemplateName: singleTemplate,
					Title:        term.Name,
					Content:      rendered,
				})
			}

			// Generate page/1/ redirect to canonical term URL
			pageOneURL := term.URL + paginatePath + "/1/"
			pages = append(pages, Page{
				URL:          pageOneURL,
				CanonicalURL: term.CanonicalURL,
				TemplateName: "redirect",
				Title:        "Redirect",
				Content:      redirectHTML(term.CanonicalURL),
			})
		}
	}

	return pages, nil
}

func renderTemplate(set templates.Set, name string, data templateContext) (string, error) {
	data.BasePath = config.BasePath(data.Config.BaseURL)
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
			CurrentURL:   "/404.html",
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
	homeURL := config.BasePath(cfg.BaseURL) + "/"

	var b strings.Builder
	b.WriteString("<!doctype html>\n")
	b.WriteString(`<html lang="` + html.EscapeString(language) + `" dir="` + html.EscapeString(direction) + `">`)
	b.WriteString("<head>")
	b.WriteString(`<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">`)
	b.WriteString(`<title>` + html.EscapeString(pageTitle) + `</title>`)
	b.WriteString(`<meta name="robots" content="noindex">`)
	b.WriteString(`<link rel="canonical" href="` + html.EscapeString(canonicalURL) + `">`)
	b.WriteString("</head><body><main><h1>Page not found</h1><p>The page you requested could not be found.</p><p><a href=\"" + html.EscapeString(homeURL) + "\">Return to the homepage</a></p></main></body></html>\n")
	return b.String()
}
