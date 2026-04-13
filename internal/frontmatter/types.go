package frontmatter

import "time"

type Metadata struct {
	Title        string
	Date         time.Time
	Draft        bool
	Tags         []string
	Categories   []string
	Description  string
	Slug         string
	Template     string
	PageTemplate string
	PaginateBy   int
	Extra        map[string]any
}

type Document struct {
	RawFrontMatter string
	BodyMarkdown   string
	Metadata       Metadata
}
