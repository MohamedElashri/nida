package render

import (
	"html/template"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/site"
	"github.com/MohamedElashri/nida/internal/taxonomies"
)

type Page struct {
	URL          string
	CanonicalURL string
	TemplateName string
	Title        string
	Content      string
}

type NavItem struct {
	Name string
	URL  string
}

type Favicon struct {
	Webmanifest    string
	Favicon16x16   string
	Favicon32x32   string
	AppleTouchIcon string
}

type Umami struct {
	Enabled   bool
	Src       string
	WebsiteID string
}

// Theme carries values derived from the merged theme and site [extra]
// tables. Extra is the general-purpose map; the named fields below predate
// it and are kept so existing themes continue to work.
type Theme struct {
	Extra      map[string]any
	InlineCSS  template.CSS
	MainMenu   []NavItem
	Social     []NavItem
	FooterText string
	DateFormat string
	AuthorName string
	Favicon    Favicon
	Umami      Umami
}

type PageLink struct {
	Number  int
	URL     string
	Current bool
}

type Paginator struct {
	CurrentIndex int
	NumberPagers int
	Previous     string
	Next         string
	PageLinks    []PageLink
	Pages        []content.Page
}

type templateContext struct {
	Title        string
	Description  string
	BasePath     string
	HomeURL      string
	CurrentURL   string
	CanonicalURL string
	Config       config.SiteConfig
	Theme        Theme
	Index        site.SiteIndex
	Page         content.Page
	Section      content.Section
	Pages        []content.Page
	Terms        []taxonomies.Term
	Taxonomy     taxonomies.Collection
	Term         taxonomies.Term
	Paginator    *Paginator
	Robots       string
}
