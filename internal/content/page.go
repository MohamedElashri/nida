package content

import (
	"time"
)

type Page struct {
	SourcePath     string
	RelativePath   string
	SectionPath    string
	BundleDir      string
	RawFrontMatter string
	BodyMarkdown   string
	BodyHTML       string
	Title          string
	Slug           string
	URL            string
	Description    string
	Date           time.Time
	Updated        time.Time
	Draft          bool
	Weight         int
	Template       string
	ReadingTime    int
	IsBundle       bool
	Resources      []string
	Extra          map[string]any
	Aliases        []string
	PrevURL        string
	PrevTitle      string
	NextURL        string
	NextTitle      string
}

type Section struct {
	SourcePath      string
	RelativePath    string
	SectionPath     string
	BodyMarkdown    string
	BodyHTML        string
	Title           string
	Description     string
	Slug            string
	URL             string
	Draft           bool
	Template        string
	PageTemplate    string
	PaginateBy      int
	PaginatePath    string
	PaginateReversed bool
	SortBy          string
	Transparent     bool
	GenerateFeeds   bool
	Sections        []Section
	Pages           []Page
	Extra           map[string]any
}
