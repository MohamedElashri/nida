package migrate

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/feeds"
	"github.com/MohamedElashri/nida/internal/frontmatter"
	"github.com/MohamedElashri/nida/internal/route"
	"github.com/MohamedElashri/nida/internal/site"
)

// fixURLs routes the written site with Nida's own router and compares
// each page's URL with its URL on the old site. A page that moved gets a
// `path` (KeepURLs) or an alias (RedirectURLs), and the file is rewritten.
// Using the real router instead of predicting Nida's URLs means the result
// is right even for permalink patterns and slugs the source set.
func fixURLs(p *Plan, target, mode string) error {
	cfg, _, err := config.Load(config.Options{SiteRoot: target})
	if err != nil {
		p.Report.Add(Manual, "", 0, "The generated config.toml does not load (%v), so URLs were not checked. Fix the config and compare the old and new sitemap.xml.", err)
		return nil
	}
	for _, warning := range cfg.Warnings {
		p.Report.Add(Review, "config.toml", 0, "Config warning: %s", warning)
	}
	cfg.Drafts, cfg.Future = true, true
	pages, sections, err := content.Discover(target, cfg)
	if err != nil {
		p.Report.Add(Manual, "", 0, "Content could not be loaded (%v), so URLs were not checked.", err)
		return nil
	}
	lookup := site.BuildPathLookup(pages, sections, cfg)
	taken := servedURLs(target, cfg, pages, sections, lookup)

	type move struct {
		f              *planned
		rel            string
		oldURL, newURL string
	}
	var moves []move
	contentPrefix := strings.Trim(filepath.ToSlash(cfg.ContentDir), "/") + "/"
	finalURL := map[string]string{} // content-relative path -> URL after fixes
	for rel, u := range lookup {
		finalURL[rel] = u
	}
	for _, dest := range p.order {
		f := p.files[dest]
		if f.doc == nil || f.oldURL == "" || !strings.HasPrefix(dest, contentPrefix) {
			continue
		}
		rel := strings.TrimPrefix(dest, contentPrefix)
		newURL, ok := lookup[rel]
		if !ok {
			p.Report.Add(Manual, f.origin, 0, "Nida could not route this page, so its old URL `%s` was not kept. `nida check` explains why.", f.oldURL)
			continue
		}
		oldURL, err := cleanOldURL(f.oldURL)
		if err != nil {
			p.Report.Add(Manual, f.origin, 0, "The old URL `%s` is not a valid Nida route (%v). Add a redirect for it on your web host.", f.oldURL, err)
			continue
		}
		if sameURL(oldURL, newURL) {
			p.Report.URLsKept++
			continue
		}
		moves = append(moves, move{f, rel, oldURL, newURL})
	}

	// A page that moves back to its old URL frees the URL Nida gave it,
	// so two pages can trade URLs. In redirect mode pages stay where Nida
	// put them and only claim their old URL for a redirect.
	if mode == KeepURLs {
		for _, m := range moves {
			if taken[m.newURL] == m.rel {
				delete(taken, m.newURL)
			}
		}
	}
	for _, m := range moves {
		if owner, clash := taken[m.oldURL]; clash && owner != m.rel {
			p.Report.Add(Manual, m.f.origin, 0, "The old URL `%s` is now used by %s, so this page moved to `%s`. Decide which one keeps the URL.", m.oldURL, owner, m.newURL)
			if mode == KeepURLs {
				if other, clash := taken[m.newURL]; clash && other != m.rel {
					p.Report.Add(Manual, m.f.origin, 0, "This page's new URL `%s` was also claimed by %s; `nida check` will report the conflict until one of them moves.", m.newURL, other)
				}
				taken[m.newURL] = m.rel
			}
			continue
		}
		if mode == RedirectURLs {
			aliases := frontmatter.StringList(m.f.doc.Values()["aliases"])
			if !slices.Contains(aliases, m.oldURL) {
				aliases = append(aliases, m.oldURL)
			}
			if err := m.f.doc.Set("aliases", aliases); err != nil {
				return err
			}
			p.Report.Add(Changed, m.f.origin, 0, "Page moved to its Nida URL; the old URL redirects there through `aliases`.")
		} else {
			if err := m.f.doc.Set("path", m.oldURL); err != nil {
				return err
			}
			finalURL[m.rel] = m.oldURL
			p.Report.Add(Changed, m.f.origin, 0, "Set `path` so the page keeps its old URL.")
		}
		taken[m.oldURL] = "`" + m.rel + "`"
		p.Report.URLsMoved++
		full := filepath.Join(target, filepath.FromSlash(contentPrefix+m.rel))
		if err := os.WriteFile(full, m.f.doc.Bytes(), 0o644); err != nil {
			return err
		}
	}
	checkBundleResources(p, pages, finalURL, contentPrefix)
	return nil
}

