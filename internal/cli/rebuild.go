package cli

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/assets"
	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/feeds"
	"github.com/MohamedElashri/nida/internal/output"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/site"
	"github.com/MohamedElashri/nida/internal/sitemap"
)

func rebuildSite(opts commandOptions, previous buildResult, changedPaths []string) (buildResult, string, error) {
	mode := rebuildMode(previous.cfg, changedPaths)
	if mode == "assets-only" {
		if err := assets.SyncChanged(opts.siteRoot, previous.cfg, changedPaths); err != nil {
			return previous, mode, err
		}
		return previous, mode, nil
	}

	next, err := buildSiteState(opts)
	if err != nil {
		return buildResult{}, mode, err
	}

	if err := writeIncrementalOutputs(opts, previous, next, changedPaths, mode); err != nil {
		return buildResult{}, mode, err
	}

	return next, mode, nil
}

func buildSiteState(opts commandOptions) (buildResult, error) {
	cfg, path, err := loadCommandConfig(opts)
	if err != nil {
		return buildResult{}, err
	}

	state, err := site.Load(opts.siteRoot, cfg)
	if err != nil {
		return buildResult{}, err
	}

	pages, err := render.RenderSite(opts.siteRoot, cfg, state)
	if err != nil {
		return buildResult{}, err
	}

	artifacts := []output.Artifact{}
	if cfg.RSS.Enabled {
		artifacts = append(artifacts, output.Artifact{Path: cfg.RSS.Filename})
	}
	if cfg.Sitemap.Enabled {
		artifacts = append(artifacts, output.Artifact{Path: cfg.Sitemap.Filename})
	}
	if err := output.ValidateWritePlan(opts.siteRoot, cfg, pages, artifacts); err != nil {
		return buildResult{}, err
	}

	return buildResult{cfg: cfg, path: path, state: state, pages: pages}, nil
}

func writeIncrementalOutputs(opts commandOptions, previous, next buildResult, changedPaths []string, mode string) error {
	changedPages, removedRoutes := diffRenderedPages(previous.pages, next.pages)

	switch mode {
	case "full":
		if err := output.WritePages(opts.siteRoot, next.cfg, next.pages); err != nil {
			return err
		}
	default:
		if len(changedPages) > 0 {
			if err := output.WritePages(opts.siteRoot, next.cfg, changedPages); err != nil {
				return err
			}
		}
	}
	if len(removedRoutes) > 0 {
		if err := output.RemovePages(opts.siteRoot, previous.cfg, removedRoutes); err != nil {
			return err
		}
	}

	prevArtifacts := artifactPaths(previous.cfg)
	nextArtifacts := artifactPaths(next.cfg)
	for _, oldPath := range prevArtifacts {
		if !slices.Contains(nextArtifacts, oldPath) {
			if err := output.RemoveFile(opts.siteRoot, previous.cfg, oldPath); err != nil {
				return err
			}
		}
	}

	feedOutput, err := feeds.Generate(next.cfg, next.state.Index)
	if err != nil {
		return err
	}
	if feedOutput != nil {
		if err := output.WriteFile(opts.siteRoot, next.cfg, feedOutput.Filename, feedOutput.Content); err != nil {
			return err
		}
	}

	sitemapOutput, err := sitemap.Generate(next.cfg, next.state, next.pages)
	if err != nil {
		return err
	}
	if sitemapOutput != nil {
		if err := output.WriteFile(opts.siteRoot, next.cfg, sitemapOutput.Filename, sitemapOutput.Content); err != nil {
			return err
		}
	}

	if err := assets.SyncChanged(opts.siteRoot, next.cfg, changedStaticPaths(next.cfg, changedPaths)); err != nil {
		return err
	}

	return nil
}

func rebuildMode(cfg config.SiteConfig, changedPaths []string) string {
	staticPrefix := filepath.ToSlash(strings.Trim(cfg.StaticDir, "/")) + "/"
	contentPrefix := filepath.ToSlash(strings.Trim(cfg.ContentDir, "/")) + "/"
	templatePrefix := filepath.ToSlash(strings.Trim(cfg.TemplateDir, "/")) + "/"
	configName := filepath.Base(config.DefaultConfigName)

	mode := "partial"
	for _, changedPath := range changedPaths {
		path := filepath.ToSlash(strings.TrimSpace(changedPath))
		switch {
		case path == configName || strings.HasSuffix(path, "/"+configName):
			return "full"
		case strings.HasPrefix(path, templatePrefix):
			return "full"
		case strings.HasPrefix(path, staticPrefix):
			if mode == "partial" {
				mode = "assets-only"
			}
		case strings.HasPrefix(path, contentPrefix):
			mode = "partial"
		default:
			return "full"
		}
	}
	return mode
}

func diffRenderedPages(previous, next []render.Page) ([]render.Page, []string) {
	prevByURL := map[string]render.Page{}
	for _, page := range previous {
		prevByURL[page.URL] = page
	}

	nextByURL := map[string]render.Page{}
	changed := make([]render.Page, 0)
	for _, page := range next {
		nextByURL[page.URL] = page
		prev, ok := prevByURL[page.URL]
		if !ok || prev.Content != page.Content || prev.CanonicalURL != page.CanonicalURL || prev.TemplateName != page.TemplateName || prev.Title != page.Title {
			changed = append(changed, page)
		}
	}

	removed := make([]string, 0)
	for url := range prevByURL {
		if _, ok := nextByURL[url]; !ok {
			removed = append(removed, url)
		}
	}

	slices.SortFunc(changed, func(a, b render.Page) int { return strings.Compare(a.URL, b.URL) })
	slices.Sort(removed)
	return changed, removed
}

func artifactPaths(cfg config.SiteConfig) []string {
	paths := make([]string, 0, 2)
	if cfg.RSS.Enabled {
		paths = append(paths, cfg.RSS.Filename)
	}
	if cfg.Sitemap.Enabled {
		paths = append(paths, cfg.Sitemap.Filename)
	}
	return paths
}

func changedStaticPaths(cfg config.SiteConfig, changedPaths []string) []string {
	if len(changedPaths) == 0 {
		return nil
	}
	prefix := filepath.ToSlash(strings.Trim(cfg.StaticDir, "/")) + "/"
	out := make([]string, 0, len(changedPaths))
	for _, path := range changedPaths {
		if strings.HasPrefix(filepath.ToSlash(path), prefix) {
			out = append(out, filepath.ToSlash(path))
		}
	}
	return out
}
