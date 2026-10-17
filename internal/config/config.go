package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	cfg.Warnings = append(cfg.Warnings, unknownKeyWarnings(data)...)

	normalize(&cfg)
	if err := Validate(cfg); err != nil {
		return SiteConfig{}, "", fmt.Errorf("validate config %q: %w", configPath, err)
	}

	return cfg, configPath, nil
}

// unknownKeyWarnings reports config keys Nida does not understand. Unknown
// keys are ignored rather than rejected so a typo never blocks a build, but
// silently dropping them hides mistakes, so each one becomes a warning.
func unknownKeyWarnings(data []byte) []string {
	var probe SiteConfig
	err := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(&probe)
	var strict *toml.StrictMissingError
	if !errors.As(err, &strict) {
		return nil
	}
	warnings := make([]string, 0, len(strict.Errors))
	for _, decodeErr := range strict.Errors {
		key := strings.Join(decodeErr.Key(), ".")
		warnings = append(warnings, fmt.Sprintf("unknown config key %q is ignored", key))
	}
	return warnings
}
