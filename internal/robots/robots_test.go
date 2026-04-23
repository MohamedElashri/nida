package robots

import (
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
)

func TestGenerateRobotsWithCustomContent(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.Robots.Enabled = true
	cfg.Robots.Content = "User-agent: TestBot\nDisallow: /"

	out := Generate(cfg)
	if out == nil {
		t.Fatal("expected robots output")
	}
	if string(out.Content) != "User-agent: TestBot\nDisallow: /\n" {
		t.Fatalf("unexpected robots content %q", string(out.Content))
	}
}

func TestGenerateDefaultRobotsIncludesSitemap(t *testing.T) {
	cfg := config.DefaultSiteConfig()
	cfg.BaseURL = "https://example.com"
	cfg.Robots.Enabled = true

	out := Generate(cfg)
	if out == nil || !strings.Contains(string(out.Content), "Sitemap: https://example.com/sitemap.xml") {
		t.Fatalf("expected sitemap in default robots, got %+v", out)
	}
}
