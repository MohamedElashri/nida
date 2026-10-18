package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/buildinfo"
	"github.com/MohamedElashri/nida/internal/config"
)

func TestBuildLoadsConfigFromSiteRoot(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{
		"build",
		"--site", filepath.Join("..", "..", "example-site"),
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "config=") {
		t.Fatalf("expected config path in output, got %q", stdout.String())
	}
}

func TestBuildLoadsArabicExampleSite(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{
		"build",
		"--site", filepath.Join("..", "..", "example-site-ar"),
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "rendered=") {
		t.Fatalf("expected build summary in output, got %q", stdout.String())
	}
}

func TestInitCreatesBuildableExampleSite(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	target := filepath.Join(t.TempDir(), "my-site")

	code := run(stdout, stderr, []string{"init", target})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nida init: created") {
		t.Fatalf("expected init summary in output, got %q", stdout.String())
	}

	for _, path := range []string{
		"config.toml",
		"content/_index.md",
		"content/pages/about.md",
		"content/pages/search.md",
		"content/posts/_index.md",
		"content/posts/welcome.md",
		"content/posts/markdown-tour.md",
		"content/posts/content-model/index.md",
		"content/posts/content-model/bundle-note.txt",
		"templates/base.html",
		"templates/index.html",
		"templates/post.html",
		"templates/page.html",
		"templates/search.html",
		"templates/list.html",
		"templates/taxonomy.html",
		"static/site.css",
	} {
		if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(path))); err != nil {
			t.Fatalf("expected scaffolded file %s: %v", path, err)
		}
	}

	buildOut := &bytes.Buffer{}
	buildErr := &bytes.Buffer{}
	code = run(buildOut, buildErr, []string{"build", "--site", target})
	if code != 0 {
		t.Fatalf("expected scaffolded site to build, got %d stderr=%s", code, buildErr.String())
	}
	if !strings.Contains(buildOut.String(), "rendered=") {
		t.Fatalf("expected build summary in output, got %q", buildOut.String())
	}
	if _, err := os.Stat(filepath.Join(target, "public", "index.html")); err != nil {
		t.Fatalf("expected generated homepage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "posts", "markdown-tour", "index.html")); err != nil {
		t.Fatalf("expected generated markdown tour: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "tags", "workflow", "index.html")); err != nil {
		t.Fatalf("expected generated taxonomy page: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "search", "index.html")); err != nil {
		t.Fatalf("expected generated search page: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "search_index.en.js")); err != nil {
		t.Fatalf("expected generated search index: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "public", "posts", "content-model", "bundle-note.txt")); err != nil {
		t.Fatalf("expected copied bundle resource: %v", err)
	}
}

func TestInitRefusesNonEmptyDirectory(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "keep.txt"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	code := run(stdout, stderr, []string{"init", target})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "is not empty") {
		t.Fatalf("expected non-empty error, got %q", stderr.String())
	}
}

func TestLoadCommandConfigAppliesServeOverrides(t *testing.T) {
	cfg, _, err := loadCommandConfig(commandOptions{
		siteRoot: filepath.Join("..", "..", "example-site"),
		drafts:   true,
		port:     1313,
	})
	if err != nil {
		t.Fatalf("loadCommandConfig returned error: %v", err)
	}
	if !cfg.Drafts {
		t.Fatal("expected drafts override")
	}
	if cfg.Server.Port != 1313 {
		t.Fatalf("expected port override 1313, got %d", cfg.Server.Port)
	}
}

func TestLoadCommandConfigUsesUpdatedDefaultServePort(t *testing.T) {
	cfg, _, err := loadCommandConfig(commandOptions{
		siteRoot: filepath.Join("..", "..", "example-site"),
	})
	if err != nil {
		t.Fatalf("loadCommandConfig returned error: %v", err)
	}
	if cfg.Server.Port != 1702 {
		t.Fatalf("expected default port 1702, got %d", cfg.Server.Port)
	}
}

func TestBuildReportsConfigErrors(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{
		"build",
		"--site", t.TempDir(),
	})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "error: load config") {
		t.Fatalf("expected config error, got %q", stderr.String())
	}
}

func TestRebuildModeTreatsContentResourcesAsFull(t *testing.T) {
	cfg := testRebuildConfig()

	got := rebuildMode(cfg, []string{"content/posts/bundle/image.png"})
	if got != "full" {
		t.Fatalf("expected content resource change to trigger full rebuild, got %q", got)
	}
}

func TestVersionReportsReleaseVersionOnly(t *testing.T) {
	originalVersion := buildinfo.Version
	originalCommit := buildinfo.Commit
	originalDate := buildinfo.Date
	originalBuiltBy := buildinfo.BuiltBy
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
		buildinfo.Commit = originalCommit
		buildinfo.Date = originalDate
		buildinfo.BuiltBy = originalBuiltBy
	})

	buildinfo.Version = "0.2.0"
	buildinfo.Commit = "abc1234"
	buildinfo.Date = "2026-04-23T10:00:00Z"
	buildinfo.BuiltBy = "goreleaser"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{"version"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	output := stdout.String()
	if output != "nida version 0.2.0\n" {
		t.Fatalf("expected release version only, got %q", output)
	}
}

