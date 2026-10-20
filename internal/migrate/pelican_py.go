package migrate

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// pyNone is Python's None in values read by readPyConfig.
type pyNone struct{}

// pyConfig holds the top-level assignments of a Pelican settings file.
// The file is never executed: only literal values are read. A name bound
// to anything else (a function call, a comprehension, an f-string) is
// recorded in bad with the reason, so it can be reported.
type pyConfig struct {
	values map[string]any
	order  []string
	bad    map[string]string
	// imports lists modules pulled in with "from X import *".
	imports []string
}

func newPyConfig() *pyConfig {
	return &pyConfig{values: map[string]any{}, bad: map[string]string{}}
}

func (c *pyConfig) set(name string, value any) {
	if _, ok := c.values[name]; !ok {
		c.order = append(c.order, name)
	}
	delete(c.bad, name)
	c.values[name] = value
}

func (c *pyConfig) setBad(name, reason string) {
	if _, ok := c.values[name]; ok {
		delete(c.values, name)
	} else if _, ok := c.bad[name]; !ok {
		c.order = append(c.order, name)
	}
	c.bad[name] = reason
}

// merge applies other's assignments on top of c, as a later file does.
func (c *pyConfig) merge(other *pyConfig) {
	for _, name := range other.order {
		if reason, ok := other.bad[name]; ok {
			c.setBad(name, reason)
		} else {
			c.set(name, other.values[name])
		}
	}
}

// readPyConfig reads the top-level literal assignments of a Python file.
// base supplies names defined before the file runs, such as those from
// "from pelicanconf import *", so expressions like SITEURL + "/x" work.
func readPyConfig(src string, base *pyConfig) *pyConfig {
	cfg := newPyConfig()
	env := func(name string) (any, bool) {
		if v, ok := cfg.values[name]; ok {
			return v, true
		}
		if _, bad := cfg.bad[name]; bad {
			return nil, false
		}
		if base != nil {
			v, ok := base.values[name]
			return v, ok
		}
		return nil, false
	}
	for _, stmt := range pyStatements(src) {
		if stmt.indented {
			continue // inside an if, def, for, or class block
		}
		text := strings.TrimSpace(stmt.text)
		if rest, ok := strings.CutPrefix(text, "from "); ok {
			if module, star, found := strings.Cut(rest, " import "); found && strings.TrimSpace(star) == "*" {
				cfg.imports = append(cfg.imports, strings.TrimSpace(module))
			}
			continue
		}
		name, expr, ok := splitPyAssignment(text)
		if !ok {
			continue
		}
		p := &pyParser{src: expr, env: env}
		value, err := p.parseExpr()
		if err == nil {
			p.skipSpace()
			if p.pos < len(p.src) {
				err = fmt.Errorf("unexpected %q", truncate(p.src[p.pos:], 20))
			}
		}
		if err != nil {
			cfg.setBad(name, err.Error())
			continue
		}
		cfg.set(name, value)
	}
	return cfg
}

// splitPyAssignment splits "NAME = expr" and "NAME: type = expr".
func splitPyAssignment(stmt string) (string, string, bool) {
	i := 0
	for i < len(stmt) && (stmt[i] == '_' || unicode.IsLetter(rune(stmt[i])) || (i > 0 && unicode.IsDigit(rune(stmt[i])))) {
		i++
	}
	if i == 0 {
		return "", "", false
	}
	name := stmt[:i]
	rest := strings.TrimLeft(stmt[i:], " \t")
	if strings.HasPrefix(rest, ":") {
		eq := strings.Index(rest, "=")
		if eq < 0 {
			return "", "", false
		}
		rest = rest[eq:]
	}
	if !strings.HasPrefix(rest, "=") || strings.HasPrefix(rest, "==") {
		return "", "", false
	}
	return name, strings.TrimSpace(rest[1:]), true
}

type pyStatement struct {
	text     string
	indented bool
}

// pyStatements splits Python source into logical lines: brackets and
// backslashes continue a line, comments are dropped, and triple-quoted
// strings may span lines.
func pyStatements(src string) []pyStatement {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	var out []pyStatement
	var cur strings.Builder
	depth := 0
	indented := false
	atLineStart := true
	for i := 0; i < len(src); i++ {
		c := src[i]
		if atLineStart && cur.Len() == 0 && depth == 0 {
			indented = c == ' ' || c == '\t'
			if c == '\n' {
				continue
			}
			atLineStart = false
		}
		switch {
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			i--
		case c == '"' || c == '\'':
			end := pyStringEnd(src, i)
			cur.WriteString(src[i:end])
			i = end - 1
		case c == '(' || c == '[' || c == '{':
			depth++
			cur.WriteByte(c)
		case c == ')' || c == ']' || c == '}':
			if depth > 0 {
				depth--
			}
			cur.WriteByte(c)
		case c == '\\' && i+1 < len(src) && src[i+1] == '\n':
			i++
			cur.WriteByte(' ')
		case c == '\n' || c == ';':
			if depth > 0 {
				cur.WriteByte(' ')
				continue
			}
			if s := strings.TrimSpace(cur.String()); s != "" {
				out = append(out, pyStatement{text: s, indented: indented})
			}
			cur.Reset()
			if c == '\n' {
				atLineStart = true
			}
		default:
			cur.WriteByte(c)
		}
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		out = append(out, pyStatement{text: s, indented: indented})
	}
	return out
}