// servedURLs maps every URL the new site will serve, other than redirects
// a fix may add, to a description of what serves it: pages, sections,
// taxonomy pages, feeds and other generated files, and static files.
func servedURLs(target string, cfg config.SiteConfig, pages []content.Page, sections []content.Section, lookup map[string]string) map[string]string {
	taken := map[string]string{}
	for rel, u := range lookup {
		if strings.HasSuffix(rel, ".md") {
			taken[u] = rel
		}
	}
	for _, page := range pages {
		for _, alias := range page.Aliases {
			if a, err := route.Page("alias", alias); err == nil {
				taken[a] = "an alias of `" + page.RelativePath + "`"
			}
		}
	}
	if index, _, err := site.BuildIndex(pages, sections, cfg); err == nil {
		for _, tax := range index.Taxonomies {
			if !tax.Render {
				continue
			}
			taken[tax.URL] = "the `" + tax.Name + "` taxonomy page"
			for _, term := range tax.Terms {
				taken[term.URL] = "the `" + tax.Name + "` term page"
			}
		}
		for _, name := range feeds.Filenames(cfg, index) {
			taken["/"+strings.TrimPrefix(name, "/")] = "Nida's generated `" + name + "`"
		}
	}
	for _, generated := range []struct {
		on   bool
		name string
	}{{cfg.Sitemap.Enabled, cfg.Sitemap.Filename}, {cfg.Robots.Enabled, cfg.Robots.Filename}, {cfg.Search.Enabled, cfg.Search.Filename}} {
		if generated.on {
			taken["/"+generated.name] = "Nida's generated `" + generated.name + "`"
		}
	}
	static := filepath.Join(target, filepath.FromSlash(cfg.StaticDir))
	_ = filepath.WalkDir(static, func(full string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(static, full)
		u := "/" + filepath.ToSlash(rel)
		taken[u] = "the static file `" + u + "`"
		if path.Base(u) == "index.html" {
			taken[path.Dir(u)+"/"] = "the static file `" + u + "`"
		}
		return nil
	})
	return taken
}

// checkBundleResources reports page bundles whose resources would land on
// the same output path. Pages written as files, such as /posts/a.html and
// /posts/b.html, share their directory, so two bundles with a cover.png
// would overwrite each other; Nida's build refuses that.
func checkBundleResources(p *Plan, pages []content.Page, finalURL map[string]string, contentPrefix string) {
	owner := map[string]string{}
	for _, page := range pages {
		if !page.IsBundle || len(page.Resources) == 0 {
			continue
		}
		u, ok := finalURL[page.RelativePath]
		if !ok {
			continue
		}
		origin := contentPrefix + page.RelativePath
		if f, ok := p.files[origin]; ok && f.origin != "" {
			origin = f.origin
		}
		if route.IsFile(u) {
			p.Report.Add(Review, origin, 0, "The page is written as the file `%s`, so its bundle files are served from `%s`, next to it.", u, route.Dir(u))
		}
		for _, res := range page.Resources {
			out := route.Dir(u) + res
			if other, clash := owner[out]; clash {
				p.Report.Add(Manual, origin, 0, "Bundle file `%s` would be written to `%s`, which `%s` also writes. Rename one of them; `nida build` refuses the clash.", res, out, other)
				continue
			}
			owner[out] = origin
		}
	}
}

// cleanOldURL turns an old site URL into a Nida route: it drops any
// scheme and host, the query and fragment, decodes percent escapes, and
// treats a trailing index.html as the directory it names.
func cleanOldURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	// Only a URL with a scheme has a host; "//2024/x/" from an empty
	// Jekyll :categories is a path with a doubled slash.
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			raw = u.EscapedPath()
		}
	}
	raw, _, _ = strings.Cut(raw, "#")
	raw, _, _ = strings.Cut(raw, "?")
	for strings.Contains(raw, "//") {
		raw = strings.ReplaceAll(raw, "//", "/")
	}
	// Decode each segment on its own, so an escaped slash (%2F) cannot
	// become a path separator.
	segments := strings.Split(raw, "/")
	for i, seg := range segments {
		decoded, err := url.PathUnescape(seg)
		if err != nil {
			continue
		}
		if strings.Contains(decoded, "/") {
			return "", fmt.Errorf("segment %q holds an escaped slash", seg)
		}
		segments[i] = decoded
	}
	raw = strings.Join(segments, "/")
	if strings.HasSuffix(raw, "/index.html") || raw == "index.html" {
		raw = strings.TrimSuffix(raw, "index.html")
	}
	if raw == "" {
		raw = "/"
	}
	return route.Page("old URL", raw)
}

func sameURL(a, b string) bool {
	if a == b {
		return true
	}
	ua, errA := url.PathUnescape(a)
	ub, errB := url.PathUnescape(b)
	return errA == nil && errB == nil && ua == ub
}
