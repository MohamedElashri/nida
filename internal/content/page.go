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
	Summary        string
	TOC            []Heading
	WordCount      int
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
	SourcePath    string
	RelativePath  string
	SectionPath   string
	BodyMarkdown  string
	BodyHTML      string
	TOC           []Heading
	Title         string
	Description   string
	Slug          string
	URL           string
	Draft         bool
	Weight        int
	Template      string
	PageTemplate  string
	PaginateBy    int
	PaginatePath  string
	SortBy        string
	Transparent   bool
	GenerateFeeds bool
	Sections      []Section
	Pages         []Page
	Extra         map[string]any
}

// Heading is one entry in a page's table of contents. Children holds the
// headings nested under it.
type Heading struct {
	Level    int
	ID       string
	Title    string
	Children []Heading
}
