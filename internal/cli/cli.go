package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/MohamedElashri/nida/internal/archetype"
	"github.com/MohamedElashri/nida/internal/assets"
	"github.com/MohamedElashri/nida/internal/buildinfo"
	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/output"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/scaffold"
	"github.com/MohamedElashri/nida/internal/server"
	"github.com/MohamedElashri/nida/internal/site"
	"github.com/MohamedElashri/nida/internal/watcher"
)

// Run executes a nida command and returns the process exit code.
func Run(args []string) int {
	return run(os.Stdout, os.Stderr, args)
}

type commandOptions struct {
	siteRoot   string
	configPath string
	drafts     bool
	future     bool
	port       int
	baseURL    string
	outputDir  string
	external   bool
}

type initOptions struct {
	targetDir string
}

type buildResult struct {
	cfg   config.SiteConfig
	path  string
	state site.State
	pages []render.Page
}

func run(stdout, stderr io.Writer, args []string) int {
	if len(args) == 0 {
		writeUsage(stderr)
		return 1
	}

	switch args[0] {
	case "init":
		opts, err := parseInitFlags(args[1:])
		if err != nil {
			return writeCommandError(stderr, err)
		}
		return runInit(stdout, stderr, opts)
	case "new":
		opts, target, err := parseNewFlags(args[1:])
		if err != nil {
			return writeCommandError(stderr, err)
		}
		return runNew(stdout, stderr, opts, target)
	case "check":
		opts, err := parseCheckFlags(args[1:])
		if err != nil {
			return writeCommandError(stderr, err)
		}
		return runCheck(stdout, stderr, opts)
	case "build":
		opts, err := parseBuildFlags(args[1:])
		if err != nil {
			return writeCommandError(stderr, err)
		}
		return runBuild(stdout, stderr, opts)
	case "serve":
		opts, err := parseServeFlags(args[1:])
		if err != nil {
			return writeCommandError(stderr, err)
		}
		return runServe(stdout, stderr, opts)
	case "version", "--version":
		writeVersion(stdout)
		return 0
	case "-h", "--help", "help":
		writeUsage(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		writeUsage(stderr)
		return 1
	}
}

func parseInitFlags(args []string) (initOptions, error) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	positional, err := parseArgs(fs, args)
	if err != nil {
		return initOptions{}, err
	}
	if len(positional) > 1 {
		return initOptions{}, fmt.Errorf("init accepts at most one path argument")
	}

	opts := initOptions{targetDir: "."}
	if len(positional) == 1 {
		opts.targetDir = positional[0]
	}
	return opts, nil
}

func parseNewFlags(args []string) (commandOptions, string, error) {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := commandOptions{}
	addSiteFlags(fs, &opts)

	positional, err := parseArgs(fs, args)
	if err != nil {
		return commandOptions{}, "", err
	}
	if len(positional) != 1 {
		return commandOptions{}, "", fmt.Errorf("new expects one content path, such as posts/my-post.md")
	}
	return opts, positional[0], nil
}

func runNew(stdout, stderr io.Writer, opts commandOptions, target string) int {
	cfg, _, err := loadCommandConfig(opts)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	result, err := archetype.Create(opts.siteRoot, cfg, target, time.Now())
	if err != nil {
		return writeCommandError(stderr, err)
	}
	colors := colorsFor(stdout)
	_, _ = fmt.Fprintf(stdout, "%s %s path=%s archetype=%s\n", colors.command("nida new:"), colors.success("created"), result.Path, result.Archetype)
	return 0
}

// addSiteFlags registers the flags shared by commands that load a site.
func addSiteFlags(fs *flag.FlagSet, opts *commandOptions) {
	fs.StringVar(&opts.siteRoot, "site", ".", "site root")
	fs.StringVar(&opts.siteRoot, "s", ".", "site root")
	fs.StringVar(&opts.configPath, "config", "", "config file path")
	fs.StringVar(&opts.configPath, "c", "", "config file path")
}

