package migrate

import (
	"bytes"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/render"
	"github.com/MohamedElashri/nida/internal/site"
)

var update = flag.Bool("update", false, "rewrite the golden migration output in testdata")

// checkSite builds a migrated site in memory, like the CLI's check hook.
func checkSite(root string) error {
	cfg, _, err := config.Load(config.Options{SiteRoot: root})
	if err != nil {
		return err
	}
	cfg.Diagnostics.Enabled = true
	cfg.Drafts, cfg.Future = true, true
	state, err := site.Load(root, cfg)
	if err != nil {
		return err
	}
	_, err = render.RenderSite(root, cfg, state)
	return err
}

// writeTree creates files under root from a path-to-content map.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// readTree returns every file under root, keyed by slash path.
func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		data, err := os.ReadFile(p)
		files[filepath.ToSlash(rel)] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// runGolden migrates testdata/<from>/site and compares the result with
// testdata/<from>/want. Run `go test ./internal/migrate -update` to
// accept new output after reviewing the diff.
func runGolden(t *testing.T, from string) *Result {
	t.Helper()
	src := filepath.Join("testdata", from, "site")
	dest := filepath.Join(t.TempDir(), "out")
	result, err := Run(Options{Source: src, Dest: dest, From: from, Check: checkSite})
	if err != nil {
		t.Fatalf("migrate %s: %v", from, err)
	}
	if !strings.HasPrefix(result.Report.Check, "passed") {
		t.Errorf("migrated %s site does not build: %s", from, result.Report.Check)
	}
	got := readTree(t, dest)
	wantDir := filepath.Join("testdata", from, "want")
	if *update {
		if err := os.RemoveAll(wantDir); err != nil {
			t.Fatal(err)
		}
		for name, body := range got {
			writeTree(t, wantDir, map[string]string{name: body})
		}
		return result
	}
	want := readTree(t, wantDir)
	for name, body := range want {
		if got[name] != body {
			if _, ok := got[name]; !ok {
				t.Errorf("missing %s", name)
			} else {
				t.Errorf("%s differs:\n--- got\n%s\n--- want\n%s", name, got[name], body)
			}
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("unexpected file %s", name)
		}
	}
	return result
}

func TestZolaGolden(t *testing.T) {
	result := runGolden(t, "zola")
	if result.Report.URLsKept == 0 {
		t.Error("expected URLs to be checked")
	}
}

func TestRunRefusesUnsafeDestinations(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src, map[string]string{"config.toml": "base_url = \"https://x\"\n", "content/_index.md": "+++\n+++\n"})

	nonEmpty := t.TempDir()
	writeTree(t, nonEmpty, map[string]string{"keep.txt": "x"})

	for name, dest := range map[string]string{
		"same as source":       src,
		"non-empty":            nonEmpty,
		"parent of the source": filepath.Dir(src),
	} {
		if _, err := Run(Options{Source: src, Dest: dest, From: "zola"}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if data, _ := os.ReadFile(filepath.Join(nonEmpty, "keep.txt")); string(data) != "x" {
		t.Error("existing file was touched")
	}
}

func TestRunNeverWritesToSource(t *testing.T) {
	src := filepath.Join("testdata", "zola", "site")
	before := readTree(t, src)
	if _, err := Run(Options{Source: src, Dest: filepath.Join(t.TempDir(), "out"), From: "zola"}); err != nil {
		t.Fatal(err)
	}
	after := readTree(t, src)
	if len(before) != len(after) {
		t.Fatalf("source file count changed: %d -> %d", len(before), len(after))
	}
	for name, body := range before {
		if after[name] != body {
			t.Errorf("source file %s changed", name)
		}
	}
}

func TestRunSkipsDestinationInsideSource(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src, map[string]string{
		"config.toml":       "base_url = \"https://x\"\n",
		"content/_index.md": "+++\ntitle = \"Home\"\n+++\n",
		"static/a.txt":      "a",
		"content/post.md":   "+++\ntitle = \"P\"\n+++\n",
	})
	dest := filepath.Join(src, "static", "migrated")
	if _, err := Run(Options{Source: src, Dest: dest, From: "zola"}); err != nil {
		t.Fatal(err)
	}
	for name := range readTree(t, dest) {
		if strings.Contains(name, "migrated/") {
			t.Errorf("destination was copied into itself: %s", name)
		}
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out")
	result, err := Run(Options{Source: filepath.Join("testdata", "zola", "site"), Dest: dest, From: "zola", DryRun: true, Check: checkSite})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("dry run created %s", dest)
	}
	if result.ReportPath != "" || !strings.Contains(result.Markdown, "# Migration report") {
		t.Errorf("dry run result: path=%q", result.ReportPath)
	}
	if result.Report.Check != "passed" {
		t.Errorf("dry run should still check the site: %s", result.Report.Check)
	}
}

