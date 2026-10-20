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

	// seen holds output paths already written; the first writer wins.
	seen := map[string]bool{}
	if cfg.Pipeline.SCSS.Enabled {
		compiled, err := compileSCSS(absSiteRoot, staticRoot, outputRoot, cfg)
		if err != nil {
			return nil, err
		}
		for _, rel := range compiled {
			seen[rel] = true
			if err := fingerprintCompiled(outputRoot, rel, cfg, manifest); err != nil {
				return nil, err
			}
		}
	}

	// Site first, then theme, then parent themes; the first root wins a path.
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
	for _, root := range roots {
		if err := processStaticFiles(outputRoot, "", root, cfg, manifest, seen); err != nil {
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

func processStaticFiles(outputRoot, relDir, absDir string, cfg config.SiteConfig, manifest Manifest, seen map[string]bool) error {
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
			if err := processStaticFiles(outputRoot, relPath, srcPath, cfg, manifest, seen); err != nil {
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
		if cfg.Pipeline.SCSS.Enabled && IsSass(name) {
			// Sources and partials are compiled, not published.
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
			if err := safepath.CopyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("copy %q: %w", relPath, err)
			}
		}
	}

	return nil
}

// IsSass reports whether a file is an SCSS or Sass source.
func IsSass(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".scss" || ext == ".sass"
}

// fingerprintCompiled moves a compiled stylesheet to its fingerprinted
// name when fingerprinting is on, as static files are written.
func fingerprintCompiled(outputRoot, rel string, cfg config.SiteConfig, manifest Manifest) error {
	if !cfg.Pipeline.Fingerprint {
		return nil
	}
	plain, err := safepath.Join(outputRoot, rel)
	if err != nil {
		return fmt.Errorf("resolve compiled css %q: %w", rel, err)
	}
	data, err := os.ReadFile(plain)
	if err != nil {
		return fmt.Errorf("read compiled css %q: %w", rel, err)
	}
	fpPath, err := fingerprintData(data, rel, outputRoot)
	if err != nil {
		return fmt.Errorf("fingerprint %q: %w", rel, err)
	}
	manifest[rel] = fpPath
	return os.Remove(plain)
}

// IsImageFile reports whether the pipeline can resize a file. WebP is read
// but not encoded, so it is copied like any other asset.
func IsImageFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif":
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
