package mathml

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderInlineAndDisplay(t *testing.T) {
	inline, err := Render(`e^{i\pi} + 1 = 0`, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(inline, "<math>") || !strings.Contains(inline, "<msup>") {
		t.Fatalf("unexpected inline MathML %s", inline)
	}
	display, err := Render(`\frac{a}{b}`, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(display, `display="block"`) || !strings.Contains(display, "<mfrac>") {
		t.Fatalf("unexpected display MathML %s", display)
	}
	if again, _ := Render(`\frac{a}{b}`, false); again == display {
		t.Fatal("inline and display results must be cached separately")
	}
}

func TestRenderReportsTeXErrors(t *testing.T) {
	for _, tex := range []string{`\frac{1}{`, `\notacommand x`} {
		_, err := Render(tex, false)
		var mathErr *Error
		if !errors.As(err, &mathErr) {
			t.Fatalf("%s: expected a math error, got %#v", tex, err)
		}
		if strings.Contains(err.Error(), "\u0332") || !strings.HasPrefix(err.Error(), "math ") {
			t.Fatalf("unexpected message %q", err.Error())
		}
	}
}
