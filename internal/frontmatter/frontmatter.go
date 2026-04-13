package frontmatter

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const delimiter = "+++"

type Metadata struct {
	Title       string    `toml:"title"`
	Date        time.Time `toml:"date"`
	Draft       bool      `toml:"draft"`
	Tags        []string  `toml:"tags"`
	Categories  []string  `toml:"categories"`
	Description string    `toml:"description"`
	Slug        string    `toml:"slug"`
}

type Document struct {
	RawFrontMatter string
	BodyMarkdown   string
	Metadata       Metadata
}

func Parse(input []byte) (Document, error) {
	raw, body, err := Split(input)
	if err != nil {
		return Document{}, err
	}

	var meta Metadata
	if err := toml.Unmarshal([]byte(raw), &meta); err != nil {
		return Document{}, fmt.Errorf("parse TOML front matter: %w", err)
	}

	return Document{
		RawFrontMatter: raw,
		BodyMarkdown:   body,
		Metadata:       meta,
	}, nil
}

func Split(input []byte) (string, string, error) {
	normalized := bytes.ReplaceAll(input, []byte("\r\n"), []byte("\n"))
	text := string(normalized)

	if !strings.HasPrefix(text, delimiter+"\n") {
		return "", "", fmt.Errorf("expected TOML front matter opening delimiter %q", delimiter)
	}

	remaining := text[len(delimiter)+1:]
	closeIndex := strings.Index(remaining, "\n"+delimiter+"\n")
	if closeIndex == -1 {
		if strings.HasSuffix(remaining, "\n"+delimiter) {
			raw := remaining[:len(remaining)-len("\n"+delimiter)]
			return raw, "", nil
		}
		return "", "", fmt.Errorf("missing TOML front matter closing delimiter %q", delimiter)
	}

	raw := remaining[:closeIndex]
	body := remaining[closeIndex+len("\n"+delimiter+"\n"):]
	return raw, body, nil
}
