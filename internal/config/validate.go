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
	if cfg.Atom.Enabled && cfg.Atom.Limit <= 0 {
		problems = append(problems, "atom.limit must be greater than 0 when Atom is enabled")
	}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		problems = append(problems, "server.port must be between 1 and 65535")
	}

	seenTaxonomyNames := map[string]bool{}
	for _, t := range cfg.Taxonomies {
		if strings.TrimSpace(t.Name) == "" {
			problems = append(problems, "each taxonomy must have a name")
		} else {
			lowered := strings.ToLower(strings.TrimSpace(t.Name))
			if seenTaxonomyNames[lowered] {
				problems = append(problems, "duplicate taxonomy name: "+t.Name)
			}
			seenTaxonomyNames[lowered] = true
		}
	}

	requiredPaths := map[string]string{
		"content_dir":      cfg.ContentDir,
		"template_dir":    cfg.TemplateDir,
		"static_dir":      cfg.StaticDir,
		"output_dir":       cfg.OutputDir,
		"rss.filename":    cfg.RSS.Filename,
		"atom.filename":   cfg.Atom.Filename,
		"sitemap.filename": cfg.Sitemap.Filename,
		"robots.filename":  cfg.Robots.Filename,
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
