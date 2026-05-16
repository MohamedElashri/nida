package templates

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/safepath"
)

const (
	baseTemplateFile = "base.html"
	templateExt      = ".html"
)

type Set struct {
	templates map[string]*template.Template
}

func Load(siteRoot string, cfg config.SiteConfig) (Set, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return Set{}, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	var templateRoots []string

	if cfg.Theme != "" {
		themeRoot, err := safepath.Join(absSiteRoot, filepath.Join(cfg.ThemesDir, cfg.Theme, "templates"))
		if err != nil {
			return Set{}, fmt.Errorf("resolve theme templates: %w", err)
		}
		if err := safepath.EnsureNoSymlinkPath(absSiteRoot, themeRoot); err != nil && !os.IsNotExist(err) {
			return Set{}, fmt.Errorf("check theme templates: %w", err)
		}
		if _, err := os.Stat(themeRoot); err == nil {
			templateRoots = append(templateRoots, themeRoot)
		}
	}

	siteTemplateRoot, err := safepath.Join(absSiteRoot, cfg.TemplateDir)
	if err != nil {
		return Set{}, fmt.Errorf("resolve template dir: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, siteTemplateRoot); err != nil {
		return Set{}, fmt.Errorf("check template dir: %w", err)
	}
	templateRoots = append(templateRoots, siteTemplateRoot)

	basePath := filepath.Join(siteTemplateRoot, baseTemplateFile)
	if _, err := os.Stat(basePath); err != nil {
		if os.IsNotExist(err) {
			return Set{}, fmt.Errorf("load templates: missing required %s", baseTemplateFile)
		}
		return Set{}, fmt.Errorf("load templates: %w", err)
	}

	return loadFromRoots(templateRoots, absSiteRoot, cfg)
}

func loadFromRoots(roots []string, siteRoot string, cfg config.SiteConfig) (Set, error) {
	var shared []string
	entries := map[string]string{}

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || filepath.Ext(path) != templateExt {
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return fmt.Errorf("refuse template symlink %q", path)
			}

			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)

			if relative == baseTemplateFile {
				shared = append(shared, path)
				return nil
			}

			if strings.Contains(relative, "/") {
				shared = append(shared, path)
				return nil
			}

			name := strings.TrimSuffix(relative, filepath.Ext(relative))
			if name == "base" {
				shared = append(shared, path)
				return nil
			}

			entries[name] = path
			return nil
		})
		if err != nil {
			return Set{}, fmt.Errorf("read template directory %q: %w", root, err)
		}
	}

	slices.Sort(shared)
	set := Set{templates: map[string]*template.Template{}}
	for name, entry := range entries {
		files := append(append([]string(nil), shared...), entry)
		tmpl := template.New("root").Funcs(funcMap(siteRoot, cfg))
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				return Set{}, fmt.Errorf("read template %q: %w", file, err)
			}
			if _, err := tmpl.Parse(string(data)); err != nil {
				return Set{}, fmt.Errorf("parse template %q: %w", file, err)
			}
		}
		set.templates[name] = tmpl
	}

	return set, nil
}

func (s Set) Has(name string) bool {
	_, ok := s.templates[name]
	return ok
}

func (s Set) Execute(name string, data any) (string, error) {
	tmpl, ok := s.templates[name]
	if !ok {
		return "", fmt.Errorf("missing template %q", name)
	}

	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, name, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", name, err)
	}
	return b.String(), nil
}

func funcMap(siteRoot string, cfgs ...config.SiteConfig) template.FuncMap {
	cfg := config.DefaultSiteConfig()
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	return template.FuncMap{
		"formatDate":        formatDate,
		"formatDateWith":    formatDateWith,
		"safeHTML":          unsafeHTML,
		"unsafeHTML":        unsafeHTML,
		"safeCSS":           unsafeCSS,
		"unsafeCSS":         unsafeCSS,
		"join":              joinValues,
		"add":               add,
		"sub":               sub,
		"hasSuffix":         strings.HasSuffix,
		"hasPrefix":         strings.HasPrefix,
		"contains":          strings.Contains,
		"lower":             strings.ToLower,
		"trimSpace":         strings.TrimSpace,
		"replace":           strings.ReplaceAll,
		"default":           defaultString,
		"slugify":           content.DeriveSlug,
		"documentDirection": config.DocumentDirection,
		"groupByYear":       groupByYear,
		"now":               time.Now,
		"readFile":          readFileFunc(siteRoot, cfg),
		"resizeImage":       resizeImageFunc(siteRoot),
		"sortDesc":          sortDesc,
		"dig":               digValue,
	}
}

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func joinValues(value any, sep string) string {
	switch values := value.(type) {
	case []string:
		return strings.Join(values, sep)
	case []any:
		parts := make([]string, 0, len(values))
		for _, item := range values {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, sep)
	default:
		return ""
	}
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func formatDateWith(value time.Time, format string) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(strftimeToGoLayout(format))
}

