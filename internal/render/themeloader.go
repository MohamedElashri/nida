package render

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/MohamedElashri/nida/internal/config"
)

type ThemeConfig struct {
	Name   string         `toml:"name"`
	Extends string        `toml:"extends"`
	Extra  map[string]any `toml:"extra"`
}

func loadThemeChain(siteRoot string, cfg config.SiteConfig) ([]ThemeConfig, error) {
	if cfg.Theme == "" {
		return nil, nil
	}

	themesRoot := filepath.Join(siteRoot, cfg.ThemesDir)
	resolved, err := resolveThemeChain(cfg.Theme, themesRoot, nil)
	if err != nil {
		return nil, fmt.Errorf("resolve theme chain: %w", err)
	}

	var chain []ThemeConfig
	for _, name := range resolved {
		tc, err := loadThemeConfig(themesRoot, name)
		if err != nil {
			return nil, err
		}
		chain = append(chain, tc)
	}

	return chain, nil
}

func resolveThemeChain(name, themesRoot string, visited []string) ([]string, error) {
	for _, v := range visited {
		if v == name {
			return nil, fmt.Errorf("circular theme inheritance: %s appears twice in chain", name)
		}
	}

	themePath := filepath.Join(themesRoot, name)
	if _, err := os.Stat(themePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("theme %q not found in %q", name, themesRoot)
		}
		return nil, fmt.Errorf("stat theme %q: %w", name, err)
	}

	configPath := filepath.Join(themePath, "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read theme config %q: %w", configPath, err)
	}

	var tc ThemeConfig
	if err := decodeTOML(string(data), &tc); err != nil {
		return nil, fmt.Errorf("parse theme config %q: %w", configPath, err)
	}

	result := []string{name}

	if tc.Extends != "" {
		parentChain, err := resolveThemeChain(tc.Extends, themesRoot, append(visited, name))
		if err != nil {
			return nil, err
		}
		result = append(parentChain, result...)
	}

	return result, nil
}

func loadThemeConfig(themesRoot, name string) (ThemeConfig, error) {
	configPath := filepath.Join(themesRoot, name, "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ThemeConfig{}, fmt.Errorf("read theme config %q: %w", configPath, err)
	}

	var tc ThemeConfig
	if err := decodeTOML(string(data), &tc); err != nil {
		return ThemeConfig{}, fmt.Errorf("parse theme config %q: %w", configPath, err)
	}

	tc.Name = name
	return tc, nil
}

func mergeThemeExtra(chain []ThemeConfig, siteExtra map[string]any) map[string]any {
	result := make(map[string]any)

	for i := len(chain) - 1; i >= 0; i-- {
		for k, v := range chain[i].Extra {
			result[k] = v
		}
	}

	for k, v := range siteExtra {
		result[k] = v
	}

	return result
}

func decodeTOML(content string, v any) error {
	if err := toml.Unmarshal([]byte(content), v); err != nil {
		return err
	}
	return nil
}
