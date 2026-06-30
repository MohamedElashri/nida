package config

func DefaultSiteConfig() SiteConfig {
	return SiteConfig{
		ConfigVersion: ConfigVersion,
		Language:      "en",
		ContentDir:    "content",
		TemplateDir:   "templates",
		StaticDir:     "static",
		OutputDir:     "public",
		ThemesDir:     "themes",
		Paginate:      10,
		Drafts:        false,
		MinifyHTML:    false,
		SyntaxTheme:   "github",
		Taxonomies:    []TaxonomyConfig{},
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
			Enabled:  true,
			Filename: "robots.txt",
		},
		Server: ServerConfig{
			Host:       "127.0.0.1",
			Port:       1702,
			Livereload: true,
		},
		Permalinks: PermalinkConfig{
			"tags":       "/tags/{slug}/",
			"categories": "/categories/{slug}/",
		},
		Sections: SectionConfig{
			DefaultSortBy: "date",
			PaginatePath:  "page",
		},
		Pipeline: PipelineConfig{
			Fingerprint: false,
			MinifyCSS:   false,
			MinifyJS:    false,
			Images: ImageConfig{
				Enabled: false,
				Widths:  []int{480, 768, 1200},
				Quality: 85,
				Presets: map[string]ImagePresetConfig{
					"thumb": {
						Widths: []int{320, 640},
						Sizes:  "(max-width: 700px) 50vw, 320px",
					},
					"content": {
						Widths: []int{480, 768, 1200},
						Sizes:  "(max-width: 760px) 100vw, 760px",
					},
					"hero": {
						Widths: []int{768, 1200, 1600},
						Sizes:  "100vw",
					},
				},
			},
			SCSS: SCSSConfig{
				Enabled:  false,
				EntryDir: "css",
			},
		},
		Search: SearchConfig{
			Enabled:  false,
			Filename: "search_index.en.js",
		},
		Diagnostics: DiagnosticsConfig{
			Enabled: false,
		},
	}
}
