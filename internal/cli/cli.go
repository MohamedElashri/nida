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

	"github.com/MohamedElashri/nida/internal/assets"
	"github.com/MohamedElashri/nida/internal/buildinfo"
	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/feeds"
	"github.com/MohamedElashri/nida/internal/output"
	"github.com/MohamedElashri/nida/internal/pipeline"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/robots"
	"github.com/MohamedElashri/nida/internal/server"
	"github.com/MohamedElashri/nida/internal/site"
	"github.com/MohamedElashri/nida/internal/sitemap"
	"github.com/MohamedElashri/nida/internal/watcher"
)

// Run executes the narrow public command surface defined in the project plan.
func Run(args []string) int {
	return run(os.Stdout, os.Stderr, args)
}

type commandOptions struct {
	siteRoot   string
	configPath string
	drafts     bool
	port       int
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

func parseBuildFlags(args []string) (commandOptions, error) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := commandOptions{}
	fs.StringVar(&opts.siteRoot, "site", ".", "site root")
	fs.StringVar(&opts.siteRoot, "s", ".", "site root")
	fs.StringVar(&opts.configPath, "config", "", "config file path")
	fs.StringVar(&opts.configPath, "c", "", "config file path")
	fs.BoolVar(&opts.drafts, "drafts", false, "include draft content")
	fs.BoolVar(&opts.drafts, "d", false, "include draft content")

	if err := fs.Parse(args); err != nil {
		return commandOptions{}, err
	}
	return opts, nil
}

func parseServeFlags(args []string) (commandOptions, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := commandOptions{}
	fs.StringVar(&opts.siteRoot, "site", ".", "site root")
	fs.StringVar(&opts.siteRoot, "s", ".", "site root")
	fs.StringVar(&opts.configPath, "config", "", "config file path")
	fs.StringVar(&opts.configPath, "c", "", "config file path")
	fs.BoolVar(&opts.drafts, "drafts", false, "include draft content")
	fs.BoolVar(&opts.drafts, "d", false, "include draft content")
	fs.IntVar(&opts.port, "port", 0, "override server port")
	fs.IntVar(&opts.port, "p", 0, "override server port")

	if err := fs.Parse(args); err != nil {
		return commandOptions{}, err
	}
	return opts, nil
}

func runBuild(stdout, stderr io.Writer, opts commandOptions) int {
	result, err := buildSite(opts)
	if err != nil {
		return writeCommandError(stderr, err)
	}

	_, _ = fmt.Fprintf(stdout, "nida build: config=%s drafts=%t output=%s posts=%d pages=%d routes=%d rendered=%d\n", result.path, result.cfg.Drafts, result.cfg.OutputDir, len(result.state.Index.Posts), len(result.state.Index.Pages), len(result.state.Index.RouteRegistry), len(result.pages))
	return 0
}

