// Package bundled ships KaTeX and Mermaid inside the binary so sites can
// render math and diagrams without a CDN. A library's files are written to
// the output only when a rendered page links to them.
package bundled

import (
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
	"strings"
)

//go:embed files
var files embed.FS

// Library is one bundled JavaScript library.
type Library struct {
	Name    string
	Version string
	// Marker is HTML that Nida's Markdown renderer emits when a page needs
	// the library.
	Marker  string
	styles  []string
	scripts []string
}

// Dir is the library's directory in the output, such as vendor/katex-0.16.47.
func (l Library) Dir() string {
	return "vendor/" + l.Name + "-" + l.Version
}

// Tags returns the link and script tags that load the library.
func (l Library) Tags(basePath string) template.HTML {
	var b strings.Builder
	prefix := basePath + "/" + l.Dir() + "/"
	for _, name := range l.styles {
		b.WriteString(`<link rel="stylesheet" href="` + template.HTMLEscapeString(prefix+name) + `">`)
	}
	for _, name := range l.scripts {
		b.WriteString(`<script defer src="` + template.HTMLEscapeString(prefix+name) + `"></script>`)
	}
	return template.HTML(b.String())
}

var (
	KaTeX = Library{
		Name: "katex", Version: "0.16.47", Marker: `class="math math-`,
		styles:  []string{"katex.min.css"},
		scripts: []string{"katex.min.js", "nida-katex.js"},
	}
	Mermaid = Library{
		Name: "mermaid", Version: "11.17.2", Marker: `<pre class="mermaid">`,
		scripts: []string{"mermaid.min.js", "nida-mermaid.js"},
	}
	Libraries = []Library{KaTeX, Mermaid}
)

// Used returns the libraries whose files any of the given pages link to.
func Used(pages []string) []Library {
	var used []Library
	for _, lib := range Libraries {
		needle := "/" + lib.Dir() + "/"
		for _, page := range pages {
			if strings.Contains(page, needle) {
				used = append(used, lib)
				break
			}
		}
	}
	return used
}

// File is one output file of a library.
type File struct {
	Path    string
	Content []byte
}

// Files returns a library's output files, decompressing gzipped sources.
func (l Library) Files() ([]File, error) {
	root := path.Join("files", l.Name)
	var out []File
	err := fs.WalkDir(files, root, func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(name, root+"/")
		if strings.HasSuffix(rel, ".gz") {
			if data, err = gunzip(data); err != nil {
				return fmt.Errorf("decompress %s: %w", name, err)
			}
			rel = strings.TrimSuffix(rel, ".gz")
		}
		out = append(out, File{Path: path.Join(l.Dir(), rel), Content: data})
		return nil
	})
	return out, err
}

func gunzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}