// pyStringEnd returns the index just past the string literal at i.
func pyStringEnd(src string, i int) int {
	q := src[i]
	if strings.HasPrefix(src[i:], strings.Repeat(string(q), 3)) {
		end := strings.Index(src[i+3:], strings.Repeat(string(q), 3))
		if end < 0 {
			return len(src)
		}
		return i + 3 + end + 3
	}
	for j := i + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case q, '\n':
			return j + 1
		}
	}
	return len(src)
}

type pyParser struct {
	src string
	pos int
	env func(string) (any, bool)
}

func (p *pyParser) skipSpace() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t' || p.src[p.pos] == '\n') {
		p.pos++
	}
}

func (p *pyParser) peek() byte {
	p.skipSpace()
	if p.pos < len(p.src) {
		return p.src[p.pos]
	}
	return 0
}

// parseExpr handles a + b (strings, lists, numbers) and tuples without
// parentheses, as in X = 'a', 'b'.
func (p *pyParser) parseExpr() (any, error) {
	first, err := p.parseSum()
	if err != nil {
		return nil, err
	}
	if p.peek() != ',' {
		return first, nil
	}
	items := []any{first}
	for p.peek() == ',' {
		p.pos++
		if c := p.peek(); c == 0 || c == ')' {
			break
		}
		item, err := p.parseSum()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (p *pyParser) parseSum() (any, error) {
	left, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	for p.peek() == '+' {
		p.pos++
		right, err := p.parseAtom()
		if err != nil {
			return nil, err
		}
		if left, err = pyAdd(left, right); err != nil {
			return nil, err
		}
	}
	return left, nil
}

func pyAdd(a, b any) (any, error) {
	switch x := a.(type) {
	case string:
		if y, ok := b.(string); ok {
			return x + y, nil
		}
	case []any:
		if y, ok := b.([]any); ok {
			return append(append([]any{}, x...), y...), nil
		}
	case int64:
		switch y := b.(type) {
		case int64:
			return x + y, nil
		case float64:
			return float64(x) + y, nil
		}
	case float64:
		switch y := b.(type) {
		case int64:
			return x + float64(y), nil
		case float64:
			return x + y, nil
		}
	}
	return nil, fmt.Errorf("cannot add %T and %T", a, b)
}

func (p *pyParser) parseAtom() (any, error) {
	c := p.peek()
	switch {
	case c == 0:
		return nil, fmt.Errorf("missing value")
	case c == '[':
		p.pos++
		return p.parseSeq(']')
	case c == '(':
		p.pos++
		return p.parseSeq(')')
	case c == '{':
		p.pos++
		return p.parseDict()
	case c == '"' || c == '\'':
		return p.parseStrings("")
	case c == '-' || c == '.' || (c >= '0' && c <= '9'):
		return p.parseNumber()
	}
	start := p.pos
	for p.pos < len(p.src) && (p.src[p.pos] == '_' || p.src[p.pos] == '.' || unicode.IsLetter(rune(p.src[p.pos])) || unicode.IsDigit(rune(p.src[p.pos]))) {
		p.pos++
	}
	word := p.src[start:p.pos]
	if word == "" {
		return nil, fmt.Errorf("unexpected %q", truncate(p.src[start:], 20))
	}
	// String prefixes: r'', u'', b'', f''.
	if p.pos < len(p.src) && (p.src[p.pos] == '"' || p.src[p.pos] == '\'') && len(word) <= 2 {
		return p.parseStrings(strings.ToLower(word))
	}
	if p.peek() == '(' {
		return nil, fmt.Errorf("calls %s()", word)
	}
	switch word {
	case "True":
		return true, nil
	case "False":
		return false, nil
	case "None":
		return pyNone{}, nil
	}
	if v, ok := p.env(word); ok {
		return v, nil
	}
	return nil, fmt.Errorf("uses %s, which is not a literal setting", word)
}

func (p *pyParser) parseSeq(closer byte) (any, error) {
	items := []any{}
	for {
		if p.peek() == closer {
			p.pos++
			return items, nil
		}
		item, err := p.parseSum()
		if err != nil {
			return nil, err
		}
		if p.peek() == 'f' && strings.HasPrefix(p.src[p.pos:], "for ") {
			return nil, fmt.Errorf("is a comprehension")
		}
		items = append(items, item)
		switch p.peek() {
		case ',':
			p.pos++
		case closer:
		default:
			return nil, fmt.Errorf("expected , or %c", closer)
		}
	}
}

func (p *pyParser) parseDict() (any, error) {
	m := map[string]any{}
	for {
		if p.peek() == '}' {
			p.pos++
			return m, nil
		}
		key, err := p.parseSum()
		if err != nil {
			return nil, err
		}
		if p.peek() != ':' {
			return nil, fmt.Errorf("is a set or malformed dict")
		}
		p.pos++
		value, err := p.parseSum()
		if err != nil {
			return nil, err
		}
		m[fmt.Sprint(key)] = value
		switch p.peek() {
		case ',':
			p.pos++
		case '}':
		default:
			return nil, fmt.Errorf("expected , or }")
		}
	}
}

// parseStrings reads one or more adjacent string literals, which Python
// concatenates.
func (p *pyParser) parseStrings(prefix string) (any, error) {
	var b strings.Builder
	for {
		if strings.Contains(prefix, "f") {
			return nil, fmt.Errorf("is an f-string")
		}
		if strings.Contains(prefix, "b") {
			return nil, fmt.Errorf("is a bytes literal")
		}
		s, err := p.parseString(strings.Contains(prefix, "r"))
		if err != nil {
			return nil, err
		}
		b.WriteString(s)
		// Another literal follows, possibly with a prefix.
		save := p.pos
		c := p.peek()
		prefix = ""
		if c != '"' && c != '\'' {
			j := p.pos
			for j < len(p.src) && j-p.pos < 2 && strings.ContainsRune("rRuUfFbB", rune(p.src[j])) {
				j++
			}
			if j > p.pos && j < len(p.src) && (p.src[j] == '"' || p.src[j] == '\'') {
				prefix = strings.ToLower(p.src[p.pos:j])
				p.pos = j
			} else {
				p.pos = save
				return b.String(), nil
			}
		}
	}
}

func (p *pyParser) parseString(raw bool) (string, error) {
	q := p.src[p.pos]
	triple := strings.HasPrefix(p.src[p.pos:], strings.Repeat(string(q), 3))
	delim := string(q)
	if triple {
		delim = strings.Repeat(string(q), 3)
	}
	p.pos += len(delim)
	var b strings.Builder
	for p.pos < len(p.src) {
		if strings.HasPrefix(p.src[p.pos:], delim) {
			p.pos += len(delim)
			return b.String(), nil
		}
		c := p.src[p.pos]
		if c == '\\' && p.pos+1 < len(p.src) {
			next := p.src[p.pos+1]
			if raw {
				b.WriteByte(c)
				b.WriteByte(next)
				p.pos += 2
				continue
			}
			p.pos += 2
			switch next {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\', '\'', '"':
				b.WriteByte(next)
			case '\n':
			default:
				b.WriteByte('\\')
				b.WriteByte(next)
			}
			continue
		}
		b.WriteByte(c)
		p.pos++
	}
	return "", fmt.Errorf("unterminated string")
}

func (p *pyParser) parseNumber() (any, error) {
	start := p.pos
	if p.src[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.src) && strings.ContainsRune("0123456789._eExXabcdefABCDEF+-", rune(p.src[p.pos])) {
		if (p.src[p.pos] == '+' || p.src[p.pos] == '-') && p.pos > start && p.src[p.pos-1] != 'e' && p.src[p.pos-1] != 'E' {
			break
		}
		p.pos++
	}
	text := strings.ReplaceAll(p.src[start:p.pos], "_", "")
	if n, err := strconv.ParseInt(text, 0, 64); err == nil {
		return n, nil
	}
	if f, err := strconv.ParseFloat(text, 64); err == nil {
		return f, nil
	}
	return nil, fmt.Errorf("bad number %q", text)
}

// pyString returns a value as a string, or "" for anything else.
func pyString(v any) string {
	s, _ := v.(string)
	return s
}

// pyTruthy follows Python truthiness for literal values.
func pyTruthy(v any) bool {
	switch x := v.(type) {
	case nil, pyNone:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int64:
		return x != 0
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

// tomlSafe converts a Python value into one go-toml can encode: None
// becomes an empty string and tuples stay lists.
func tomlSafe(v any) any {
	switch x := v.(type) {
	case pyNone:
		return ""
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = tomlSafe(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			out[k] = tomlSafe(item)
		}
		return out
	}
	return v
}
