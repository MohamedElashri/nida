package assets

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
)

func Copy(siteRoot string, cfg config.SiteConfig) error {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	outputRoot := filepath.Join(absSiteRoot, cfg.OutputDir)

	staticRoots := []string{filepath.Join(absSiteRoot, cfg.StaticDir)}

	if cfg.Theme != "" {
		themeStaticRoot := filepath.Join(absSiteRoot, cfg.ThemesDir, cfg.Theme, "static")
		if _, err := os.Stat(themeStaticRoot); err == nil {
			staticRoots = append([]string{themeStaticRoot}, staticRoots...)
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

		target := filepath.Join(outputRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
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

	outputRoot := filepath.Join(absSiteRoot, cfg.OutputDir)

	staticRoots := []string{filepath.Join(absSiteRoot, cfg.StaticDir)}
	if cfg.Theme != "" {
		themeStaticRoot := filepath.Join(absSiteRoot, cfg.ThemesDir, cfg.Theme, "static")
		if _, err := os.Stat(themeStaticRoot); err == nil {
			staticRoots = append([]string{themeStaticRoot}, staticRoots...)
		}
	}

	for _, staticRoot := range staticRoots {
		staticPrefix := filepath.ToSlash(strings.Trim(staticRoot, "/")) + "/"
		for _, changedPath := range changedPaths {
			normalized := filepath.ToSlash(strings.TrimSpace(changedPath))
			if !strings.HasPrefix(normalized, staticPrefix) {
				continue
			}

			rel := strings.TrimPrefix(normalized, staticPrefix)
			if strings.TrimSpace(rel) == "" {
				continue
			}

			source := filepath.Join(staticRoot, filepath.FromSlash(rel))
			target := filepath.Join(outputRoot, filepath.FromSlash(rel))

			info, err := os.Stat(source)
			if err != nil {
				if os.IsNotExist(err) {
					if removeErr := os.Remove(target); removeErr != nil && !os.IsNotExist(removeErr) {
						return fmt.Errorf("remove stale asset %q: %w", target, removeErr)
					}
					continue
				}
				return fmt.Errorf("stat static asset %q: %w", source, err)
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open static file %q: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create output directory for %q: %w", dst, err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %q to %q: %w", src, dst, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("close output file %q: %w", dst, err)
	}
	return nil
}