func testRebuildConfig() config.SiteConfig {
	cfg := config.DefaultSiteConfig()
	cfg.ContentDir = "content"
	cfg.StaticDir = "static"
	cfg.TemplateDir = "templates"
	return cfg
}

func TestVersionReportsDevBuildMetadata(t *testing.T) {
	originalVersion := buildinfo.Version
	originalCommit := buildinfo.Commit
	originalDate := buildinfo.Date
	originalBuiltBy := buildinfo.BuiltBy
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
		buildinfo.Commit = originalCommit
		buildinfo.Date = originalDate
		buildinfo.BuiltBy = originalBuiltBy
	})

	buildinfo.Version = "dev"
	buildinfo.Commit = "abc1234"
	buildinfo.Date = "2026-04-23T10:00:00Z"
	buildinfo.BuiltBy = "local"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{"version"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "nida version dev") {
		t.Fatalf("expected version in output, got %q", output)
	}
	if !strings.Contains(output, "commit=abc1234") {
		t.Fatalf("expected commit in output, got %q", output)
	}
	if !strings.Contains(output, "builtBy=local") {
		t.Fatalf("expected builtBy in output, got %q", output)
	}
}

func TestHelpIncludesVersionCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(stdout, stderr, []string{"help"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nida version") {
		t.Fatalf("expected version command in help, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "nida init [PATH]") {
		t.Fatalf("expected init command in help, got %q", stdout.String())
	}
}

func TestColorsDoNotDecorateNonTerminalWriters(t *testing.T) {
	colors := colorsFor(&bytes.Buffer{})

	got := colors.command("nida build:")
	if got != "nida build:" {
		t.Fatalf("expected plain command for non-terminal writer, got %q", got)
	}
}

func TestColorsRespectNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	colors := terminalColors{enabled: writerSupportsColor(os.Stdout)}
	got := colors.error("error:")
	if got != "error:" {
		t.Fatalf("expected plain error with NO_COLOR, got %q", got)
	}
}

