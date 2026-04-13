package assets

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/MohamedElashri/nida/internal/config"
)

func Copy(siteRoot string, cfg config.SiteConfig) error {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	staticRoot := filepath.Join(absSiteRoot, cfg.StaticDir)
	outputRoot := filepath.Join(absSiteRoot, cfg.OutputDir)

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
			return fmt.Errorf("static asset %q conflicts with existing output %q", path, target)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat output path %q: %w", target, err)
		}

		return copyFile(path, target)
	})
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