// addContentFlags registers flags that change which content is built.
func addContentFlags(fs *flag.FlagSet, opts *commandOptions) {
	fs.BoolVar(&opts.drafts, "drafts", false, "include draft content")
	fs.BoolVar(&opts.drafts, "d", false, "include draft content")
	fs.BoolVar(&opts.future, "future", false, "include future-dated content")
	fs.BoolVar(&opts.future, "f", false, "include future-dated content")
	fs.StringVar(&opts.baseURL, "base-url", "", "override base_url")
	fs.StringVar(&opts.outputDir, "output", "", "override output_dir")
	fs.StringVar(&opts.outputDir, "o", "", "override output_dir")
}

// parseArgs parses flags wherever they appear in args, not only before the
// first positional argument as flag.FlagSet.Parse does, and returns the
// positional arguments in order. Everything after "--" is positional.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		if consumed := len(args) - len(rest); consumed > 0 && args[consumed-1] == "--" {
			return append(positional, rest...), nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

// parseSiteArgs parses flags for commands that load a site and accepts the
// site root as an optional positional argument, so "nida serve PATH" works
// like "nida serve --site PATH".
func parseSiteArgs(fs *flag.FlagSet, opts *commandOptions, args []string) error {
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	switch {
	case len(positional) == 0:
		return nil
	case len(positional) > 1:
		return fmt.Errorf("%s accepts at most one site path argument, got %q", fs.Name(), positional)
	}
	siteFlagSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "site" || f.Name == "s" {
			siteFlagSet = true
		}
	})
	if siteFlagSet {
		return fmt.Errorf("%s: pass the site path as an argument or with --site, not both", fs.Name())
	}
	opts.siteRoot = positional[0]
	return nil
}

func parseBuildFlags(args []string) (commandOptions, error) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := commandOptions{}
	addSiteFlags(fs, &opts)
	addContentFlags(fs, &opts)

	if err := parseSiteArgs(fs, &opts, args); err != nil {
		return commandOptions{}, err
	}
	return opts, nil
}

func parseServeFlags(args []string) (commandOptions, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := commandOptions{}
	addSiteFlags(fs, &opts)
	addContentFlags(fs, &opts)
	fs.IntVar(&opts.port, "port", 0, "override server port")
	fs.IntVar(&opts.port, "p", 0, "override server port")

	if err := parseSiteArgs(fs, &opts, args); err != nil {
		return commandOptions{}, err
	}
	return opts, nil
}

func parseCheckFlags(args []string) (commandOptions, error) {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := commandOptions{}
	addSiteFlags(fs, &opts)
	fs.BoolVar(&opts.drafts, "drafts", false, "include draft content")
	fs.BoolVar(&opts.drafts, "d", false, "include draft content")
	fs.BoolVar(&opts.future, "future", false, "include future-dated content")
	fs.BoolVar(&opts.future, "f", false, "include future-dated content")
	fs.BoolVar(&opts.external, "external", false, "also check external links over the network")

	if err := parseSiteArgs(fs, &opts, args); err != nil {
		return commandOptions{}, err
	}
	return opts, nil
}

func runInit(stdout, stderr io.Writer, opts initOptions) int {
	result, err := scaffold.Init(scaffold.Options{TargetDir: opts.targetDir})
	if err != nil {
		return writeCommandError(stderr, err)
	}

	colors := colorsFor(stdout)
	_, _ = fmt.Fprintf(stdout, "%s %s path=%s files=%d\n", colors.command("nida init:"), colors.success("created"), result.TargetDir, len(result.Files))
	_, _ = fmt.Fprintf(stdout, "Next: nida serve --site %s\n", result.TargetDir)
	return 0
}

