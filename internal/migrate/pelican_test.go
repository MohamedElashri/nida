package migrate

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPelicanGolden(t *testing.T) {
	result := runGolden(t, "pelican")
	if result.Report.URLsKept < 6 {
		t.Errorf("expected published URLs to be kept, got %d", result.Report.URLsKept)
	}
}

func TestPelicanDetect(t *testing.T) {
	if got := (pelicanSource{}).Detect("testdata/pelican/site"); got < 4 {
		t.Errorf("Detect = %d, want a strong match", got)
	}
	if got := (pelicanSource{}).Detect("testdata/zola/site"); got != 0 {
		t.Errorf("Detect on a Zola site = %d, want 0", got)
	}
	if s, err := pickSource("testdata/pelican/site", "auto"); err != nil || s.Name() != "pelican" {
		t.Errorf("auto detection picked %v, %v", s, err)
	}
}

func TestReadPyConfig(t *testing.T) {
	src := `
import os
AUTHOR = 'Ada'  # trailing comment
SITENAME = "It's " 'joined'
SITEURL = 'https://x.org'
FEED = SITEURL + '/feed.xml'
N = 10
F = -1.5
FLAG = True
NOTHING = None
PATHS = ['a', "b",
         'c']
TUPLE = 'x', 'y'
LINKS = (('A', 'https://a'),
         ('B', 'https://b'),)
META = {
    'extra/robots.txt': {'path': 'robots.txt'},
}
LONG = """multi
line"""
RAW = r'\d+'
ENV = os.environ.get('X', '')
FSTR = f'{AUTHOR}'
COMP = [x for x in range(3)]
UNKNOWN = OTHER_NAME
if os.environ.get('DEBUG'):
    SITENAME = 'Debug'
TYPED: int = 3
A = 1; B = 2
`
	cfg := readPyConfig(src, nil)
	want := map[string]any{
		"AUTHOR":   "Ada",
		"SITENAME": "It's joined",
		"SITEURL":  "https://x.org",
		"FEED":     "https://x.org/feed.xml",
		"N":        int64(10),
		"F":        -1.5,
		"FLAG":     true,
		"NOTHING":  pyNone{},
		"PATHS":    []any{"a", "b", "c"},
		"TUPLE":    []any{"x", "y"},
		"LINKS":    []any{[]any{"A", "https://a"}, []any{"B", "https://b"}},
		"META":     map[string]any{"extra/robots.txt": map[string]any{"path": "robots.txt"}},
		"LONG":     "multi\nline",
		"RAW":      `\d+`,
		"TYPED":    int64(3),
		"A":        int64(1),
		"B":        int64(2),
	}
	for k, v := range want {
		if got := cfg.values[k]; !reflect.DeepEqual(got, v) {
			t.Errorf("%s = %#v, want %#v", k, got, v)
		}
	}
	for _, k := range []string{"ENV", "FSTR", "COMP", "UNKNOWN"} {
		if _, ok := cfg.bad[k]; !ok {
			t.Errorf("%s should be reported as not literal, got %#v", k, cfg.values[k])
		}
	}
	if cfg.values["SITENAME"] == "Debug" {
		t.Error("assignments inside an if block must be ignored")
	}

	// publishconf.py: from pelicanconf import *, then overrides that use
	// pelicanconf's names.
	pub := readPyConfig("from pelicanconf import *\nSITEURL = 'https://prod'\nFEED2 = SITEURL + '/f'\n", cfg)
	if len(pub.imports) != 1 || pub.imports[0] != "pelicanconf" {
		t.Errorf("imports = %v", pub.imports)
	}
	cfg.merge(pub)
	if cfg.values["SITEURL"] != "https://prod" || cfg.values["FEED2"] != "https://prod/f" || cfg.values["AUTHOR"] != "Ada" {
		t.Errorf("merge: %v", cfg.values)
	}
}

