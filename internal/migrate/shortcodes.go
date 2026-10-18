package migrate

import (
	"fmt"
	"slices"
	"strings"
)

// builtinShortcodes are the shortcodes Nida ships, which never get a
// placeholder.
var builtinShortcodes = []string{"figure", "youtube", "callout", "rawhtml", "details"}

// Arg is one shortcode argument. An empty Key is a positional argument.
type Arg struct {
	Key   string
	Value string
}

// Shortcode formats an opening Nida shortcode tag, such as
// {{< figure src="a.png" >}}.
func Shortcode(name string, args []Arg) string {
	var b strings.Builder
	b.WriteString("{{< ")
	b.WriteString(name)
	for _, arg := range args {
		b.WriteByte(' ')
		if arg.Key != "" {
			b.WriteString(arg.Key)
			b.WriteByte('=')
		}
		b.WriteString(quoteArg(arg.Value))
	}
	b.WriteString(" >}}")
	return b.String()
}

// ClosingShortcode formats a closing tag, such as {{< /callout >}}.
func ClosingShortcode(name string) string {
	return "{{< /" + name + " >}}"
}

func quoteArg(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

// EscapedShortcode returns text that renders as the literal shortcode
// source, for generators' escaped shortcode syntax. Nida does not process
// a shortcode whose first brace is an HTML entity.
func EscapedShortcode(inner string) string {
	return "&#123;{< " + strings.TrimSpace(inner) + " >}}"
}

// Placeholder makes sure a shortcode Nida does not have yet renders a
// visible "needs a template" box with its inner content, instead of
// failing the build, and reports the place it is used. Templates added
// with Plan.Template replace placeholders when they are used.
func (p *Plan) Placeholder(name, file string, line int) {
	if slices.Contains(builtinShortcodes, name) {
		return
	}
	p.Report.Add(Manual, file, line, "Shortcode `%s` has no Nida template yet, so it renders a placeholder box. Write `templates/shortcodes/%s.html`; the original is in `%s/`.", name, name, preservedDir)
	if p.placeholders == nil {
		p.placeholders = map[string][]byte{}
	}
	p.placeholders["templates/shortcodes/"+name+".html"] = []byte(fmt.Sprintf(placeholderTemplate, name, name))
}

const placeholderTemplate = `<div class="nida-migrate-todo" style="border: 2px dashed #c0392b; padding: 0.75rem; margin: 1rem 0;">
  <p><strong>Migration TODO:</strong> the <code>%s</code> shortcode needs a Nida template in <code>templates/shortcodes/%s.html</code>.</p>
  {{ .InnerHTML }}
</div>
`
