package config

const ConfigVersion = "0.4"

type SiteConfig struct {
	ConfigVersion string              `toml:"config_version"`
	BaseURL       string              `toml:"base_url"`
	Title         string              `toml:"title"`
	Description   string              `toml:"description"`
	Language      string              `toml:"language"`
	Author        string              `toml:"author"`
	ContentDir    string              `toml:"content_dir"`
	TemplateDir   string              `toml:"template_dir"`
	StaticDir     string              `toml:"static_dir"`
	OutputDir     string              `toml:"output_dir"`
	Paginate      int                `toml:"paginate"`
	Drafts        bool               `toml:"drafts"`
	MinifyHTML    bool               `toml:"minify_html"`
	SyntaxTheme   string              `toml:"syntax_theme"`
	Markdown      MarkdownConfig     `toml:"markdown"`
	Taxonomies    []TaxonomyConfig    `toml:"taxonomies"`
	RSS           RSSConfig          `toml:"rss"`
	Atom          AtomConfig         `toml:"atom"`
	Sitemap       SitemapConfig      `toml:"sitemap"`
	Robots        RobotsConfig       `toml:"robots"`
	Server        ServerConfig       `toml:"server"`
	Permalinks    PermalinkConfig    `toml:"permalinks"`
	Sections      SectionConfig      `toml:"sections"`
	Pipeline      PipelineConfig     `toml:"pipeline"`
	Extra         map[string]any     `toml:"extra"`
}

type TaxonomyConfig struct {
	Name          string `toml:"name"`
	PaginateBy    int    `toml:"paginate_by"`
	PaginatePath  string `toml:"paginate_path"`
	Feed          bool   `toml:"feed"`
	Render        bool   `toml:"render"`
}

type MarkdownConfig struct {
	ExternalLinksTargetBlank bool `toml:"external_links_target_blank"`
	ExternalLinksNoFollow    bool `toml:"external_links_no_follow"`
	ExternalLinksNoReferrer   bool `toml:"external_links_no_referrer"`
}

type SectionConfig struct {
	DefaultPageTemplate string `toml:"default_page_template"`
	DefaultSortBy       string `toml:"default_sort_by"`
	PaginateBy          int    `toml:"paginate_by"`
	PaginatePath        string `toml:"paginate_path"`
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

type PermalinkConfig map[string]string

type PipelineConfig struct {
	Fingerprint bool          `toml:"fingerprint"`
	MinifyCSS   bool          `toml:"minify_css"`
	MinifyJS    bool          `toml:"minify_js"`
	Images      ImageConfig   `toml:"images"`
	SCSS        SCSSConfig    `toml:"scss"`
}

type ImageConfig struct {
	Enabled bool  `toml:"enabled"`
	Widths  []int `toml:"widths"`
	Quality int   `toml:"quality"`
}

type SCSSConfig struct {
	Enabled  bool   `toml:"enabled"`
	EntryDir string `toml:"entry_dir"`
}

type Options struct {
	SiteRoot string
	Path    string
}
