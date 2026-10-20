package frontmatter

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	doc, err := Parse([]byte(`+++
title = "Hello"
date = 2026-04-12T10:00:00Z
draft = true
tags = ["go", "ssg"]
categories = ["software"]
description = "A post"
slug = "hello"
+++

# Heading
`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if doc.Metadata.Title != "Hello" {
		t.Fatalf("expected title Hello, got %q", doc.Metadata.Title)
	}
	if !doc.Metadata.Draft {
		t.Fatal("expected draft to be true")
	}
	if !strings.Contains(doc.BodyMarkdown, "# Heading") {
		t.Fatalf("expected markdown body, got %q", doc.BodyMarkdown)
	}
}

func TestParseQuotedBooleanValues(t *testing.T) {
	doc, err := Parse([]byte(`---
draft: "true"
---

Body
`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !doc.Metadata.Draft {
		t.Fatal("expected draft to be true when supplied as a quoted YAML string")
	}
}

func TestSplitMissingOpeningDelimiter(t *testing.T) {
	_, _, _, err := Split([]byte("title = \"oops\""))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "opening delimiter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSplitMissingClosingDelimiter(t *testing.T) {
	_, _, _, err := Split([]byte(`+++
title = "oops"
`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "closing delimiter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseReportsInvalidTOML(t *testing.T) {
	_, err := Parse([]byte(`+++
title = "Broken
+++
`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse TOML front matter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseYAMLFrontMatter(t *testing.T) {
	doc, err := Parse([]byte(`---
title: "Optional vs Optional-Ref"
description: "A short look"
date: 2025-10-01
slug: "std-optional-vs-optional-ref"
draft: false
---

Body
`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if doc.Metadata.Title != "Optional vs Optional-Ref" {
		t.Fatalf("unexpected title %q", doc.Metadata.Title)
	}
	if doc.Metadata.Slug != "std-optional-vs-optional-ref" {
		t.Fatalf("unexpected slug %q", doc.Metadata.Slug)
	}
	if doc.Metadata.Date.IsZero() {
		t.Fatal("expected date to parse")
	}
}

func TestParseYAMLBlockListsAndMaps(t *testing.T) {
	doc, err := Parse([]byte(`---
title: Migrated post
date: 2024-03-05T09:30:00Z
tags:
  - go
  - static sites
aliases:
  - /old/path/
author:
  name: Ada
---

Body
`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	tags, ok := doc.Metadata.Extra["tags"].([]any)
	if !ok || len(tags) != 2 || tags[1] != "static sites" {
		t.Fatalf("expected block list tags, got %#v", doc.Metadata.Extra["tags"])
	}
	if len(doc.Metadata.Aliases) != 1 || doc.Metadata.Aliases[0] != "/old/path/" {
		t.Fatalf("unexpected aliases %#v", doc.Metadata.Aliases)
	}
	author, ok := doc.Metadata.Extra["author"].(map[string]any)
	if !ok || author["name"] != "Ada" {
		t.Fatalf("expected nested author map, got %#v", doc.Metadata.Extra["author"])
	}
	if doc.Metadata.Date.Year() != 2024 {
		t.Fatalf("unexpected date %v", doc.Metadata.Date)
	}
}

func TestParseFlattensZolaTaxonomiesTable(t *testing.T) {
	doc, err := Parse([]byte(`+++
title = "Zola post"

[taxonomies]
tags = ["rust", "zola"]
+++
`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if _, ok := doc.Metadata.Extra["taxonomies"]; ok {
		t.Fatal("expected taxonomies table to be flattened")
	}
	tags, ok := doc.Metadata.Extra["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "rust" {
		t.Fatalf("expected flattened tags, got %#v", doc.Metadata.Extra["tags"])
	}
}

func TestSplitAcceptsEmptyBlocksAndByteOrderMark(t *testing.T) {
	for in, want := range map[string][2]string{
		"+++\n+++\nbody\n":                       {"", "body\n"},
		"---\n---\n":                             {"", ""},
		"+++\n+++":                               {"", ""},
		"+++\r\n+++\r\nbody":                     {"", "body"},
		"\xef\xbb\xbf+++\ntitle = \"x\"\n+++\nb": {`title = "x"`, "b"},
		"+++\na = 1\n+++":                        {"a = 1", ""},
	} {
		raw, _, body, err := Split([]byte(in))
		if err != nil || raw != want[0] || body != want[1] {
			t.Errorf("Split(%q) = %q, %q, %v; want %q, %q", in, raw, body, err, want[0], want[1])
		}
	}
}
