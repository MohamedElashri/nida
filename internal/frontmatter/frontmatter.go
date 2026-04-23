package frontmatter

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	tomlDelimiter = "+++"
	yamlDelimiter = "---"
)

func Parse(input []byte) (Document, error) {
	raw, format, body, err := Split(input)
	if err != nil {
		return Document{}, err
	}

	var values map[string]any
	switch format {
	case tomlDelimiter:
		if err := toml.Unmarshal([]byte(raw), &values); err != nil {
			return Document{}, fmt.Errorf("parse TOML front matter: %w", err)
		}
	case yamlDelimiter:
		parsed, err := parseSimpleYAML(raw)
		if err != nil {
			return Document{}, fmt.Errorf("parse YAML front matter: %w", err)
		}
		values = parsed
	default:
		return Document{}, fmt.Errorf("unsupported front matter delimiter %q", format)
	}

	meta, err := decodeMetadata(values)
	if err != nil {
		return Document{}, fmt.Errorf("parse front matter: %w", err)
	}

	return Document{
		RawFrontMatter: raw,
		BodyMarkdown:   body,
		Metadata:       meta,
	}, nil
}

func Split(input []byte) (string, string, string, error) {
	normalized := bytes.ReplaceAll(input, []byte("\r\n"), []byte("\n"))
	text := string(normalized)
	text = strings.TrimLeft(text, "\n")

	delimiter := ""
	switch {
	case strings.HasPrefix(text, tomlDelimiter+"\n"):
		delimiter = tomlDelimiter
	case strings.HasPrefix(text, yamlDelimiter+"\n"):
		delimiter = yamlDelimiter
	default:
		return "", "", "", fmt.Errorf("expected front matter opening delimiter %q or %q", tomlDelimiter, yamlDelimiter)
	}

	remaining := text[len(delimiter)+1:]
	closeIndex := strings.Index(remaining, "\n"+delimiter+"\n")
	if closeIndex == -1 {
		if strings.HasSuffix(remaining, "\n"+delimiter) {
			raw := remaining[:len(remaining)-len("\n"+delimiter)]
			return raw, delimiter, "", nil
		}
		return "", "", "", fmt.Errorf("missing front matter closing delimiter %q", delimiter)
	}

	raw := remaining[:closeIndex]
	body := remaining[closeIndex+len("\n"+delimiter+"\n"):]
	return raw, delimiter, body, nil
}

func parseSimpleYAML(raw string) (map[string]any, error) {
	values := map[string]any{}
	for lineNum, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key: value", lineNum+1)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: key is required", lineNum+1)
		}
		values[key] = parseYAMLScalar(strings.TrimSpace(value))
	}
	return values, nil
}

func parseYAMLScalar(value string) any {
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
		if inner == "" {
			return []any{}
		}
		parts := strings.Split(inner, ",")
		items := make([]any, 0, len(parts))
		for _, part := range parts {
			items = append(items, parseYAMLScalar(strings.TrimSpace(part)))
		}
		return items
	}
	if unquoted, err := strconv.Unquote(value); err == nil {
		return unquoted
	}
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	default:
		return strings.Trim(value, `'"`)
	}
}
