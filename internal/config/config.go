package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const DefaultConfigName = "config.toml"

func Load(opts Options) (SiteConfig, string, error) {
	siteRoot := opts.SiteRoot
	if siteRoot == "" {
		siteRoot = "."
	}

	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return SiteConfig{}, "", fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	configPath := opts.Path
	if configPath == "" {
		configPath = filepath.Join(absSiteRoot, DefaultConfigName)
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(absSiteRoot, configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SiteConfig{}, "", fmt.Errorf("load config %q: file does not exist", configPath)
		}
		return SiteConfig{}, "", fmt.Errorf("load config %q: %w", configPath, err)
	}

	cfg := DefaultSiteConfig()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return SiteConfig{}, "", fmt.Errorf("parse config %q: %w", configPath, err)
	}

	normalize(&cfg)
	if err := Validate(cfg); err != nil {
		return SiteConfig{}, "", fmt.Errorf("validate config %q: %w", configPath, err)
	}

	return cfg, configPath, nil
}
