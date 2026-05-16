package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Options struct {
	TargetDir string
}

type Result struct {
	TargetDir string
	Files     []string
}

type fileSpec struct {
	path    string
	content string
}

var exampleSiteFiles = []fileSpec{
	{
		path: "config.toml",
		content: `config_version = "0.4"
base_url = "http://127.0.0.1:2906"
title = "Nida Website"
description = "A small example site built with Nida."
language = "en"
author = "Nida Author"

content_dir = "content"
template_dir = "templates"
static_dir = "static"
output_dir = "public"

paginate = 10

[[taxonomies]]
name = "tags"
render = true
paginate_by = 20

[rss]
enabled = true
filename = "rss.xml"
limit = 20
sections = ["posts"]

[atom]
enabled = true
filename = "atom.xml"
limit = 20
sections = ["posts"]

[sitemap]
enabled = true
filename = "sitemap.xml"

[robots]
enabled = true
filename = "robots.txt"

[server]
host = "127.0.0.1"
port = 2906
livereload = true

[permalinks]
posts = "/posts/{slug}/"
pages = "/{slug}/"

[search]
enabled = true
filename = "search_index.en.js"
`,
	},
	{
		path: "content/_index.md",
		content: `+++
title = "Home"
description = "A small example site built with Nida."
+++

This example is intentionally small, but it includes the shape of a real Nida
site: sections, posts, tags, aliases, internal links, syntax highlighting,
footnotes, generated feeds, search data, and static assets.

Edit anything under ` + "`content/`" + `, adjust the templates, then run:

~~~bash
nida serve --site .
~~~
`,
	},
	{
		path: "content/pages/about.md",
		content: `+++
title = "About"
description = "A simple standalone page using the page template."
template = "page"
slug = "about"
+++

This is a regular page. Pages are useful for things like about pages,
colophons, project notes, and documentation that should not appear in the post
archive.

The route for this page comes from the ` + "`pages`" + ` permalink in
` + "`config.toml`" + `.
`,
	},
	{
		path: "content/pages/search.md",
		content: `+++
title = "Search"
description = "Search this example site."
template = "search"
slug = "search"
+++
`,
	},
	{
		path: "content/posts/_index.md",
		content: `+++
title = "Posts"
description = "Notes from the example site."
template = "list"
page_template = "post"
sort_by = "date"
paginate_by = 10
+++

This archive is generated from a section file. Section front matter controls
the list template, default page template, sorting, and pagination.
`,
	},
	{
		path: "content/posts/welcome.md",
		content: `+++
title = "Welcome to Nida"
description = "A quick tour of the generated example site."
date = "2026-01-03"
aliases = ["/hello/"]

[extra]
tags = ["start", "workflow"]
+++

This example site is meant to be read and edited. It uses the core pieces you
will touch most often:

- content files under ` + "`content/`" + `
- templates under ` + "`templates/`" + `
- static assets under ` + "`static/`" + `
- site behavior in ` + "`config.toml`" + `

Nida resolves internal links written with Zola-style paths. For example, read
the [Markdown tour](@/posts/markdown-tour.md) or visit the [About page](@/pages/about.md).

The old ` + "`/hello/`" + ` path is listed as an alias in this post's front
matter, so Nida also emits a small redirect page for it.
`,
	},
	{
		path: "content/posts/markdown-tour.md",
		content: `+++
title = "Markdown Tour"
description = "GitHub-flavored Markdown, code blocks, footnotes, and details."
date = "2026-01-02"

[extra]
tags = ["markdown", "writing"]
+++

Nida renders GitHub-flavored Markdown with heading IDs, tables, task lists,
footnotes, and highlighted code fences.

## A small checklist

- [x] Write Markdown
- [x] Render templates
- [ ] Publish the generated ` + "`public/`" + ` directory

## Tables

| Feature | Where it lives |
| --- | --- |
| Content | ` + "`content/`" + ` |
| Templates | ` + "`templates/`" + ` |
| Static files | ` + "`static/`" + ` |

## Highlighted code

~~~go
package main

import "fmt"

func main() {
    fmt.Println("hello from Nida")
}
~~~

{% details(summary="A collapsible note") %}

Details blocks are useful when you want extra context without interrupting the
main reading flow.

{% end %}

Footnotes are supported too.[^note]

[^note]: This note is rendered at the bottom of the page.
`,
	},
	{
		path: "content/posts/content-model/index.md",
		content: `+++
title = "Content Model"
description = "Sections, pages, bundles, resources, and tags in one place."
date = "2026-01-01"

[extra]
tags = ["content", "workflow"]
+++

This post is a page bundle because it lives at
` + "`content/posts/content-model/index.md`" + `. Files beside it are bundle
resources and are copied next to the rendered page.

Open the bundled [note file](bundle-note.txt) after building the site to see the
resource copy in action.

The post also has tags in front matter:

~~~toml
[extra]
tags = ["content", "workflow"]
~~~

Because ` + "`config.toml`" + ` defines a ` + "`tags`" + ` taxonomy, Nida builds
both the tag index and each tag page.
`,
	},
	{
		path: "content/posts/content-model/bundle-note.txt",
		content: `This file sits beside content/posts/content-model/index.md.
Nida copies page bundle resources next to the rendered page.
`,
	},
	{
		path: "templates/base.html",
		content: `{{- define "base" -}}
<!doctype html>
<html lang="{{ default .Config.Language "en" }}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ if .Title }}{{ .Title }} | {{ end }}{{ .Config.Title }}</title>
  <meta name="description" content="{{ default .Description .Config.Description }}">
  <link rel="stylesheet" href="{{ .BasePath }}/site.css">
</head>
<body>
  <header class="site-header">
    <a class="site-title" href="{{ .BasePath }}/">{{ .Config.Title }}</a>
    <nav class="site-nav" aria-label="Primary">
      <a href="{{ .BasePath }}/posts/">Posts</a>
      <a href="{{ .BasePath }}/tags/">Tags</a>
      <a href="{{ .BasePath }}/search/">Search</a>
      <a href="{{ .BasePath }}/about/">About</a>
    </nav>
  </header>
  <main>
    {{- template "content" . }}
  </main>
  <footer class="site-footer">
    <p>{{ .Config.Description }}</p>
  </footer>
</body>
</html>
{{- end }}
`,
	},
	{
		path: "templates/index.html",
		content: `{{- define "index" -}}{{ template "base" . }}{{- end }}
{{- define "content" }}
  <section class="home">
    <h1>{{ .Config.Title }}</h1>
    <p class="lede">{{ .Config.Description }}</p>
    <div class="prose">{{ unsafeHTML .Section.BodyHTML }}</div>
  </section>
  {{ if .Pages }}
  <section>
    <h2>Latest</h2>
    <ul class="post-list">
      {{ range .Pages }}
      <li>
        <a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a>
        <span>{{ formatDate .Date }}</span>
        {{ if .Description }}<p>{{ .Description }}</p>{{ end }}
      </li>
      {{ end }}
    </ul>
  </section>
  {{ end }}
{{- end }}
`,
	},
	{
		path: "templates/post.html",
		content: `{{- define "post" -}}{{ template "base" . }}{{- end }}
{{- define "content" }}
  <article class="prose">
    <h1>{{ .Page.Title }}</h1>
    {{ if .Page.Description }}<p class="lede">{{ .Page.Description }}</p>{{ end }}
    {{ if not .Page.Date.IsZero }}<p class="post-meta">{{ formatDate .Page.Date }}</p>{{ end }}
    {{ with .Page.Extra.tags }}
    <ul class="tag-list">
      {{ range . }}
      <li><a href="{{ $.BasePath }}/tags/{{ slugify . }}/">{{ . }}</a></li>
      {{ end }}
    </ul>
    {{ end }}
    {{ unsafeHTML .Page.BodyHTML }}
  </article>
{{- end }}
`,
	},
	{
		path: "templates/page.html",
		content: `{{- define "page" -}}{{ template "base" . }}{{- end }}
{{- define "content" }}
  <article class="prose">
    <h1>{{ .Page.Title }}</h1>
    {{ if .Page.Description }}<p class="lede">{{ .Page.Description }}</p>{{ end }}
    {{ unsafeHTML .Page.BodyHTML }}
  </article>
{{- end }}
`,
	},
	{
		path: "templates/search.html",
		content: `{{- define "search" -}}{{ template "base" . }}{{- end }}
{{- define "content" }}
  <section class="search-page">
    <h1>{{ .Page.Title }}</h1>
    {{ if .Page.Description }}<p class="lede">{{ .Page.Description }}</p>{{ end }}
    <form class="search-form" role="search" data-search-form data-base-path="{{ .BasePath }}">
      <label for="search-query">Search</label>
      <input id="search-query" type="search" name="q" autocomplete="off" placeholder="Search posts and pages" data-search-input>
    </form>
    <p class="search-status" data-search-status></p>
    <ol class="search-results" data-search-results></ol>
  </section>
  <script src="{{ .BasePath }}/{{ .Config.Search.Filename }}"></script>
  <script>
  (function () {
    var form = document.querySelector("[data-search-form]");
    var input = document.querySelector("[data-search-input]");
    var results = document.querySelector("[data-search-results]");
    var status = document.querySelector("[data-search-status]");
    if (!form || !input || !results || !status) return;

    var basePath = form.getAttribute("data-base-path") || "";
    var store = window.searchIndex && window.searchIndex.documentStore;
    var docs = store && store.docs ? store.docs : {};
    var entries = Object.keys(docs).map(function (url) {
      var doc = docs[url] || {};
      return {
        url: url,
        title: doc.title || url,
        description: doc.description || "",
        body: doc.body || ""
      };
    });

    function termsFor(query) {
      return query.toLowerCase().trim().split(/\s+/).filter(Boolean);
    }

    function entryScore(entry, terms) {
      var title = entry.title.toLowerCase();
      var description = entry.description.toLowerCase();
      var body = entry.body.toLowerCase();
      var total = 0;
      terms.forEach(function (term) {
        if (title.indexOf(term) !== -1) total += 8;
        if (description.indexOf(term) !== -1) total += 4;
        if (body.indexOf(term) !== -1) total += 1;
      });
      return total;
    }

    function summaryFor(entry) {
      if (entry.description) return entry.description;
      return entry.body.slice(0, 160);
    }

    function render(query) {
      var terms = termsFor(query);
      results.replaceChildren();
      if (terms.length === 0) {
        status.textContent = "";
        return;
      }

      var matches = entries.map(function (entry) {
        return { entry: entry, score: entryScore(entry, terms) };
      }).filter(function (match) {
        return match.score > 0;
      }).sort(function (a, b) {
        return b.score - a.score || a.entry.title.localeCompare(b.entry.title);
      }).slice(0, 10);

      status.textContent = matches.length === 1 ? "1 result" : matches.length + " results";
      matches.forEach(function (match) {
        var item = document.createElement("li");
        var link = document.createElement("a");
        var summary = document.createElement("p");
        link.href = basePath + match.entry.url;
        link.textContent = match.entry.title;
        summary.textContent = summaryFor(match.entry);
        item.append(link, summary);
        results.append(item);
      });
    }

    form.addEventListener("submit", function (event) {
      event.preventDefault();
      render(input.value);
    });
    input.addEventListener("input", function () {
      render(input.value);
    });
  })();
  </script>
{{- end }}
`,
	},
	{
		path: "templates/list.html",
		content: `{{- define "list" -}}{{ template "base" . }}{{- end }}
{{- define "content" }}
  <section>
    <h1>{{ .Section.Title }}</h1>
    {{ if .Section.Description }}<p class="lede">{{ .Section.Description }}</p>{{ end }}
    <div class="prose">{{ unsafeHTML .Section.BodyHTML }}</div>
    <ul class="post-list">
      {{ range .Pages }}
      <li>
        <a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a>
        <span>{{ formatDate .Date }}</span>
        {{ if .Description }}<p>{{ .Description }}</p>{{ end }}
      </li>
      {{ end }}
    </ul>
  </section>
{{- end }}
`,
	},
	{
		path: "templates/taxonomy.html",
		content: `{{- define "taxonomy" -}}{{ template "base" . }}{{- end }}
{{- define "content" }}
  {{ if .Term.Name }}
  <section>
    <p class="eyebrow">{{ .Taxonomy.Name }}</p>
    <h1>{{ .Term.Name }}</h1>
    <ul class="post-list">
      {{ range .Pages }}
      <li>
        <a href="{{ $.BasePath }}{{ .URL }}">{{ .Title }}</a>
        <span>{{ formatDate .Date }}</span>
        {{ if .Description }}<p>{{ .Description }}</p>{{ end }}
      </li>
      {{ end }}
    </ul>
  </section>
  {{ else }}
  <section>
    <h1>{{ .Taxonomy.Name }}</h1>
    <ul class="tag-list tag-list-large">
      {{ range .Terms }}
      <li><a href="{{ $.BasePath }}{{ .URL }}">{{ .Name }}</a></li>
      {{ end }}
    </ul>
  </section>
  {{ end }}
{{- end }}
`,
	},
	{
		path: "static/site.css",
		content: `:root {
  color-scheme: light;
  font-family: system-ui, sans-serif;
  line-height: 1.6;
  color: #1f2933;
  background: #ffffff;
}

body {
  max-width: 44rem;
  margin: 0 auto;
  padding: 2rem 1rem;
}

a {
  color: #0969da;
}

.site-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 3rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #d9e2ec;
}

.site-title {
  font-weight: 700;
  text-decoration: none;
}

.site-nav {
  display: flex;
  gap: 0.75rem;
  font-size: 0.95rem;
}

.site-nav a,
.tag-list a {
  text-decoration: none;
}

.prose {
  font-size: 1.05rem;
}

.lede {
  color: #52606d;
  font-size: 1.1rem;
}

.post-meta,
.eyebrow,
.post-list span {
  color: #627d98;
  font-size: 0.9rem;
}

.post-list,
.tag-list {
  list-style: none;
  padding-left: 0;
}

.post-list {
  display: grid;
  gap: 1rem;
}

.post-list li {
  padding-bottom: 1rem;
  border-bottom: 1px solid #d9e2ec;
}

.post-list p {
  margin: 0.25rem 0 0;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin: 1rem 0;
}

.tag-list a {
  display: inline-block;
  padding: 0.15rem 0.55rem;
  border: 1px solid #bcccdc;
  border-radius: 999px;
  color: #334e68;
  background: #ffffff;
}

.tag-list-large a {
  font-size: 1.05rem;
}

.search-form {
  display: grid;
  gap: 0.35rem;
  margin: 1.5rem 0 0.5rem;
}

.search-form label {
  color: #627d98;
  font-size: 0.9rem;
  font-weight: 700;
}

.search-form input {
  border: 1px solid #bcccdc;
  border-radius: 4px;
  color: #1f2933;
  font: inherit;
  padding: 0.65rem 0.75rem;
}

.search-form input:focus {
  border-color: #0969da;
  outline: 3px solid #dbeafe;
}

.search-status {
  color: #627d98;
  min-height: 1.5rem;
}

.search-results {
  display: grid;
  gap: 0.75rem;
  list-style: none;
  padding-left: 0;
}

.search-results li {
  border-bottom: 1px solid #d9e2ec;
  padding-bottom: 0.75rem;
}

.search-results a {
  font-weight: 700;
}

.search-results p {
  color: #52606d;
  margin: 0.25rem 0 0;
}

.site-footer {
  margin-top: 4rem;
  padding-top: 1rem;
  border-top: 1px solid #d9e2ec;
  color: #627d98;
}

pre {
  overflow-x: auto;
  padding: 1rem;
  background: #f6f8fa;
  color: #24292f;
}

code {
  font-size: 0.95em;
}

.z-c,
.z-c1 {
  color: #6e7781;
}

.z-k,
.z-kc,
.z-kd,
.z-kn,
.z-kp,
.z-kr,
.z-kt {
  color: #cf222e;
}

.z-s,
.z-s1,
.z-s2,
.z-sa {
  color: #0a3069;
}

.z-m,
.z-mi {
  color: #0550ae;
}

.z-nf {
  color: #8250df;
}

.z-nx,
.z-na {
  color: #953800;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  padding: 0.4rem;
  border: 1px solid #bcccdc;
  text-align: left;
}

details {
  padding: 0.75rem 1rem;
  border: 1px solid #bcccdc;
  background: #ffffff;
}
`,
	},
}

func Init(opts Options) (Result, error) {
	target := opts.TargetDir
	if target == "" {
		target = "."
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return Result{}, fmt.Errorf("resolve init path %q: %w", target, err)
	}

	if err := ensureWritableTarget(absTarget); err != nil {
		return Result{}, err
	}

	written := make([]string, 0, len(exampleSiteFiles))
	for _, file := range exampleSiteFiles {
		path := filepath.Join(absTarget, filepath.FromSlash(file.path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Result{}, fmt.Errorf("create directory for %q: %w", file.path, err)
		}
		if err := os.WriteFile(path, []byte(file.content), 0o644); err != nil {
			return Result{}, fmt.Errorf("write %q: %w", file.path, err)
		}
		written = append(written, file.path)
	}

	return Result{TargetDir: absTarget, Files: written}, nil
}

func ensureWritableTarget(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create init path %q: %w", target, err)
			}
			return nil
		}
		return fmt.Errorf("read init path %q: %w", target, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("init path %q is not a directory", target)
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return fmt.Errorf("read init path %q: %w", target, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("init path %q is not empty", target)
	}

	return nil
}
