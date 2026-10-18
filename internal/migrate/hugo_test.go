package migrate

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestHugoGolden(t *testing.T) {
	result := runGolden(t, "hugo")
	if result.Report.URLsKept+result.Report.URLsMoved == 0 {
		t.Error("expected URLs to be checked")
	}
}

func testHugoSite() *hugoSite {
	return &hugoSite{p: &Plan{Report: &Report{}, Config: &Config{}}, pathToLower: true}
}

func hugoDoc(t *testing.T, fm string) *Document {
	t.Helper()
	doc, err := ParseDocument([]byte("---\n" + fm + "\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestHugoTranslatePermalink(t *testing.T) {
	h := testHugoSite()
	for pattern, want := range map[string]string{
		"/:year/:month/:slug/":    "/{year}/{month}/{slug}/",
		":section/:title/":        "/posts/{slug}/",
		"/:sections/:filename/":   "/{sections}/{slug}/",
		"/blog/:contentbasename/": "/blog/{slug}/",
		"/:year/:yearday/:slug/":  "",
		"/:monthname/:slug/":      "",
		"/:slug/:title/":          "", // two slug-like tokens
	} {
		got, ok := h.translatePermalink(pattern, "posts")
		if want == "" {
			if ok {
				t.Errorf("%s: expected no translation, got %s", pattern, got)
			}
			continue
		}
		if !ok || got != want {
			t.Errorf("%s: got %q ok=%t, want %q", pattern, got, ok, want)
		}
	}
	h.uglyURLs = true
	if got, _ := h.translatePermalink("/:year/:slug/", "posts"); got != "/{year}/{slug}.html" {
		t.Errorf("ugly: got %q", got)
	}
}

func TestHugoExpandPermalink(t *testing.T) {
	h := testHugoSite()
	f := hugoContentFile{rel: "posts/2024/My Post.md", dest: "content/posts/2024/My Post.md"}
	doc := hugoDoc(t, "title: Hello World!\ndate: 2024-02-03T10:00:00+02:00")
	cases := map[string]string{
		"/:year/:month/:day/:slug/": "/2024/02/03/hello-world/",
		"/:section/:filename/":      "/posts/my-post/",
		"/:sections/:title/":        "/posts/2024/hello-world/",
		"/:yearday/:weekdayname/":   "/34/saturday/",
	}
	for pattern, want := range cases {
		got, ok := h.expandPermalink(pattern, doc, f)
		if !ok || got != want {
			t.Errorf("%s: got %q ok=%t, want %q", pattern, got, ok, want)
		}
	}
	slugged := hugoDoc(t, "title: T\nslug: Custom Slug")
	if got, _ := h.expandPermalink("/:slugorfilename/", slugged, f); got != "/custom-slug/" {
		t.Errorf("slugorfilename: got %q", got)
	}
	h.pathToLower = false
	if got, _ := h.expandPermalink("/:section/:filename/", doc, f); got != "/posts/My-Post/" {
		t.Errorf("disablePathToLower: got %q", got)
	}
}

func TestHugoPageURLDefaults(t *testing.T) {
	h := testHugoSite()
	cases := []struct {
		rel, fm, want string
	}{
		{"posts/My_Post.md", "title: x", "/posts/my_post/"},
		{"posts/bundle/index.md", "title: x", "/posts/bundle/"},
		{"about.md", "title: x", "/about/"},
		{"posts/x.md", "title: x\nslug: Other Name", "/posts/other-name/"},
		{"posts/x.md", "title: x\npath: /exact.html", "/exact.html"},
	}
	for _, tc := range cases {
		f := hugoContentFile{rel: tc.rel, dest: "content/" + tc.rel}
		if got := h.pageURL(hugoDoc(t, tc.fm), f); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.rel, got, tc.want)
		}
	}
	h.uglyURLs = true
	f := hugoContentFile{rel: "posts/x.md", dest: "content/posts/x.md"}
	if got := h.pageURL(hugoDoc(t, "title: x"), f); got != "/posts/x.html" {
		t.Errorf("ugly: got %q", got)
	}
}

func TestHugoAbsoluteAlias(t *testing.T) {
	h := testHugoSite()
	page := hugoContentFile{rel: "Posts/x.md"}
	bundle := hugoContentFile{rel: "posts/b/index.md"}
	root := hugoContentFile{rel: "about.md"}
	for _, tc := range []struct {
		f          hugoContentFile
		alias, out string
	}{
		{page, "old", "/posts/old"},
		{page, "old/", "/posts/old/"},
		{page, "/abs/", "/abs/"},
		{bundle, "older.html", "/posts/older.html"},
		{root, "me", "/me"},
	} {
		if got := h.absoluteAlias(tc.alias, tc.f); got != tc.out {
			t.Errorf("%s from %s: got %q, want %q", tc.alias, tc.f.rel, got, tc.out)
		}
	}
}

