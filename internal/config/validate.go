package config

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/bundled"
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

	switch cfg.Markdown.MathFont {
	case bundled.MathFontLatinModern, bundled.MathFontLocal, bundled.MathFontNone:
	default:
		problems = append(problems, `markdown.math_font must be "latin-modern", "local", or "none"`)
	}
	if cfg.Markdown.DiagramWidth < 0 {
		problems = append(problems, "markdown.diagram_width must not be negative")
	}

	if cfg.RSS.Enabled && cfg.RSS.Limit <= 0 {
		problems = append(problems, "rss.limit must be greater than 0 when rss is enabled")
	}
	if cfg.Feed.Enabled && cfg.Feed.Limit <= 0 {
		problems = append(problems, "feed.limit must be greater than 0 when feed is enabled")
	}
	if cfg.RSS.Enabled && cfg.Feed.Enabled && cfg.RSS.Filename == cfg.Feed.Filename {
		problems = append(problems, "rss.filename and feed.filename must differ")
	}
	if cfg.Search.BodyLimit < 0 {
		problems = append(problems, "search.body_limit must not be negative")
	}
	if cfg.Home.Limit < 0 {
		problems = append(problems, "home.limit must not be negative")
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
		"data_dir":         cfg.DataDir,
		"rss.filename":     cfg.RSS.Filename,
		"feed.filename":    cfg.Feed.Filename,
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
	problems = append(problems, outputOverlaps(cfg)...)
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
	if cfg.Pipeline.Images.Enabled {
		for _, width := range cfg.Pipeline.Images.Widths {
			if width <= 0 {
				problems = append(problems, "pipeline.images.widths values must be greater than 0")
				break
			}
		}
		for name, preset := range cfg.Pipeline.Images.Presets {
			if strings.TrimSpace(name) == "" {
				problems = append(problems, "pipeline.images.presets names must not be empty")
				continue
			}
			for _, width := range preset.Widths {
				if width <= 0 {
					problems = append(problems, "pipeline.images.presets."+name+".widths values must be greater than 0")
					break
				}
			}
		}
	}

	if len(problems) == 0 {
		return nil
	}

	return errors.New(strings.Join(problems, "; "))
}

// outputOverlaps reports source directories that output_dir would delete.
// Every build empties output_dir first, so it must not be, hold, or sit
// inside a directory the site is built from, or the Git repository.
func outputOverlaps(cfg SiteConfig) []string {
	output := cleanSlash(cfg.OutputDir)
	if output == "" || output == "." {
		return nil
	}
	sources := []struct{ field, dir string }{
		{"content_dir", cfg.ContentDir},
		{"template_dir", cfg.TemplateDir},
		{"static_dir", cfg.StaticDir},
		{"themes_dir", cfg.ThemesDir},
		{"data_dir", cfg.DataDir},
		{"the Git repository", ".git"},
	}
	var problems []string
	for _, source := range sources {
		dir := cleanSlash(source.dir)
		if dir == "" || dir == "." {
			continue
		}
		if output == dir || within(output, dir) || within(dir, output) {
			problems = append(problems, "output_dir "+cfg.OutputDir+" overlaps "+source.field+" "+source.dir+": every build deletes output_dir first")
		}
	}
	return problems
}

func cleanSlash(dir string) string {
	return filepath.ToSlash(safepath.CleanRelative(dir))
}

// within reports whether path is inside dir.
func within(path, dir string) bool {
	return strings.HasPrefix(path, dir+"/")
}
