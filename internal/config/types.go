package config

type SiteConfig struct {
	BaseURL     string          `toml:"base_url"`
	Title       string          `toml:"title"`
	Description string          `toml:"description"`
	Language    string          `toml:"language"`
	Author      string          `toml:"author"`
	ContentDir  string          `toml:"content_dir"`
	TemplateDir string          `toml:"template_dir"`
	StaticDir   string          `toml:"static_dir"`
	OutputDir   string          `toml:"output_dir"`
	PostsDir    string          `toml:"posts_dir"`
	PagesDir    string          `toml:"pages_dir"`
	Paginate    int             `toml:"paginate"`
	Drafts      bool            `toml:"drafts"`
	MinifyHTML  bool            `toml:"minify_html"`
	SyntaxTheme string          `toml:"syntax_theme"`
	Markdown    MarkdownConfig  `toml:"markdown"`
	Taxonomies  TaxonomyConfig  `toml:"taxonomies"`
	RSS         RSSConfig       `toml:"rss"`
	Atom        AtomConfig      `toml:"atom"`
	Sitemap     SitemapConfig   `toml:"sitemap"`
	Robots      RobotsConfig    `toml:"robots"`
	Server      ServerConfig    `toml:"server"`
	Permalinks  PermalinkConfig `toml:"permalinks"`
	Pipeline    PipelineConfig  `toml:"pipeline"`
	Extra       map[string]any  `toml:"extra"`
}

type TaxonomyConfig struct {
	Tags       bool `toml:"tags"`
	Categories bool `toml:"categories"`
}

type MarkdownConfig struct {
	ExternalLinksTargetBlank bool `toml:"external_links_target_blank"`
	ExternalLinksNoFollow    bool `toml:"external_links_no_follow"`
	ExternalLinksNoReferrer  bool `toml:"external_links_no_referrer"`
}

type RSSConfig struct {
	Enabled  bool   `toml:"enabled"`
	Filename string `toml:"filename"`
	Limit    int    `toml:"limit"`
}

type AtomConfig struct {
	Enabled  bool   `toml:"enabled"`
	Filename string `toml:"filename"`
	Limit    int    `toml:"limit"`
}

type SitemapConfig struct {
	Enabled  bool   `toml:"enabled"`
	Filename string `toml:"filename"`
}

type RobotsConfig struct {
	Enabled  bool   `toml:"enabled"`
	Filename string `toml:"filename"`
	Content  string `toml:"content"`
}

type ServerConfig struct {
	Host       string `toml:"host"`
	Port       int    `toml:"port"`
	Livereload bool   `toml:"livereload"`
}

type PermalinkConfig struct {
	Posts      string `toml:"posts"`
	Pages      string `toml:"pages"`
	Tags       string `toml:"tags"`
	Categories string `toml:"categories"`
}

type PipelineConfig struct {
	Fingerprint bool              `toml:"fingerprint"`
	MinifyCSS   bool              `toml:"minify_css"`
	MinifyJS    bool              `toml:"minify_js"`
	Images      ImageConfig       `toml:"images"`
	SCSS        SCSSConfig        `toml:"scss"`
}

type ImageConfig struct {
	Enabled bool    `toml:"enabled"`
	Widths  []int   `toml:"widths"`
	Quality int     `toml:"quality"`
}

type SCSSConfig struct {
	Enabled   bool   `toml:"enabled"`
	EntryDir  string `toml:"entry_dir"`
}

type Options struct {
	SiteRoot string
	Path     string
}
