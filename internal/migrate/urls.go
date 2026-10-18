package migrate

import (
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
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

	// taken maps every URL the new site serves to the content file that
	// serves it, so a fix never steals another page's URL.
	taken := map[string]string{}
	for rel, u := range lookup {
		if strings.HasSuffix(rel, ".md") {
			taken[u] = rel
		}
	}
	for _, page := range pages {
		for _, alias := range page.Aliases {
			if a, err := route.Page("alias", alias); err == nil {
				taken[a] = page.RelativePath
			}
		}
	}

	contentPrefix := strings.Trim(filepath.ToSlash(cfg.ContentDir), "/") + "/"
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
		if owner, clash := taken[oldURL]; clash && owner != rel {
			p.Report.Add(Manual, f.origin, 0, "The old URL `%s` is now used by `%s`, so this page moved to `%s`. Decide which page keeps the URL.", oldURL, owner, newURL)
			continue
		}
		if mode == RedirectURLs {
			aliases := frontmatter.StringList(f.doc.Values()["aliases"])
			if !slices.Contains(aliases, oldURL) {
				aliases = append(aliases, oldURL)
			}
			if err := f.doc.Set("aliases", aliases); err != nil {
				return err
			}
			p.Report.Add(Changed, f.origin, 0, "Page moved to its Nida URL; the old URL redirects there through `aliases`.")
		} else {
			if err := f.doc.Set("path", oldURL); err != nil {
				return err
			}
			p.Report.Add(Changed, f.origin, 0, "Set `path` so the page keeps its old URL.")
		}
		taken[oldURL] = rel
		p.Report.URLsMoved++
		full := filepath.Join(target, filepath.FromSlash(dest))
		if err := os.WriteFile(full, f.doc.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// cleanOldURL turns an old site URL into a Nida route: it drops any
// scheme and host, the query and fragment, decodes percent escapes, and
// treats a trailing index.html as the directory it names.
func cleanOldURL(raw string) (string, error) {
	if u, err := url.Parse(raw); err == nil {
		raw = u.Path
		if u.RawPath != "" {
			raw = u.RawPath
		}
	}
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}
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
