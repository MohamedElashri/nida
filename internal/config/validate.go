package config

import (
	"errors"
	"net/url"
	"strings"

	"github.com/MohamedElashri/nida/internal/safepath"
)

func Validate(cfg SiteConfig) error {
	var problems []string

	if strings.TrimSpace(cfg.BaseURL) == "" {
		problems = append(problems, "base_url is required")
	} else {
		parsed, err := url.Parse(cfg.BaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			problems = append(problems, "base_url must be an absolute URL")
		} else if parsed.Scheme != "http" && parsed.Scheme != "https" {
			problems = append(problems, "base_url scheme must be http or https")
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
		"template_dir":     cfg.TemplateDir,
		"static_dir":       cfg.StaticDir,
		"output_dir":       cfg.OutputDir,
		"themes_dir":       cfg.ThemesDir,
		"rss.filename":     cfg.RSS.Filename,
		"atom.filename":    cfg.Atom.Filename,
		"sitemap.filename": cfg.Sitemap.Filename,
		"robots.filename":  cfg.Robots.Filename,
		"search.filename":  cfg.Search.Filename,
	}

	for field, value := range requiredPaths {
		allowDot := field != "output_dir" && !strings.Contains(field, ".filename")
		if err := safepath.ValidateRelative(field, value, allowDot); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if strings.TrimSpace(cfg.Theme) != "" {
		if err := safepath.ValidateRelative("theme", cfg.Theme, false); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if cfg.Pipeline.SCSS.Enabled {
		if err := safepath.ValidateRelative("pipeline.scss.entry_dir", cfg.Pipeline.SCSS.EntryDir, false); err != nil {
			problems = append(problems, err.Error())
		}
	}

	if len(problems) == 0 {
		return nil
	}

	return errors.New(strings.Join(problems, "; "))
}