func runBuild(stdout, stderr io.Writer, opts commandOptions) int {
	started := time.Now()
	result, err := buildSite(opts)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	writeWarnings(stderr, result.cfg.Warnings)

	colors := colorsFor(stdout)
	_, _ = fmt.Fprintf(stdout, "%s %s config=%s drafts=%t output=%s pages=%d routes=%d rendered=%d time=%s\n", colors.command("nida build:"), colors.success("complete"), result.path, result.cfg.Drafts, result.cfg.OutputDir, len(result.state.Index.AllPages), len(result.state.Index.RouteRegistry), len(result.pages), formatElapsed(time.Since(started)))
	return 0
}

func runServe(stdout, stderr io.Writer, opts commandOptions) int {
	current, err := buildSite(opts)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	writeWarnings(stderr, current.cfg.Warnings)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	absSiteRoot, err := filepath.Abs(opts.siteRoot)
	if err != nil {
		return writeCommandError(stderr, fmt.Errorf("resolve site root %q: %w", opts.siteRoot, err))
	}
	outputDir, err := safepath.Join(absSiteRoot, current.cfg.OutputDir)
	if err != nil {
		return writeCommandError(stderr, fmt.Errorf("resolve output dir: %w", err))
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, outputDir); err != nil {
		return writeCommandError(stderr, fmt.Errorf("check output dir: %w", err))
	}
	instance, err := server.Start(ctx, outputDir, current.cfg.Server.Host, current.cfg.Server.Port, current.cfg.Server.Livereload, config.BasePath(current.cfg.BaseURL))
	if err != nil {
		return writeCommandError(stderr, err)
	}

	stdoutColors := colorsFor(stdout)
	stderrColors := colorsFor(stderr)
	_, _ = fmt.Fprintf(stdout, "%s %s config=%s drafts=%t host=%s port=%d routes=%d rendered=%d address=%s\n", stdoutColors.command("nida serve:"), stdoutColors.success("listening"), current.path, current.cfg.Drafts, current.cfg.Server.Host, current.cfg.Server.Port, len(current.state.Index.RouteRegistry), len(current.pages), stdoutColors.highlight(instance.Address))

	var rebuildMu sync.Mutex
	go func() {
		err := watcher.Run(ctx, watcher.Options{
			SiteRoot:  opts.siteRoot,
			OutputDir: current.cfg.OutputDir,
			OnChange: func(paths []string) {
				rebuildMu.Lock()
				defer rebuildMu.Unlock()

				_, _ = fmt.Fprintf(stdout, "%s rebuild triggered by %s\n", stdoutColors.command("nida serve:"), strings.Join(paths, ", "))
				next, mode, buildErr := rebuildSite(opts, current, paths)
				if buildErr != nil {
					_, _ = fmt.Fprintf(stderr, "%s rebuild failed: %v\n", stderrColors.error("error:"), buildErr)
					return
				}
				if next.cfg.Server.Host != current.cfg.Server.Host || next.cfg.Server.Port != current.cfg.Server.Port || config.BasePath(next.cfg.BaseURL) != config.BasePath(current.cfg.BaseURL) {
					_, _ = fmt.Fprintf(stdout, "%s %s config changed server address to %s; restart required to apply\n", stdoutColors.command("nida serve:"), stdoutColors.warning("warning:"), stdoutColors.highlight(fmt.Sprintf("%s:%d", next.cfg.Server.Host, next.cfg.Server.Port)))
				}
				current = next
				instance.Reload()
				_, _ = fmt.Fprintf(stdout, "%s %s mode=%s pages=%d routes=%d rendered=%d\n", stdoutColors.command("nida serve:"), stdoutColors.success("rebuild complete"), stdoutColors.highlight(mode), len(current.state.Index.AllPages), len(current.state.Index.RouteRegistry), len(current.pages))
			},
			OnError: func(err error) {
				_, _ = fmt.Fprintf(stderr, "%s watcher snapshot failed: %v\n", stderrColors.error("error:"), err)
			},
		})
		if err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(stderr, "%s watcher failed: %v\n", stderrColors.error("error:"), err)
			stop()
		}
	}()

	<-ctx.Done()
	_, _ = fmt.Fprintln(stdout, stdoutColors.command("nida serve:"), "shutting down")
	return 0
}

