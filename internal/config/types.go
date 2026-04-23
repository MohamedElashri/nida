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
	SyntaxTheme string          `toml:"syntax_theme"`
	Taxonomies  TaxonomyConfig  `toml:"taxonomies"`
	RSS         RSSConfig       `toml:"rss"`
	Atom        AtomConfig      `toml:"atom"`
	Sitemap     SitemapConfig   `toml:"sitemap"`
	Server      ServerConfig    `toml:"server"`
	Permalinks  PermalinkConfig `toml:"permalinks"`
	Extra       map[string]any  `toml:"extra"`
}

type TaxonomyConfig struct {
	Tags       bool `toml:"tags"`
	Categories bool `toml:"categories"`
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

type Options struct {
	SiteRoot string
	Path     string
}
