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
