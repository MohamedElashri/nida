package migrate

import (
	"bytes"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	yaml "go.yaml.in/yaml/v3"
)

// Format is a front matter format.
type Format int

const (
	// NoFrontMatter is a file without front matter.
	NoFrontMatter Format = iota
	TOML
	YAML
	// JSON front matter (Hugo) is converted to YAML on parse, since Nida
	// reads only TOML and YAML.
	JSON
)

// Document is a content file whose front matter can be edited without
// disturbing the rest of it. A document that is never edited is written
// back byte for byte, apart from a leading byte order mark, which Nida's
// parser does not accept.
type Document struct {
	Format Format
	// Body is everything after the front matter.
	Body string
	// BodyLine is the 1-based line in the original file where Body starts,
	// for reporting positions inside the body.
	BodyLine int

	original []byte
	crlf     bool
	changed  bool

	// Front matter state for the current format.
	values   map[string]any // decoded values, for reads
	yamlNode *yaml.Node     // mapping node, for YAML edits
	tomlText string         // raw TOML, for line-level edits
}

var (
	bomPrefix   = []byte("\xef\xbb\xbf")
	tomlHeader  = regexp.MustCompile(`^\s*\[`)
	// jsonFrontMatter requires a key or an empty object after the brace,
	// so a body that starts with a Liquid tag such as {% include %} is not
	// mistaken for Hugo's JSON front matter.
	jsonFrontMatter = regexp.MustCompile(`^\{\s*("|\})`)
	tomlKeyLine = regexp.MustCompile(`^\s*("[^"]*"|'[^']*'|[A-Za-z0-9_-]+)\s*=`)
)

// ParseDocument splits a content file into front matter and body. It
// accepts TOML (+++), YAML (---), and JSON ({ ... }) front matter, a
// leading byte order mark, CRLF line endings, and trailing spaces after the
// delimiters. A file without front matter parses with NoFrontMatter.
func ParseDocument(data []byte) (*Document, error) {
	doc := &Document{original: data}
	text := data
	if bytes.HasPrefix(text, bomPrefix) {
		text = text[len(bomPrefix):]
		doc.changed = true
	}
	doc.crlf = bytes.Contains(text, []byte("\r\n"))
	s := strings.ReplaceAll(string(text), "\r\n", "\n")

	// Count blank lines before the front matter; Nida tolerates them.
	leading := len(s) - len(strings.TrimLeft(s, "\n"))
	s = s[leading:]
	lineOffset := leading

	firstLine, rest, _ := strings.Cut(s, "\n")
	switch delim := strings.TrimRight(firstLine, " \t"); {
	case delim == "+++" || delim == "---":
		raw, body, closeLine, ok := cutClosing(rest, delim)
		if !ok {
			return nil, fmt.Errorf("missing closing %q front matter delimiter", delim)
		}
		doc.Body = body
		doc.BodyLine = lineOffset + closeLine + 3
		if firstLine != delim {
			doc.changed = true // trailing spaces after the delimiter
		}
		if delim == "+++" {
			return doc, doc.initTOML(raw)
		}
		return doc, doc.initYAML(raw)
	case jsonFrontMatter.MatchString(s):
		raw, body, lines, err := cutJSON(s)
		if err != nil {
			return nil, err
		}
		doc.Body = body
		doc.BodyLine = lineOffset + lines + 1
		if err := doc.initYAML(raw); err != nil {
			return nil, fmt.Errorf("parse JSON front matter: %w", err)
		}
		blockStyle(doc.yamlNode)
		doc.Format = JSON
		doc.changed = true
		return doc, nil
	}
	doc.Body = strings.ReplaceAll(string(text), "\r\n", "\n")
	doc.BodyLine = 1
	doc.values = map[string]any{}
	return doc, nil
}

// NewDocument returns a document with empty front matter in the given
// format, for files that are built from scratch.
func NewDocument(format Format, body string) *Document {
	doc := &Document{Body: body, BodyLine: 1, changed: true}
	if format == TOML {
		_ = doc.initTOML("")
	} else {
		_ = doc.initYAML("")
	}
	return doc
}

// cutClosing finds the closing delimiter line, allowing trailing spaces.
func cutClosing(rest, delim string) (raw, body string, closeLine int, ok bool) {
	lines := strings.SplitAfter(rest, "\n")
	offset := 0
	for i, line := range lines {
		if strings.TrimRight(line, " \t\n") == delim {
			return rest[:offset], rest[offset+len(line):], i, true
		}
		offset += len(line)
	}
	return "", "", 0, false
}

