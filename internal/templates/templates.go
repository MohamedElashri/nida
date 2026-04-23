package templates

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
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

	templateRoot := filepath.Join(absSiteRoot, cfg.TemplateDir)
	if _, err := os.Stat(filepath.Join(templateRoot, baseTemplateFile)); err != nil {
		if os.IsNotExist(err) {
			return Set{}, fmt.Errorf("load templates from %q: missing required %s", templateRoot, baseTemplateFile)
		}
		return Set{}, fmt.Errorf("load templates from %q: %w", templateRoot, err)
	}

	var shared []string
	entries := map[string]string{}
	err = filepath.WalkDir(templateRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != templateExt {
			return nil
		}

		relative, err := filepath.Rel(templateRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == baseTemplateFile || strings.Contains(relative, "/") {
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
		return Set{}, fmt.Errorf("read template directory %q: %w", templateRoot, err)
	}

	slices.Sort(shared)
	set := Set{templates: map[string]*template.Template{}}
	for name, entry := range entries {
		files := append(append([]string(nil), shared...), entry)
		tmpl := template.New("root").Funcs(funcMap())
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

func funcMap() template.FuncMap {
	return template.FuncMap{
		"formatDate":        formatDate,
		"formatDateWith":    formatDateWith,
		"safeHTML":          safeHTML,
		"safeCSS":           safeCSS,
		"join":              joinValues,
		"default":           defaultString,
		"slugify":           content.DeriveSlug,
		"documentDirection": config.DocumentDirection,
	}
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

func safeHTML(value string) template.HTML {
	return template.HTML(value)
}

func safeCSS(value string) template.CSS {
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
