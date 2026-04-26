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
	processed, err := processShortcodes(source, cfg)
	if err != nil {
		return "", err
	}
	return renderMarkdownCore(processed, cfg)
}

func renderMarkdownCore(source string, cfg config.SiteConfig) (string, error) {
	engine := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(&fencedCodeRenderer{theme: cfg.SyntaxTheme}, 500),
				util.Prioritized(&linkRenderer{cfg: cfg.Markdown}, 600),
				util.Prioritized(&imageRenderer{}, 700),
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

type linkRenderer struct {
	cfg config.MarkdownConfig
}

func (r *linkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
}

func (r *linkRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		destination := string(n.Destination)
		if _, err := w.WriteString(`<a href="` + string(util.EscapeHTML(n.Destination)) + `"`); err != nil {
			return ast.WalkStop, err
		}
		if isExternalLink(destination) {
			if r.cfg.ExternalLinksTargetBlank {
				if _, err := w.WriteString(` target="_blank"`); err != nil {
					return ast.WalkStop, err
				}
			}
			rel := externalLinkRel(r.cfg)
			if rel != "" {
				if _, err := w.WriteString(` rel="` + rel + `"`); err != nil {
					return ast.WalkStop, err
				}
			}
		}
		if n.Title != nil {
			if _, err := w.WriteString(` title="` + string(util.EscapeHTML(n.Title)) + `"`); err != nil {
				return ast.WalkStop, err
			}
		}
		if _, err := w.WriteString(">"); err != nil {
			return ast.WalkStop, err
		}
		return ast.WalkContinue, nil
	}
	if _, err := w.WriteString("</a>"); err != nil {
		return ast.WalkStop, err
	}
	return ast.WalkContinue, nil
}

func isExternalLink(destination string) bool {
	destination = strings.ToLower(strings.TrimSpace(destination))
	return strings.HasPrefix(destination, "http://") || strings.HasPrefix(destination, "https://")
}

func externalLinkRel(cfg config.MarkdownConfig) string {
	parts := make([]string, 0, 2)
	if cfg.ExternalLinksNoFollow {
		parts = append(parts, "nofollow")
	}
	if cfg.ExternalLinksNoReferrer {
		parts = append(parts, "noreferrer")
	}
	if cfg.ExternalLinksTargetBlank && cfg.ExternalLinksNoReferrer {
		parts = append(parts, "noopener")
	}
	return strings.Join(parts, " ")
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

type imageRenderer struct{}

func (r *imageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *imageRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Image)
	_, _ = w.WriteString(`<img src="` + string(util.EscapeHTML(n.Destination)) + `" alt="` + string(util.EscapeHTML(n.Text(source))) + `" loading="lazy" decoding="async"`)
	if n.Title != nil {
		_, _ = w.WriteString(` title="` + string(util.EscapeHTML(n.Title)) + `"`)
	}
	_, _ = w.WriteString(">")
	return ast.WalkSkipChildren, nil
}
