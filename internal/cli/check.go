package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/MohamedElashri/nida/internal/linkcheck"
	"github.com/MohamedElashri/nida/internal/output"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/site"
)

// runCheck runs everything a build does up to writing files: config
// validation, content diagnostics, Markdown and template rendering, and
// route conflict checks. With --external it also requests external links.
func runCheck(stdout, stderr io.Writer, opts commandOptions) int {
	started := time.Now()
	cfg, path, err := loadCommandConfig(opts)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	writeWarnings(stderr, cfg.Warnings)
	cfg.Diagnostics.Enabled = true

	state, err := site.Load(opts.siteRoot, cfg)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	pages, err := render.RenderSite(opts.siteRoot, cfg, state)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if err := output.ValidateWritePlan(opts.siteRoot, cfg, pages, buildArtifactList(cfg, state.Index)); err != nil {
		return writeCommandError(stderr, err)
	}

	colors := colorsFor(stdout)
	external := ""
	if opts.external {
		sources := make([]linkcheck.Source, 0, len(state.Index.AllPages))
		for _, page := range state.Index.AllPages {
			sources = append(sources, linkcheck.Source{Path: page.RelativePath, HTML: page.BodyHTML})
		}
		links := linkcheck.Collect(sources)
		problems := linkcheck.Check(context.Background(), links, nil)
		for _, problem := range problems {
			_, _ = fmt.Fprintf(stderr, "%s %s: %s (in %v)\n", colorsFor(stderr).error("broken link:"), problem.URL, problem.Message, problem.Pages)
		}
		if len(problems) > 0 {
			return writeCommandError(stderr, fmt.Errorf("%d of %d external links failed", len(problems), len(links)))
		}
		external = fmt.Sprintf(" external_links=%d", len(links))
	}

	_, _ = fmt.Fprintf(stdout, "%s %s config=%s pages=%d routes=%d warnings=%d%s time=%s\n", colors.command("nida check:"), colors.success("ok"), path, len(state.Index.AllPages), len(pages), len(cfg.Warnings), external, formatElapsed(time.Since(started)))
	return 0
}

func formatElapsed(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
