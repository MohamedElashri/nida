package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMergeDataFollowsTheCascade(t *testing.T) {
	got := mergeData(
		map[string]any{"tags": []any{"posts"}, "layout": "a", "nav": map[string]any{"key": "x", "order": 1}},
		map[string]any{"tags": []any{"go"}, "layout": "b", "nav": map[string]any{"order": 2}},
	)
	if tags := got["tags"].([]any); len(tags) != 2 || tags[0] != "posts" || tags[1] != "go" {
		t.Errorf("tags = %v", tags)
	}
	if got["layout"] != "b" {
		t.Errorf("layout = %v", got["layout"])
	}
	if nav := got["nav"].(map[string]any); nav["key"] != "x" || nav["order"] != 2 {
		t.Errorf("nav = %v", nav)
	}
}

func TestReadTemplateFrontMatterVariants(t *testing.T) {
	for in, want := range map[string]string{
		"---json\n{\"title\": \"J\"}\n---\nbody\n":             "J",
		"---toml\ntitle = \"T\"\n---\nbody\n":                  "T",
		"---yaml\ntitle: Y\n---\nbody\n":                       "Y",
		"---\ntitle: plain\n---\nbody\n":                       "plain",
		"\xef\xbb\xbf---json\r\n{\"title\": \"B\"}\r\n---\r\n": "B",
	} {
		doc, _, err := readTemplate([]byte(in))
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if doc.String("title") != want {
			t.Errorf("%q: title = %q", in, doc.String("title"))
		}
		if _, err := ParseDocument(doc.Bytes()); err != nil {
			t.Errorf("%q: output does not reparse: %v", in, err)
		}
	}
	if _, variant, err := readTemplate([]byte("---js\n{ title: 'x' }\n---\n")); err == nil || variant != "js" {
		t.Errorf("JavaScript front matter must be refused, got variant %q err %v", variant, err)
	}
}

func TestPassthroughDestinations(t *testing.T) {
	r := &eleventyRun{cfg: &eleventyConfig{input: "src", passthrough: []passRule{
		{src: "src/img"},
		{src: "fonts"},
		{src: "public", dest: ".", set: true},
		{src: "src/robots.txt", dest: "robots.txt", set: true},
		{src: "src/**/*.{png,jpg}"},
	}}}
	for rel, want := range map[string]string{
		"src/img/a/b.svg":     "img/a/b.svg",
		"fonts/x.woff2":       "fonts/x.woff2",
		"public/css/site.css": "css/site.css",
		"src/robots.txt":      "robots.txt",
		"src/posts/pic.png":   "posts/pic.png",
	} {
		if got, ok := r.passthroughDest(rel); !ok || got != want {
			t.Errorf("passthroughDest(%q) = %q, %t; want %q", rel, got, ok, want)
		}
	}
	if _, ok := r.passthroughDest("src/posts/post.md"); ok {
		t.Error("a template is not a passthrough copy")
	}
}

func TestEleventyIgnores(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".eleventyignore":     "README.md\n./drafts/\n# comment\n!keep.md\nsrc/private/*.md\n",
		"src/.eleventyignore": "notes.md\n",
	})
	r := &eleventyRun{p: &Plan{Root: root}, cfg: &eleventyConfig{input: "src"}}
	r.ignore = r.readIgnores()
	for rel, want := range map[string]bool{
		"README.md":          true,
		"src/README.md":      true,
		"drafts/a.md":        true,
		"src/drafts/a.md":    false,
		"src/private/x.md":   true,
		"src/private/x.png":  false,
		"src/notes.md":       true,
		"src/deep/notes.md":  true,
		"notes.md":           false,
		"src/posts/hello.md": false,
	} {
		if got := r.ignored(rel); got != want {
			t.Errorf("ignored(%q) = %t, want %t", rel, got, want)
		}
	}
}

func TestEleventySpecialDateFallsBackToModTime(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"post.md": "x"})
	when := time.Date(2021, 4, 5, 6, 7, 8, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, "post.md"), when, when); err != nil {
		t.Fatal(err)
	}
	r := &eleventyRun{p: &Plan{Root: root}}
	for _, kind := range []string{"last modified", "created", "git last modified"} {
		got, how := r.specialDate("post.md", kind)
		if !got.Equal(when) || how == "" {
			t.Errorf("%s: got %v (%s), want %v", kind, got, how, when)
		}
	}
}

func TestEleventySiteWithoutConfigUsesDefaults(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src, map[string]string{
		"package.json":              `{"name": "tiny", "devDependencies": {"@11ty/eleventy": "^2.0.1"}}`,
		"index.md":                  "# Tiny\n\nHome page.\n",
		"posts/posts.json":          `{"tags": "post", "layout": "post.liquid"}`,
		"posts/hello.md":            "---\ntitle: Hello\ndate: 2020-02-02\n---\nHi {{ title }}.\n",
		"posts/hello.11tydata.json": `{"permalink": "/hello.html"}`,
		"_includes/post.liquid":     "{{ content }}",
		"_site/index.html":          "old",
	})
	dest := filepath.Join(t.TempDir(), "out")
	result, err := Run(Options{Source: src, Dest: dest, From: "eleventy", Check: checkSite})
	if err != nil {
		t.Fatal(err)
	}
	if result.Report.Check != "passed" {
		t.Fatalf("check: %s\n%s", result.Report.Check, result.Markdown)
	}
	files := readTree(t, dest)
	hello := files["content/posts/hello.md"]
	for _, want := range []string{"template: post", "collections:\n  - post", "path: /hello.html", "Hi Hello."} {
		if !contains(hello, want) {
			t.Errorf("hello.md lacks %q:\n%s", want, hello)
		}
	}
	if !contains(files["content/_index.md"], `title = "Tiny"`) {
		t.Errorf("home page:\n%s", files["content/_index.md"])
	}
	if !contains(files["config.toml"], `title = "tiny"`) {
		t.Errorf("title should fall back to the package name:\n%s", files["config.toml"])
	}
	for name := range files {
		if contains(name, "_site") {
			t.Errorf("build output was migrated: %s", name)
		}
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
