package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/safepath"
)

func compileSCSS(siteRoot, staticRoot, outputRoot string, cfg config.SiteConfig) error {
	entryDir := cfg.Pipeline.SCSS.EntryDir
	if entryDir == "" {
		entryDir = "css"
	}

	scssRoots := []string{}

		if cfg.Theme != "" {
			themeSCSSRoot, err := safepath.Join(siteRoot, filepath.Join(cfg.ThemesDir, cfg.Theme, "scss"))
			if err != nil {
				return fmt.Errorf("resolve theme scss dir: %w", err)
			}
			if err := safepath.EnsureNoSymlinkPath(siteRoot, themeSCSSRoot); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("check theme scss dir: %w", err)
			}
			if _, err := os.Stat(themeSCSSRoot); err == nil {
				scssRoots = append(scssRoots, themeSCSSRoot)
			}
		}

	siteSCSSRoot, err := safepath.Join(staticRoot, entryDir)
	if err != nil {
		return fmt.Errorf("resolve site scss dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(siteRoot, siteSCSSRoot); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check site scss dir: %w", err)
	}
	if _, err := os.Stat(siteSCSSRoot); err == nil {
		scssRoots = append(scssRoots, siteSCSSRoot)
	}

	for _, scssRoot := range scssRoots {
		if err := compileSCSSDir(scssRoot, outputRoot, entryDir); err != nil {
			return err
		}
	}

	return nil
}

func compileSCSSDir(scssRoot, outputRoot, entryDir string) error {
	if err := safepath.RejectSymlink(scssRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("check scss dir: %w", err)
	}
	if _, err := os.Stat(scssRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat scss dir: %w", err)
	}

	sassPath, err := exec.LookPath("sass")
	if err != nil {
		return fmt.Errorf("SCSS compilation requires the 'sass' CLI (https://sass-lang.com/install): %w", err)
	}

	cssOutput, err := safepath.Join(outputRoot, entryDir)
	if err != nil {
		return fmt.Errorf("resolve css output dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(outputRoot, cssOutput); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check css output dir: %w", err)
	}
	if err := os.MkdirAll(cssOutput, 0o755); err != nil {
		return fmt.Errorf("create css output dir: %w", err)
	}

	entries, err := os.ReadDir(scssRoot)
	if err != nil {
		return fmt.Errorf("read scss dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".scss") && !strings.HasSuffix(name, ".sass") {
			continue
		}
		if strings.HasPrefix(name, "_") {
			continue
		}

		srcPath := filepath.Join(scssRoot, name)
		outName := strings.TrimSuffix(name, filepath.Ext(name)) + ".css"
		outPath := filepath.Join(cssOutput, outName)

		cmd := exec.Command(sassPath, "--no-source-map", "--style=compressed", srcPath, outPath)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("compile %q: %w", name, err)
		}
	}

	return nil
}
