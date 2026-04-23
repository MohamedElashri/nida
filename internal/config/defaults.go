package config

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
		Atom: AtomConfig{
			Enabled:  false,
			Filename: "atom.xml",
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