func TestDetect(t *testing.T) {
	zola := t.TempDir()
	writeTree(t, zola, map[string]string{"config.toml": "base_url = \"https://x\"\ncompile_sass = true\n", "content/_index.md": "+++\n+++\n"})
	if s, err := pickSource(zola, "auto"); err != nil || s.Name() != "zola" {
		t.Errorf("detect zola: %v %v", s, err)
	}
	if _, err := pickSource(t.TempDir(), ""); err == nil {
		t.Error("an empty directory must not be detected as any generator")
	}
	if _, err := pickSource(zola, "wordpress"); err == nil {
		t.Error("unknown --from must fail")
	}
}

// urlSite is a Zola site whose page URLs the test controls through the
// old-URL predictions.
func urlPlan(t *testing.T, mode string, oldURLs map[string]string) (map[string]string, *Report) {
	t.Helper()
	root := t.TempDir()
	report := &Report{}
	p := newPlan(root, report)
	p.Config.Set("base_url", "https://x")
	p.Config.Set("title", "T")
	p.Page("content/_index.md", NewDocument(TOML, ""), "", "")
	for _, name := range slices.Sorted(mapKeys(oldURLs)) {
		doc := NewDocument(TOML, "body\n")
		_ = doc.Set("title", name)
		p.Page("content/posts/"+name+".md", doc, name+".md", oldURLs[name])
	}
	if err := finishPlan(p); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := p.apply(dest); err != nil {
		t.Fatal(err)
	}
	if err := fixURLs(p, dest, mode); err != nil {
		t.Fatal(err)
	}
	return readTree(t, dest), report
}

func mapKeys(m map[string]string) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func TestFixURLsKeepsOldURLs(t *testing.T) {
	files, report := urlPlan(t, KeepURLs, map[string]string{
		"same":    "/posts/same/",
		"moved":   "/2024/05/01/moved.html",
		"encoded": "https://old.example/blog/caf%C3%A9/?utm=x#top",
		"legacy":  "/legacy/page/index.html",
	})
	if report.URLsKept != 1 || report.URLsMoved != 3 {
		t.Errorf("kept=%d moved=%d, notes=%v", report.URLsKept, report.URLsMoved, report.Notes)
	}
	for name, want := range map[string]string{
		"content/posts/moved.md":   `path = "/2024/05/01/moved.html"`,
		"content/posts/encoded.md": `path = "/blog/café/"`,
		"content/posts/legacy.md":  `path = "/legacy/page/"`,
	} {
		if !strings.Contains(files[name], want) {
			t.Errorf("%s: want %s in:\n%s", name, want, files[name])
		}
	}
	if strings.Contains(files["content/posts/same.md"], "path") {
		t.Error("a page whose URL matches must not get a path")
	}
}

func TestFixURLsRedirectMode(t *testing.T) {
	files, _ := urlPlan(t, RedirectURLs, map[string]string{"moved": "/old/moved/"})
	if !strings.Contains(files["content/posts/moved.md"], `aliases = ["/old/moved/"]`) {
		t.Errorf("want an alias:\n%s", files["content/posts/moved.md"])
	}
}

func TestFixURLsReportsCollisions(t *testing.T) {
	files, report := urlPlan(t, KeepURLs, map[string]string{
		"a": "/posts/b/", // b's new URL
		"b": "/posts/b/",
	})
	if strings.Contains(files["content/posts/a.md"], "path") {
		t.Error("a page must not take another page's URL")
	}
	if report.Count(Manual) != 1 {
		t.Errorf("expected one collision note, got %v", report.Notes)
	}
}

