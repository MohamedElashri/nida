package robots

import (
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/route"
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
		b.WriteString("\nSitemap: ")
		b.WriteString(route.CanonicalURL(strings.TrimSpace(cfg.BaseURL), "/"+strings.TrimLeft(cfg.Sitemap.Filename, "/")))
	}
	return b.String()
}
