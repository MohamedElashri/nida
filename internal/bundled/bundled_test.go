package bundled

import (
	"regexp"
	"strings"
	"testing"
)

func TestMathFilesResolve(t *testing.T) {
	for _, lib := range Libraries {
		files, err := lib.Files()
		if err != nil {
			t.Fatal(err)
		}
		byPath := map[string][]byte{}
		for _, f := range files {
			byPath[f.Path] = f.Content
		}
		css := byPath[lib.Dir()+"/"+lib.styles[0]]
		if !strings.Contains(string(css), "math {") {
			t.Fatalf("%s: expected the Temml stylesheet, got %d bytes", lib.Name, len(css))
		}
		for _, m := range regexp.MustCompile(`url\('(?:\./)?([^']+\.woff2)'\)`).FindAllStringSubmatch(string(css), -1) {
			if _, ok := byPath[lib.Dir()+"/"+m[1]]; !ok {
				t.Fatalf("%s: CSS references missing font %s", lib.Name, m[1])
			}
		}
		if _, ok := byPath[lib.Dir()+"/LICENSE"]; !ok {
			t.Fatalf("%s: expected the license to be shipped", lib.Name)
		}
	}
	local, _ := MathLocal.Files()
	for _, f := range local {
		if strings.HasSuffix(f.Path, "latinmodernmath.woff2") {
			t.Fatal("the local font option must not ship Latin Modern")
		}
	}
}

func TestUsedDetectsLinkedLibraries(t *testing.T) {
	pages := []string{`<p>no styles</p>`, `<link rel="stylesheet" href="/blog/` + MathLocal.Dir() + `/Temml-Local.css">`}
	used := Used(pages)
	if len(used) != 1 || used[0].Name != "temml-local" {
		t.Fatalf("unexpected libraries %+v", used)
	}
	if _, ok := Math("none"); ok {
		t.Fatal(`math_font = "none" must ship no stylesheet`)
	}
	lib, _ := Math("latin-modern")
	if got := string(lib.Tags("/blog")); got != `<link rel="stylesheet" href="/blog/vendor/temml-`+TemmlVersion+`/Temml-Latin-Modern.css">` {
		t.Fatalf("unexpected tags %s", got)
	}
}
