package templates

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"path"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/MohamedElashri/nida/internal/bundled"
	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/highlight"
	"github.com/MohamedElashri/nida/internal/markdown"
)

// dict builds a map from alternating keys and values, mainly so partials can
// receive several named arguments: {{ template "card" (dict "Page" . "Wide" true) }}.
func dict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict expects key/value pairs, got %d arguments", len(pairs))
	}
	out := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %T", pairs[i])
		}
		out[key] = pairs[i+1]
	}
	return out, nil
}

// merge returns a new map with the keys of b laid over a.
func merge(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func sliceOf(items ...any) []any {
	return items
}

// seq returns 1..n, for loops that need a counter.
func seq(n int) []int {
	if n <= 0 {
		return nil
	}
	out := make([]int, n)
	for i := range out {
		out[i] = i + 1
	}
	return out
}

func listValue(items any) (reflect.Value, error) {
	v := reflect.ValueOf(items)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return v, nil
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return reflect.Value{}, fmt.Errorf("expected a list, got %T", items)
	}
	return v, nil
}

// first returns at most n items from the start of a list.
func first(n int, items any) (any, error) {
	v, err := listValue(items)
	if err != nil || !v.IsValid() {
		return items, err
	}
	if n < 0 {
		return nil, fmt.Errorf("first: negative count %d", n)
	}
	return v.Slice(0, min(n, v.Len())).Interface(), nil
}

// last returns at most n items from the end of a list.
func last(n int, items any) (any, error) {
	v, err := listValue(items)
	if err != nil || !v.IsValid() {
		return items, err
	}
	if n < 0 {
		return nil, fmt.Errorf("last: negative count %d", n)
	}
	return v.Slice(max(v.Len()-n, 0), v.Len()).Interface(), nil
}

// where keeps items whose field path (such as "Extra.tags") equals, or for
// lists contains, the value. An optional "!=" operator inverts the match.
func where(items any, field string, args ...any) (any, error) {
	op := "=="
	var value any
	switch len(args) {
	case 1:
		value = args[0]
	case 2:
		opStr, ok := args[0].(string)
		if !ok || (opStr != "==" && opStr != "!=" && opStr != "=") {
			return nil, fmt.Errorf(`where: operator must be "==" or "!=", got %v`, args[0])
		}
		op, value = opStr, args[1]
	default:
		return nil, fmt.Errorf("where expects a value or an operator and a value")
	}

	v, err := listValue(items)
	if err != nil || !v.IsValid() {
		return items, err
	}
	out := reflect.MakeSlice(reflect.SliceOf(v.Type().Elem()), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		matched := matchesValue(fieldValue(item, field), value)
		if (op == "!=") != matched {
			out = reflect.Append(out, item)
		}
	}
	return out.Interface(), nil
}

func fieldValue(v reflect.Value, fieldPath string) reflect.Value {
	for _, name := range strings.Split(fieldPath, ".") {
		for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		switch v.Kind() {
		case reflect.Struct:
			v = v.FieldByName(name)
		case reflect.Map:
			if v.Type().Key().Kind() != reflect.String {
				return reflect.Value{}
			}
			v = v.MapIndex(reflect.ValueOf(name).Convert(v.Type().Key()))
		default:
			return reflect.Value{}
		}
		if !v.IsValid() {
			return v
		}
	}
	for v.IsValid() && v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func matchesValue(field reflect.Value, want any) bool {
	if !field.IsValid() {
		return want == nil
	}
	if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
		for i := 0; i < field.Len(); i++ {
			if matchesValue(field.Index(i), want) {
				return true
			}
		}
		return false
	}
	if field.CanInterface() && reflect.DeepEqual(field.Interface(), want) {
		return true
	}
	return fmt.Sprint(field.Interface()) == fmt.Sprint(want)
}

// truncate shortens text to at most n characters, cutting at a word
// boundary when possible and appending an ellipsis.
func truncate(n int, value string) string {
	if n <= 0 || utf8.RuneCountInString(value) <= n {
		return value
	}
	runes := []rune(value)
	cut := n
	for i := n; i > n/2; i-- {
		if unicode.IsSpace(runes[i]) {
			cut = i
			break
		}
	}
	return strings.TrimRightFunc(string(runes[:cut]), unicode.IsSpace) + "…"
}

// markdownifyFunc renders a short Markdown string, unwrapping a single
// paragraph so the result can sit inline.
func markdownifyFunc(cfg config.SiteConfig) func(string) (template.HTML, error) {
	return func(value string) (template.HTML, error) {
		html, err := markdown.Render(value, cfg, nil)
		if err != nil {
			return "", err
		}
		html = strings.TrimSpace(html)
		if strings.HasPrefix(html, "<p>") && strings.HasSuffix(html, "</p>") && strings.Count(html, "<p>") == 1 {
			html = strings.TrimSuffix(strings.TrimPrefix(html, "<p>"), "</p>")
		}
		return template.HTML(html), nil
	}
}

// absURLFunc joins a site-relative path onto base_url.
func absURLFunc(cfg config.SiteConfig) func(string) string {
	return func(value string) string {
		if isAbsoluteURL(value) {
			return value
		}
		base, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
		if err != nil {
			return value
		}
		joined := path.Join("/", base.Path, value)
		if strings.HasSuffix(value, "/") && joined != "/" {
			joined += "/"
		}
		base.Path = joined
		return base.String()
	}
}

// relURLFunc prefixes a site-relative path with the base_url path, which
// keeps links working when a site is published under a subpath.
func relURLFunc(cfg config.SiteConfig) func(string) string {
	basePath := config.BasePath(cfg.BaseURL)
	return func(value string) string {
		if isAbsoluteURL(value) {
			return value
		}
		joined := path.Join("/", basePath, value)
		if strings.HasSuffix(value, "/") && joined != "/" {
			joined += "/"
		}
		return joined
	}
}

func isAbsoluteURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme != "" || strings.HasPrefix(value, "//"))
}

// jsonify encodes a value as JSON for use in scripts, such as JSON-LD.
// encoding/json escapes <, >, and &, so the output is safe in a script tag.
func jsonify(value any) (template.JS, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return template.JS(data), nil
}

// syntaxCSSFunc returns highlighting CSS for syntax_theme, or for the style
// named in the optional argument, such as syntaxCSS "github-dark".
func syntaxCSSFunc(cfg config.SiteConfig) func(...string) (template.CSS, error) {
	return func(names ...string) (template.CSS, error) {
		theme := cfg.SyntaxTheme
		if len(names) > 0 && strings.TrimSpace(names[0]) != "" {
			theme = names[0]
		}
		css, err := highlight.CSS(theme)
		return template.CSS(css), err
	}
}

// bundledScriptsFunc returns tags for the bundled files a site needs, today
// the math stylesheet. It is site-wide: with markdown.math on, every page
// gets the stylesheet, so pages need no per-page detection. Arguments are
// accepted for compatibility with templates that pass the page; the string
// "math" loads the stylesheet even while markdown.math is off.
func bundledScriptsFunc(cfg config.SiteConfig) func(...any) template.HTML {
	basePath := config.BasePath(cfg.BaseURL)
	return func(sources ...any) template.HTML {
		forced := false
		for _, source := range sources {
			if name, ok := source.(string); ok && name == "math" {
				forced = true
			}
		}
		if !cfg.Markdown.Math && !forced {
			return ""
		}
		math, ok := bundled.Math(cfg.Markdown.MathFont)
		if !ok {
			return ""
		}
		return math.Tags(basePath)
	}
}
