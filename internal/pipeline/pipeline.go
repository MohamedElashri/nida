package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/themes"
)

type Manifest map[string]string

func Process(siteRoot string, cfg config.SiteConfig) (Manifest, error) {
	if !cfg.Pipeline.Active() {
		return nil, nil
	}

	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve site root: %w", err)
	}

	staticRoot, err := safepath.Join(absSiteRoot, cfg.StaticDir)
	if err != nil {
		return nil, fmt.Errorf("resolve static dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, staticRoot); err != nil {
		return nil, fmt.Errorf("check static dir: %w", err)
	}
	outputRoot, err := safepath.Join(absSiteRoot, cfg.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve output dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, outputRoot); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("check output dir: %w", err)
	}

	staticExists := true
	if _, err := os.Stat(staticRoot); err != nil {
		if os.IsNotExist(err) {
			staticExists = false
		} else {
			return nil, fmt.Errorf("stat static dir: %w", err)
		}
	}

	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	manifest := make(Manifest)

	if cfg.Pipeline.SCSS.Enabled {
		if err := compileSCSS(absSiteRoot, staticRoot, outputRoot, cfg); err != nil {
			return nil, err
		}
	}

	// The site's static files come first, then the selected theme's, then
	// each parent theme's. A path handled by an earlier root is skipped,
	// matching how plain asset copying lets the site override a theme.
	var roots []string
	if staticExists {
		roots = append(roots, staticRoot)
	}
	themeStatic, err := themes.Dirs(absSiteRoot, cfg, "static")
	if err != nil {
		return nil, err
	}
	for i := len(themeStatic) - 1; i >= 0; i-- {
		roots = append(roots, themeStatic[i])
	}
	seen := map[string]bool{}
	for _, root := range roots {
		if err := processStaticFiles(root, outputRoot, "", root, cfg, manifest, seen); err != nil {
			return nil, err
		}
	}

	if len(manifest) > 0 {
		if err := writeManifest(outputRoot, manifest); err != nil {
			return nil, err
		}
	}

	return manifest, nil
}

func processStaticFiles(staticRoot, outputRoot, relDir, absDir string, cfg config.SiteConfig, manifest Manifest, seen map[string]bool) error {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return fmt.Errorf("read dir %q: %w", absDir, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." || name[0] == '.' {
			continue
		}

		srcPath := filepath.Join(absDir, name)
		relPath := filepath.ToSlash(filepath.Join(relDir, name))

		if entry.IsDir() {
			if err := processStaticFiles(staticRoot, outputRoot, relPath, srcPath, cfg, manifest, seen); err != nil {
				return err
			}
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse static symlink %q", srcPath)
		}
		if seen[relPath] {
			continue
		}
		seen[relPath] = true

		if cfg.Pipeline.Images.Enabled && IsImageFile(name) {
			if err := processImage(srcPath, relPath, outputRoot, cfg, manifest); err != nil {
				return fmt.Errorf("process image %q: %w", relPath, err)
			}
			continue
		}

		dstPath, err := safepath.Join(outputRoot, relPath)
		if err != nil {
			return fmt.Errorf("resolve output path %q: %w", relPath, err)
		}
		if err := safepath.EnsureNoSymlinkPath(outputRoot, dstPath); err != nil {
			return fmt.Errorf("check output path %q: %w", relPath, err)
		}
		if cfg.Pipeline.MinifyCSS || cfg.Pipeline.MinifyJS {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("read %q: %w", relPath, err)
			}
			minified := minifyAsset(name, data, cfg.Pipeline)
			if !bytes.Equal(minified, data) {
				if err := writeProcessed(minified, relPath, dstPath, outputRoot, cfg, manifest); err != nil {
					return err
				}
				continue
			}
		}

		if cfg.Pipeline.Fingerprint && isFingerprintable(name) {
			fpPath, err := fingerprintFile(srcPath, relPath, outputRoot)
			if err != nil {
				return fmt.Errorf("fingerprint %q: %w", relPath, err)
			}
			manifest[relPath] = fpPath
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("copy %q: %w", relPath, err)
			}
		}
	}

	return nil
}

// IsImageFile reports whether the pipeline can resize a file by extension.
func IsImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	}
	return false
}

func isFingerprintable(name string) bool {
	ext := filepath.Ext(name)
	switch ext {
	case ".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".woff", ".woff2", ".ttf", ".eot":
		return true
	}
	return false
}

// writeProcessed writes transformed asset bytes, fingerprinting them when
// that is enabled so the hash reflects the bytes actually served.
func writeProcessed(data []byte, relPath, dstPath, outputRoot string, cfg config.SiteConfig, manifest Manifest) error {
	if cfg.Pipeline.Fingerprint && isFingerprintable(relPath) {
		fpPath, err := fingerprintData(data, relPath, outputRoot)
		if err != nil {
			return fmt.Errorf("fingerprint %q: %w", relPath, err)
		}
		manifest[relPath] = fpPath
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create dir for %q: %w", relPath, err)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", relPath, err)
	}
	return nil
}
