package config

import (
	"strings"

	"github.com/MohamedElashri/nida/internal/safepath"
)

func normalize(cfg *SiteConfig) {
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Title = strings.TrimSpace(cfg.Title)
	cfg.Description = strings.TrimSpace(cfg.Description)
	cfg.Language = strings.TrimSpace(cfg.Language)
	cfg.Author = strings.TrimSpace(cfg.Author)
	cfg.ContentDir = cleanRelativePath(cfg.ContentDir)
	cfg.TemplateDir = cleanRelativePath(cfg.TemplateDir)
	cfg.StaticDir = cleanRelativePath(cfg.StaticDir)
	cfg.OutputDir = cleanRelativePath(cfg.OutputDir)
	cfg.ThemesDir = cleanRelativePath(cfg.ThemesDir)
	cfg.Theme = strings.TrimSpace(cfg.Theme)
	cfg.SyntaxTheme = strings.TrimSpace(cfg.SyntaxTheme)
	cfg.RSS.Filename = cleanRelativePath(cfg.RSS.Filename)
	cfg.Atom.Filename = cleanRelativePath(cfg.Atom.Filename)
	cfg.Sitemap.Filename = cleanRelativePath(cfg.Sitemap.Filename)
	cfg.Robots.Filename = cleanRelativePath(cfg.Robots.Filename)
	cfg.Search.Filename = cleanRelativePath(cfg.Search.Filename)
	cfg.Pipeline.SCSS.EntryDir = cleanRelativePath(cfg.Pipeline.SCSS.EntryDir)
	cfg.Server.Host = strings.TrimSpace(cfg.Server.Host)

	if cfg.Sections.PaginatePath == "" {
		cfg.Sections.PaginatePath = "page"
	}
	if cfg.Sections.DefaultSortBy == "" {
		cfg.Sections.DefaultSortBy = "date"
	}

	if cfg.Taxonomies == nil {
		cfg.Taxonomies = []TaxonomyConfig{}
	}
}

func cleanRelativePath(value string) string {
	return safepath.CleanRelative(value)
}
