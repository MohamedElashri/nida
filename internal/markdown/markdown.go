package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/highlight"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	renderhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

func Render(source string, cfg config.SiteConfig) (string, error) {
	engine := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(&fencedCodeRenderer{theme: cfg.SyntaxTheme}, 500),
			),
			renderhtml.WithHardWraps(),
			renderhtml.WithUnsafe(),
		),
	)

	var buf bytes.Buffer
	if err := engine.Convert([]byte(source), &buf); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}

	return buf.String(), nil
}

func RenderItem(item content.Item, cfg config.SiteConfig) (content.Item, error) {
	html, err := Render(item.BodyMarkdown, cfg)
	if err != nil {
		return content.Item{}, fmt.Errorf("render %q markdown: %w", item.RelativePath, err)
	}

	item.BodyHTML = html
	return item, nil
}

func RenderItems(items []content.Item, cfg config.SiteConfig) ([]content.Item, error) {
	rendered := make([]content.Item, 0, len(items))
	for _, item := range items {
		next, err := RenderItem(item, cfg)
		if err != nil {
			return nil, err
		}
		rendered = append(rendered, next)
	}

	return rendered, nil
}

type fencedCodeRenderer struct {
	theme string
}

func (r *fencedCodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func (r *fencedCodeRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.FencedCodeBlock)
	html, err := highlight.Render(blockText(source, n), string(n.Language(source)), r.theme)
	if err != nil {
		return ast.WalkStop, err
	}

	if _, err := w.WriteString(html); err != nil {
		return ast.WalkStop, err
	}

	return ast.WalkSkipChildren, nil
}

func blockText(source []byte, node ast.Node) string {
	var b strings.Builder
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		b.Write(line.Value(source))
	}
	return b.String()
}