func TestTemplateFallback(t *testing.T) {
	root := t.TempDir()
	report := &Report{}
	p := newPlan(root, report)
	p.Config.Set("base_url", "https://x")
	p.Page("content/_index.md", NewDocument(TOML, ""), "", "")
	p.Template("templates/base.html", []byte(`{{ define "base" }}{{ .NoSuchField }}{{ end }}`))
	p.Template("templates/index.html", []byte(`{{ define "index" }}{{ template "base" . }}{{ end }}`))
	if err := finishPlan(p); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := p.apply(dest); err != nil {
		t.Fatal(err)
	}
	if err := chooseTemplates(p, dest, zolaSource{}, checkSite); err != nil {
		t.Fatal(err)
	}
	files := readTree(t, dest)
	if !strings.Contains(files["templates/base.html"], "nida-migrate.css") {
		t.Error("expected the starter base template after the translated set failed")
	}
	if _, ok := files[translatedDir+"/base.html"]; !ok {
		t.Error("translated templates must be kept for reference")
	}
	if report.Check != "passed" || report.Count(Review) != 1 {
		t.Errorf("check=%q notes=%v", report.Check, report.Notes)
	}
}

func TestReportGroupsNotes(t *testing.T) {
	r := &Report{From: "Zola"}
	r.Add(Changed, "b.md", 0, "renamed lastmod")
	r.Add(Changed, "a.md", 3, "renamed lastmod")
	r.Add(Changed, "a.md", 3, "renamed lastmod") // duplicate
	r.Add(Manual, "", 0, "set base_url")
	md := r.Markdown()
	if !strings.Contains(md, "- renamed lastmod (2 places)\n  - `a.md:3`\n  - `b.md`\n") {
		t.Errorf("notes not grouped:\n%s", md)
	}
	if strings.Index(md, "## Needs manual work") > strings.Index(md, "## Changed automatically") {
		t.Error("manual work must come first")
	}
}

