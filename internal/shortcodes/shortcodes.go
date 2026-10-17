// Package shortcodes loads shortcode templates and renders shortcode calls
// found in Markdown.
//
// Templates come from three places, later ones overriding earlier ones:
// built-in shortcodes, templates/shortcodes/ in each theme (parent first),
// and the site's templates/shortcodes/.
package shortcodes

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/markdown"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/templates"
	"github.com/MohamedElashri/nida/internal/themes"
)

//go:embed builtin/*.html
var builtin embed.FS

const dirName = "shortcodes"

type Set struct {
	templates map[string]*template.Template
}

func Load(siteRoot string, cfg config.SiteConfig) (*Set, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}
	set := &Set{templates: map[string]*template.Template{}}
	funcs := templates.FuncMap(absSiteRoot, cfg)

	entries, err := fs.ReadDir(builtin, "builtin")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		data, err := fs.ReadFile(builtin, path.Join("builtin", entry.Name()))
		if err != nil {
			return nil, err
		}
		if err := set.add(entry.Name(), string(data), funcs); err != nil {
			return nil, err
		}
	}

	themeTemplates, err := themes.Dirs(absSiteRoot, cfg, "templates")
	if err != nil {
		return nil, err
	}
	siteTemplates, err := safepath.Join(absSiteRoot, cfg.TemplateDir)
	if err != nil {
		return nil, fmt.Errorf("resolve template dir: %w", err)
	}
	for _, root := range append(themeTemplates, siteTemplates) {
		if err := set.loadDir(filepath.Join(root, dirName), funcs); err != nil {
			return nil, err
		}
	}
	return set, nil
}

func (s *Set) loadDir(dir string, funcs template.FuncMap) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read shortcode directory %q: %w", dir, err)
	}
	if err := safepath.RejectSymlink(dir); err != nil {
		return fmt.Errorf("check shortcode directory %q: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".html" {
			continue
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refuse shortcode template symlink %q", filepath.Join(dir, entry.Name()))
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read shortcode template %q: %w", entry.Name(), err)
		}
		if err := s.add(entry.Name(), string(data), funcs); err != nil {
			return err
		}
	}
	return nil
}

func (s *Set) add(file, body string, funcs template.FuncMap) error {
	name := strings.TrimSuffix(file, filepath.Ext(file))
	tmpl, err := template.New(name).Funcs(funcs).Parse(body)
	if err != nil {
		return fmt.Errorf("parse shortcode template %q: %w", file, err)
	}
	s.templates[name] = tmpl
	return nil
}

func (s *Set) Has(name string) bool {
	_, ok := s.templates[name]
	return ok
}

func (s *Set) Render(call *markdown.ShortcodeCall) (string, error) {
	tmpl, ok := s.templates[call.Name]
	if !ok {
		return "", fmt.Errorf("unknown shortcode %q", call.Name)
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, call); err != nil {
		return "", err
	}
	return strings.TrimSpace(b.String()), nil
}
