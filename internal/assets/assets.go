package assets

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/safepath"
)

type BundlePage struct {
	URL       string
	BundleDir string
	Resources []string
}

func Copy(siteRoot string, cfg config.SiteConfig) error {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	outputRoot, err := safepath.Join(absSiteRoot, cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("resolve output dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, outputRoot); err != nil {
		return fmt.Errorf("check output dir: %w", err)
	}

	staticRoot, err := safepath.Join(absSiteRoot, cfg.StaticDir)
	if err != nil {
		return fmt.Errorf("resolve static dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, staticRoot); err != nil {
		return fmt.Errorf("check static dir: %w", err)
	}
	staticRoots := []string{staticRoot}

	if cfg.Theme != "" {
		themeRoot, err := safepath.Join(absSiteRoot, filepath.Join(cfg.ThemesDir, cfg.Theme))
		if err != nil {
			return fmt.Errorf("resolve theme dir: %w", err)
		}
		themeStaticRoot := filepath.Join(themeRoot, "static")
		if _, err := os.Stat(themeStaticRoot); err == nil {
			staticRoots = append(staticRoots, themeStaticRoot)
		}
	}

	for _, staticRoot := range staticRoots {
		if err := copyDir(staticRoot, outputRoot); err != nil {
			return err
		}
	}

	return nil
}

func copyDir(staticRoot, outputRoot string) error {
	if _, err := os.Stat(staticRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat static directory %q: %w", staticRoot, err)
	}

	return filepath.WalkDir(staticRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(staticRoot, path)
		if err != nil {
			return fmt.Errorf("compute static relative path for %q: %w", path, err)
		}
		if rel == "." {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refuse static symlink %q", path)
		}

		target, err := safepath.Join(outputRoot, rel)
		if err != nil {
			return fmt.Errorf("resolve output path %q: %w", rel, err)
		}
		if d.IsDir() {
			if err := safepath.EnsureNoSymlinkPath(outputRoot, target); err != nil {
				return fmt.Errorf("check output directory %q: %w", target, err)
			}
			return os.MkdirAll(target, 0o755)
		}

		if err := safepath.EnsureNoSymlinkPath(outputRoot, target); err != nil {
			return fmt.Errorf("check output path %q: %w", target, err)
		}
		if _, err := os.Stat(target); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat output path %q: %w", target, err)
		}

		return copyFile(path, target)
	})
}