func TestHugoFindContent(t *testing.T) {
	h := testHugoSite()
	index := map[string]string{
		"posts/a.md":           "content/posts/a.md",
		"posts/b/index.md":     "content/posts/b/index.md",
		"posts/_index.md":      "content/posts/_index.md",
		"docs/a.md":            "content/docs/a.md",
		"about.en.md":          "content/about.md",
		"about.md":             "content/about.md",
		"docs/guide/_index.md": "content/docs/guide/_index.md",
	}
	from := hugoContentFile{rel: "posts/c.md"}
	for target, want := range map[string]string{
		"a":            "content/posts/a.md", // relative to the page first
		"/docs/a.md":   "content/docs/a.md",
		"b":            "content/posts/b/index.md",
		"/posts":       "content/posts/_index.md",
		"docs/guide":   "content/docs/guide/_index.md",
		"about":        "content/about.md",
		"about.en.md":  "content/about.md",
		"../docs/a.md": "content/docs/a.md",
	} {
		got, _ := h.findContent(target, from, index)
		if got != want {
			t.Errorf("%s: got %q, want %q", target, got, want)
		}
	}
	elsewhere := hugoContentFile{rel: "other/x.md"}
	if got, ambiguous := h.findContent("a", elsewhere, index); got == "" || !ambiguous {
		t.Errorf("a bare name matching two files must be flagged: %q %t", got, ambiguous)
	}
	if got, _ := h.findContent("missing", from, index); got != "" {
		t.Errorf("missing: got %q", got)
	}
}

func TestHugoHighlight(t *testing.T) {
	r := &Report{}
	tag := hugoShortcode{name: "highlight", args: []Arg{{Value: "python"}, {Value: "linenos=table, hl_lines=1 3-4, anchorlinenos=true"}}}
	got := hugoHighlight(tag, "\nprint(1)\n```\nx\n", r, "f.md", 3)
	want := "````python {linenos=true, hl_lines=\"1 3-4\"}\nprint(1)\n```\nx\n````"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if r.Count(Review) != 1 {
		t.Errorf("anchorlinenos must be reported: %v", r.Notes)
	}
	plain := hugoHighlight(hugoShortcode{args: []Arg{{Value: "go"}}}, "x := 1\n", &Report{}, "f.md", 1)
	if plain != "```go\nx := 1\n```" {
		t.Errorf("plain: %q", plain)
	}
}

func TestHugoShortcodeScanning(t *testing.T) {
	body := "a {{< figure src=\"x.png\" caption=`multi\nline` >}} b {{% notice \"warn\" %}}in{{% /notice %}} {{</* esc */>}} {{< self />}} {{ not a shortcode }}"
	tags := scanHugoShortcodes(body)
	if len(tags) != 5 {
		t.Fatalf("found %d tags: %+v", len(tags), tags)
	}
	if tags[0].name != "figure" || tags[0].args[1].Value != "multi\nline" {
		t.Errorf("raw string argument: %+v", tags[0])
	}
	if !tags[1].markdown || tags[1].args[0].Value != "warn" || !tags[2].closing {
		t.Errorf("markdown shortcode: %+v %+v", tags[1], tags[2])
	}
	if !tags[3].escaped || tags[3].inner != "esc" {
		t.Errorf("escaped: %+v", tags[3])
	}
	if !tags[4].selfClose || tags[4].name != "self" {
		t.Errorf("self-closing: %+v", tags[4])
	}
}

func TestHugoConfigMergeAndCaseInsensitiveKeys(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src, map[string]string{
		"config.yaml":                    "BaseURL: https://x.org/\nTitle: Root wins\nParams:\n  FooBar: 1\n",
		"config/_default/hugo.toml":      "title = \"Dir loses\"\nuglyURLs = true\n",
		"config/_default/params.toml":    "fooBar = 2\nOther = \"o\"\n",
		"config/_default/markup.toml":    "[goldmark.renderer]\nunsafe = true\n",
		"config/_default/languages.toml": "[en]\ncontentDir = \"content/en\"\n[de]\ncontentDir = \"content/de\"\n",
		"content/en/_index.md":           "---\ntitle: Home\n---\n",
		"content/en/posts/a.md":          "---\ntitle: A\n---\n",
		"content/de/posts/a.md":          "---\ntitle: A (de)\n---\n",
	})
	p := newPlan(src, &Report{})
	h := &hugoSite{p: p}
	if err := h.loadConfig(); err != nil {
		t.Fatal(err)
	}
	if h.cfg["title"] != "Root wins" || h.cfg["baseurl"] != "https://x.org/" || !h.uglyURLs || !h.unsafeHTML {
		t.Errorf("merged config: %v", h.cfg)
	}
	if h.params["foobar"] != uint64(1) && h.params["foobar"] != 1 {
		t.Errorf("root params must win and be lowercased: %v", h.params)
	}
	if h.params["other"] != "o" {
		t.Errorf("params from config/_default: %v", h.params)
	}
	if h.contentDir != "content/en" || len(h.langDirs) != 1 || h.langDirs[0] != "content/de" {
		t.Errorf("language content dirs: %q %v", h.contentDir, h.langDirs)
	}

	dest := filepath.Join(t.TempDir(), "out")
	result, err := Run(Options{Source: src, Dest: dest, From: "hugo", Check: checkSite})
	if err != nil {
		t.Fatal(err)
	}
	files := readTree(t, dest)
	if _, ok := files["content/posts/a.md"]; !ok {
		t.Errorf("default language content must move to content/: %v", keysOf(files))
	}
	for name := range files {
		if strings.Contains(name, "(de)") || strings.Contains(files[name], "A (de)") {
			t.Errorf("other-language content migrated: %s", name)
		}
	}
	if !strings.Contains(files["config.toml"], `posts = "/posts/{slug}.html"`) {
		t.Errorf("uglyURLs must become .html permalinks:\n%s", files["config.toml"])
	}
	if result.Report.Check != "passed" {
		t.Errorf("check: %s", result.Report.Check)
	}
}

