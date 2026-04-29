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

func Render(source string, cfg config.SiteConfig, pathLookup PathLookup) (string, error) {
	processed, err := processShortcodes(source, cfg, pathLookup)
	if err != nil {
		return "", err
	}
	return renderMarkdownCore(processed, cfg, pathLookup)
}

func renderMarkdownCore(source string, cfg config.SiteConfig, pathLookup PathLookup) (string, error) {
	engine := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(&fencedCodeRenderer{theme: cfg.SyntaxTheme}, 500),
				util.Prioritized(&linkRenderer{cfg: cfg.Markdown, pathLookup: pathLookup}, 600),
				util.Prioritized(&imageRenderer{pathLookup: pathLookup}, 700),
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
	cfg        config.MarkdownConfig
	pathLookup PathLookup
}

func (r *linkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
}

func (r *linkRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		destination := string(n.Destination)
		resolved := ResolveInternalPath(destination, r.pathLookup)
		if _, err := w.WriteString(`<a href="` + string(util.EscapeHTML([]byte(resolved))) + `"`); err != nil {
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

func RenderItem(item content.Page, cfg config.SiteConfig, pathLookup PathLookup) (content.Page, error) {
	html, err := Render(item.BodyMarkdown, cfg, pathLookup)
	if err != nil {
		return content.Page{}, fmt.Errorf("render %q markdown: %w", item.RelativePath, err)
	}

	item.BodyHTML = html
	item.ReadingTime = readingTime(item.BodyMarkdown)
	return item, nil
}

func readingTime(markdown string) int {
	words := len(strings.Fields(markdown))
	minutes := words / 200
	if minutes < 1 {
		return 1
	}
	return minutes
}

func RenderItems(items []content.Page, cfg config.SiteConfig, pathLookup PathLookup) ([]content.Page, error) {
	rendered := make([]content.Page, 0, len(items))
	for _, item := range items {
		next, err := RenderItem(item, cfg, pathLookup)
		if err != nil {
			return nil, err
		}
		rendered = append(rendered, next)
	}

	return rendered, nil
}

func RenderPages(pages []content.Page, cfg config.SiteConfig, pathLookup PathLookup) ([]content.Page, error) {
	return RenderItems(pages, cfg, pathLookup)
}

func RenderSections(sections []content.Section, cfg config.SiteConfig, pathLookup PathLookup) ([]content.Section, error) {
	rendered := make([]content.Section, len(sections))
	for i, s := range sections {
		if s.BodyMarkdown == "" {
			rendered[i] = s
			continue
		}
		html, err := Render(s.BodyMarkdown, cfg, pathLookup)
		if err != nil {
			return nil, fmt.Errorf("render section %q markdown: %w", s.RelativePath, err)
		}
		s.BodyHTML = html
		rendered[i] = s
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

type imageRenderer struct {
	pathLookup PathLookup
}

func (r *imageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *imageRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Image)
	dest := string(n.Destination)
	resolved := ResolveInternalPath(dest, r.pathLookup)
	_, _ = w.WriteString(`<img src="` + string(util.EscapeHTML([]byte(resolved))) + `" alt="` + string(util.EscapeHTML(n.Text(source))) + `" loading="lazy" decoding="async"`)
	if n.Title != nil {
		_, _ = w.WriteString(` title="` + string(util.EscapeHTML(n.Title)) + `"`)
	}
	_, _ = w.WriteString(">")
	return ast.WalkSkipChildren, nil
}
