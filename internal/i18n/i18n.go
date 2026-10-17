// Package i18n loads translation strings for theme and template UI text.
package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/themes"
	"github.com/pelletier/go-toml/v2"
)

const dirName = "i18n"

// Translations maps keys to strings for the site language.
type Translations map[string]string

// Load reads i18n/<lang>.toml from each theme (parent first) and the site.
// For language "ar-EG" it layers en.toml, then ar.toml, then ar-eg.toml, so
// a key missing from a more specific file falls back to a broader one.
// Nested tables become dotted keys: [nav] home = "..." is "nav.home".
func Load(siteRoot string, cfg config.SiteConfig) (Translations, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}
	roots, err := themes.Dirs(absSiteRoot, cfg, dirName)
	if err != nil {
		return nil, err
	}
	siteDir, err := safepath.Join(absSiteRoot, dirName)
	if err != nil {
		return nil, err
	}
	roots = append(roots, siteDir)

	out := Translations{}
	for _, lang := range languageChain(cfg.Language) {
		for _, root := range roots {
			path := filepath.Join(root, lang+".toml")
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("read translations %q: %w", path, err)
			}
			if err := safepath.RejectSymlink(path); err != nil {
				return nil, fmt.Errorf("check translations %q: %w", path, err)
			}
			var table map[string]any
			if err := toml.Unmarshal(data, &table); err != nil {
				return nil, fmt.Errorf("parse translations %q: %w", path, err)
			}
			flatten("", table, out)
		}
	}
	return out, nil
}

func languageChain(language string) []string {
	language = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(language), "_", "-"))
	chain := []string{"en"}
	if language == "" || language == "en" {
		return chain
	}
	primary, _, hasRegion := strings.Cut(language, "-")
	if primary != "en" {
		chain = append(chain, primary)
	}
	if hasRegion {
		chain = append(chain, language)
	}
	return chain
}

func flatten(prefix string, table map[string]any, out Translations) {
	for key, value := range table {
		full := key
		if prefix != "" {
			full = prefix + "." + key
		}
		switch v := value.(type) {
		case map[string]any:
			flatten(full, v, out)
		case string:
			out[full] = v
		default:
			out[full] = fmt.Sprint(v)
		}
	}
}

// Lookup returns the translation for key and whether one exists.
func (t Translations) Lookup(key string) (string, bool) {
	value, ok := t[key]
	return value, ok
}

// T returns the translation for key, or key itself when there is none.
// Placeholders such as {count} are replaced from the optional args map.
func (t Translations) T(key string, args ...map[string]any) string {
	value, ok := t[key]
	if !ok {
		value = key
	}
	for _, m := range args {
		for name, arg := range m {
			value = strings.ReplaceAll(value, "{"+name+"}", fmt.Sprint(arg))
		}
	}
	return value
}
