// Package bundled ships the stylesheet and fonts for math inside the binary,
// so sites need no CDN. Math itself is MathML rendered at build time by
// package mathml, and Mermaid diagrams are SVG rendered by package diagram,
// so no library here is JavaScript. A library's files are written to the
// output only when a rendered page links to them.
package bundled

import (
	"embed"
	"html/template"
	"path"
	"strings"
)

//go:embed files
var files embed.FS

// Library is one set of bundled files.
type Library struct {
	Name    string
	Version string
	// src is the directory under files/ that holds the library.
	src     string
	styles  []string
	scripts []string
	// files lists every file written to the output, relative to src.
	files []string
}

// Dir is the library's directory in the output, such as vendor/temml-0.14.0.
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

// Math font choices for markdown.math_font.
const (
	MathFontLatinModern = "latin-modern"
	MathFontLocal       = "local"
	// MathFontNone ships no math stylesheet, for themes that style MathML
	// themselves.
	MathFontNone = "none"
)

var (
	// MathLatinModern styles MathML with the Latin Modern Math font, the
	// look of LaTeX, the same in every browser.
	MathLatinModern = Library{
		Name: "temml", Version: TemmlVersion, src: "temml",
		styles: []string{"Temml-Latin-Modern.css"},
		files:  []string{"Temml-Latin-Modern.css", "Temml.woff2", "latinmodernmath.woff2", "LICENSE", "GUST-FONT-LICENSE.txt"},
	}
	// MathLocal styles MathML with the reader's installed math fonts, such
	// as Cambria Math or STIX Two Math, and downloads only a 10 KB font.
	MathLocal = Library{
		Name: "temml-local", Version: TemmlVersion, src: "temml",
		styles: []string{"Temml-Local.css"},
		files:  []string{"Temml-Local.css", "Temml.woff2", "LICENSE"},
	}
	Libraries = []Library{MathLatinModern, MathLocal}
)

// Math returns the math library for a markdown.math_font value, or false
// for "none".
func Math(font string) (Library, bool) {
	switch font {
	case MathFontNone:
		return Library{}, false
	case MathFontLocal:
		return MathLocal, true
	}
	return MathLatinModern, true
}

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

// Files returns a library's output files.
func (l Library) Files() ([]File, error) {
	out := make([]File, 0, len(l.files))
	for _, name := range l.files {
		data, err := files.ReadFile(path.Join("files", l.src, name))
		if err != nil {
			return nil, err
		}
		out = append(out, File{Path: path.Join(l.Dir(), name), Content: data})
	}
	return out, nil
}