func buildSite(opts commandOptions) (buildResult, error) {
	result, err := buildSiteState(opts)
	if err != nil {
		return buildResult{}, err
	}
	if err := output.WriteSite(opts.siteRoot, result.cfg, result.pages); err != nil {
		return buildResult{}, err
	}
	if err := writeGenerated(opts.siteRoot, result); err != nil {
		return buildResult{}, err
	}
	if err := assets.Copy(opts.siteRoot, result.cfg); err != nil {
		return buildResult{}, err
	}
	if err := assets.CopyPageBundles(opts.siteRoot, result.cfg, bundlePagesFromIndex(result.state.Index)); err != nil {
		return buildResult{}, err
	}
	return result, nil
}

func loadCommandConfig(opts commandOptions) (config.SiteConfig, string, error) {
	cfg, path, err := config.Load(config.Options{
		SiteRoot: opts.siteRoot,
		Path:     opts.configPath,
	})
	if err != nil {
		return config.SiteConfig{}, "", err
	}

	if opts.drafts {
		cfg.Drafts = true
	}
	if opts.future {
		cfg.Future = true
	}
	if opts.port != 0 {
		cfg.Server.Port = opts.port
	}
	if opts.baseURL != "" || opts.outputDir != "" {
		if opts.baseURL != "" {
			cfg.BaseURL = strings.TrimSpace(opts.baseURL)
		}
		if opts.outputDir != "" {
			cfg.OutputDir = safepath.CleanRelative(opts.outputDir)
		}
		if err := config.Validate(cfg); err != nil {
			return config.SiteConfig{}, "", fmt.Errorf("invalid command-line override: %w", err)
		}
	}
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return config.SiteConfig{}, "", errors.New("server.port must be between 1 and 65535")
	}

	return cfg, path, nil
}

func writeCommandError(stderr io.Writer, err error) int {
	colors := colorsFor(stderr)
	_, _ = fmt.Fprintf(stderr, "%s %v\n", colors.error("error:"), err)
	return 1
}

func writeWarnings(stderr io.Writer, warnings []string) {
	colors := colorsFor(stderr)
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(stderr, "%s %s\n", colors.warning("warning:"), warning)
	}
}

func writeUsage(w io.Writer) {
	_, _ = io.WriteString(w, `Usage:
  nida init [PATH]
  nida new [-s PATH] [--site PATH] [-c PATH] [--config PATH] CONTENT_PATH
  nida serve [PATH] [-s PATH] [--site PATH] [-c PATH] [--config PATH] [-d] [--drafts] [-f] [--future] [--base-url URL] [-o DIR] [--output DIR] [-p PORT] [--port PORT]
  nida build [PATH] [-s PATH] [--site PATH] [-c PATH] [--config PATH] [-d] [--drafts] [-f] [--future] [--base-url URL] [-o DIR] [--output DIR]
  nida check [PATH] [-s PATH] [--site PATH] [-c PATH] [--config PATH] [-d] [--drafts] [-f] [--future] [--external]
  nida version

Commands:
  init    Create an example site to start from
  new     Create a content file from an archetype, e.g. posts/my-post.md
  serve   Build, watch, and serve the local site
  build   Build the site into the configured output directory
  check   Validate config, content, links, and templates without writing output
  version Show nida build and version information

serve, build, and check use the current directory as the site root unless
a PATH argument or --site is given. Flags may come before or after PATH.
`)
}

func writeVersion(w io.Writer) {
	_, _ = fmt.Fprintln(w, buildinfo.Summary())
}

func bundlePagesFromIndex(index site.SiteIndex) []assets.BundlePage {
	var bundles []assets.BundlePage
	for _, p := range index.AllPages {
		if p.IsBundle && len(p.Resources) > 0 {
			bundles = append(bundles, assets.BundlePage{
				URL:       p.URL,
				BundleDir: p.BundleDir,
				Resources: p.Resources,
			})
		}
	}
	return bundles
}
