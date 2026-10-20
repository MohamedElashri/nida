package assets

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/pipeline"
	"github.com/MohamedElashri/nida/internal/route"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/themes"
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

	// Copying skips files that already exist, so the site comes first, then
	// the selected theme, then the themes it extends.
	themeStatic, err := themes.Dirs(absSiteRoot, cfg, "static")
	if err != nil {
		return err
	}
	for i := len(themeStatic) - 1; i >= 0; i-- {
		staticRoots = append(staticRoots, themeStatic[i])
	}

	for _, staticRoot := range staticRoots {
		if err := copyDir(staticRoot, outputRoot, cfg); err != nil {
			return err
		}
	}

	return nil
}

func copyDir(staticRoot, outputRoot string, cfg config.SiteConfig) error {
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
		if !d.IsDir() && skipStatic(cfg, rel) {
			return nil
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

		return safepath.CopyFile(path, target)
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
	// Copying skips files that already exist, so the site comes first, then
	// the selected theme, then the themes it extends.
	themeStatic, err := themes.Dirs(absSiteRoot, cfg, "static")
	if err != nil {
		return err
	}
	for i := len(themeStatic) - 1; i >= 0; i-- {
		staticRoots = append(staticRoots, themeStatic[i])
	}

	prefixes := make([]string, len(staticRoots))
	for i, root := range staticRoots {
		if prefixes[i], err = staticRootPrefix(absSiteRoot, root); err != nil {
			return err
		}
	}

	for _, changedPath := range changedPaths {
		normalized, err := changedPathRelativeToSite(absSiteRoot, changedPath)
		if err != nil {
			return err
		}
		if normalized == "" || safepath.Escapes(normalized) {
			continue
		}
		rel := ""
		for _, prefix := range prefixes {
			if prefix != "" && strings.HasPrefix(normalized, prefix) {
				rel = strings.TrimPrefix(normalized, prefix)
				break
			}
		}
		if rel == "" && slices.Contains(prefixes, "") {
			// static_dir = "." serves the whole site root.
			rel = normalized
		}
		if strings.TrimSpace(rel) == "" || skipStatic(cfg, rel) {
			continue
		}
		if err := syncStatic(staticRoots, outputRoot, rel, cfg); err != nil {
			return err
		}
	}

	return nil
}

// syncStatic updates one output path from the first static root that has
// it, so the site overrides its theme as in a full build, and removes the
// output when no root has it.
func syncStatic(staticRoots []string, outputRoot, rel string, cfg config.SiteConfig) error {
	target, err := safepath.Join(outputRoot, rel)
	if err != nil {
		return fmt.Errorf("resolve output path %q: %w", rel, err)
	}
	if err := safepath.EnsureNoSymlinkPath(outputRoot, target); err != nil {
		return fmt.Errorf("check output path %q: %w", target, err)
	}
	for _, root := range staticRoots {
		source, err := safepath.Join(root, rel)
		if err != nil {
			return fmt.Errorf("resolve static asset %q: %w", rel, err)
		}
		info, err := os.Lstat(source)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("stat static asset %q: %w", source, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse static symlink %q", source)
		}
		if info.IsDir() {
			// A directory moved in brings files that report no changes
			// of their own.
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create output directory for %q: %w", target, err)
			}
			entries, err := os.ReadDir(source)
			if err != nil {
				return fmt.Errorf("read static directory %q: %w", source, err)
			}
			for _, entry := range entries {
				child := path.Join(rel, entry.Name())
				if !entry.IsDir() && skipStatic(cfg, child) {
					continue
				}
				if err := syncStatic(staticRoots, outputRoot, child, cfg); err != nil {
					return err
				}
			}
			return nil
		}
		return safepath.CopyFile(source, target)
	}
	if info, err := os.Lstat(target); err == nil && info.IsDir() {
		// Rendered pages may share the directory, so only an empty one
		// is removed; the next full build drops anything left.
		_ = os.Remove(target)
		return nil
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale asset %q: %w", target, err)
	}
	return nil
}

// skipStatic reports static files that are not published as they are:
// SCSS sources, which the pipeline compiles.
func skipStatic(cfg config.SiteConfig, rel string) bool {
	return cfg.Pipeline.SCSS.Enabled && pipeline.IsSass(rel)
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

	// Pages written as files, such as /posts/a.html and /posts/b.html,
	// share a directory for their resources; refuse to let one page's
	// cover.png silently replace another's.
	written := map[string]string{}
	for _, page := range pages {
		if len(page.Resources) == 0 {
			continue
		}

		sourceDir, err := safepath.Join(contentRoot, page.BundleDir)
		if err != nil {
			return fmt.Errorf("resolve bundle dir %q: %w", page.BundleDir, err)
		}
		outputURL := strings.TrimSuffix(route.Dir(page.URL), "/")
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
			if other, ok := written[dst]; ok && other != page.URL {
				return fmt.Errorf("page resource %q of %s would overwrite the one of %s; rename one of them", res, page.URL, other)
			}
			written[dst] = page.URL
			if err := safepath.CopyFile(src, dst); err != nil {
				return fmt.Errorf("copy page resource %q: %w", res, err)
			}
			if cfg.Pipeline.Images.Enabled {
				if err := pipeline.ResizeVariants(src, filepath.Dir(dst), cfg); err != nil {
					return fmt.Errorf("resize page resource %q: %w", res, err)
				}
			}
		}
	}

	return nil
}
