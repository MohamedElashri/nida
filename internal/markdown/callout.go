package markdown

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// calloutMarker matches GitHub's "> [!WARNING] Optional title" syntax.
var calloutMarker = regexp.MustCompile(`^\[!([A-Za-z]+)\][ \t]*(.*)$`)

var calloutTitles = map[string]string{
	"note":      "Note",
	"tip":       "Tip",
	"important": "Important",
	"warning":   "Warning",
	"caution":   "Caution",
}

var kindCallout = ast.NewNodeKind("Callout")

type calloutNode struct {
	ast.BaseBlock
	Variant string
	Title   string
}

func (n *calloutNode) Kind() ast.NodeKind { return kindCallout }

func (n *calloutNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Variant": n.Variant, "Title": n.Title}, nil)
}

type calloutTransformer struct {
	translate func(key string) (string, bool)
}

func (t calloutTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	var quotes []*ast.Blockquote
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if quote, ok := node.(*ast.Blockquote); ok && entering {
			quotes = append(quotes, quote)
		}
		return ast.WalkContinue, nil
	})
	for _, quote := range quotes {
		convertCallout(quote, source, t.translate)
	}
}

func convertCallout(quote *ast.Blockquote, source []byte, translate func(string) (string, bool)) {
	para, ok := quote.FirstChild().(*ast.Paragraph)
	if !ok || para.Lines().Len() == 0 {
		return
	}
	firstLine := para.Lines().At(0)
	match := calloutMarker.FindSubmatch(util.TrimRightSpace(firstLine.Value(source)))
	if match == nil {
		return
	}

	kind := strings.ToLower(string(match[1]))
	title := strings.TrimSpace(string(match[2]))
	if title == "" && translate != nil {
		title, _ = translate("callout." + kind)
	}
	if title == "" {
		title = calloutTitles[kind]
	}
	if title == "" {
		title = strings.ToUpper(kind[:1]) + kind[1:]
	}

	// Drop the marker line's inline nodes; an emptied paragraph is removed.
	lineEnd := firstLine.Stop
	for child := para.FirstChild(); child != nil; {
		next := child.NextSibling()
		endsLine := false
		if t, ok := child.(*ast.Text); ok {
			endsLine = t.SoftLineBreak() || t.HardLineBreak() || t.Segment.Stop >= lineEnd
		}
		para.RemoveChild(para, child)
		if endsLine {
			break
		}
		child = next
	}
	if para.FirstChild() == nil {
		quote.RemoveChild(quote, para)
	}

	callout := &calloutNode{Variant: kind, Title: title}
	for child := quote.FirstChild(); child != nil; {
		next := child.NextSibling()
		callout.AppendChild(callout, child)
		child = next
	}
	quote.Parent().ReplaceChild(quote.Parent(), quote, callout)
}

type calloutRenderer struct{}

func (calloutRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindCallout, renderCallout)
}

func renderCallout(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*calloutNode)
	if entering {
		kind := string(util.EscapeHTML([]byte(n.Variant)))
		_, _ = w.WriteString(`<div class="callout callout-` + kind + `" data-callout="` + kind + `">` + "\n")
		_, _ = w.WriteString(`<p class="callout-title">` + string(util.EscapeHTML([]byte(n.Title))) + "</p>\n")
		return ast.WalkContinue, nil
	}
	_, _ = w.WriteString("</div>\n")
	return ast.WalkContinue, nil
}
