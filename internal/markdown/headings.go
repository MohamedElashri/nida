package markdown

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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

// collectTOC nests each heading under the closest preceding heading with a
// smaller level, so skipped levels still form a tree.
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
			Title: stripPlaceholders(string(extractTextFromInline(heading, source))),
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

// placeholderPattern matches the markers shortcodes leave in the Markdown
// source until their HTML is restored after rendering.
var placeholderPattern = regexp.MustCompile(`\s*@@NIDA_SHORTCODE_HTML_\d+@@`)

func stripPlaceholders(value string) string {
	return strings.TrimSpace(placeholderPattern.ReplaceAllString(value, ""))
}

// headingIDs follows goldmark's ID rules for ASCII but keeps Unicode letters,
// which goldmark drops, and ignores shortcode placeholders.
type headingIDs struct {
	values map[string]bool
}

func newHeadingIDs() *headingIDs {
	return &headingIDs{values: map[string]bool{}}
}

func (s *headingIDs) Generate(value []byte, kind ast.NodeKind) []byte {
	clean := stripPlaceholders(string(value))
	var b strings.Builder
	for _, r := range clean {
		switch {
		case r < utf8.RuneSelf && (r >= 'a' && r <= 'z' || r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 'a' - 'A')
		case r == ' ' || r == '\t' || r == '-' || r == '_':
			b.WriteByte('-')
		case r >= utf8.RuneSelf && (unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r)):
			b.WriteRune(unicode.ToLower(r))
		}
	}
	result := b.String()
	if result == "" {
		result = "id"
		if kind == ast.KindHeading {
			result = "heading"
		}
	}
	if !s.values[result] {
		s.values[result] = true
		return []byte(result)
	}
	for i := 1; ; i++ {
		candidate := result + "-" + strconv.Itoa(i)
		if !s.values[candidate] {
			s.values[candidate] = true
			return []byte(candidate)
		}
	}
}

func (s *headingIDs) Put(value []byte) {
	s.values[string(value)] = true
}
