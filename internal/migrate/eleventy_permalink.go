package migrate

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/MohamedElashri/nida/internal/content"
)

// eleventyPage is what Eleventy knows about a template while it renders
// its permalink: page.fileSlug, page.filePathStem, page.date, and the
// template's data.
type eleventyPage struct {
	inputRel string // path relative to the input directory, such as posts/2024-01-05-hi.md
	date     time.Time
	hasDate  bool
	data     map[string]any
}

var eleventyDatePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
var eleventyDateInName = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// eleventyStem is a file name without its template extension; .11ty.js
// counts as one extension.
func eleventyStem(name string) string {
	base := path.Base(name)
	if strings.HasSuffix(base, ".11ty.js") {
		return strings.TrimSuffix(base, ".11ty.js")
	}
	return strings.TrimSuffix(base, path.Ext(base))
}

// fileSlug follows Eleventy's page.fileSlug: the file name without its
// extension or a leading YYYY-MM-DD- date, or the parent directory's name
// for index files.
func (pg eleventyPage) fileSlug() string {
	stem := eleventyStem(pg.inputRel)
	if stem == "index" {
		dir := path.Dir(pg.inputRel)
		if dir == "." {
			return ""
		}
		stem = path.Base(dir)
	}
	if trimmed := eleventyDatePrefix.ReplaceAllString(stem, ""); trimmed != "" {
		return trimmed
	}
	return stem
}

// filePathStem follows Eleventy's page.filePathStem: the input-relative
// path without its extension, with a leading slash and dates kept.
func (pg eleventyPage) filePathStem() string {
	dir := path.Dir(pg.inputRel)
	stem := eleventyStem(pg.inputRel)
	if dir == "." {
		return "/" + stem
	}
	return "/" + dir + "/" + stem
}

// defaultURL is the URL Eleventy gives a template without a permalink:
// the path without extension as a directory, with index files standing
// for their directory. Dates and case in the file name are kept.
func (pg eleventyPage) defaultURL() string {
	stem := pg.filePathStem()
	if eleventyStem(pg.inputRel) == "index" {
		return strings.TrimSuffix(stem, "index")
	}
	return stem + "/"
}

var liquidOutput = regexp.MustCompile(`\{\{-?\s*(.*?)\s*-?\}\}`)

// errNotEvaluable means a template string uses something nida cannot
// work out without running Eleventy.
type errNotEvaluable struct{ expr string }

func (e errNotEvaluable) Error() string { return fmt.Sprintf("cannot evaluate `%s`", e.expr) }

// evalPermalink renders a permalink template string for a page. It knows
// the variables and filters that Eleventy permalinks use in practice:
// page.fileSlug, page.filePathStem, page.date, data keys such as title,
// and the slugify, slug, lower, date, url, and htmlDateString filters.
func evalPermalink(tmpl string, pg eleventyPage) (string, error) {
	if strings.Contains(tmpl, "{%") {
		return "", errNotEvaluable{tmpl}
	}
	var err error
	out := liquidOutput.ReplaceAllStringFunc(tmpl, func(m string) string {
		if err != nil {
			return ""
		}
		v, e := evalLiquidExpr(liquidOutput.FindStringSubmatch(m)[1], pg)
		if e != nil {
			err = e
			return ""
		}
		return v
	})
	if err != nil {
		return "", err
	}
	return out, nil
}

// evalLiquidExpr evaluates `value | filter: arg | filter("arg")`.
func evalLiquidExpr(expr string, pg eleventyPage) (string, error) {
	parts := splitFilters(expr)
	value, isDate, err := lookupLiquid(strings.TrimSpace(parts[0]), pg)
	if err != nil {
		return "", err
	}
	for _, f := range parts[1:] {
		name, arg := parseFilter(f)
		switch name {
		case "slugify":
			value = eleventySlugify(value)
		case "slug":
			value = content.DeriveSlug(value)
		case "lower", "downcase":
			value = strings.ToLower(value)
		case "upper", "upcase":
			value = strings.ToUpper(value)
		case "url", "safe", "htmlBaseUrl", "strip", "trim":
			value = strings.TrimSpace(value)
		case "date", "dateFilter", "postDate", "dateFormat":
			if !isDate {
				return "", errNotEvaluable{expr}
			}
			layout, ok := strftimeLayout(arg)
			if !ok {
				return "", errNotEvaluable{expr}
			}
			value = pg.date.UTC().Format(layout)
			isDate = false
		case "htmlDateString":
			if !isDate {
				return "", errNotEvaluable{expr}
			}
			value = pg.date.UTC().Format("2006-01-02")
			isDate = false
		default:
			return "", errNotEvaluable{expr}
		}
	}
	if isDate {
		return "", errNotEvaluable{expr} // a raw date prints in a locale format
	}
	return value, nil
}

func lookupLiquid(name string, pg eleventyPage) (string, bool, error) {
	if s, ok := unquoteLiquid(name); ok {
		return s, false, nil
	}
	switch name {
	case "page.fileSlug":
		return pg.fileSlug(), false, nil
	case "page.filePathStem":
		return pg.filePathStem(), false, nil
	case "page.date", "date":
		if !pg.hasDate {
			return "", false, errNotEvaluable{name}
		}
		return "", true, nil
	}
	v, ok := lookupData(pg.data, name)
	if !ok {
		return "", false, errNotEvaluable{name}
	}
	switch x := v.(type) {
	case string:
		return x, false, nil
	case int, int64, float64, bool:
		return fmt.Sprint(x), false, nil
	}
	return "", false, errNotEvaluable{name}
}