func TestCheckReportsBrokenLinksWithoutWritingOutput(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := run(stdout, stderr, []string{"init", dir}); code != 0 {
		t.Fatalf("init failed: %s", stderr.String())
	}

	stdout.Reset()
	if code := run(stdout, stderr, []string{"check", "--site", dir}); code != 0 {
		t.Fatalf("expected a clean check, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nida check: ok") {
		t.Fatalf("unexpected output %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "public")); !os.IsNotExist(err) {
		t.Fatalf("check must not write output, stat err=%v", err)
	}

	about := filepath.Join(dir, "content", "pages", "about.md")
	data, err := os.ReadFile(about)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(about, append(data, []byte("\n[gone](@/posts/missing.md)\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if code := run(stdout, stderr, []string{"check", "--site", dir}); code == 0 {
		t.Fatal("expected check to fail on a broken internal link")
	}
	if !strings.Contains(stderr.String(), "posts/missing.md") {
		t.Fatalf("expected the broken link in the error, got %q", stderr.String())
	}
}

func TestLoadCommandConfigAppliesBuildOverrides(t *testing.T) {
	cfg, _, err := loadCommandConfig(commandOptions{
		siteRoot:  filepath.Join("..", "..", "example-site"),
		baseURL:   "https://preview.example.org/pr-1/",
		outputDir: "./preview/",
	})
	if err != nil {
		t.Fatalf("loadCommandConfig returned error: %v", err)
	}
	if cfg.BaseURL != "https://preview.example.org/pr-1/" || cfg.OutputDir != "preview" {
		t.Fatalf("overrides not applied: base_url=%q output_dir=%q", cfg.BaseURL, cfg.OutputDir)
	}
	if _, _, err := loadCommandConfig(commandOptions{siteRoot: filepath.Join("..", "..", "example-site"), baseURL: "not a url"}); err == nil {
		t.Fatal("expected invalid base URL override to fail")
	}
}

func TestBuildWritesBundledLibrariesOnlyWhenUsed(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := run(stdout, stderr, []string{"init", dir}); code != 0 {
		t.Fatalf("init failed: %s", stderr.String())
	}
	if code := run(stdout, stderr, []string{"build", "--site", dir}); code != 0 {
		t.Fatalf("build failed: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "public", "vendor")); !os.IsNotExist(err) {
		t.Fatalf("a site without math or diagrams must not get bundled files, stat err=%v", err)
	}

	post := filepath.Join(dir, "content", "posts", "diagram.md")
	if err := os.WriteFile(post, []byte("+++\ntitle = \"Diagram\"\ndate = 2025-01-01\n+++\n\n```mermaid\ngraph LR\n  A --> B\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run(stdout, stderr, []string{"build", "--site", dir}); code != 0 {
		t.Fatalf("build failed: %s", stderr.String())
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "public", "vendor", "mermaid-*", "mermaid.min.js"))
	if len(matches) != 1 {
		t.Fatalf("expected mermaid to be written, got %v", matches)
	}
	if katex, _ := filepath.Glob(filepath.Join(dir, "public", "vendor", "katex-*")); len(katex) != 0 {
		t.Fatalf("KaTeX is unused and must not be written, got %v", katex)
	}
	html, err := os.ReadFile(filepath.Join(dir, "public", "posts", "diagram", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "/vendor/mermaid-") {
		t.Fatal("expected the diagram page to load the bundled script")
	}
}

func TestSiteCommandsAcceptPositionalSitePath(t *testing.T) {
	parsers := map[string]func([]string) (commandOptions, error){
		"build": parseBuildFlags,
		"serve": parseServeFlags,
		"check": parseCheckFlags,
	}
	cases := []struct {
		name   string
		args   []string
		site   string
		drafts bool
	}{
		{name: "default", args: nil, site: "."},
		{name: "positional", args: []string{"my-site"}, site: "my-site"},
		{name: "flags before path", args: []string{"--drafts", "my-site"}, site: "my-site", drafts: true},
		{name: "flags after path", args: []string{"my-site", "-d"}, site: "my-site", drafts: true},
		{name: "long site flag", args: []string{"--site", "my-site"}, site: "my-site"},
		{name: "short site flag", args: []string{"-s", "my-site", "--drafts"}, site: "my-site", drafts: true},
		{name: "after double dash", args: []string{"-d", "--", "-my-site"}, site: "-my-site", drafts: true},
	}
	for command, parse := range parsers {
		for _, tc := range cases {
			t.Run(command+"/"+tc.name, func(t *testing.T) {
				opts, err := parse(tc.args)
				if err != nil {
					t.Fatalf("parse %v: %v", tc.args, err)
				}
				if opts.siteRoot != tc.site {
					t.Fatalf("expected site root %q, got %q", tc.site, opts.siteRoot)
				}
				if opts.drafts != tc.drafts {
					t.Fatalf("expected drafts=%t, got %t", tc.drafts, opts.drafts)
				}
			})
		}
	}
}

func TestSiteCommandsRejectAmbiguousSitePaths(t *testing.T) {
	parsers := map[string]func([]string) (commandOptions, error){
		"build": parseBuildFlags,
		"serve": parseServeFlags,
		"check": parseCheckFlags,
	}
	for command, parse := range parsers {
		for _, args := range [][]string{
			{"one", "two"},
			{"one", "--site", "two"},
			{"-s", "two", "one"},
		} {
			if _, err := parse(args); err == nil {
				t.Fatalf("%s %v: expected an error, got none", command, args)
			}
		}
	}
}

func TestServeParsesPortAfterPositionalSitePath(t *testing.T) {
	opts, err := parseServeFlags([]string{"my-site", "--port", "4000"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if opts.siteRoot != "my-site" || opts.port != 4000 {
		t.Fatalf("expected site=my-site port=4000, got site=%q port=%d", opts.siteRoot, opts.port)
	}
}

func TestNewAcceptsFlagsAfterContentPath(t *testing.T) {
	opts, target, err := parseNewFlags([]string{"posts/hello.md", "--site", "my-site"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if target != "posts/hello.md" || opts.siteRoot != "my-site" {
		t.Fatalf("expected target=posts/hello.md site=my-site, got target=%q site=%q", target, opts.siteRoot)
	}
}

func TestSiteCommandsUsePositionalSitePath(t *testing.T) {
	target := filepath.Join(t.TempDir(), "my-site")
	if code := run(&bytes.Buffer{}, &bytes.Buffer{}, []string{"init", target}); code != 0 {
		t.Fatalf("init failed with code %d", code)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := run(stdout, stderr, []string{"check", target}); code != 0 {
		t.Fatalf("check %s: expected exit code 0, got %d stderr=%s", target, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), filepath.Join(target, "config.toml")) {
		t.Fatalf("expected check to load the site's config, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run(stdout, stderr, []string{"build", target, "--output", "out"}); code != 0 {
		t.Fatalf("build %s: expected exit code 0, got %d stderr=%s", target, code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(target, "out", "index.html")); err != nil {
		t.Fatalf("expected output written inside the site root: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := run(stdout, stderr, []string{"new", "posts/hello.md", "-s", target}); code != 0 {
		t.Fatalf("new: expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(target, "content", "posts", "hello.md")); err != nil {
		t.Fatalf("expected new content inside the site root: %v", err)
	}
}

func TestServeLoadsPositionalSitePath(t *testing.T) {
	// A missing site fails before the server starts, so the error shows
	// which root serve tried to load without blocking the test.
	missing := filepath.Join(t.TempDir(), "missing-site")
	stderr := &bytes.Buffer{}
	if code := run(&bytes.Buffer{}, stderr, []string{"serve", missing}); code == 0 {
		t.Fatalf("expected serve of a missing site to fail")
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("expected serve to load %s, got stderr=%q", missing, stderr.String())
	}
}

func TestBuildWritesFileRoutesForPathsPermalinksAndAliases(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"config.toml": `base_url = "https://example.com"
title = "Routes"

[permalinks]
notes = "/{year}/{month}/{slug}.html"
`,
		"content/_index.md":             "+++\ntitle = \"Home\"\n+++\n",
		"content/posts/_index.md":       "+++\ntitle = \"Posts\"\n+++\n",
		"content/posts/exact.md":        "+++\ntitle = \"Exact\"\npath = \"/legacy/exact.html\"\n+++\nbody\n",
		"content/posts/folder.md":       "+++\ntitle = \"Folder\"\npath = \"elsewhere/folder\"\naliases = [\"/old/folder.html\", \"/old/folder/\"]\n+++\nbody\n",
		"content/posts/bundle/index.md": "+++\ntitle = \"Bundle\"\npath = \"/2024/01/02/bundle.html\"\n+++\n![pic](pic.txt)\n",
		"content/posts/bundle/pic.txt":  "resource",
		"content/notes/_index.md":       "+++\ntitle = \"Notes\"\n+++\n",
		"content/notes/dated.md":        "+++\ntitle = \"Dated\"\ndate = 2023-07-09\n+++\nbody\n",
		"content/posts/dotted.md":       "+++\ntitle = \"Dotted\"\nslug = \"node.js\"\naliases = [\"/old/v2.0\"]\n+++\nbody\n",
		"templates/base.html":           `{{ define "base" }}{{ end }}`,
		"templates/index.html":          `{{ define "index" }}home{{ end }}`,
		"templates/list.html":           `{{ define "list" }}list{{ end }}`,
		"templates/page.html":           `{{ define "page" }}page {{ .Page.URL }}{{ end }}`,
		"templates/post.html":           `{{ define "post" }}post {{ .Page.URL }}{{ end }}`,
	}
	for name, body := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	stderr := &bytes.Buffer{}
	if code := run(&bytes.Buffer{}, stderr, []string{"build", root}); code != 0 {
		t.Fatalf("build failed: %s", stderr.String())
	}

	public := filepath.Join(root, "public")
	for path, want := range map[string]string{
		"legacy/exact.html":           "post /legacy/exact.html",
		"elsewhere/folder/index.html": "post /elsewhere/folder/",
		"2024/01/02/bundle.html":      "post /2024/01/02/bundle.html",
		"2024/01/02/pic.txt":          "resource",
		"2023/07/dated.html":          "post /2023/07/dated.html",
		"old/folder.html":             "https://example.com/elsewhere/folder/",
		"old/folder/index.html":       "https://example.com/elsewhere/folder/",
		"posts/node.js/index.html":    "post /posts/node.js/",
		"old/v2.0/index.html":         "https://example.com/posts/node.js/",
	} {
		data, err := os.ReadFile(filepath.Join(public, filepath.FromSlash(path)))
		if err != nil {
			t.Errorf("expected %s: %v", path, err)
			continue
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s: expected %q in %q", path, want, data)
		}
	}
	sitemap, err := os.ReadFile(filepath.Join(public, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sitemap), "https://example.com/legacy/exact.html<") {
		t.Errorf("expected file route in sitemap, got %s", sitemap)
	}
}

func TestMigrateWritesBuildableSite(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "migrated")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	src := filepath.Join("..", "migrate", "testdata", "zola", "site")
	if code := run(stdout, stderr, []string{"migrate", src, dest}); code != 0 {
		t.Fatalf("migrate failed: %s", stderr.String())
	}
	for _, want := range []string{"nida migrate: complete", "from=zola", "check=passed", "Report: "} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("expected %q in %q", want, stdout.String())
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "MIGRATION.md")); err != nil {
		t.Fatalf("expected report: %v", err)
	}
	stderr.Reset()
	if code := run(&bytes.Buffer{}, stderr, []string{"build", dest}); code != 0 {
		t.Fatalf("migrated site does not build: %s", stderr.String())
	}
}

func TestMigrateDryRunPrintsReport(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "migrated")
	stdout := &bytes.Buffer{}
	src := filepath.Join("..", "migrate", "testdata", "zola", "site")
	if code := run(stdout, &bytes.Buffer{}, []string{"migrate", "-n", src, dest}); code != 0 {
		t.Fatal("dry run failed")
	}
	if !strings.Contains(stdout.String(), "# Migration report") || !strings.Contains(stdout.String(), "dry run") {
		t.Errorf("expected the report on stdout, got %q", stdout.String())
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("dry run must not create the destination")
	}
}

func TestMigrateRejectsBadArguments(t *testing.T) {
	for _, args := range [][]string{
		{"migrate"},
		{"migrate", "only-one"},
		{"migrate", "a", "b", "c"},
		{"migrate", "--urls", "sideways", filepath.Join("..", "migrate", "testdata", "zola", "site"), filepath.Join(t.TempDir(), "y")},
		{"migrate", "--from", "wordpress", ".", filepath.Join(t.TempDir(), "x")},
	} {
		stderr := &bytes.Buffer{}
		if code := run(&bytes.Buffer{}, stderr, args); code == 0 {
			t.Errorf("%v: expected failure", args)
		}
	}
}