func TestPelicanSlugify(t *testing.T) {
	s := pelicanSlugger{}
	for in, want := range map[string]string{
		"The Difference Engine, Revisited!": "the-difference-engine-revisited",
		"Café Ünïcode & Ölçü":               "cafe-unicode-olcu",
		"snake_case stays":                  "snake_case-stays",
		"  spaces  --  and dashes ":         "spaces-and-dashes",
		"C++ tips":                          "c-tips",
		"Привет мир":                        "privet-mir",
		"Straße":                            "strasse",
	} {
		if got, _ := s.slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
	if _, exact := s.slugify("日本語"); exact {
		t.Error("CJK text must be reported as an approximate slug")
	}
	if got, _ := (pelicanSlugger{preserveCase: true}).slugify("Mixed Case"); got != "Mixed-Case" {
		t.Errorf("preserve case: %q", got)
	}
	uni := pelicanSlugger{useUnicode: true, subs: unicodePelicanSubs}
	if got, _ := uni.slugify("Café crème!"); got != "café-crème" {
		t.Errorf("unicode slug: %q", got)
	}
}

func TestPelicanFormat(t *testing.T) {
	date := time.Date(2024, 3, 5, 10, 20, 0, 0, time.UTC)
	fields := map[string]string{"slug": "hello", "category": "misc", "lang": "en"}
	for pattern, want := range map[string]string{
		"{slug}.html":                                     "hello.html",
		"{date:%Y}/{date:%m}/{slug}.html":                 "2024/03/hello.html",
		"{date:%Y}/{date:%b}/{date:%d}/{slug}/index.html": "2024/Mar/05/hello/index.html",
		"{category}/{slug}-{lang}.html":                   "misc/hello-en.html",
		"posts/{{literal}}/{slug}":                        "posts/{literal}/hello",
	} {
		got, err := pelicanFormat(pattern, fields, date)
		if err != nil || got != want {
			t.Errorf("pelicanFormat(%q) = %q, %v; want %q", pattern, got, err, want)
		}
	}
	if _, err := pelicanFormat("{unknown}.html", fields, date); err == nil {
		t.Error("unknown fields must fail")
	}
	if _, err := pelicanFormat("{date:%Y}/{slug}", fields, time.Time{}); err == nil {
		t.Error("a date field without a date must fail")
	}
}

func TestPelicanPermalink(t *testing.T) {
	for in, want := range map[string]string{
		"{slug}.html":                     "/{slug}.html",
		"{date:%Y}/{date:%m}/{slug}.html": "/{year}/{month}/{slug}.html",
		"posts/{slug}/index.html":         "/posts/{slug}/",
		"{category}/{slug}.html":          "/{categories}/{slug}.html",
	} {
		if got, ok := pelicanPermalink(in, true); !ok || got != want {
			t.Errorf("pelicanPermalink(%q) = %q, %t; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"{date:%b}/{slug}.html", "{author}/{slug}.html", "fixed.html"} {
		if _, ok := pelicanPermalink(in, true); ok {
			t.Errorf("pelicanPermalink(%q) should not map", in)
		}
	}
}

func TestPelicanMetaHeaders(t *testing.T) {
	site := &pelicanSite{p: newPlan(t.TempDir(), &Report{})}
	it := &pelicanItem{meta: map[string]any{}}
	site.readMarkdown(it, "Title: Hello: World\nTAGS: a, b\nSummary: one\n    two\n\nBody line\n")
	if it.str("title") != "Hello: World" || it.str("tags") != "a, b" || it.str("summary") != "one two" {
		t.Errorf("meta = %v", it.meta)
	}
	if it.body != "Body line\n" || it.bodyLine != 6 {
		t.Errorf("body = %q line %d", it.body, it.bodyLine)
	}

	it = &pelicanItem{meta: map[string]any{}}
	site.readMarkdown(it, "# Just a heading\n\nNo metadata.\n")
	if len(it.meta) != 0 || !strings.HasPrefix(it.body, "# Just a heading") {
		t.Errorf("a file without metadata: %v %q", it.meta, it.body)
	}

	// Python-Markdown also accepts --- around meta lines that are not YAML.
	it = &pelicanItem{meta: map[string]any{}}
	site.readMarkdown(it, "---\nTitle: Colon: here: too\nDate: 2020-01-01\n---\nText\n")
	if it.str("title") != "Colon: here: too" || it.body != "Text\n" {
		t.Errorf("--- meta: %v %q", it.meta, it.body)
	}

	if got := pelicanList("a; b, c"); !reflect.DeepEqual(got, []string{"a", "b, c"}) {
		t.Errorf("semicolon list: %v", got)
	}
	if got := pelicanList("a, b ,c"); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("comma list: %v", got)
	}
}

func TestPelicanDatesUseTimezone(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tz database")
	}
	site := &pelicanSite{loc: loc}
	got, dateOnly, err := site.parseDate("2024-07-01 09:30")
	if err != nil || dateOnly || got.Format(time.RFC3339) != "2024-07-01T09:30:00-04:00" {
		t.Errorf("naive date: %v %v %v", got, dateOnly, err)
	}
	got, _, _ = site.parseDate("2024-07-01T09:30:00+02:00")
	if got.Format(time.RFC3339) != "2024-07-01T09:30:00+02:00" {
		t.Errorf("an explicit offset must win: %v", got)
	}
}

func TestPythonMarkdownRewrites(t *testing.T) {
	site := &pelicanSite{p: newPlan(t.TempDir(), &Report{})}
	it := &pelicanItem{rel: "a.md", bodyLine: 1}
	in := "Intro\n\n    #!python\n    x = 1\n\n## H {: #h .c }\n\n```{.go}\nfunc() {}\n```\n\n    plain indented\n    code {: .x}\n"
	got := site.pythonMarkdown(it, in)
	want := "Intro\n\n```python {linenos}\nx = 1\n```\n\n## H {#h .c}\n\n```go\nfunc() {}\n```\n\n    plain indented\n    code {: .x}\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestConvertRST(t *testing.T) {
	doc := convertRST(`=======
 Title
=======

:date: 2020-01-01
:tags: a, b

Section
-------

Para with ` + "``code`` and a `link <https://x>`_." + `

* item one
  continued
* item two

::

    literal

.. code:: go

   fmt.Println()

.. tip:: Short tip.

.. unknown:: arg

   body
`)
	if doc.meta["title"] != "Title" || doc.meta["date"] != "2020-01-01" || doc.meta["tags"] != "a, b" {
		t.Errorf("meta = %v", doc.meta)
	}
	want := "## Section\n\nPara with `code` and a [link](https://x).\n\n- item one\n  continued\n- item two\n\n```\nliteral\n```\n\n```go\nfmt.Println()\n```\n\n> [!TIP]\n> Short tip.\n\n```rst\n.. unknown:: arg\n\n   body\n```\n"
	if doc.body != want {
		t.Errorf("body:\n%s\nwant:\n%s", doc.body, want)
	}
	if len(doc.notes) != 1 || doc.notes[0].sev != Manual || doc.notes[0].line != 27 {
		t.Errorf("notes = %+v", doc.notes)
	}
}

func TestConvertRSTEscapesMarkdownSyntax(t *testing.T) {
	doc := convertRST("A paragraph\n# not a heading\n1. not a list, wrapped\n<div> is text\n")
	want := "A paragraph\n\\# not a heading\n1\\. not a list, wrapped\n&lt;div> is text\n"
	if doc.body != want {
		t.Errorf("got %q, want %q", doc.body, want)
	}
}

func TestConvertRSTMultipleTopSectionsAreHeadings(t *testing.T) {
	doc := convertRST(":title: From Field\n\nOne\n===\n\nText.\n\nTwo\n===\n")
	if doc.meta["title"] != "From Field" {
		t.Errorf("title = %q", doc.meta["title"])
	}
	if !strings.Contains(doc.body, "## One") || !strings.Contains(doc.body, "## Two") {
		t.Errorf("body = %q", doc.body)
	}
}

func TestRSTEllipsisIsNotExplicitMarkup(t *testing.T) {
	doc := convertRST("Title\n=====\n\n...and then it worked.\n\n..\n\n.. not-a-directive\n")
	if doc == nil {
		t.Fatal("no document")
	}
}
