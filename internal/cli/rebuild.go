package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/assets"
	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/feeds"
	"github.com/MohamedElashri/nida/internal/markdown"
	"github.com/MohamedElashri/nida/internal/output"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/robots"
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

	var next buildResult
	var err error
	if mode == "partial" {
		next, err = buildIncremental(opts, previous, changedPaths)
	} else {
		next, err = buildSiteState(opts)
	}
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

	if err := output.ValidateWritePlan(opts.siteRoot, cfg, pages, buildArtifactList(cfg)); err != nil {
		return buildResult{}, err
	}

	return buildResult{cfg: cfg, path: path, state: state, pages: pages}, nil
}

func buildArtifactList(cfg config.SiteConfig) []output.Artifact {
	artifacts := make([]output.Artifact, 0, 4)
	if cfg.RSS.Enabled {
		artifacts = append(artifacts, output.Artifact{Path: cfg.RSS.Filename})
	}
	if cfg.Atom.Enabled {
		artifacts = append(artifacts, output.Artifact{Path: cfg.Atom.Filename})
	}
	if cfg.Sitemap.Enabled {
		artifacts = append(artifacts, output.Artifact{Path: cfg.Sitemap.Filename})
	}
	if cfg.Robots.Enabled {
		artifacts = append(artifacts, output.Artifact{Path: cfg.Robots.Filename})
	}
	return artifacts
}

func buildIncremental(opts commandOptions, previous buildResult, changedPaths []string) (buildResult, error) {
	cfg, path, err := loadCommandConfig(opts)
	if err != nil {
		return buildResult{}, err
	}

	absSiteRoot, err := filepath.Abs(opts.siteRoot)
	if err != nil {
		return buildResult{}, err
	}
	contentRoot := filepath.Join(absSiteRoot, cfg.ContentDir)
	contentPrefix := filepath.ToSlash(cfg.ContentDir + "/")

	changedByRelPath := make(map[string]content.Item)
	removedByRelPath := make(map[string]bool)

	for _, p := range changedPaths {
		normalized := filepath.ToSlash(strings.TrimSpace(p))
		if !strings.HasPrefix(normalized, contentPrefix) {
			continue
		}
		if !strings.HasSuffix(normalized, ".md") {
			continue
		}

		relPath := strings.TrimPrefix(normalized, contentPrefix)
		fullPath := filepath.Join(contentRoot, filepath.FromSlash(relPath))

		info, statErr := os.Stat(fullPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				removedByRelPath[relPath] = true
				continue
			}
			return buildResult{}, statErr
		}
		if info.IsDir() {
			continue
		}

		item, loadErr := content.LoadFile(contentRoot, fullPath, cfg)
		if loadErr != nil {
			return buildResult{}, loadErr
		}

		item, renderErr := markdown.RenderItem(item, cfg)
		if renderErr != nil {
			return buildResult{}, renderErr
		}

		changedByRelPath[item.RelativePath] = item
	}

	changedItems := make([]content.Item, 0, len(changedByRelPath))
	for _, item := range changedByRelPath {
		changedItems = append(changedItems, item)
	}
	removedPaths := make([]string, 0, len(removedByRelPath))
	for p := range removedByRelPath {
		removedPaths = append(removedPaths, p)
	}

	merged := make([]content.Item, 0, len(previous.state.Items))
	for _, prevItem := range previous.state.Items {
		if removedByRelPath[prevItem.RelativePath] {
			continue
		}
		if updated, ok := changedByRelPath[prevItem.RelativePath]; ok {
			merged = append(merged, updated)
			delete(changedByRelPath, prevItem.RelativePath)
		} else {
			merged = append(merged, prevItem)
		}
	}
	for _, item := range changedByRelPath {
		merged = append(merged, item)
	}

	index, err := site.BuildIndex(merged, cfg)
	if err != nil {
		return buildResult{}, err
	}

	newState := site.State{Items: merged, Index: index}

	pages, err := render.RenderIncremental(opts.siteRoot, cfg, newState, previous.pages, changedItems, removedPaths)
	if err != nil {
		return buildResult{}, err
	}

	if err := output.ValidateWritePlan(opts.siteRoot, cfg, pages, buildArtifactList(cfg)); err != nil {
		return buildResult{}, err
	}

	return buildResult{cfg: cfg, path: path, state: newState, pages: pages}, nil
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

	feedOutputs, err := feeds.GenerateAll(next.cfg, next.state.Index)
	if err != nil {
		return err
	}
	for _, feedOutput := range feedOutputs {
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
	if robotsOutput := robots.Generate(next.cfg); robotsOutput != nil {
		if err := output.WriteFile(opts.siteRoot, next.cfg, robotsOutput.Filename, robotsOutput.Content); err != nil {
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
	if cfg.Atom.Enabled {
		paths = append(paths, cfg.Atom.Filename)
	}
	if cfg.Sitemap.Enabled {
		paths = append(paths, cfg.Sitemap.Filename)
	}
	if cfg.Robots.Enabled {
		paths = append(paths, cfg.Robots.Filename)
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
