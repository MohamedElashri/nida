package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Math support passes TeX through untouched so a client-side renderer such as
// KaTeX or MathJax can typeset it. Without this, Markdown would turn
// underscores and asterisks inside formulas into emphasis.
//
// Inline math is $...$ or $$...$$ on one line. Display math is a block that
// opens with $$ and closes with a line ending in $$. Output uses the \( \)
// and \[ \] delimiters that KaTeX auto-render and MathJax look for.

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

	// Pandoc's rules keep prices like "$5 and $10" as plain text: the opening
	// $ must not be followed by a space, and the closing $ must not follow a
	// space or precede a digit.
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

type mathRenderer struct{}

func (mathRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMathInline, renderMathInline)
	reg.Register(kindMathBlock, renderMathBlock)
}

func renderMathInline(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*mathInline)
	if n.Display {
		_, _ = w.WriteString(`<span class="math math-display">\[`)
		_, _ = w.Write(util.EscapeHTML(n.TeX))
		_, _ = w.WriteString(`\]</span>`)
	} else {
		_, _ = w.WriteString(`<span class="math math-inline">\(`)
		_, _ = w.Write(util.EscapeHTML(n.TeX))
		_, _ = w.WriteString(`\)</span>`)
	}
	return ast.WalkSkipChildren, nil
}

func renderMathBlock(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*mathBlock)
	_, _ = w.WriteString(`<div class="math math-display">\[`)
	_, _ = w.Write(util.EscapeHTML(bytes.TrimSpace(n.TeX.Bytes())))
	_, _ = w.WriteString("\\]</div>\n")
	return ast.WalkSkipChildren, nil
}
