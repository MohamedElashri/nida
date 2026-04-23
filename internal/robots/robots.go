package robots

import (
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
)

type Output struct {
	Filename string
	Content  []byte
}

func Generate(cfg config.SiteConfig) *Output {
	if !cfg.Robots.Enabled {
		return nil
	}

	content := strings.TrimSpace(cfg.Robots.Content)
	if content == "" {
		content = defaultRobots(cfg)
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return &Output{
		Filename: cfg.Robots.Filename,
		Content:  []byte(content),
	}
}

func defaultRobots(cfg config.SiteConfig) string {
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	b.WriteString("Allow: /\n")
	if cfg.Sitemap.Enabled && strings.TrimSpace(cfg.Sitemap.Filename) != "" {
		baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
		b.WriteString("\nSitemap: ")
		b.WriteString(baseURL)
		b.WriteString("/")
		b.WriteString(strings.TrimLeft(cfg.Sitemap.Filename, "/"))
	}
	return b.String()
}
