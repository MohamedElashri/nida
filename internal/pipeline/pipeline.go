package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MohamedElashri/nida/internal/config"
)

type Manifest map[string]string

func Process(siteRoot string, cfg config.SiteConfig) (Manifest, error) {
	if !cfg.Pipeline.Fingerprint && !cfg.Pipeline.Images.Enabled && !cfg.Pipeline.SCSS.Enabled {
		return nil, nil
	}

	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve site root: %w", err)
	}

	staticRoot := filepath.Join(absSiteRoot, cfg.StaticDir)
	outputRoot := filepath.Join(absSiteRoot, cfg.OutputDir)

	if _, err := os.Stat(staticRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat static dir: %w", err)
	}

	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	manifest := make(Manifest)

	if cfg.Pipeline.SCSS.Enabled {
		if err := compileSCSS(staticRoot, outputRoot, cfg); err != nil {
			return nil, err
		}
	}

	if err := processStaticFiles(staticRoot, outputRoot, "", staticRoot, cfg, manifest); err != nil {
		return nil, err
	}

	if len(manifest) > 0 {
		if err := writeManifest(outputRoot, manifest); err != nil {
			return nil, err
		}
	}

	return manifest, nil
}

func processStaticFiles(staticRoot, outputRoot, relDir, absDir string, cfg config.SiteConfig, manifest Manifest) error {
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
			if err := processStaticFiles(staticRoot, outputRoot, relPath, srcPath, cfg, manifest); err != nil {
				return err
			}
			continue
		}

		if cfg.Pipeline.Images.Enabled && isImageFile(name) {
			if err := processImage(srcPath, relPath, outputRoot, cfg, manifest); err != nil {
				return fmt.Errorf("process image %q: %w", relPath, err)
			}
			continue
		}

		dstPath := filepath.Join(outputRoot, filepath.FromSlash(relPath))
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

func isImageFile(name string) bool {
	ext := filepath.Ext(name)
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
