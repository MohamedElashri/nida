package templates

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
)

const baseTemplateFile = "base.tmpl"

type Set struct {
	templates map[string]*template.Template
}

func Load(siteRoot string, cfg config.SiteConfig) (Set, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return Set{}, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	templateRoot := filepath.Join(absSiteRoot, cfg.TemplateDir)
	basePath := filepath.Join(templateRoot, baseTemplateFile)
	if _, err := os.Stat(basePath); err != nil {
		if os.IsNotExist(err) {
			return Set{}, fmt.Errorf("load templates from %q: missing required %s", templateRoot, baseTemplateFile)
		}
		return Set{}, fmt.Errorf("load templates from %q: %w", templateRoot, err)
	}

	entries, err := os.ReadDir(templateRoot)
	if err != nil {
		return Set{}, fmt.Errorf("read template directory %q: %w", templateRoot, err)
	}

	set := Set{templates: map[string]*template.Template{}}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == baseTemplateFile || filepath.Ext(entry.Name()) != ".tmpl" {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		path := filepath.Join(templateRoot, entry.Name())
		tmpl, err := template.New("base").Funcs(funcMap()).ParseFiles(basePath, path)
		if err != nil {
			return Set{}, fmt.Errorf("parse template %q: %w", path, err)
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
		"formatDate": formatDate,
		"safeHTML":   safeHTML,
		"join":       strings.Join,
		"default":    defaultString,
		"slugify":    content.DeriveSlug,
		"title":      strings.Title,
	}
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func safeHTML(value string) template.HTML {
	return template.HTML(value)
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
