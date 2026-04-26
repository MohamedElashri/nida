package frontmatter

import "time"

type Metadata struct {
	Title          string
	Description   string
	Date          time.Time
	Updated       time.Time
	Draft         bool
	Weight        int
	Slug          string
	Template      string
	PageTemplate  string
	SortBy        string
	Transparent   bool
	GenerateFeeds bool
	PaginateBy    int
	PaginatePath  string
	Extra         map[string]any
}

type Document struct {
	RawFrontMatter string
	BodyMarkdown   string
	Metadata       Metadata
}
