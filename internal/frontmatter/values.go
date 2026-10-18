package frontmatter

import "strings"

// StringList reads a front matter value as a list of non-empty, trimmed
// strings. A single string becomes a one-item list.
func StringList(value any) []string {
	var raw []any
	switch v := value.(type) {
	case string:
		raw = []any{v}
	case []string:
		for _, s := range v {
			raw = append(raw, s)
		}
	case []any:
		raw = v
	}
	var out []string
	for _, item := range raw {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