// cutJSON returns the JSON object at the start of s and the text after it.
func cutJSON(s string) (raw, body string, lines int, err error) {
	depth, inString, escaped := 0, false, false
	for i, r := range s {
		switch {
		case escaped:
			escaped = false
		case inString && r == '\\':
			escaped = true
		case r == '"':
			inString = !inString
		case inString:
		case r == '{':
			depth++
		case r == '}':
			depth--
			if depth == 0 {
				raw = s[:i+1]
				body = strings.TrimPrefix(s[i+1:], "\n")
				return raw, body, strings.Count(raw, "\n") + 1, nil
			}
		}
	}
	return "", "", 0, fmt.Errorf("unterminated JSON front matter")
}

// blockStyle clears flow and quoting styles, so JSON front matter is
// written as ordinary block YAML. The encoder still quotes values that
// would otherwise change type.
func blockStyle(n *yaml.Node) {
	n.Style &^= yaml.FlowStyle | yaml.DoubleQuotedStyle
	for _, c := range n.Content {
		blockStyle(c)
	}
}

func (d *Document) initTOML(raw string) error {
	d.Format = TOML
	d.tomlText = raw
	return d.decodeTOML()
}

func (d *Document) decodeTOML() error {
	values := map[string]any{}
	if err := toml.Unmarshal([]byte(d.tomlText), &values); err != nil {
		return fmt.Errorf("parse TOML front matter: %w", err)
	}
	d.values = values
	return nil
}

func (d *Document) initYAML(raw string) error {
	d.Format = YAML
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &root); err != nil {
		return fmt.Errorf("parse YAML front matter: %w", err)
	}
	switch {
	case root.Kind == 0:
		d.yamlNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	case root.Kind == yaml.DocumentNode && len(root.Content) == 1 && root.Content[0].Kind == yaml.MappingNode:
		d.yamlNode = root.Content[0]
	case root.Kind == yaml.DocumentNode && len(root.Content) == 1 && root.Content[0].Tag == "!!null":
		d.yamlNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	default:
		return fmt.Errorf("parse YAML front matter: expected a mapping")
	}
	return d.decodeYAML()
}

func (d *Document) decodeYAML() error {
	values := map[string]any{}
	if err := d.yamlNode.Decode(&values); err != nil {
		return fmt.Errorf("parse YAML front matter: %w", err)
	}
	d.values = values
	return nil
}

// Changed reports whether the document differs from the file it was
// parsed from.
func (d *Document) Changed() bool { return d.changed }

// Values returns the decoded front matter. Callers must not modify it.
func (d *Document) Values() map[string]any { return d.values }

// Get returns a top-level front matter value.
func (d *Document) Get(key string) (any, bool) {
	v, ok := d.values[key]
	return v, ok
}

// String returns a top-level value as a string, or "" if it is missing
// or not a string.
func (d *Document) String(key string) string {
	s, _ := d.values[key].(string)
	return s
}

// Has reports whether a top-level key is set.
func (d *Document) Has(key string) bool {
	_, ok := d.values[key]
	return ok
}

// Keys returns the top-level keys in file order.
func (d *Document) Keys() []string {
	switch d.Format {
	case YAML, JSON:
		var keys []string
		for i := 0; i+1 < len(d.yamlNode.Content); i += 2 {
			keys = append(keys, d.yamlNode.Content[i].Value)
		}
		return keys
	case TOML:
		var keys []string
		for _, line := range strings.Split(d.tomlText, "\n") {
			if tomlHeader.MatchString(line) {
				break
			}
			if m := tomlKeyLine.FindStringSubmatch(line); m != nil {
				keys = append(keys, strings.Trim(m[1], `"'`))
			}
		}
		for key := range d.values {
			if !slices.Contains(keys, key) {
				keys = append(keys, key) // tables, in map order
			}
		}
		return keys
	}
	return nil
}

// Set adds or replaces a top-level key. A document without front matter
// gains TOML front matter.
func (d *Document) Set(key string, value any) error {
	if d.Format == NoFrontMatter {
		if err := d.initTOML(""); err != nil {
			return err
		}
	}
	d.changed = true
	switch d.Format {
	case YAML, JSON:
		var node yaml.Node
		if err := node.Encode(value); err != nil {
			return fmt.Errorf("encode %s: %w", key, err)
		}
		if i := d.yamlIndex(key); i >= 0 {
			node.HeadComment = d.yamlNode.Content[i+1].HeadComment
			node.LineComment = d.yamlNode.Content[i+1].LineComment
			d.yamlNode.Content[i+1] = &node
		} else {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
			d.yamlNode.Content = append(d.yamlNode.Content, keyNode, &node)
		}
		return d.decodeYAML()
	default:
		return d.setTOML(key, value)
	}
}

