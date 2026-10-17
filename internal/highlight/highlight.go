package highlight

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

const DefaultTheme = "github"

// Options are per-block rendering settings taken from a fenced code block's
// info string.
type Options struct {
	// HighlightLines lists inclusive [start, end] line ranges to emphasize.
	HighlightLines [][2]int
	LineNumbers    bool
	// LineStart is the number shown for the first line. Zero means 1.
	LineStart int
}

func Render(code, language, theme string) (string, error) {
	return RenderWithOptions(code, language, theme, Options{})
}

func RenderWithOptions(code, language, theme string, opts Options) (string, error) {
	language = normalizeLanguage(language)
	if language == "" {
		return renderPlain(code, ""), nil
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		return renderPlain(code, language), nil
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", fmt.Errorf("tokenize %q code block: %w", language, err)
	}

	style := styles.Get(strings.TrimSpace(theme))
	if style == nil {
		style = styles.Get(DefaultTheme)
	}
	if style == nil {
		style = styles.Fallback
	}

	formatterOpts := []chromahtml.Option{
		chromahtml.WithClasses(true),
		chromahtml.ClassPrefix("z-"),
		chromahtml.WithLineNumbers(opts.LineNumbers),
	}
	if len(opts.HighlightLines) > 0 {
		formatterOpts = append(formatterOpts, chromahtml.HighlightLines(opts.HighlightLines))
	}
	if opts.LineStart > 0 {
		formatterOpts = append(formatterOpts, chromahtml.BaseLineNumber(opts.LineStart))
	}
	formatter := chromahtml.New(formatterOpts...)

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return "", fmt.Errorf("format %q code block: %w", language, err)
	}

	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	return out, nil
}

func normalizeLanguage(language string) string {
	language = strings.TrimSpace(language)
	if language == "" {
		return ""
	}

	return strings.ToLower(strings.Fields(language)[0])
}

func renderPlain(code, language string) string {
	classAttr := ""
	if language != "" {
		classAttr = ` class="language-` + stdhtml.EscapeString(normalizeLanguage(language)) + `"`
	}

	return "<pre><code" + classAttr + ">" + stdhtml.EscapeString(code) + "</code></pre>\n"
}
