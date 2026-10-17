// Package datafiles loads structured data from the data directory for use
// in templates as .Data.
package datafiles

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/themes"
	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

// Load reads .toml, .json, .yaml, and .yml files under data_dir in each
// theme (parent first) and then the site. data/authors.toml becomes
// .Data.authors and data/team/core.json becomes .Data.team.core. A file
// from a later directory replaces the same key from an earlier one.
func Load(siteRoot string, cfg config.SiteConfig) (map[string]any, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}
	roots, err := themes.Dirs(absSiteRoot, cfg, cfg.DataDir)
	if err != nil {
		return nil, err
	}
	siteData, err := safepath.Join(absSiteRoot, cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}
	roots = append(roots, siteData)

	data := map[string]any{}
	for _, root := range roots {
		if err := loadDir(root, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func loadDir(root string, data map[string]any) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.HasPrefix(d.Name(), ".") && path != root {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refuse data symlink %q", path)
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".toml" && ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read data file %q: %w", path, err)
		}
		var value any
		switch ext {
		case ".toml":
			var table map[string]any
			err = toml.Unmarshal(raw, &table)
			value = table
		case ".json":
			err = json.Unmarshal(raw, &value)
		default:
			err = yaml.Unmarshal(raw, &value)
		}
		if err != nil {
			return fmt.Errorf("parse data file %q: %w", path, err)
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		keys := strings.Split(filepath.ToSlash(strings.TrimSuffix(rel, filepath.Ext(rel))), "/")
		target := data
		for _, key := range keys[:len(keys)-1] {
			next, ok := target[key].(map[string]any)
			if !ok {
				next = map[string]any{}
				target[key] = next
			}
			target = next
		}
		target[keys[len(keys)-1]] = value
		return nil
	})
}