func unsafeHTML(value string) template.HTML {
	return template.HTML(value)
}

func unsafeCSS(value string) template.CSS {
	return template.CSS(value)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func AvailableNames(set Set) []string {
	names := make([]string, 0, len(set.templates))
	for name := range set.templates {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func strftimeToGoLayout(format string) string {
	if strings.TrimSpace(format) == "" {
		return "2006-01-02"
	}
	replacer := strings.NewReplacer(
		"%Y", "2006",
		"%m", "01",
		"%d", "02",
		"%b", "Jan",
		"%B", "January",
		"%H", "15",
		"%M", "04",
		"%S", "05",
		"%+", time.RFC3339,
	)
	return replacer.Replace(format)
}

// YearGroup holds pages grouped by a year string.
type YearGroup struct {
	Year  string
	Pages []content.Page
}

func groupByYear(pages []content.Page) []YearGroup {
	groups := make(map[string][]content.Page)
	for _, p := range pages {
		if !p.Date.IsZero() {
			year := p.Date.Format("2006")
			groups[year] = append(groups[year], p)
		}
	}

	var years []string
	for y := range groups {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(years)))

	result := make([]YearGroup, 0, len(years))
	for _, y := range years {
		result = append(result, YearGroup{Year: y, Pages: groups[y]})
	}
	return result
}

func readFileFunc(siteRoot string, cfg config.SiteConfig) func(string) (string, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		absSiteRoot = siteRoot
	}
	allowedRoots := readFileAllowedRoots(absSiteRoot, cfg)
	return func(path string) (string, error) {
		if containsHiddenPathSegment(path) {
			return "", fmt.Errorf("readFile refuses hidden path segment in %q", path)
		}
		fullPath, err := safepath.Join(absSiteRoot, path)
		if err != nil {
			return "", err
		}
		if !pathUnderAnyRoot(allowedRoots, fullPath) {
			return "", fmt.Errorf("readFile path %q is outside allowed template/static roots", path)
		}
		if err := safepath.EnsureNoSymlinkPath(absSiteRoot, fullPath); err != nil {
			return "", err
		}
		if err := safepath.RejectSymlink(fullPath); err != nil {
			return "", err
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func readFileAllowedRoots(absSiteRoot string, cfg config.SiteConfig) []string {
	candidates := []string{
		cfg.TemplateDir,
		cfg.StaticDir,
	}
	if strings.TrimSpace(cfg.Theme) != "" {
		candidates = append(candidates,
			filepath.Join(cfg.ThemesDir, cfg.Theme, "templates"),
			filepath.Join(cfg.ThemesDir, cfg.Theme, "static"),
		)
	}

	roots := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		root, err := safepath.Join(absSiteRoot, candidate)
		if err != nil {
			continue
		}
		if containsHiddenPathSegment(candidate) {
			continue
		}
		roots = append(roots, root)
	}
	return roots
}

func pathUnderAnyRoot(roots []string, path string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}

func containsHiddenPathSegment(path string) bool {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if clean == "." || clean == "" {
		return false
	}
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func sortDesc(values []string) []string {
	out := append([]string(nil), values...)
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out
}

func digValue(values map[string]any, path string) any {
	if values == nil {
		return nil
	}
	keys := strings.Split(path, ".")
	for i, key := range keys {
		v, ok := values[key]
		if !ok {
			return nil
		}
		if i == len(keys)-1 {
			return v
		}
		values, ok = v.(map[string]any)
		if !ok {
			return nil
		}
	}
	return values
}