func runServe(stdout, stderr io.Writer, opts commandOptions) int {
	current, err := buildSite(opts)
	if err != nil {
		return writeCommandError(stderr, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	outputDir := filepath.Join(opts.siteRoot, current.cfg.OutputDir)
	instance, err := server.Start(ctx, outputDir, current.cfg.Server.Host, current.cfg.Server.Port, current.cfg.Server.Livereload)
	if err != nil {
		return writeCommandError(stderr, err)
	}

	_, _ = fmt.Fprintf(stdout, "nida serve: config=%s drafts=%t host=%s port=%d routes=%d rendered=%d address=%s\n", current.path, current.cfg.Drafts, current.cfg.Server.Host, current.cfg.Server.Port, len(current.state.Index.RouteRegistry), len(current.pages), instance.Address)

	var rebuildMu sync.Mutex
	go func() {
		err := watcher.Run(ctx, watcher.Options{
			SiteRoot:  opts.siteRoot,
			OutputDir: current.cfg.OutputDir,
			OnChange: func(paths []string) {
				rebuildMu.Lock()
				defer rebuildMu.Unlock()

				_, _ = fmt.Fprintf(stdout, "nida serve: rebuild triggered by %s\n", strings.Join(paths, ", "))
				next, mode, buildErr := rebuildSite(opts, current, paths)
				if buildErr != nil {
					_, _ = fmt.Fprintf(stderr, "error: rebuild failed: %v\n", buildErr)
					return
				}
				if next.cfg.Server.Host != current.cfg.Server.Host || next.cfg.Server.Port != current.cfg.Server.Port {
					_, _ = fmt.Fprintf(stdout, "nida serve: config changed server address to %s:%d; restart required to apply\n", next.cfg.Server.Host, next.cfg.Server.Port)
				}
				current = next
				instance.Reload()
				_, _ = fmt.Fprintf(stdout, "nida serve: rebuild complete mode=%s posts=%d pages=%d routes=%d rendered=%d\n", mode, len(current.state.Index.Posts), len(current.state.Index.Pages), len(current.state.Index.RouteRegistry), len(current.pages))
			},
			OnError: func(err error) {
				_, _ = fmt.Fprintf(stderr, "error: watcher snapshot failed: %v\n", err)
			},
		})
		if err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(stderr, "error: watcher failed: %v\n", err)
			stop()
		}
	}()

	<-ctx.Done()
	_, _ = fmt.Fprintln(stdout, "nida serve: shutting down")
	return 0
}

func buildSite(opts commandOptions) (buildResult, error) {
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

	artifacts := make([]output.Artifact, 0, 2)
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
	if err := output.ValidateWritePlan(opts.siteRoot, cfg, pages, artifacts); err != nil {
		return buildResult{}, err
	}

	if err := output.WriteSite(opts.siteRoot, cfg, pages); err != nil {
		return buildResult{}, err
	}

	if cfg.Pipeline.Fingerprint || cfg.Pipeline.Images.Enabled || cfg.Pipeline.SCSS.Enabled {
		manifest, pipeErr := pipeline.Process(opts.siteRoot, cfg)
		if pipeErr != nil {
			return buildResult{}, pipeErr
		}
		if manifest != nil {
			if err := pipeline.RewriteOutputFiles(opts.siteRoot, cfg, manifest); err != nil {
				return buildResult{}, err
			}
		}
	}

	feedOutputs, err := feeds.GenerateAll(cfg, state.Index)
	if err != nil {
		return buildResult{}, err
	}
	for _, feedOutput := range feedOutputs {
		if err := output.WriteFile(opts.siteRoot, cfg, feedOutput.Filename, feedOutput.Content); err != nil {
			return buildResult{}, err
		}
	}
	sitemapOutput, err := sitemap.Generate(cfg, state, pages)
	if err != nil {
		return buildResult{}, err
	}
	if sitemapOutput != nil {
		if err := output.WriteFile(opts.siteRoot, cfg, sitemapOutput.Filename, sitemapOutput.Content); err != nil {
			return buildResult{}, err
		}
	}
	if robotsOutput := robots.Generate(cfg); robotsOutput != nil {
		if err := output.WriteFile(opts.siteRoot, cfg, robotsOutput.Filename, robotsOutput.Content); err != nil {
			return buildResult{}, err
		}
	}
	if err := assets.Copy(opts.siteRoot, cfg); err != nil {
		return buildResult{}, err
	}

	return buildResult{cfg: cfg, path: path, state: state, pages: pages}, nil
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
	if opts.port != 0 {
		cfg.Server.Port = opts.port
	}
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return config.SiteConfig{}, "", errors.New("server.port must be between 1 and 65535")
	}

	return cfg, path, nil
}

func writeCommandError(stderr io.Writer, err error) int {
	_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
	return 1
}

func writeUsage(w io.Writer) {
	_, _ = io.WriteString(w, `Usage:
  nida serve [-s PATH] [--site PATH] [-c PATH] [--config PATH] [-d] [--drafts] [-p PORT] [--port PORT]
  nida build [-s PATH] [--site PATH] [-c PATH] [--config PATH] [-d] [--drafts]
  nida version

Commands:
  serve   Build, watch, and serve the local site
  build   Build the site into the configured output directory
  version Show nida build and version information
`)
}

func writeVersion(w io.Writer) {
	_, _ = fmt.Fprintln(w, buildinfo.Summary())
}
