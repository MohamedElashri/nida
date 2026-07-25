package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/safepath"
)

type Options struct {
	SiteRoot     string
	OutputDir    string
	PollInterval time.Duration
	OnChange     func([]string)
	OnError      func(error)
}

type fileState struct {
	size    int64
	modTime time.Time
}

func Run(ctx context.Context, opts Options) error {
	if opts.OnChange == nil {
		return fmt.Errorf("watcher OnChange callback is required")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = time.Second
	}

	if err := runEventWatcher(ctx, opts); err == nil {
		return nil
	}

	return runPollingWatcher(ctx, opts)
}

func runPollingWatcher(ctx context.Context, opts Options) error {
	previous, err := snapshot(opts.SiteRoot, opts.OutputDir)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			current, err := snapshot(opts.SiteRoot, opts.OutputDir)
			if err != nil {
				if opts.OnError != nil {
					opts.OnError(err)
					continue
				}
				return err
			}

			changed := diff(previous, current)
			if len(changed) > 0 {
				opts.OnChange(changed)
				previous = current
			}
		}
	}
}

func snapshot(siteRoot, outputDir string) (map[string]fileState, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}
	absOutputDir, err := resolveOutputRoot(absSiteRoot, outputDir)
	if err != nil {
		return nil, err
	}

	files := make(map[string]fileState)
	err = filepath.WalkDir(absSiteRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if shouldSkipPath(path, absOutputDir) {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if walkErr != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(absSiteRoot, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = fileState{
			size:    info.Size(),
			modTime: info.ModTime().UTC(),
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("watch snapshot for %q: %w", siteRoot, err)
	}

	return files, nil
}

func resolveOutputRoot(absSiteRoot, outputDir string) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		return "", nil
	}

	absOutputDir, err := safepath.Join(absSiteRoot, outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output dir %q: %w", outputDir, err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, absOutputDir); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("check output dir %q: %w", outputDir, err)
	}
	return absOutputDir, nil
}

func shouldSkipPath(path, outputDir string) bool {
	cleanPath := filepath.Clean(path)
	if outputDir != "" && (cleanPath == outputDir || strings.HasPrefix(cleanPath, outputDir+string(filepath.Separator))) {
		return true
	}

	parts := strings.Split(cleanPath, string(filepath.Separator))
	for _, part := range parts {
		switch part {
		case ".git", ".hg", ".svn":
			return true
		}
	}

	return false
}

func diff(previous, current map[string]fileState) []string {
	changed := make([]string, 0)

	for path, state := range current {
		prev, ok := previous[path]
		if !ok || prev != state {
			changed = append(changed, path)
		}
	}
	for path := range previous {
		if _, ok := current[path]; !ok {
			changed = append(changed, path)
		}
	}

	sort.Strings(changed)
	return changed
}
