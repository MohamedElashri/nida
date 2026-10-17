// Package themes resolves the selected theme and the themes it extends.
package themes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/pelletier/go-toml/v2"
)

// Theme is one theme in an inheritance chain.
type Theme struct {
	Name    string         `toml:"-"`
	Dir     string         `toml:"-"`
	Title   string         `toml:"name"`
	Extends string         `toml:"extends"`
	Extra   map[string]any `toml:"extra"`
}

// Chain returns the configured theme and its ancestors, parent first, or nil
// when no theme is set.
func Chain(siteRoot string, cfg config.SiteConfig) ([]Theme, error) {
	if cfg.Theme == "" {
		return nil, nil
	}
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve site root: %w", err)
	}
	themesRoot, err := safepath.Join(absSiteRoot, cfg.ThemesDir)
	if err != nil {
		return nil, fmt.Errorf("resolve themes root: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, themesRoot); err != nil {
		return nil, fmt.Errorf("check themes root: %w", err)
	}

	var chain []Theme
	seen := map[string]bool{}
	for name := cfg.Theme; name != ""; {
		if seen[name] {
			return nil, fmt.Errorf("resolve theme chain: circular theme inheritance: %s appears twice in chain", name)
		}
		seen[name] = true
		theme, err := load(themesRoot, name)
		if err != nil {
			return nil, fmt.Errorf("resolve theme chain: %w", err)
		}
		chain = append([]Theme{theme}, chain...)
		name = theme.Extends
	}
	return chain, nil
}

// Dirs returns the named subdirectory (such as "templates" or "static") of
// every theme in the chain that has one, parent first.
func Dirs(siteRoot string, cfg config.SiteConfig, sub string) ([]string, error) {
	chain, err := Chain(siteRoot, cfg)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, theme := range chain {
		dir := filepath.Join(theme.Dir, sub)
		if err := safepath.EnsureNoSymlinkPath(theme.Dir, dir); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("check theme %s directory %q: %w", sub, dir, err)
		}
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dirs = append(dirs, dir)
		}
	}
	return dirs, nil
}

// MergeExtra layers theme [extra] tables parent first, then the site's
// [extra] on top, so the most specific value wins.
func MergeExtra(chain []Theme, siteExtra map[string]any) map[string]any {
	result := make(map[string]any)
	for _, theme := range chain {
		for k, v := range theme.Extra {
			result[k] = v
		}
	}
	for k, v := range siteExtra {
		result[k] = v
	}
	return result
}

func load(themesRoot, name string) (Theme, error) {
	themePath, err := safepath.Join(themesRoot, name)
	if err != nil {
		return Theme{}, fmt.Errorf("resolve theme %q: %w", name, err)
	}
	if err := safepath.EnsureNoSymlinkPath(themesRoot, themePath); err != nil {
		return Theme{}, fmt.Errorf("check theme %q: %w", name, err)
	}
	if _, err := os.Stat(themePath); err != nil {
		if os.IsNotExist(err) {
			return Theme{}, fmt.Errorf("theme %q not found in %q", name, themesRoot)
		}
		return Theme{}, fmt.Errorf("stat theme %q: %w", name, err)
	}

	// config.toml is optional; a theme without one extends nothing.
	var theme Theme
	configPath := filepath.Join(themePath, "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return Theme{}, fmt.Errorf("read theme config %q: %w", configPath, err)
	}
	if err == nil {
		if err := toml.Unmarshal(data, &theme); err != nil {
			return Theme{}, fmt.Errorf("parse theme config %q: %w", configPath, err)
		}
	}
	theme.Name = name
	theme.Dir = themePath
	return theme, nil
}
