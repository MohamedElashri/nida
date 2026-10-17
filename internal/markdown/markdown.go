package markdown

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/highlight"
	"github.com/MohamedElashri/nida/internal/parallel"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	renderhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Result is rendered Markdown plus the structure templates need from it.
type Result struct {
	HTML string
	TOC  []content.Heading
}

// Render renders a Markdown snippet without template shortcodes.
func Render(source string, cfg config.SiteConfig, pathLookup PathLookup) (string, error) {
	result, err := RenderDocument(source, cfg, Env{Lookup: pathLookup})
	return result.HTML, err
}

// RenderDocument renders Markdown and also returns its table of contents.
func RenderDocument(source string, cfg config.SiteConfig, env Env) (Result, error) {
	processed, err := processShortcodes(source, cfg, env)
	if err != nil {
		return Result{}, err
	}
	result, err := renderMarkdownCore(processed.Source, cfg, env)
	if err != nil {
		return Result{}, err
	}
	result.HTML = restoreShortcodeHTML(result.HTML, processed.Replacements)
	return result, nil
}

func newEngine(cfg config.SiteConfig, env Env) goldmark.Markdown {
	pathLookup := env.Lookup
	options := []renderer.Option{
		renderer.WithNodeRenderers(
			util.Prioritized(&fencedCodeRenderer{theme: cfg.SyntaxTheme}, 500),
			util.Prioritized(&linkRenderer{cfg: cfg.Markdown, pathLookup: pathLookup, basePath: config.BasePath(cfg.BaseURL)}, 600),
			util.Prioritized(&imageRenderer{pathLookup: pathLookup, basePath: config.BasePath(cfg.BaseURL), images: env.Images, page: env.Page}, 700),
			util.Prioritized(&headingRenderer{anchors: cfg.Markdown.HeadingAnchors}, 500),
			util.Prioritized(calloutRenderer{}, 500),
		),
	}
	if cfg.Markdown.HardWraps {
		options = append(options, renderhtml.WithHardWraps())
	}
	if cfg.Markdown.UnsafeHTML {
		options = append(options, renderhtml.WithUnsafe())
	}

	parserOptions := []parser.Option{
		parser.WithAutoHeadingID(),
		parser.WithAttribute(),
		parser.WithASTTransformers(util.Prioritized(calloutTransformer{translate: env.Translate}, 100)),
	}
	if cfg.Markdown.Math {
		parserOptions = append(parserOptions,
			parser.WithBlockParsers(util.Prioritized(mathBlockParser{}, 150)),
			parser.WithInlineParsers(util.Prioritized(mathInlineParser{}, 150)),
		)
		options = append(options, renderer.WithNodeRenderers(util.Prioritized(mathRenderer{}, 500)))
	}

	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(parserOptions...),
		goldmark.WithRendererOptions(options...),
	)
}

func renderMarkdownCore(source string, cfg config.SiteConfig, env Env) (Result, error) {
	engine := newEngine(cfg, env)
	src := []byte(source)
	doc := engine.Parser().Parse(text.NewReader(src))

	var buf bytes.Buffer
	if err := engine.Renderer().Render(&buf, src, doc); err != nil {
		return Result{}, fmt.Errorf("render markdown: %w", err)
	}

	return Result{HTML: buf.String(), TOC: collectTOC(doc, src)}, nil
}

type linkRenderer struct {
	cfg        config.MarkdownConfig
	pathLookup PathLookup
	basePath   string
}

func (r *linkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
}

