package render

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
)

func buildTheme(siteRoot string, cfg config.SiteConfig) (Theme, error) {
	chain, err := loadThemeChain(siteRoot, cfg)
	if err != nil {
		return Theme{}, fmt.Errorf("load theme chain: %w", err)
	}

	mergedExtra := mergeThemeExtra(chain, cfg.Extra)

	inlineCSS, err := loadInlineCSS(siteRoot, cfg)
	if err != nil && !os.IsNotExist(err) {
		return Theme{}, fmt.Errorf("load inline css: %w", err)
	}
	return Theme{
		InlineCSS:  template.CSS(inlineCSS),
		MainMenu:   navItems(mergedExtra["main_menu"]),
		Social:     navItems(mergedExtra["social_icons"]),
		FooterText: nestedStringOr(mergedExtra, "footer", "text", cfg.Title),
		DateFormat: nestedStringOr(mergedExtra, "", "date_format", "%Y-%m-%d"),
		AuthorName: nestedStringOr(mergedExtra, "author", "name", cfg.Title),
		Favicon: Favicon{
			Webmanifest:    nestedStringOr(mergedExtra, "favicon", "webmanifest", ""),
			Favicon16x16:   nestedStringOr(mergedExtra, "favicon", "favicon_16x16", ""),
			Favicon32x32:   nestedStringOr(mergedExtra, "favicon", "favicon_32x32", ""),
			AppleTouchIcon: nestedStringOr(mergedExtra, "favicon", "apple_touch_icon", ""),
		},
		Umami: Umami{
			Enabled:   nestedBoolOr(mergedExtra, "umami", "enabled", false),
			Src:       nestedStringOr(mergedExtra, "umami", "src", ""),
			WebsiteID: nestedStringOr(mergedExtra, "umami", "website_id", ""),
		},
	}, nil
}

func loadInlineCSS(siteRoot string, cfg config.SiteConfig) (string, error) {
	if cfg.Theme != "" {
		themeCSS := filepath.Join(siteRoot, cfg.ThemesDir, cfg.Theme, "style.css.html")
		expanded, err := expandTemplateIncludes(themeCSS)
		if err == nil {
			return expanded, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}

	stylePath := filepath.Join(siteRoot, cfg.TemplateDir, "style.css.html")
	expanded, err := expandTemplateIncludes(stylePath)
	if err == nil {
		return expanded, nil
	}
	fallbackPath := filepath.Join(siteRoot, cfg.StaticDir, "site.css")
	data, fallbackErr := os.ReadFile(fallbackPath)
	if fallbackErr != nil {
		return "", err
	}
	return string(data), nil
}

func expandTemplateIncludes(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	text := string(data)
	dir := filepath.Dir(path)
	re := regexp.MustCompile(`\{%\s*include\s+"([^"]+)"(?:\s+ignore missing)?\s*%\}`)
	matches := re.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil
	}

	var b strings.Builder
	last := 0
	for _, match := range matches {
		b.WriteString(text[last:match[0]])
		includeName := text[match[2]:match[3]]
		includePath := filepath.Join(dir, filepath.FromSlash(includeName))
		expanded, err := expandTemplateIncludes(includePath)
		if err != nil {
			if os.IsNotExist(err) {
				last = match[1]
				continue
			}
			return "", err
		}
		b.WriteString(expanded)
		last = match[1]
	}
	b.WriteString(text[last:])
	return b.String(), nil
}

func navItems(value any) []NavItem {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	items := make([]NavItem, 0, len(raw))
	for _, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		urlValue, _ := m["url"].(string)
		name = strings.TrimSpace(name)
		urlValue = strings.TrimSpace(urlValue)
		if name == "" || urlValue == "" {
			continue
		}
		items = append(items, NavItem{Name: name, URL: urlValue})
	}
	return items
}

func nestedStringOr(values map[string]any, mapKey, key, fallback string) string {
	if mapKey == "" {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
		return fallback
	}
	raw, ok := values[mapKey].(map[string]any)
	if !ok {
		return fallback
	}
	value, ok := raw[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func nestedBoolOr(values map[string]any, mapKey, key string, fallback bool) bool {
	raw, ok := values[mapKey].(map[string]any)
	if !ok {
		return fallback
	}
	value, ok := raw[key].(bool)
	if !ok {
		return fallback
	}
	return value
}
