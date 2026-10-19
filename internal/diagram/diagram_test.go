package diagram

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestRenderProducesThemeableSVG(t *testing.T) {
	svg, err := Render("flowchart LR\n  A[Markdown] --> B[SVG]", Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<svg `, `class="pele pele-flowchart"`, `var(--pele-fg`, "Markdown", "</svg>"} {
		if !strings.Contains(svg, want) {
			t.Fatalf("expected %q in %s", want, svg)
		}
	}
	again, err := Render("flowchart LR\n  A[Markdown] --> B[SVG]", Options{})
	if err != nil || again != svg {
		t.Fatalf("expected identical output for identical source, err %v", err)
	}
}

func TestRenderSupportsOtherDiagramTypes(t *testing.T) {
	for _, source := range []string{
		"sequenceDiagram\n  Alice->>Bob: hi",
		"pie title Pets\n  \"Dogs\": 3\n  \"Cats\": 2",
		"classDiagram\n  Animal <|-- Duck",
	} {
		if _, err := Render(source, Options{}); err != nil {
			t.Fatalf("render %q: %v", source, err)
		}
	}
}

func TestRenderFitsWideDiagramsToMaxWidth(t *testing.T) {
	source := "flowchart LR\n  A[first step] --> B[second step] --> C[third step] --> D[fourth step] --> E[fifth step] --> F[sixth step] --> G[seventh step]"
	natural, err := Render(source, Options{})
	if err != nil {
		t.Fatal(err)
	}
	fitted, err := Render(source, Options{MaxWidth: 400})
	if err != nil {
		t.Fatal(err)
	}
	if width(t, natural) <= 400 || width(t, fitted) > 400 {
		t.Fatalf("expected a wide natural layout and a fitted one, got %d and %d", width(t, natural), width(t, fitted))
	}
}

func width(t *testing.T, svg string) int {
	t.Helper()
	var w int
	i := strings.Index(svg, ` width="`)
	if i < 0 {
		t.Fatalf("no width in %.200s", svg)
	}
	if _, err := fmt.Sscanf(svg[i+len(` width="`):], "%d", &w); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestRenderReportsDiagramErrors(t *testing.T) {
	_, err := Render("flowchart LR\n  A --> B[[", Options{})
	var diagErr *Error
	if !errors.As(err, &diagErr) || diagErr.Code != "syntax" || diagErr.Line == 0 {
		t.Fatalf("expected a syntax error with a line, got %#v", err)
	}
	if !strings.HasPrefix(err.Error(), "mermaid diagram") {
		t.Fatalf("unexpected message %q", err.Error())
	}

	_, err = Render("not a diagram", Options{})
	if !errors.As(err, &diagErr) || diagErr.Code != "unsupported-diagram" {
		t.Fatalf("expected unsupported-diagram, got %#v", err)
	}
}

func TestRenderIsSafeConcurrently(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svg, err := Render(fmt.Sprintf("flowchart TD\n  N%d --> M%d", i, i), Options{})
			if err == nil && !strings.Contains(svg, fmt.Sprintf("N%d", i)) {
				err = fmt.Errorf("diagram %d got another diagram's SVG", i)
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
