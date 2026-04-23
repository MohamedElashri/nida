package config

import (
	"path/filepath"
	"strings"
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
	cfg.PostsDir = cleanRelativePath(cfg.PostsDir)
	cfg.PagesDir = cleanRelativePath(cfg.PagesDir)
	cfg.SyntaxTheme = strings.TrimSpace(cfg.SyntaxTheme)
	cfg.RSS.Filename = cleanRelativePath(cfg.RSS.Filename)
	cfg.Atom.Filename = cleanRelativePath(cfg.Atom.Filename)
	cfg.Sitemap.Filename = cleanRelativePath(cfg.Sitemap.Filename)
	cfg.Robots.Filename = cleanRelativePath(cfg.Robots.Filename)
	cfg.Server.Host = strings.TrimSpace(cfg.Server.Host)
	cfg.Permalinks.Posts = normalizePermalink(cfg.Permalinks.Posts)
	cfg.Permalinks.Pages = normalizePermalink(cfg.Permalinks.Pages)
	cfg.Permalinks.Tags = normalizePermalink(cfg.Permalinks.Tags)
	cfg.Permalinks.Categories = normalizePermalink(cfg.Permalinks.Categories)
}

func cleanRelativePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	return filepath.Clean(value)
}

func normalizePermalink(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	if !strings.HasSuffix(value, "/") {
		value += "/"
	}
	return value
}
