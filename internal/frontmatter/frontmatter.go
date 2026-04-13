package frontmatter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const tomlDelimiter = "+++"

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
	default:
		return Document{}, fmt.Errorf("unsupported front matter delimiter %q", format)
	}

	meta, err := decodeMetadata(values)
	if err != nil {
		return Document{}, fmt.Errorf("parse TOML front matter: %w", err)
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
	default:
		return "", "", "", fmt.Errorf("expected front matter opening delimiter %q", tomlDelimiter)
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
