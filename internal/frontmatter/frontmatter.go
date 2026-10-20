package frontmatter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
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
		if err := yaml.Unmarshal([]byte(raw), &values); err != nil {
			return Document{}, fmt.Errorf("parse YAML front matter: %w", err)
		}
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
	// Windows editors such as Notepad may start the file with a byte
	// order mark.
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.TrimLeft(text, "\n")

	var delimiter string
	switch {
	case strings.HasPrefix(text, tomlDelimiter+"\n"):
		delimiter = tomlDelimiter
	case strings.HasPrefix(text, yamlDelimiter+"\n"):
		delimiter = yamlDelimiter
	default:
		return "", "", "", fmt.Errorf("expected front matter opening delimiter %q or %q", tomlDelimiter, yamlDelimiter)
	}

	// Keep the newline after the opening delimiter, so an empty block's
	// closing delimiter is found too.
	remaining := text[len(delimiter):]
	closeIndex := strings.Index(remaining, "\n"+delimiter+"\n")
	if closeIndex == -1 {
		if strings.HasSuffix(remaining, "\n"+delimiter) {
			raw := remaining[1 : len(remaining)-len(delimiter)]
			return strings.TrimSuffix(raw, "\n"), delimiter, "", nil
		}
		return "", "", "", fmt.Errorf("missing front matter closing delimiter %q", delimiter)
	}

	raw := remaining[1:max(closeIndex, 1)]
	body := remaining[closeIndex+len("\n"+delimiter+"\n"):]
	return raw, delimiter, body, nil
}
