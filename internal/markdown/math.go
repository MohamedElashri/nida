package markdown

import (
	"bytes"
	"fmt"

	"github.com/MohamedElashri/nida/internal/mathml"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Math reads $...$ and $$...$$ TeX without Markdown touching it, and renders
// it to MathML at build time with package mathml. With markdown.math off,
// TeX is an error, while dollar signs that do not look like TeX, such as
// "$HOME and $PATH", stay text.

var (
	kindMathInline = ast.NewNodeKind("MathInline")
	kindMathBlock  = ast.NewNodeKind("MathBlock")
)

type mathInline struct {
	ast.BaseInline
	Display bool
	TeX     []byte
}

func (n *mathInline) Kind() ast.NodeKind { return kindMathInline }

func (n *mathInline) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"TeX": string(n.TeX)}, nil)
}

type mathBlock struct {
	ast.BaseBlock
	TeX    bytes.Buffer
	closed bool
}

func (n *mathBlock) Kind() ast.NodeKind { return kindMathBlock }

func (n *mathBlock) IsRaw() bool { return true }

func (n *mathBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"TeX": n.TeX.String()}, nil)
}

type mathInlineParser struct{}

func (mathInlineParser) Trigger() []byte { return []byte{'$'} }

func (mathInlineParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 2 {
		return nil
	}

	if line[1] == '$' {
		end := bytes.Index(line[2:], []byte("$$"))
		if end <= 0 {
			return nil
		}
		tex := line[2 : 2+end]
		block.Advance(end + 4)
		return &mathInline{Display: true, TeX: append([]byte(nil), tex...)}
	}

	// Pandoc's rules, so "$5 and $10" stays text: no space after the opening
	// $, and no space before or digit after the closing $.
	if line[1] == ' ' || line[1] == '\t' {
		return nil
	}
	for i := 1; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '$':
			if line[i-1] == ' ' || line[i-1] == '\t' {
				return nil
			}
			if i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9' {
				return nil
			}
			tex := line[1:i]
			block.Advance(i + 1)
			return &mathInline{TeX: append([]byte(nil), tex...)}
		case '\n':
			return nil
		}
	}
	return nil
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }

func (mathBlockParser) Open(_ ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || !bytes.HasPrefix(line[pos:], []byte("$$")) {
		return nil, parser.NoChildren
	}
	rest := bytes.TrimSpace(line[pos+2:])
	node := &mathBlock{}
	if bytes.HasSuffix(rest, []byte("$$")) {
		// $$ ... $$ on one line.
		node.TeX.Write(bytes.TrimSpace(rest[:len(rest)-2]))
		node.closed = true
	} else if len(rest) > 0 {
		node.TeX.Write(rest)
		node.TeX.WriteByte('\n')
	}
	reader.AdvanceToEOL()
	return node, parser.NoChildren
}

func (mathBlockParser) Continue(node ast.Node, reader text.Reader, _ parser.Context) parser.State {
	n := node.(*mathBlock)
	if n.closed {
		return parser.Close
	}
	line, _ := reader.PeekLine()
	trimmed := bytes.TrimRight(line, " \t\r\n")
	if bytes.HasSuffix(trimmed, []byte("$$")) {
		n.TeX.Write(bytes.TrimSpace(trimmed[:len(trimmed)-2]))
		reader.AdvanceToEOL()
		return parser.Close
	}
	n.TeX.Write(line)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (mathBlockParser) Close(ast.Node, text.Reader, parser.Context) {}

func (mathBlockParser) CanInterruptParagraph() bool { return true }

func (mathBlockParser) CanAcceptIndentedLine() bool { return false }

type mathRenderer struct {
	enabled bool
}

func (r mathRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMathInline, r.renderInline)
	reg.Register(kindMathBlock, r.renderBlock)
}

func (r mathRenderer) renderInline(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*mathInline)
	if !r.enabled {
		if !n.Display && !looksLikeTeX(n.TeX) {
			_, _ = w.Write(util.EscapeHTML([]byte("$" + string(n.TeX) + "$")))
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkStop, mathOffError(n.TeX, n.Display)
	}
	class := "math math-inline"
	if n.Display {
		class = "math math-display"
	}
	return writeMath(w, "span", class, string(n.TeX), n.Display)
}

func (r mathRenderer) renderBlock(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*mathBlock)
	tex := bytes.TrimSpace(n.TeX.Bytes())
	if !r.enabled {
		return ast.WalkStop, mathOffError(tex, true)
	}
	status, err := writeMath(w, "div", "math math-display", string(tex), true)
	if err == nil {
		_ = w.WriteByte('\n')
	}
	return status, err
}

// looksLikeTeX reports whether $...$ text is TeX rather than prose that
// happens to sit between dollar signs.
func looksLikeTeX(tex []byte) bool {
	return bytes.ContainsAny(tex, `\^_{}`)
}

func mathOffError(tex []byte, display bool) error {
	delim := "$"
	if display {
		delim = "$$"
	}
	return fmt.Errorf("math %s%s%s found, but markdown.math is off: set math = true under [markdown] in config.toml, or write \\$ for a literal dollar sign", delim, bytes.TrimSpace(tex), delim)
}

func writeMath(w util.BufWriter, tag, class, tex string, display bool) (ast.WalkStatus, error) {
	out, err := mathml.Render(tex, display)
	if err != nil {
		return ast.WalkStop, err
	}
	if _, err := w.WriteString("<" + tag + ` class="` + class + `">` + out + "</" + tag + ">"); err != nil {
		return ast.WalkStop, err
	}
	return ast.WalkSkipChildren, nil
}
