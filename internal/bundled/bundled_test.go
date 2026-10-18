package bundled

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestFilesDecompressAndFontsResolve(t *testing.T) {
	files, err := KaTeX.Files()
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string][]byte{}
	for _, f := range files {
		byPath[f.Path] = f.Content
	}
	css := byPath["vendor/katex-"+KaTeX.Version+"/katex.min.css"]
	if !bytes.Contains(css, []byte(".katex")) {
		t.Fatalf("expected decompressed CSS, got %d bytes", len(css))
	}
	for _, m := range regexp.MustCompile(`url\((fonts/[^)]+)\)`).FindAllSubmatch(css, -1) {
		if _, ok := byPath["vendor/katex-"+KaTeX.Version+"/"+string(m[1])]; !ok {
			t.Fatalf("CSS references missing font %s", m[1])
		}
	}
	if _, ok := byPath["vendor/katex-"+KaTeX.Version+"/LICENSE"]; !ok {
		t.Fatal("expected the license to be shipped")
	}

	mermaid, err := Mermaid.Files()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range mermaid {
		if strings.HasSuffix(f.Path, "/mermaid.min.js") && bytes.Contains(f.Content, []byte(`globalThis["mermaid"]`)) {
			found = true
		}
	}
	if !found {
		t.Fatal("expected mermaid.min.js exposing the mermaid global")
	}
}

func TestUsedDetectsLinkedLibraries(t *testing.T) {
	pages := []string{`<p>no scripts</p>`, `<script defer src="/blog/` + Mermaid.Dir() + `/mermaid.min.js"></script>`}
	used := Used(pages)
	if len(used) != 1 || used[0].Name != "mermaid" {
		t.Fatalf("unexpected libraries %+v", used)
	}
	if got := string(KaTeX.Tags("/blog")); !strings.Contains(got, `href="/blog/vendor/katex-`) || !strings.Contains(got, "nida-katex.js") {
		t.Fatalf("unexpected tags %s", got)
	}
}