func keysOf(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestHugoUnsafeFalseKeepsHTMLOff(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src, map[string]string{
		"hugo.toml":         "baseURL = \"https://x/\"\ntitle = \"T\"\n",
		"content/_index.md": "---\ntitle: Home\n---\n<div>html</div>\n",
	})
	dest := filepath.Join(t.TempDir(), "out")
	result, err := Run(Options{Source: src, Dest: dest, From: "hugo"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readTree(t, dest)["config.toml"], "unsafe_html = false") {
		t.Error("Hugo's default unsafe = false must carry over")
	}
	if !strings.Contains(result.Markdown, "Hugo left it out of pages") {
		t.Errorf("expected a note about omitted HTML:\n%s", result.Markdown)
	}
}

func TestHugoDetect(t *testing.T) {
	hugo := t.TempDir()
	writeTree(t, hugo, map[string]string{"config.toml": "baseURL = \"https://x/\"\n", "layouts/index.html": "x", "archetypes/default.md": "x"})
	if s, err := pickSource(hugo, "auto"); err != nil || s.Name() != "hugo" {
		t.Errorf("hugo with config.toml: %v %v", s, err)
	}
	modern := t.TempDir()
	writeTree(t, modern, map[string]string{"hugo.yaml": "title: x\n"})
	if s, err := pickSource(modern, ""); err != nil || s.Name() != "hugo" {
		t.Errorf("hugo.yaml: %v %v", s, err)
	}
	zola := t.TempDir()
	writeTree(t, zola, map[string]string{"config.toml": "base_url = \"https://x\"\n"})
	if (hugoSource{}).Detect(zola) != 0 {
		t.Error("a Zola config must not look like Hugo")
	}
}

func TestHugoTranslations(t *testing.T) {
	toml := "home = \"Home\"\n[readMore]\nother = \"Read more\"\none = \"Read one\"\n"
	got, err := hugoTranslations("en.toml", []byte(toml))
	if err != nil || got["home"] != "Home" || got["readMore"] != "Read more" {
		t.Errorf("toml: %v %v", got, err)
	}
	list := "- id: next\n  translation: Next\n- id: prev\n  translation:\n    other: Previous\n"
	got, err = hugoTranslations("en.yaml", []byte(list))
	if err != nil || got["next"] != "Next" || got["prev"] != "Previous" {
		t.Errorf("yaml list: %v %v", got, err)
	}
}

func TestCutLang(t *testing.T) {
	for in, want := range map[string]string{"post.fr": "fr", "post.zh-cn": "zh-cn", "v1.2": "", "post": "", "file.backup2": ""} {
		_, lang, _ := cutLang(in)
		if lang != want {
			t.Errorf("%s: got %q, want %q", in, lang, want)
		}
	}
}

func TestHugoReportsUnreadableShortcodes(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src, map[string]string{
		"hugo.toml":          "baseURL = \"https://x/\"\ntitle = \"T\"\n",
		"content/posts/a.md": "---\ntitle: A\n---\nbefore {{< ../../../evil >}} after\n\n`{{< also-fine-in-code >}}`\n",
	})
	result, err := Run(Options{Source: src, Dest: filepath.Join(t.TempDir(), "out"), From: "hugo"})
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, note := range result.Report.Notes {
		if strings.Contains(note.Message, "could not be read") {
			found++
			if note.Line != 4 {
				t.Errorf("line %d, want 4", note.Line)
			}
		}
	}
	if found != 1 {
		t.Errorf("expected one note, got %d: %v", found, result.Report.Notes)
	}
}

func TestHugoShortcodeOnlyLayoutsUseStarter(t *testing.T) {
	src := t.TempDir()
	writeTree(t, src, map[string]string{
		"hugo.toml":                   "baseURL = \"https://x/\"\ntitle = \"T\"\n",
		"layouts/shortcodes/box.html": `<div class="box">{{ .Inner }}</div>`,
		"content/posts/a.md":          "---\ntitle: A\n---\n{{< box >}}hi{{< /box >}}\n",
	})
	result, err := Run(Options{Source: src, Dest: filepath.Join(t.TempDir(), "out"), From: "hugo", Check: checkSite})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.Report.TemplateSet, "Nida starter") || result.Report.Check != "passed" {
		t.Errorf("templates=%q check=%q", result.Report.TemplateSet, result.Report.Check)
	}
}
