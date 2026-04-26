package config

func DefaultSiteConfig() SiteConfig {
	return SiteConfig{
		ConfigVersion: ConfigVersion,
		Language:      "en",
		ContentDir:    "content",
		TemplateDir:   "templates",
		StaticDir:    "static",
		OutputDir:    "public",
		Paginate:     10,
		Drafts:       false,
		MinifyHTML:   false,
		SyntaxTheme:  "github",
		Taxonomies:   []TaxonomyConfig{},
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
		Robots: RobotsConfig{
			Enabled:  false,
			Filename: "robots.txt",
		},
		Server: ServerConfig{
			Host:       "127.0.0.1",
			Port:       1307,
			Livereload: true,
		},
		Permalinks: PermalinkConfig{
			"tags":       "/tags/{slug}/",
			"categories": "/categories/{slug}/",
		},
		Sections: SectionConfig{
			DefaultSortBy:   "date",
			PaginatePath:    "page",
		},
		Pipeline: PipelineConfig{
			Fingerprint: false,
			MinifyCSS:   false,
			MinifyJS:    false,
			Images: ImageConfig{
				Enabled: false,
				Widths:  []int{480, 768, 1200},
				Quality: 85,
			},
			SCSS: SCSSConfig{
				Enabled:  false,
				EntryDir: "css",
			},
		},
	}
}
