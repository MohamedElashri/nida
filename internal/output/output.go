package output

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/safepath"
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
		if err := safepath.EnsureNoSymlinkPath(outputDir, targetPath); err != nil {
			return fmt.Errorf("check output path for %q: %w", page.URL, err)
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
		if err := safepath.EnsureNoSymlinkPath(outputDir, targetPath); err != nil {
			return fmt.Errorf("check output path for %q: %w", route, err)
		}
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove rendered page %q: %w", targetPath, err)
		}
		if err := removeEmptyParents(outputDir, targetPath); err != nil {
			return err
		}
	}

	return nil
}

// removeEmptyParents removes ancestor directories left empty by a page
// removal, stopping at (and never including) the output root. Without this,
// Go's http.FileServer serves a directory listing (HTTP 200) for a removed
// page's URL instead of 404.
func removeEmptyParents(outputDir, removedPath string) error {
	dir := filepath.Clean(filepath.Dir(removedPath))
	for dir != outputDir {
		rel, err := filepath.Rel(outputDir, dir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return fmt.Errorf("refuse to remove directory %q outside output root %q", dir, outputDir)
		}

		err = os.Remove(dir)
		switch {
		case err == nil:
		case os.IsNotExist(err), errors.Is(err, syscall.ENOTEMPTY), errors.Is(err, syscall.EEXIST):
			return nil
		default:
			return fmt.Errorf("remove empty directory %q: %w", dir, err)
		}

		dir = filepath.Dir(dir)
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

	targetPath, err := safepath.Join(outputDir, relativePath)
	if err != nil {
		return fmt.Errorf("resolve output path for %q: %w", relativePath, err)
	}
	if err := safepath.EnsureNoSymlinkPath(outputDir, targetPath); err != nil {
		return fmt.Errorf("check output path for %q: %w", relativePath, err)
	}
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

	targetPath, err := safepath.Join(outputDir, relativePath)
	if err != nil {
		return fmt.Errorf("resolve output path for %q: %w", relativePath, err)
	}
	if err := safepath.EnsureNoSymlinkPath(outputDir, targetPath); err != nil {
		return fmt.Errorf("check output path for %q: %w", relativePath, err)
	}
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
		targetPath, err := safepath.Join(outputDir, artifact.Path)
		if err != nil {
			return fmt.Errorf("resolve artifact output path for %q: %w", artifact.Path, err)
		}
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
	outputDir, err := safepath.Join(absSiteRoot, cfg.OutputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory %q: %w", cfg.OutputDir, err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, outputDir); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("check output directory %q: %w", cfg.OutputDir, err)
	}
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
