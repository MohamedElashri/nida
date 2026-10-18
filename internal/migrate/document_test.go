package migrate

import (
	"strings"
	"testing"
	"time"

	"github.com/MohamedElashri/nida/internal/frontmatter"
)

func TestDocumentTOMLDeleteTableKeepsTheRest(t *testing.T) {
	in := "+++\ntitle = \"A\" # c\ndraft = true\n\n[params]\nmood = \"x\"\n\n[params.deep]\nk = 1\n\n[extra]\nimage = \"y\"\n+++\nbody\n"
	doc, err := ParseDocument([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Delete("params") {
		t.Fatal("delete failed")
	}
	want := "+++\ntitle = \"A\" # c\ndraft = true\n\n[extra]\nimage = \"y\"\n+++\nbody\n"
	if got := string(doc.Bytes()); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDocumentUnchangedIsByteForByte(t *testing.T) {
	inputs := []string{
		"+++\ntitle = \"A\"\n# comment\n+++\nbody\n",
		"---\ntitle: A # comment\ntags: [a, b]\n---\nbody\r\n",
		"no front matter at all\n",
		"\n\n+++\ntitle = \"leading blank lines\"\n+++\n",
	}
	for _, in := range inputs {
		doc, err := ParseDocument([]byte(in))
		if err != nil {
			t.Fatalf("parse %q: %v", in, err)
		}
		if got := string(doc.Bytes()); got != in {
			t.Errorf("unchanged document rewritten:\n got %q\nwant %q", got, in)
		}
	}
}

func TestDocumentStripsBOMAndKeepsCRLF(t *testing.T) {
	doc, err := ParseDocument([]byte("\xef\xbb\xbf---\r\ntitle: A\r\n---\r\nline one\r\nline two\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Changed() {
		t.Fatal("a BOM must count as a change")
	}
	if err := doc.Set("draft", true); err != nil {
		t.Fatal(err)
	}
	want := "---\r\ntitle: A\r\ndraft: true\r\n---\r\nline one\r\nline two\r\n"
	if got := string(doc.Bytes()); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDocumentToleratesTrailingSpacesOnDelimiters(t *testing.T) {
	doc, err := ParseDocument([]byte("---  \ntitle: A\n--- \nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.String("title") != "A" || doc.Body != "body\n" {
		t.Fatalf("got title=%q body=%q", doc.String("title"), doc.Body)
	}
	if got := string(doc.Bytes()); got != "---\ntitle: A\n---\nbody\n" {
		t.Errorf("delimiters not normalized: %q", got)
	}
}

func TestDocumentYAMLEditsKeepCommentsAndOrder(t *testing.T) {
	in := "---\n# heading comment\ntitle: Hello # trailing\nlastmod: 2024-01-02\ntags:\n  - a\n  - b\n---\nbody\n"
	doc, err := ParseDocument([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Rename("lastmod", "updated") {
		t.Fatal("rename failed")
	}
	if err := doc.Set("slug", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := doc.Set("title", "Hi"); err != nil {
		t.Fatal(err)
	}
	got := string(doc.Bytes())
	for _, want := range []string{"# heading comment", "title: Hi # trailing", "updated: 2024-01-02", "slug: hello", "  - a"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Index(got, "title:") > strings.Index(got, "updated:") || strings.Index(got, "updated:") > strings.Index(got, "slug:") {
		t.Errorf("key order changed:\n%s", got)
	}
}

func TestDocumentTOMLLineEdits(t *testing.T) {
	in := `+++
title = "Hello"
# keep me
lastmod = 2024-01-02
tags = [
  "a",
  "b",
]
draft = false

[extra]
image = "x.png"
+++
body
`
	doc, err := ParseDocument([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	doc.Rename("lastmod", "updated")
	if err := doc.Set("tags", []any{"c"}); err != nil {
		t.Fatal(err)
	}
	doc.Delete("draft")
	if err := doc.Set("slug", "hello world"); err != nil {
		t.Fatal(err)
	}
	got := string(doc.Bytes())
	want := `+++
title = "Hello"
# keep me
updated = 2024-01-02
tags = ["c"]
slug = "hello world"

[extra]
image = "x.png"
+++
body
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if extra, _ := doc.Get("extra"); extra.(map[string]any)["image"] != "x.png" {
		t.Errorf("table lost: %v", extra)
	}
}

func TestDocumentTOMLSetTableAppendsAtEnd(t *testing.T) {
	doc, err := ParseDocument([]byte("+++\ntitle = \"A\"\n+++\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Set("extra", map[string]any{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	if err := doc.Set("draft", true); err != nil {
		t.Fatal(err)
	}
	got := string(doc.Bytes())
	if !strings.Contains(got, "title = \"A\"\ndraft = true\n\n[extra]\nk = ") {
		t.Errorf("plain keys must stay above tables:\n%s", got)
	}
}

func TestDocumentJSONBecomesYAML(t *testing.T) {
	doc, err := ParseDocument([]byte("{\n  \"title\": \"J\",\n  \"tags\": [\"x\"],\n  \"note\": \"has } brace\"\n}\nbody text\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Format != JSON || doc.String("title") != "J" || doc.String("note") != "has } brace" {
		t.Fatalf("parsed %v %v", doc.Format, doc.Values())
	}
	got := string(doc.Bytes())
	if !strings.HasPrefix(got, "---\n") || !strings.Contains(got, "title: J") || !strings.HasSuffix(got, "---\nbody text\n") {
		t.Errorf("got:\n%s", got)
	}
}

func TestDocumentWithoutFrontMatterGainsTOML(t *testing.T) {
	doc, err := ParseDocument([]byte("# Heading\n\ntext\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Format != NoFrontMatter {
		t.Fatalf("format %v", doc.Format)
	}
	if err := doc.Set("title", "Heading"); err != nil {
		t.Fatal(err)
	}
	if got := string(doc.Bytes()); got != "+++\ntitle = \"Heading\"\n+++\n# Heading\n\ntext\n" {
		t.Errorf("got %q", got)
	}
}

func TestDocumentBodyLine(t *testing.T) {
	doc, err := ParseDocument([]byte("\n+++\na = 1\nb = 2\n+++\nfirst body line\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.BodyLine != 6 {
		t.Errorf("BodyLine = %d, want 6", doc.BodyLine)
	}
}

func TestDocumentRejectsUnclosedFrontMatter(t *testing.T) {
	if _, err := ParseDocument([]byte("+++\ntitle = \"x\"\nbody\n")); err == nil {
		t.Error("expected an error for missing closing delimiter")
	}
}

func TestDocumentRenameRefusesToOverwrite(t *testing.T) {
	doc, _ := ParseDocument([]byte("---\nlastmod: 2024-01-01\nupdated: 2024-02-02\n---\n"))
	if doc.Rename("lastmod", "updated") {
		t.Error("rename onto an existing key must fail")
	}
	if doc.Changed() {
		t.Error("failed rename must not change the document")
	}
}

func TestNormalizeDate(t *testing.T) {
	cases := []struct {
		in, format, want string
	}{
		{"---\ndate: 2024-05-01 10:00:00 -0500\n---\n", "yaml", "date: \"2024-05-01T10:00:00-05:00\""},
		{"---\ndate: 2024-05-01\n---\n", "yaml", "date: 2024-05-01"},
		{"+++\ndate = \"2024/05/01\"\n+++\n", "toml", "date = 2024-05-01\n"},
	}
	for _, tc := range cases {
		doc, err := ParseDocument([]byte(tc.in))
		if err != nil {
			t.Fatal(err)
		}
		r := &Report{}
		if !NormalizeDate(doc, "date", r, "f.md") {
			t.Fatalf("%q: normalize failed: %v", tc.in, r.Notes)
		}
		if got := string(doc.Bytes()); !strings.Contains(got, tc.want) {
			t.Errorf("%q: got %q, want it to contain %q", tc.in, got, tc.want)
		}
	}

	doc, _ := ParseDocument([]byte("---\ndate: next tuesday\n---\n"))
	r := &Report{}
	if NormalizeDate(doc, "date", r, "f.md") || r.Count(Manual) != 1 {
		t.Errorf("unparseable date must be reported: %v", r.Notes)
	}
}

func TestParseDateLayouts(t *testing.T) {
	for in, want := range map[string]string{
		"2024-05-01T10:00:00Z":      "2024-05-01T10:00:00Z",
		"2024-05-01 10:00:00 +0200": "2024-05-01T10:00:00+02:00",
		"2024-05-01 10:00":          "2024-05-01T10:00:00Z",
		"January 2, 2006":           "2006-01-02T00:00:00Z",
	} {
		got, _, err := ParseDate(in)
		if err != nil {
			t.Errorf("ParseDate(%q): %v", in, err)
			continue
		}
		if got.Format(time.RFC3339) != want {
			t.Errorf("ParseDate(%q) = %s, want %s", in, got.Format(time.RFC3339), want)
		}
	}
	if _, dateOnly, _ := ParseDate("2024-01-10"); !dateOnly {
		t.Error("a bare date must be date-only")
	}
	if _, dateOnly, _ := ParseDate("2024-01-10 12:00"); dateOnly {
		t.Error("a date with a time must not be date-only")
	}
}

func TestDocumentJSONKeepsStringTypes(t *testing.T) {
	doc, err := ParseDocument([]byte(`{"version": "1.0", "flag": "true", "n": 3, "empty": ""}` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	reparsed, err := ParseDocument(doc.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{"version": "1.0", "flag": "true", "n": 3, "empty": ""} {
		if got, _ := reparsed.Get(key); got != want {
			t.Errorf("%s = %#v, want %#v\n%s", key, got, want, doc.Bytes())
		}
	}
}

func TestEmptyFrontMatterParsesInNida(t *testing.T) {
	for _, format := range []Format{TOML, YAML} {
		data := NewDocument(format, "body\n").Bytes()
		if _, err := frontmatter.Parse(data); err != nil {
			t.Errorf("format %d: Nida cannot parse %q: %v", format, data, err)
		}
	}
}
