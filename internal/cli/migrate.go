package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/migrate"
	"github.com/MohamedElashri/nida/internal/output"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/site"
)

type migrateOptions struct {
	source string
	dest   string
	from   string
	urls   string
	dryRun bool
	check  bool
}

func parseMigrateFlags(args []string) (migrateOptions, error) {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := migrateOptions{}
	fs.StringVar(&opts.from, "from", "auto", "source generator")
	fs.StringVar(&opts.urls, "urls", migrate.KeepURLs, "keep or redirect")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "print the report without writing")
	fs.BoolVar(&opts.dryRun, "n", false, "print the report without writing")
	noCheck := fs.Bool("no-check", false, "skip building the migrated site")

	positional, err := parseArgs(fs, args)
	if err != nil {
		return migrateOptions{}, err
	}
	if len(positional) != 2 {
		return migrateOptions{}, fmt.Errorf("migrate expects a source site and a new directory, such as: nida migrate ./old-site ./new-site")
	}
	opts.source, opts.dest = positional[0], positional[1]
	opts.check = !*noCheck
	return opts, nil
}

func runMigrate(stdout, stderr io.Writer, opts migrateOptions) int {
	started := time.Now()
	run := migrate.Options{
		Source: opts.source,
		Dest:   opts.dest,
		From:   opts.from,
		URLs:   opts.urls,
		DryRun: opts.dryRun,
	}
	if opts.check {
		run.Check = checkMigratedSite
	}
	result, err := migrate.Run(run)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	r := result.Report
	if opts.dryRun {
		_, _ = io.WriteString(stdout, result.Markdown)
		_, _ = io.WriteString(stdout, "\n")
	}

	colors := colorsFor(stdout)
	status := colors.success("complete")
	if opts.dryRun {
		status = colors.success("dry run")
	}
	check := r.Check
	if check == "" {
		check = "skipped"
	}
	_, _ = fmt.Fprintf(stdout, "%s %s from=%s pages=%d sections=%d copied=%d urls_kept=%d urls_moved=%d manual=%d unsupported=%d review=%d check=%s time=%s\n",
		colors.command("nida migrate:"), status, strings.ToLower(r.From), r.Pages, r.Sections, r.Copied, r.URLsKept, r.URLsMoved,
		r.Count(migrate.Manual), r.Count(migrate.Unsupported), r.Count(migrate.Review), firstWord(check), formatElapsed(time.Since(started)))
	if result.ReportPath != "" {
		_, _ = fmt.Fprintf(stdout, "Report: %s\n", result.ReportPath)
		_, _ = fmt.Fprintf(stdout, "Next: nida serve %s\n", filepath.Clean(opts.dest))
	}
	if strings.HasPrefix(check, "failed") {
		_, _ = fmt.Fprintf(stderr, "%s the migrated site does not build yet: %s\n", colorsFor(stderr).warning("warning:"), strings.TrimPrefix(check, "failed: "))
	}
	return 0
}

func firstWord(s string) string {
	word, _, _ := strings.Cut(s, " ")
	return strings.TrimSuffix(word, ":")
}

// checkMigratedSite builds a migrated site in memory the way nida check
// does, with drafts and future pages included so every page is tried.
func checkMigratedSite(siteRoot string) error {
	cfg, _, err := config.Load(config.Options{SiteRoot: siteRoot})
	if err != nil {
		return err
	}
	cfg.Diagnostics.Enabled = true
	cfg.Drafts, cfg.Future = true, true
	state, err := site.Load(siteRoot, cfg)
	if err != nil {
		return err
	}
	pages, err := render.RenderSite(siteRoot, cfg, state)
	if err != nil {
		return err
	}
	return output.ValidateWritePlan(siteRoot, cfg, pages, buildArtifactList(cfg, state.Index))
}
