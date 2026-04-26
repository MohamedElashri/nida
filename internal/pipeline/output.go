package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
)

func RewriteOutputFiles(siteRoot string, cfg config.SiteConfig, manifest Manifest) error {
	if len(manifest) == 0 {
		return nil
	}

	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root: %w", err)
	}

	outputDir := filepath.Join(absSiteRoot, cfg.OutputDir)

	return filepath.WalkDir(outputDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %q: %w", path, err)
		}

		rewritten := RewriteHTML(string(data), manifest)
		if rewritten == string(data) {
			return nil
		}

		if err := os.WriteFile(path, []byte(rewritten), 0o644); err != nil {
			return fmt.Errorf("write %q: %w", path, err)
		}

		return nil
	})
}
