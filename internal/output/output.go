package output

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/render"
)

type Artifact struct {
	Path string
}

func WriteSite(siteRoot string, cfg config.SiteConfig, pages []render.Page) error {
	outputDir, err := outputDirectory(siteRoot, cfg)
	if err != nil {
		return err
	}

	if err := cleanOutputDir(outputDir); err != nil {
		return err
	}

	return WritePages(siteRoot, cfg, pages)
}

func WritePages(siteRoot string, cfg config.SiteConfig, pages []render.Page) error {
	outputDir, err := outputDirectory(siteRoot, cfg)
	if err != nil {
		return err
	}

	sorted := append([]render.Page(nil), pages...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].URL < sorted[j].URL
	})

	for _, page := range sorted {
		targetPath, err := pagePath(outputDir, page.URL)
		if err != nil {
			return fmt.Errorf("resolve output path for %q: %w", page.URL, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create output directory for %q: %w", targetPath, err)
		}
		if err := os.WriteFile(targetPath, []byte(page.Content), 0o644); err != nil {
			return fmt.Errorf("write rendered page %q: %w", targetPath, err)
		}
	}

	return nil
}

func RemovePages(siteRoot string, cfg config.SiteConfig, routes []string) error {
	outputDir, err := outputDirectory(siteRoot, cfg)
	if err != nil {
		return err
	}

	for _, route := range routes {
		targetPath, err := pagePath(outputDir, route)
		if err != nil {
			return fmt.Errorf("resolve output path for %q: %w", route, err)
		}
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove rendered page %q: %w", targetPath, err)
		}
	}

	return nil
}

func RemoveFile(siteRoot string, cfg config.SiteConfig, relativePath string) error {
	outputDir, err := outputDirectory(siteRoot, cfg)
	if err != nil {
		return err
	}
	if relativePath == "" {
		return fmt.Errorf("relative output path is required")
	}

	targetPath := filepath.Join(outputDir, filepath.FromSlash(relativePath))
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove output file %q: %w", targetPath, err)
	}
	return nil
}

func WriteFile(siteRoot string, cfg config.SiteConfig, relativePath string, content []byte) error {
	outputDir, err := outputDirectory(siteRoot, cfg)
	if err != nil {
		return err
	}

	if relativePath == "" {
		return fmt.Errorf("relative output path is required")
	}

	targetPath := filepath.Join(outputDir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create output directory for %q: %w", targetPath, err)
	}
	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		return fmt.Errorf("write output file %q: %w", targetPath, err)
	}
	return nil
}

func ValidateWritePlan(siteRoot string, cfg config.SiteConfig, pages []render.Page, artifacts []Artifact) error {
	outputDir, err := outputDirectory(siteRoot, cfg)
	if err != nil {
		return err
	}

	seen := make(map[string]string)
	for _, page := range pages {
		targetPath, err := pagePath(outputDir, page.URL)
		if err != nil {
			return fmt.Errorf("resolve output path for %q: %w", page.URL, err)
		}
		if existing, ok := seen[targetPath]; ok {
			return fmt.Errorf("output path conflict for %q between %s and page %q", targetPath, existing, page.URL)
		}
		seen[targetPath] = fmt.Sprintf("page %q", page.URL)
	}

	for _, artifact := range artifacts {
		if artifact.Path == "" {
			return fmt.Errorf("artifact output path is required")
		}
		targetPath := filepath.Join(outputDir, filepath.FromSlash(artifact.Path))
		if existing, ok := seen[targetPath]; ok {
			return fmt.Errorf("output path conflict for %q between %s and artifact %q", targetPath, existing, artifact.Path)
		}
		seen[targetPath] = fmt.Sprintf("artifact %q", artifact.Path)
	}

	return nil
}

func outputDirectory(siteRoot string, cfg config.SiteConfig) (string, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return "", fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}
	outputDir := filepath.Join(absSiteRoot, cfg.OutputDir)
	return outputDir, nil
}

func cleanOutputDir(outputDir string) error {
	if outputDir == "" || outputDir == string(filepath.Separator) {
		return fmt.Errorf("refuse to clean unsafe output directory %q", outputDir)
	}
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("clean output directory %q: %w", outputDir, err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("recreate output directory %q: %w", outputDir, err)
	}
	return nil
}

func pagePath(outputDir, route string) (string, error) {
	if route == "" || !strings.HasPrefix(route, "/") {
		return "", fmt.Errorf("route must start with /")
	}

	trimmed := strings.TrimPrefix(route, "/")
	if trimmed == "" {
		return filepath.Join(outputDir, "index.html"), nil
	}

	clean := filepath.Clean(trimmed)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe route %q", route)
	}

	if strings.HasSuffix(route, "/") {
		return filepath.Join(outputDir, filepath.FromSlash(clean), "index.html"), nil
	}
	return filepath.Join(outputDir, filepath.FromSlash(clean)), nil
}