func SyncChanged(siteRoot string, cfg config.SiteConfig, changedPaths []string) error {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	outputRoot, err := safepath.Join(absSiteRoot, cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("resolve output dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, outputRoot); err != nil {
		return fmt.Errorf("check output dir: %w", err)
	}

	staticRoot, err := safepath.Join(absSiteRoot, cfg.StaticDir)
	if err != nil {
		return fmt.Errorf("resolve static dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, staticRoot); err != nil {
		return fmt.Errorf("check static dir: %w", err)
	}
	staticRoots := []string{staticRoot}
	if cfg.Theme != "" {
		themeRoot, err := safepath.Join(absSiteRoot, filepath.Join(cfg.ThemesDir, cfg.Theme))
		if err != nil {
			return fmt.Errorf("resolve theme dir: %w", err)
		}
		themeStaticRoot := filepath.Join(themeRoot, "static")
		if _, err := os.Stat(themeStaticRoot); err == nil {
			staticRoots = append(staticRoots, themeStaticRoot)
		}
	}

	for _, staticRoot := range staticRoots {
		staticPrefix, err := staticRootPrefix(absSiteRoot, staticRoot)
		if err != nil {
			return err
		}
		for _, changedPath := range changedPaths {
			normalized, err := changedPathRelativeToSite(absSiteRoot, changedPath)
			if err != nil {
				return err
			}
			if normalized == "" || pathEscapesRoot(normalized) {
				continue
			}
			if staticPrefix != "" && normalized != strings.TrimSuffix(staticPrefix, "/") && !strings.HasPrefix(normalized, staticPrefix) {
				continue
			}

			rel := strings.TrimPrefix(normalized, staticPrefix)
			if rel == normalized && staticPrefix != "" {
				rel = ""
			}
			if strings.TrimSpace(rel) == "" {
				continue
			}

			source, err := safepath.Join(staticRoot, rel)
			if err != nil {
				return fmt.Errorf("resolve static asset %q: %w", rel, err)
			}
			target, err := safepath.Join(outputRoot, rel)
			if err != nil {
				return fmt.Errorf("resolve output path %q: %w", rel, err)
			}
			if err := safepath.EnsureNoSymlinkPath(outputRoot, target); err != nil {
				return fmt.Errorf("check output path %q: %w", target, err)
			}

			info, err := os.Lstat(source)
			if err != nil {
				if os.IsNotExist(err) {
					if removeErr := os.Remove(target); removeErr != nil && !os.IsNotExist(removeErr) {
						return fmt.Errorf("remove stale asset %q: %w", target, removeErr)
					}
					continue
				}
				return fmt.Errorf("stat static asset %q: %w", source, err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("refuse static symlink %q", source)
			}
			if info.IsDir() {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return fmt.Errorf("create output directory for %q: %w", target, err)
				}
				continue
			}

			if err := copyFile(source, target); err != nil {
				return err
			}
		}
	}

	return nil
}

func staticRootPrefix(absSiteRoot, staticRoot string) (string, error) {
	rel, err := filepath.Rel(absSiteRoot, staticRoot)
	if err != nil {
		return "", fmt.Errorf("compute static root relative path for %q: %w", staticRoot, err)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return "", nil
	}
	return strings.Trim(rel, "/") + "/", nil
}

func changedPathRelativeToSite(absSiteRoot, changedPath string) (string, error) {
	trimmed := strings.TrimSpace(changedPath)
	if trimmed == "" {
		return "", nil
	}

	clean := filepath.Clean(trimmed)
	if filepath.IsAbs(clean) {
		rel, err := filepath.Rel(absSiteRoot, clean)
		if err != nil {
			return "", fmt.Errorf("compute changed path relative path for %q: %w", changedPath, err)
		}
		clean = rel
	}
	if clean == "." {
		return "", nil
	}

	return filepath.ToSlash(clean), nil
}

func pathEscapesRoot(path string) bool {
	clean := filepath.Clean(filepath.FromSlash(path))
	return clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean)
}

func CopyPageBundles(siteRoot string, cfg config.SiteConfig, pages []BundlePage) error {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	outputRoot, err := safepath.Join(absSiteRoot, cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("resolve output dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, outputRoot); err != nil {
		return fmt.Errorf("check output dir: %w", err)
	}
	contentRoot, err := safepath.Join(absSiteRoot, cfg.ContentDir)
	if err != nil {
		return fmt.Errorf("resolve content dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, contentRoot); err != nil {
		return fmt.Errorf("check content dir: %w", err)
	}

	for _, page := range pages {
		if len(page.Resources) == 0 {
			continue
		}

		sourceDir, err := safepath.Join(contentRoot, page.BundleDir)
		if err != nil {
			return fmt.Errorf("resolve bundle dir %q: %w", page.BundleDir, err)
		}
		outputURL := strings.TrimSuffix(page.URL, "/")
		outputDir, err := safepath.Join(outputRoot, strings.TrimPrefix(outputURL, "/"))
		if err != nil {
			return fmt.Errorf("resolve bundle output dir %q: %w", page.URL, err)
		}

		for _, res := range page.Resources {
			src, err := safepath.Join(sourceDir, res)
			if err != nil {
				return fmt.Errorf("resolve page resource %q: %w", res, err)
			}
			dst, err := safepath.Join(outputDir, res)
			if err != nil {
				return fmt.Errorf("resolve page resource output %q: %w", res, err)
			}
			if err := safepath.EnsureNoSymlinkPath(outputRoot, dst); err != nil {
				return fmt.Errorf("check page resource output %q: %w", res, err)
			}
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("copy page resource %q: %w", res, err)
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	if err := safepath.RejectSymlink(src); err != nil {
		return fmt.Errorf("check source file %q: %w", src, err)
	}
	if err := safepath.RejectSymlink(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check output file %q: %w", dst, err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open static file %q: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create output directory for %q: %w", dst, err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", dst, err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %q to %q: %w", src, dst, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("close output file %q: %w", dst, err)
	}
	return nil
}
