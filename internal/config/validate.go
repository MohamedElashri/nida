package config

import (
	"errors"
	"net/url"
	"strings"
)

func Validate(cfg SiteConfig) error {
	var problems []string

	if strings.TrimSpace(cfg.BaseURL) == "" {
		problems = append(problems, "base_url is required")
	} else {
		parsed, err := url.Parse(cfg.BaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			problems = append(problems, "base_url must be an absolute URL")
		}
	}

	if strings.TrimSpace(cfg.Title) == "" {
		problems = append(problems, "title is required")
	}

	if cfg.Paginate <= 0 {
		problems = append(problems, "paginate must be greater than 0")
	}

	if cfg.RSS.Enabled && cfg.RSS.Limit <= 0 {
		problems = append(problems, "rss.limit must be greater than 0 when RSS is enabled")
	}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		problems = append(problems, "server.port must be between 1 and 65535")
	}

	requiredPaths := map[string]string{
		"content_dir":           cfg.ContentDir,
		"template_dir":          cfg.TemplateDir,
		"static_dir":            cfg.StaticDir,
		"output_dir":            cfg.OutputDir,
		"posts_dir":             cfg.PostsDir,
		"pages_dir":             cfg.PagesDir,
		"rss.filename":          cfg.RSS.Filename,
		"sitemap.filename":      cfg.Sitemap.Filename,
		"permalinks.posts":      cfg.Permalinks.Posts,
		"permalinks.pages":      cfg.Permalinks.Pages,
		"permalinks.tags":       cfg.Permalinks.Tags,
		"permalinks.categories": cfg.Permalinks.Categories,
	}

	for field, value := range requiredPaths {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, field+" must not be empty")
		}
	}

	if len(problems) == 0 {
		return nil
	}

	return errors.New(strings.Join(problems, "; "))
}