// lookupData resolves a dotted path such as metadata.author.name.
func lookupData(data map[string]any, name string) (any, bool) {
	if !regexp.MustCompile(`^[A-Za-z_$][\w$]*(\.[A-Za-z_$][\w$]*)*$`).MatchString(name) {
		return nil, false
	}
	var cur any = data
	for _, key := range strings.Split(name, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = m[key]; !ok {
			return nil, false
		}
	}
	return cur, true
}

func unquoteLiquid(s string) (string, bool) {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1], true
	}
	return "", false
}

// splitFilters splits on | outside quotes.
func splitFilters(expr string) []string {
	var parts []string
	var quote byte
	start := 0
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '|':
			parts = append(parts, expr[start:i])
			start = i + 1
		}
	}
	return append(parts, expr[start:])
}

// parseFilter reads `name: "arg"` (Liquid) or `name("arg")` (Nunjucks).
func parseFilter(f string) (string, string) {
	f = strings.TrimSpace(f)
	if name, arg, ok := strings.Cut(f, ":"); ok {
		a, _ := unquoteLiquid(strings.TrimSpace(arg))
		return strings.TrimSpace(name), a
	}
	if open := strings.IndexByte(f, '('); open > 0 && strings.HasSuffix(f, ")") {
		a, _ := unquoteLiquid(strings.TrimSpace(f[open+1 : len(f)-1]))
		return strings.TrimSpace(f[:open]), a
	}
	return f, ""
}

// strftimeLayout converts a strftime format (Liquid's date filter) or a
// Luxon format (common Nunjucks date filters) to a Go layout. Only
// numeric date parts are supported.
func strftimeLayout(format string) (string, bool) {
	if format == "" {
		return "", false
	}
	replacer := strings.NewReplacer("%Y", "2006", "%m", "01", "%d", "02", "%y", "06", "%F", "2006-01-02",
		"yyyy", "2006", "LL", "01", "MM", "01", "dd", "02")
	layout := replacer.Replace(format)
	if strings.ContainsAny(layout, "%") || strings.ContainsAny(stripDigits(layout), "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return "", false
	}
	return layout, true
}

func stripDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return -1
		}
		return r
	}, s)
}

// eleventySlugify approximates the slugify filter (@sindresorhus/slugify):
// camelCase splits into words, and the result is lowercase and hyphenated.
// Its transliteration of non-ASCII letters is not reproduced; the URL check
// catches pages whose URL then differs.
func eleventySlugify(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			b.WriteByte('-')
		}
		if t, ok := latinTransliteration[unicode.ToLower(r)]; ok {
			b.WriteString(t)
			continue
		}
		b.WriteRune(r)
	}
	return content.DeriveSlug(b.String())
}

// latinTransliteration covers the accented Latin letters that
// @sindresorhus/transliterate replaces, with its German defaults.
var latinTransliteration = map[rune]string{
	'ä': "ae", 'ö': "oe", 'ü': "ue", 'ß': "ss",
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'å': "a", 'æ': "ae",
	'ç': "c", 'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ñ': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ø': "o", 'œ': "oe",
	'ù': "u", 'ú': "u", 'û': "u", 'ý': "y", 'ÿ': "y",
}

// permalinkPattern translates a permalink template shared by a directory
// into a Nida permalink pattern. slug, if not nil, returns the slug each
// page needs for {slug} to match. ok is false when the template uses
// anything without a Nida equivalent.
func permalinkPattern(tmpl string) (pattern string, slug func(eleventyPage) (string, bool), ok bool) {
	if strings.Contains(tmpl, "{%") {
		return "", nil, false
	}
	slugSource := ""
	failed := false
	out := liquidOutput.ReplaceAllStringFunc(tmpl, func(m string) string {
		expr := strings.TrimSpace(liquidOutput.FindStringSubmatch(m)[1])
		parts := splitFilters(expr)
		head := strings.TrimSpace(parts[0])
		filters := []string{}
		for _, f := range parts[1:] {
			name, arg := parseFilter(f)
			if name == "url" || name == "safe" {
				continue
			}
			filters = append(filters, name+"\x00"+arg)
		}
		switch {
		case head == "page.fileSlug" && (len(filters) == 0 || filters[0] == "slugify\x00" || filters[0] == "slug\x00") && len(filters) <= 1:
			if slugSource != "" && slugSource != expr {
				failed = true
			}
			slugSource = expr
			return "{slug}"
		case head == "title" && len(filters) == 1 && (filters[0] == "slugify\x00" || filters[0] == "slug\x00"):
			if slugSource != "" && slugSource != expr {
				failed = true
			}
			slugSource = expr
			return "{slug}"
		case (head == "page.date" || head == "date") && len(filters) == 1:
			name, arg, _ := strings.Cut(filters[0], "\x00")
			var layout string
			switch name {
			case "date", "dateFilter", "postDate", "dateFormat":
				var good bool
				if layout, good = strftimeLayout(arg); !good {
					failed = true
					return ""
				}
			case "htmlDateString":
				layout = "2006-01-02"
			default:
				failed = true
				return ""
			}
			r := strings.NewReplacer("2006", "{year}", "01", "{month}", "02", "{day}")
			if strings.Contains(layout, "06") && !strings.Contains(layout, "2006") {
				failed = true // two-digit years have no Nida placeholder
			}
			return r.Replace(layout)
		}
		failed = true
		return ""
	})
	if failed || strings.ContainsAny(strings.NewReplacer("{slug}", "", "{year}", "", "{month}", "", "{day}", "").Replace(out), "{}") {
		return "", nil, false
	}
	if slugSource != "" {
		src := slugSource
		slug = func(pg eleventyPage) (string, bool) {
			v, err := evalLiquidExpr(src, pg)
			return v, err == nil
		}
	}
	return out, slug, true
}
