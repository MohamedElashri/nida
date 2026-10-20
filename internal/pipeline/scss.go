package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/themes"
)

// compileSCSS compiles each SCSS entry into the output and returns the
// compiled files relative to the output root, such as css/main.css.
func compileSCSS(siteRoot, staticRoot, outputRoot string, cfg config.SiteConfig) ([]string, error) {
	entryDir := cfg.Pipeline.SCSS.EntryDir
	if entryDir == "" {
		entryDir = "css"
	}

	scssRoots := []string{}

	themeSCSS, err := themes.Dirs(siteRoot, cfg, "scss")
	if err != nil {
		return nil, err
	}
	scssRoots = append(scssRoots, themeSCSS...)

	siteSCSSRoot, err := safepath.Join(staticRoot, entryDir)
	if err != nil {
		return nil, fmt.Errorf("resolve site scss dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(siteRoot, siteSCSSRoot); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("check site scss dir: %w", err)
	}
	if _, err := os.Stat(siteSCSSRoot); err == nil {
		scssRoots = append(scssRoots, siteSCSSRoot)
	}

	var compiled []string
	for _, scssRoot := range scssRoots {
		names, err := compileSCSSDir(scssRoot, outputRoot, entryDir)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			if rel := path.Join(filepath.ToSlash(entryDir), name); !slices.Contains(compiled, rel) {
				compiled = append(compiled, rel)
			}
		}
	}
	return compiled, nil
}

func compileSCSSDir(scssRoot, outputRoot, entryDir string) ([]string, error) {
	if err := safepath.RejectSymlink(scssRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("check scss dir: %w", err)
	}
	if _, err := os.Stat(scssRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat scss dir: %w", err)
	}

	sassPath, err := exec.LookPath("sass")
	if err != nil {
		return nil, fmt.Errorf("SCSS compilation requires the 'sass' CLI (https://sass-lang.com/install): %w", err)
	}

	cssOutput, err := safepath.Join(outputRoot, entryDir)
	if err != nil {
		return nil, fmt.Errorf("resolve css output dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(outputRoot, cssOutput); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("check css output dir: %w", err)
	}
	if err := os.MkdirAll(cssOutput, 0o755); err != nil {
		return nil, fmt.Errorf("create css output dir: %w", err)
	}

	entries, err := os.ReadDir(scssRoot)
	if err != nil {
		return nil, fmt.Errorf("read scss dir: %w", err)
	}

	var compiled []string
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
			return nil, fmt.Errorf("compile %q: %w", name, err)
		}
		compiled = append(compiled, outName)
	}

	return compiled, nil
}