// Delete removes a top-level key and reports whether it was present.
func (d *Document) Delete(key string) bool {
	if !d.Has(key) {
		return false
	}
	switch d.Format {
	case YAML, JSON:
		i := d.yamlIndex(key)
		if i < 0 {
			return false // the key comes from a merge (<<: *anchor)
		}
		d.changed = true
		d.yamlNode.Content = slices.Delete(d.yamlNode.Content, i, i+2)
		_ = d.decodeYAML()
	case TOML:
		d.changed = true
		before, beforeValues := d.tomlText, maps.Clone(d.values)
		want := maps.Clone(d.values)
		delete(want, key)
		if start, end, ok := d.tomlExtent(key); ok {
			lines := strings.Split(d.tomlText, "\n")
			d.tomlText = strings.Join(slices.Delete(lines, start, end), "\n")
		} else {
			d.tomlText = deleteTOMLTable(d.tomlText, key)
		}
		if d.tomlEditKept(want) {
			return true
		}
		d.tomlText, d.values = before, beforeValues
		// Re-encoding values that were just decoded cannot fail.
		_ = d.reencodeTOML(func(values map[string]any) { delete(values, key) })
	}
	return true
}

// tomlTableHeader matches [name], [name.sub], [[name]], and [[name.sub]].
var tomlTableHeader = regexp.MustCompile(`^\s*\[\[?\s*("[^"]*"|'[^']*'|[A-Za-z0-9_-]+)\s*[.\]]`)