func TestConfigBytesLoadInNida(t *testing.T) {
	c := &Config{}
	c.Set("base_url", "https://example.com/blog/")
	c.Set("title", `Quote "and" backslash \`)
	c.Taxonomy("tags").Set("render", true)
	c.Table("extra").Set("menu", []any{map[string]any{"name": "Blog", "url": "/blog/"}})
	c.Table("markdown").Set("unsafe_html", true)
	data, err := c.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeTree(t, root, map[string]string{"config.toml": string(data)})
	cfg, _, err := config.Load(config.Options{SiteRoot: root})
	if err != nil {
		t.Fatalf("load:\n%s\n%v", data, err)
	}
	if cfg.Title != `Quote "and" backslash \` || !cfg.Markdown.UnsafeHTML || len(cfg.Taxonomies) != 1 || len(cfg.Warnings) != 0 {
		t.Errorf("config round trip: %+v\n%s", cfg, data)
	}
	if !bytes.Contains(data, []byte(`menu = [{ name = "Blog", url = "/blog/" }]`)) {
		t.Errorf("nested values must stay inline:\n%s", data)
	}
}

func TestDetectEveryFixture(t *testing.T) {
	for _, src := range Sources {
		root := filepath.Join("testdata", src.Name(), "site")
		if _, err := os.Stat(root); err != nil {
			continue
		}
		got, err := pickSource(root, "auto")
		if err != nil {
			t.Errorf("%s fixture: %v", src.Name(), err)
			continue
		}
		if got.Name() != src.Name() {
			t.Errorf("%s fixture detected as %s", src.Name(), got.Name())
		}
		for _, other := range Sources {
			if other.Name() != src.Name() && other.Detect(root) >= src.Detect(root) {
				t.Errorf("%s fixture: %s scores %d, not below %d", src.Name(), other.Name(), other.Detect(root), src.Detect(root))
			}
		}
	}
}

func TestDetectRefusesNidaSites(t *testing.T) {
	for _, site := range []string{"example-site", "example-site-ar", "docs"} {
		root := filepath.Join("..", "..", site)
		if _, err := pickSource(root, "auto"); err == nil || !strings.Contains(err.Error(), "already a Nida site") {
			t.Errorf("%s: expected it to be recognized as a Nida site, got %v", site, err)
		}
		if s, err := pickSource(root, "zola"); err != nil || s.Name() != "zola" {
			t.Errorf("%s: an explicit --from must still work: %v", site, err)
		}
	}
}

func TestPlanRefusesPathsOutsideBothSites(t *testing.T) {
	outside := t.TempDir()
	writeTree(t, outside, map[string]string{"secret.txt": "secret"})
	root := filepath.Join(t.TempDir(), "src")
	writeTree(t, root, map[string]string{"ok.txt": "ok"})
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}

	report := &Report{}
	p := newPlan(root, report)
	p.Write("content/../../escape.txt", []byte("x"), "a.md")
	p.Write("../escape.txt", []byte("x"), "b.md")
	p.Copy("static/secret.txt", "../"+filepath.Base(outside)+"/secret.txt")
	p.Copy("static/link.txt", "link.txt")
	p.Copy("static/ok.txt", "ok.txt")
	if _, err := p.ReadFile("../x"); err == nil {
		t.Error("ReadFile must refuse paths outside the source")
	}
	if _, err := p.ReadFile("link.txt"); err == nil {
		t.Error("ReadFile must refuse symlinks")
	}
	if report.Count(Manual) != 4 {
		t.Errorf("expected 4 refusals, got %v", report.Notes)
	}

	dest := filepath.Join(t.TempDir(), "dest")
	if err := p.apply(dest); err != nil {
		t.Fatal(err)
	}
	files := readTree(t, dest)
	if len(files) != 1 || files["static/ok.txt"] != "ok" {
		t.Errorf("only the safe file may be written, got %v", files)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "escape.txt")); err == nil {
		t.Error("a file escaped the destination")
	}
}

func TestFixURLsRespectsGeneratedFiles(t *testing.T) {
	files, report := urlPlan(t, KeepURLs, map[string]string{"map": "/sitemap.xml", "feed": "/feed.xml"})
	for _, name := range []string{"content/posts/map.md", "content/posts/feed.md"} {
		if strings.Contains(files[name], "path") {
			t.Errorf("%s must not take a generated file's URL:\n%s", name, files[name])
		}
	}
	if report.Count(Manual) != 2 {
		t.Errorf("expected two clash notes, got %v", report.Notes)
	}
}

func TestFixURLsLetsPagesSwapURLs(t *testing.T) {
	files, report := urlPlan(t, KeepURLs, map[string]string{"a": "/posts/b/", "b": "/posts/a/"})
	if !strings.Contains(files["content/posts/a.md"], `path = "/posts/b/"`) || !strings.Contains(files["content/posts/b.md"], `path = "/posts/a/"`) {
		t.Errorf("pages should trade URLs:\n%s\n%s\n%v", files["content/posts/a.md"], files["content/posts/b.md"], report.Notes)
	}
}

func TestBundleResourceClashIsReported(t *testing.T) {
	root := t.TempDir()
	report := &Report{}
	p := newPlan(root, report)
	p.Config.Set("base_url", "https://x")
	p.Config.Set("title", "T")
	p.Page("content/_index.md", NewDocument(TOML, ""), "", "")
	for _, name := range []string{"a", "b"} {
		doc := NewDocument(TOML, "![c](cover.png)\n")
		_ = doc.Set("title", name)
		p.Page("content/posts/"+name+"/index.md", doc, name+"/index.md", "/posts/"+name+".html")
		p.Write("content/posts/"+name+"/cover.png", []byte(name), name+"/cover.png")
	}
	if err := finishPlan(p); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := p.apply(dest); err != nil {
		t.Fatal(err)
	}
	if err := fixURLs(p, dest, KeepURLs); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, note := range report.Notes {
		if note.Severity == Manual && strings.Contains(note.Message, "cover.png") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a resource clash note, got %v", report.Notes)
	}
}
