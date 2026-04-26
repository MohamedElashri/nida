package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const DefaultConfigName = "config.toml"

type ConfigMigrationError struct {
	Path string
}

func (e *ConfigMigrationError) Error() string {
	return "nida: config is from v0.3.x which is not compatible with this version of nida.\nRun 'nida migrate' to upgrade your config to v0.4.\nSee https://nida.blog/docs/migrations for details."
}

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

	if strings.TrimSpace(cfg.ConfigVersion) == "" {
		return SiteConfig{}, configPath, &ConfigMigrationError{Path: configPath}
	}

	normalize(&cfg)
	if err := Validate(cfg); err != nil {
		return SiteConfig{}, "", fmt.Errorf("validate config %q: %w", configPath, err)
	}

	return cfg, configPath, nil
}