// deleteTOMLTable removes every table section belonging to key, from its
// header to the next header of another table, and the blank lines before
// it, keeping everything else as written.
func deleteTOMLTable(text, key string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inside := false
	for _, line := range lines {
		if tomlHeader.MatchString(line) {
			m := tomlTableHeader.FindStringSubmatch(line)
			wasInside := inside
			inside = m != nil && strings.Trim(m[1], `"'`) == key
			if inside {
				out = trimTrailingBlank(out)
			} else if wasInside && len(out) > 0 {
				out = append(out, "") // keep a blank line between tables
			}
		}
		if !inside {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// Rename moves a top-level key to a new name, keeping its position and
// comments where the format allows. It does nothing if from is missing,
// and returns false if to is already set, leaving both.
func (d *Document) Rename(from, to string) bool {
	if !d.Has(from) || d.Has(to) {
		return false
	}
	switch d.Format {
	case YAML, JSON:
		i := d.yamlIndex(from)
		if i < 0 {
			// The key comes from a merge (<<: *anchor), so there is no node
			// to rename; set the new key alongside it instead.
			return d.Set(to, d.values[from]) == nil
		}
		d.changed = true
		d.yamlNode.Content[i].Value = to
		_ = d.decodeYAML()
		return true
	case TOML:
		d.changed = true
		before, beforeValues := d.tomlText, maps.Clone(d.values)
		want := maps.Clone(d.values)
		want[to] = want[from]
		delete(want, from)
		if start, _, ok := d.tomlExtent(from); ok {
			lines := strings.Split(d.tomlText, "\n")
			loc := tomlKeyLine.FindStringSubmatchIndex(lines[start])
			lines[start] = lines[start][:loc[2]] + tomlKey(to) + lines[start][loc[3]:]
			d.tomlText = strings.Join(lines, "\n")
			if d.tomlEditKept(want) {
				return true
			}
		}
		d.tomlText, d.values = before, beforeValues
		value := d.values[from]
		_ = d.reencodeTOML(func(values map[string]any) {
			delete(values, from)
			values[to] = value
		})
		return true
	}
	return false
}

func (d *Document) yamlIndex(key string) int {
	for i := 0; i+1 < len(d.yamlNode.Content); i += 2 {
		if d.yamlNode.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// setTOML replaces the lines of a top-level key, or inserts it at the end
// of the top-level keys, before the first table.
func (d *Document) setTOML(key string, value any) error {
	var text string
	if _, isMap := value.(map[string]any); isMap {
		encoded, err := toml.Marshal(map[string]any{key: value})
		if err != nil {
			return fmt.Errorf("encode %s: %w", key, err)
		}
		text = strings.TrimRight(string(encoded), "\n")
	} else {
		inline, err := inlineValue(value)
		if err != nil {
			return fmt.Errorf("encode %s: %w", key, err)
		}
		text = tomlKey(key) + " = " + inline
	}
	// A table value marshals as "[key]" with its own lines; it has to go
	// after every plain key, so append it to the end of the document.
	isTable := strings.HasPrefix(text, "[")
	before, beforeValues := d.tomlText, maps.Clone(d.values)
	want := maps.Clone(d.values)
	want[key] = roundTripTOML(key, value)

	lines := strings.Split(d.tomlText, "\n")
	if start, end, ok := d.tomlExtent(key); ok && !isTable {
		lines = slices.Replace(lines, start, end, text)
	} else {
		if ok {
			lines = slices.Delete(lines, start, end)
		} else if _, isTableNow := d.values[key].(map[string]any); isTableNow {
			// An existing table cannot be edited line by line.
			return d.reencodeTOML(func(values map[string]any) { values[key] = value })
		}
		if isTable {
			lines = append(trimTrailingBlank(lines), "", text)
		} else {
			at := len(lines)
			for i, line := range lines {
				if tomlHeader.MatchString(line) {
					at = i
					break
				}
			}
			for at > 0 && strings.TrimSpace(lines[at-1]) == "" {
				at--
			}
			lines = slices.Insert(lines, at, text)
		}
	}
	d.tomlText = strings.Trim(strings.Join(lines, "\n"), "\n")
	if !d.tomlEditKept(want) {
		// The line edit broke the document or changed more than the key,
		// such as a line inside a multi-line string that looked like a
		// key or table header. Fall back to re-encoding.
		d.tomlText, d.values = before, beforeValues
		return d.reencodeTOML(func(values map[string]any) { values[key] = value })
	}
	return nil
}

// tomlEditKept decodes the edited TOML and reports whether it holds
// exactly the wanted values. Parsing alone is not enough: a regex-located
// edit can land inside a multi-line string and still parse.
func (d *Document) tomlEditKept(want map[string]any) bool {
	return d.decodeTOML() == nil && reflect.DeepEqual(d.values, want)
}

// roundTripTOML returns value as TOML decodes it, so it can be compared
// with decoded front matter.
func roundTripTOML(key string, value any) any {
	encoded, err := toml.Marshal(map[string]any{key: value})
	if err != nil {
		return value
	}
	var decoded map[string]any
	if toml.Unmarshal(encoded, &decoded) != nil {
		return value
	}
	return decoded[key]
}

// tomlExtent finds the lines [start, end) holding a top-level key and its
// value, which may span lines for arrays and multi-line strings.
func (d *Document) tomlExtent(key string) (int, int, bool) {
	lines := strings.Split(d.tomlText, "\n")
	for i, line := range lines {
		if tomlHeader.MatchString(line) {
			return 0, 0, false
		}
		m := tomlKeyLine.FindStringSubmatch(line)
		if m == nil || strings.Trim(m[1], `"'`) != key {
			continue
		}
		for end := i + 1; end <= len(lines); end++ {
			var probe map[string]any
			if toml.Unmarshal([]byte(strings.Join(lines[i:end], "\n")), &probe) == nil {
				return i, end, true
			}
		}
		return 0, 0, false
	}
	return 0, 0, false
}

// reencodeTOML rewrites the whole front matter from its values. It loses
// comments and key order, so it is only a fallback.
func (d *Document) reencodeTOML(edit func(map[string]any)) error {
	values := map[string]any{}
	for k, v := range d.values {
		values[k] = v
	}
	edit(values)
	encoded, err := toml.Marshal(values)
	if err != nil {
		return fmt.Errorf("encode TOML front matter: %w", err)
	}
	d.tomlText = strings.TrimRight(string(encoded), "\n")
	return d.decodeTOML()
}

func tomlKey(key string) string {
	if regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(key) {
		return key
	}
	return fmt.Sprintf("%q", key)
}

func trimTrailingBlank(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// SetBody replaces the body.
func (d *Document) SetBody(body string) {
	if body != d.Body {
		d.Body = body
		d.changed = true
	}
}

// Bytes returns the file contents. Line endings follow the original file.
func (d *Document) Bytes() []byte {
	if !d.changed && d.original != nil {
		return d.original
	}
	var b strings.Builder
	switch d.Format {
	case TOML:
		b.WriteString("+++\n")
		// Nida needs a line between the delimiters, even an empty one.
		b.WriteString(strings.Trim(d.tomlText, "\n"))
		b.WriteString("\n")
		b.WriteString("+++\n")
	case YAML, JSON:
		b.WriteString("---\n")
		if len(d.yamlNode.Content) == 0 {
			b.WriteString("\n")
		} else {
			var buf bytes.Buffer
			enc := yaml.NewEncoder(&buf)
			enc.SetIndent(2)
			_ = enc.Encode(d.yamlNode)
			_ = enc.Close()
			b.WriteString(buf.String())
		}
		b.WriteString("---\n")
	}
	b.WriteString(d.Body)
	out := b.String()
	if d.crlf {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return []byte(out)
}
