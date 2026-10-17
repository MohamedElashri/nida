package markdown

import (
	"strconv"

	"github.com/MohamedElashri/nida/internal/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	renderhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// headingRenderer renders headings with their generated id and, when
// markdown.heading_anchors is enabled, a trailing self-link.
type headingRenderer struct {
	anchors bool
}

func (r *headingRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, r.renderHeading)
}

func (r *headingRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	tag := "h" + strconv.Itoa(n.Level)
	if entering {
		_, _ = w.WriteString("<" + tag)
		if n.Attributes() != nil {
			renderhtml.RenderAttributes(w, node, renderhtml.HeadingAttributeFilter)
		}
		_ = w.WriteByte('>')
		return ast.WalkContinue, nil
	}
	if r.anchors {
		if id := headingID(n); id != "" {
			escaped := string(util.EscapeHTML([]byte(id)))
			_, _ = w.WriteString(`<a class="heading-anchor" href="#` + escaped + `" aria-hidden="true">#</a>`)
		}
	}
	_, _ = w.WriteString("</" + tag + ">\n")
	return ast.WalkContinue, nil
}

func headingID(n *ast.Heading) string {
	value, ok := n.AttributeString("id")
	if !ok {
		return ""
	}
	if id, ok := value.([]byte); ok {
		return string(id)
	}
	if id, ok := value.(string); ok {
		return id
	}
	return ""
}

// collectTOC returns the document's headings nested by level. A heading
// becomes a child of the closest preceding heading with a smaller level, so
// documents that skip levels (h2 then h4) still produce a sensible tree.
func collectTOC(doc ast.Node, source []byte) []content.Heading {
	var flat []content.Heading
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		flat = append(flat, content.Heading{
			Level: heading.Level,
			ID:    headingID(heading),
			Title: string(extractTextFromInline(heading, source)),
		})
		return ast.WalkSkipChildren, nil
	})
	return nestHeadings(flat)
}

func nestHeadings(flat []content.Heading) []content.Heading {
	var roots []content.Heading
	// path holds pointers along the current branch, from root to leaf.
	var path []*content.Heading
	for _, heading := range flat {
		for len(path) > 0 && path[len(path)-1].Level >= heading.Level {
			path = path[:len(path)-1]
		}
		if len(path) == 0 {
			roots = append(roots, heading)
			path = append(path, &roots[len(roots)-1])
			continue
		}
		parent := path[len(path)-1]
		parent.Children = append(parent.Children, heading)
		path = append(path, &parent.Children[len(parent.Children)-1])
	}
	return roots
}
