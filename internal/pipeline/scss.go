package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
)

func compileSCSS(staticRoot, outputRoot string, cfg config.SiteConfig) error {
	entryDir := cfg.Pipeline.SCSS.EntryDir
	if entryDir == "" {
		entryDir = "css"
	}

	scssRoots := []string{}

	if cfg.Theme != "" {
		themeSCSSRoot := filepath.Join(staticRoot, "..", cfg.ThemesDir, cfg.Theme, "scss")
		if _, err := os.Stat(themeSCSSRoot); err == nil {
			scssRoots = append(scssRoots, themeSCSSRoot)
		}
	}

	siteSCSSRoot := filepath.Join(staticRoot, entryDir)
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

	cssOutput := filepath.Join(outputRoot, entryDir)
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