func (r *linkRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		destination := string(n.Destination)
		resolved := resolveWithBasePath(destination, r.pathLookup, r.basePath)
		resolved = safeMarkdownURL(resolved, linkURL)
		if _, err := w.WriteString(`<a href="` + string(util.EscapeHTML([]byte(resolved))) + `"`); err != nil {
			return ast.WalkStop, err
		}
		if isExternalLink(resolved) {
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

type markdownURLKind int

const (
	linkURL markdownURLKind = iota
	imageURL
)

func safeMarkdownURL(destination string, kind markdownURLKind) string {
	trimmed := strings.TrimSpace(stdhtml.UnescapeString(destination))
	if trimmed == "" || containsControl(trimmed) {
		if kind == imageURL {
			return ""
		}
		return "#"
	}
	if strings.HasPrefix(trimmed, "//") {
		if kind == imageURL {
			return ""
		}
		return "#"
	}
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/") {
		return trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		if kind == imageURL {
			return ""
		}
		return "#"
	}
	if parsed.Scheme == "" {
		return trimmed
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return trimmed
	case "mailto", "tel":
		if kind == linkURL {
			return trimmed
		}
	}
	if kind == imageURL {
		return ""
	}
	return "#"
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
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

func RenderItem(item content.Page, cfg config.SiteConfig, env Env) (content.Page, error) {
	body, summarySource, hasMarker := splitSummary(item.BodyMarkdown)
	env.Page = &item

	result, err := RenderDocument(body, cfg, env)
	if err != nil {
		return content.Page{}, fmt.Errorf("render %q markdown: %w", item.RelativePath, err)
	}
	item.BodyHTML = result.HTML
	item.TOC = result.TOC

	if hasMarker {
		summaryResult, err := RenderDocument(summarySource, cfg, env)
		if err != nil {
			return content.Page{}, fmt.Errorf("render %q summary: %w", item.RelativePath, err)
		}
		item.Summary = summaryResult.HTML
	} else {
		item.Summary = firstParagraph(result.HTML)
	}

	item.ReadingTime = content.EstimateReadingTime(body)
	item.WordCount = content.CountWords(body)
	return item, nil
}

// RenderItems renders pages concurrently; results keep the input order.
func RenderItems(items []content.Page, cfg config.SiteConfig, env Env) ([]content.Page, error) {
	return parallel.Map(items, func(item content.Page) (content.Page, error) {
		return RenderItem(item, cfg, env)
	})
}

func RenderPages(pages []content.Page, cfg config.SiteConfig, env Env) ([]content.Page, error) {
	return RenderItems(pages, cfg, env)
}

func RenderSections(sections []content.Section, cfg config.SiteConfig, env Env) ([]content.Section, error) {
	rendered := make([]content.Section, len(sections))
	for i, s := range sections {
		if s.BodyMarkdown == "" {
			rendered[i] = s
			continue
		}
		result, err := RenderDocument(s.BodyMarkdown, cfg, env)
		if err != nil {
			return nil, fmt.Errorf("render section %q markdown: %w", s.RelativePath, err)
		}
		s.BodyHTML = result.HTML
		s.TOC = result.TOC
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
	code := blockText(source, n)
	isCollapsible, labels := extractCollapseMarker(code)
	if isCollapsible {
		code = stripCollapseMarker(code)
	}

	var info codeInfo
	if n.Info != nil {
		info = parseCodeInfo(string(n.Info.Value(source)))
	}

	if strings.EqualFold(info.Language, "mermaid") {
		// Mermaid reads the diagram source from the element text.
		_, err := w.WriteString(`<pre class="mermaid">` + stdhtml.EscapeString(code) + "</pre>\n")
		if err != nil {
			return ast.WalkStop, err
		}
		return ast.WalkSkipChildren, nil
	}

	// hl_lines count from the first line of the block even when linenostart
	// changes the displayed numbers.
	opts := highlight.Options{LineNumbers: info.LineNumbers, LineStart: info.LineStart}
	offset := max(info.LineStart-1, 0)
	for _, r := range info.HighlightLn {
		opts.HighlightLines = append(opts.HighlightLines, [2]int{r[0] + offset, r[1] + offset})
	}

	html, err := highlight.RenderWithOptions(code, info.Language, r.theme, opts)
	if err != nil {
		return ast.WalkStop, err
	}
	if info.Title != "" {
		html = `<figure class="code-block"><figcaption class="code-title">` + stdhtml.EscapeString(info.Title) + "</figcaption>" + html + "</figure>\n"
	}

	if isCollapsible {
		expandLabel := labels[0]
		collapseLabel := labels[1]
		if expandLabel == "" {
			expandLabel = "Show code"
		}
		if collapseLabel == "" {
			collapseLabel = "Hide code"
		}
		wrapped := `<div class="collapsible-code-wrapper collapsed" data-expand="` + stdhtml.EscapeString(expandLabel) + `" data-collapse="` + stdhtml.EscapeString(collapseLabel) + `"><button class="collapsible-code-toggle" aria-expanded="false"><span class="toggle-icon">▶</span><span class="toggle-label">` + stdhtml.EscapeString(expandLabel) + `</span></button><div class="collapsible-code-content">` + html + `</div></div>`
		if _, err := w.WriteString(wrapped); err != nil {
			return ast.WalkStop, err
		}
	} else {
		if _, err := w.WriteString(html); err != nil {
			return ast.WalkStop, err
		}
	}

	return ast.WalkSkipChildren, nil
}

var collapsePattern = regexp.MustCompile(`^\[!COLLAPSE\]\s*([^\n|]+)(?:\s*\|\s*([^\n]+))?`)

func extractCollapseMarker(code string) (bool, [2]string) {
	match := collapsePattern.FindStringSubmatch(code)
	if match == nil {
		return false, [2]string{}
	}
	labels := [2]string{strings.TrimSpace(match[1]), ""}
	if len(match) > 2 {
		labels[1] = strings.TrimSpace(match[2])
	}
	return true, labels
}

func stripCollapseMarker(code string) string {
	return collapsePattern.ReplaceAllString(code, "")
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
	basePath   string
	images     Images
	page       *content.Page
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
	resolved := resolveWithBasePath(dest, r.pathLookup, r.basePath)
	resolved = safeMarkdownURL(resolved, imageURL)
	altText := extractTextFromInline(n, source)
	_, _ = w.WriteString(`<img src="` + string(util.EscapeHTML([]byte(resolved))) + `" alt="` + string(util.EscapeHTML(altText)) + `"`)
	if r.images != nil && resolved != "" {
		if info, ok := r.images.Image(resolved, r.page); ok {
			if info.Srcset != "" {
				_, _ = w.WriteString(` srcset="` + string(util.EscapeHTML([]byte(info.Srcset))) + `"`)
				if info.Sizes != "" {
					_, _ = w.WriteString(` sizes="` + string(util.EscapeHTML([]byte(info.Sizes))) + `"`)
				}
			}
			_, _ = w.WriteString(` width="` + strconv.Itoa(info.Width) + `" height="` + strconv.Itoa(info.Height) + `"`)
		}
	}
	_, _ = w.WriteString(` loading="lazy" decoding="async"`)
	if n.Title != nil {
		_, _ = w.WriteString(` title="` + string(util.EscapeHTML(n.Title)) + `"`)
	}
	_, _ = w.WriteString(">")
	return ast.WalkSkipChildren, nil
}

func extractTextFromInline(n ast.Node, source []byte) []byte {
	var b []byte
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if text, ok := c.(*ast.Text); ok {
			b = append(b, text.Segment.Value(source)...)
		} else {
			b = append(b, extractTextFromInline(c, source)...)
		}
	}
	return b
}
