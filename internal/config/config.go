package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const DefaultConfigName = "config.toml"

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
	Sitemap     SitemapConfig   `toml:"sitemap"`
	Server      ServerConfig    `toml:"server"`
	Permalinks  PermalinkConfig `toml:"permalinks"`
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

func DefaultSiteConfig() SiteConfig {
	return SiteConfig{
		Language:    "en",
		ContentDir:  "content",
		TemplateDir: "templates",
		StaticDir:   "static",
		OutputDir:   "public",
		PostsDir:    "posts",
		PagesDir:    "pages",
		Paginate:    10,
		Drafts:      false,
		SyntaxTheme: "github",
		Taxonomies: TaxonomyConfig{
			Tags:       true,
			Categories: true,
		},
		RSS: RSSConfig{
			Enabled:  true,
			Filename: "rss.xml",
			Limit:    20,
		},
		Sitemap: SitemapConfig{
			Enabled:  true,
			Filename: "sitemap.xml",
		},
		Server: ServerConfig{
			Host:       "127.0.0.1",
			Port:       2906,
			Livereload: true,
		},
		Permalinks: PermalinkConfig{
			Posts:      "/posts/{slug}/",
			Pages:      "/{slug}/",
			Tags:       "/tags/{slug}/",
			Categories: "/categories/{slug}/",
		},
	}
}

func Load(opts Options) (SiteConfig, string, error) {
	siteRoot := opts.SiteRoot
	if siteRoot == "" {
		siteRoot = "."
	}

	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return SiteConfig{}, "", fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	configPath := opts.Path
	if configPath == "" {
		configPath = filepath.Join(absSiteRoot, DefaultConfigName)
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(absSiteRoot, configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SiteConfig{}, "", fmt.Errorf("load config %q: file does not exist", configPath)
		}
		return SiteConfig{}, "", fmt.Errorf("load config %q: %w", configPath, err)
	}

	cfg := DefaultSiteConfig()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return SiteConfig{}, "", fmt.Errorf("parse config %q: %w", configPath, err)
	}

	normalize(&cfg)
	if err := Validate(cfg); err != nil {
		return SiteConfig{}, "", fmt.Errorf("validate config %q: %w", configPath, err)
	}

	return cfg, configPath, nil
}

func Validate(cfg SiteConfig) error {
	var problems []string

	if strings.TrimSpace(cfg.BaseURL) == "" {
		problems = append(problems, "base_url is required")
	} else {
		parsed, err := url.Parse(cfg.BaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			problems = append(problems, "base_url must be an absolute URL")
		}
	}

	if strings.TrimSpace(cfg.Title) == "" {
		problems = append(problems, "title is required")
	}

	if cfg.Paginate <= 0 {
		problems = append(problems, "paginate must be greater than 0")
	}

	if cfg.RSS.Enabled && cfg.RSS.Limit <= 0 {
		problems = append(problems, "rss.limit must be greater than 0 when RSS is enabled")
	}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		problems = append(problems, "server.port must be between 1 and 65535")
	}

	requiredPaths := map[string]string{
		"content_dir":           cfg.ContentDir,
		"template_dir":          cfg.TemplateDir,
		"static_dir":            cfg.StaticDir,
		"output_dir":            cfg.OutputDir,
		"posts_dir":             cfg.PostsDir,
		"pages_dir":             cfg.PagesDir,
		"rss.filename":          cfg.RSS.Filename,
		"sitemap.filename":      cfg.Sitemap.Filename,
		"permalinks.posts":      cfg.Permalinks.Posts,
		"permalinks.pages":      cfg.Permalinks.Pages,
		"permalinks.tags":       cfg.Permalinks.Tags,
		"permalinks.categories": cfg.Permalinks.Categories,
	}

	for field, value := range requiredPaths {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, field+" must not be empty")
		}
	}

	if len(problems) == 0 {
		return nil
	}

	return errors.New(strings.Join(problems, "; "))
}

func normalize(cfg *SiteConfig) {
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Title = strings.TrimSpace(cfg.Title)
	cfg.Description = strings.TrimSpace(cfg.Description)
	cfg.Language = strings.TrimSpace(cfg.Language)
	cfg.Author = strings.TrimSpace(cfg.Author)
	cfg.ContentDir = cleanRelativePath(cfg.ContentDir)
	cfg.TemplateDir = cleanRelativePath(cfg.TemplateDir)
	cfg.StaticDir = cleanRelativePath(cfg.StaticDir)
	cfg.OutputDir = cleanRelativePath(cfg.OutputDir)
	cfg.PostsDir = cleanRelativePath(cfg.PostsDir)
	cfg.PagesDir = cleanRelativePath(cfg.PagesDir)
	cfg.SyntaxTheme = strings.TrimSpace(cfg.SyntaxTheme)
	cfg.RSS.Filename = cleanRelativePath(cfg.RSS.Filename)
	cfg.Sitemap.Filename = cleanRelativePath(cfg.Sitemap.Filename)
	cfg.Server.Host = strings.TrimSpace(cfg.Server.Host)
	cfg.Permalinks.Posts = normalizePermalink(cfg.Permalinks.Posts)
	cfg.Permalinks.Pages = normalizePermalink(cfg.Permalinks.Pages)
	cfg.Permalinks.Tags = normalizePermalink(cfg.Permalinks.Tags)
	cfg.Permalinks.Categories = normalizePermalink(cfg.Permalinks.Categories)
}

func cleanRelativePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	return filepath.Clean(value)
}

func normalizePermalink(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	if !strings.HasSuffix(value, "/") {
		value += "/"
	}
	return value
}

func DocumentDirection(language string) string {
	primary := strings.ToLower(strings.TrimSpace(language))
	if primary == "" {
		return "ltr"
	}

	if index := strings.IndexAny(primary, "-_"); index >= 0 {
		primary = primary[:index]
	}

	switch primary {
	case "ar", "fa", "he", "ur", "ps", "sd", "ug", "yi":
		return "rtl"
	default:
		return "ltr"
	}
}
