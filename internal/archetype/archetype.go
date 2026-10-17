// Package archetype creates new content files from archetype templates.
package archetype

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/themes"
)

const dirName = "archetypes"

const defaultArchetype = `+++
title = "{{ .Title }}"
date = {{ .Date }}
draft = true
+++

`

// Data is what an archetype template receives.
type Data struct {
	Title   string
	Date    string
	Slug    string
	Section string
	Path    string
}

// Result reports the created file.
type Result struct {
	Path      string
	Archetype string
}

// Create writes target, such as "posts/my-post.md", into the content
// directory. Without ".md" it creates a bundle: posts/my-post/index.md.
func Create(siteRoot string, cfg config.SiteConfig, target string, now time.Time) (Result, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return Result{}, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	rel := path.Clean(filepath.ToSlash(strings.TrimSpace(target)))
	rel = strings.TrimPrefix(rel, cfg.ContentDir+"/")
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") || rel == ".." || path.IsAbs(rel) {
		return Result{}, fmt.Errorf("new content path %q must be relative to the content directory", target)
	}
	name := path.Base(rel)
	if strings.EqualFold(path.Ext(rel), ".md") {
		name = strings.TrimSuffix(name, path.Ext(name))
	} else {
		rel = path.Join(rel, "index.md")
	}
	if name == "index" || name == "_index" {
		return Result{}, fmt.Errorf("new content path %q needs a page name", target)
	}

	contentRoot, err := safepath.Join(absSiteRoot, cfg.ContentDir)
	if err != nil {
		return Result{}, fmt.Errorf("resolve content dir: %w", err)
	}
	outPath, err := safepath.Join(contentRoot, rel)
	if err != nil {
		return Result{}, fmt.Errorf("resolve new content path: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(absSiteRoot, outPath); err != nil && !os.IsNotExist(err) {
		return Result{}, fmt.Errorf("check new content path: %w", err)
	}
	if _, err := os.Stat(outPath); err == nil {
		return Result{}, fmt.Errorf("%s already exists", outPath)
	}

	section := ""
	if dir := path.Dir(rel); dir != "." {
		section = strings.Split(dir, "/")[0]
		if strings.HasSuffix(rel, "/index.md") {
			section = strings.Split(path.Dir(path.Dir(rel)), "/")[0]
			if section == "." {
				section = ""
			}
		}
	}

	body, source, err := findArchetype(absSiteRoot, cfg, section)
	if err != nil {
		return Result{}, err
	}
	tmpl, err := template.New(source).Parse(body)
	if err != nil {
		return Result{}, fmt.Errorf("parse archetype %s: %w", source, err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, Data{
		Title:   titleFromName(name),
		Date:    now.Format(time.RFC3339),
		Slug:    content.DeriveSlug(name),
		Section: section,
		Path:    rel,
	}); err != nil {
		return Result{}, fmt.Errorf("render archetype %s: %w", source, err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("create directory: %w", err)
	}
	file, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Result{}, fmt.Errorf("create %s: %w", outPath, err)
	}
	if _, err := file.Write(out.Bytes()); err != nil {
		_ = file.Close()
		return Result{}, fmt.Errorf("write %s: %w", outPath, err)
	}
	if err := file.Close(); err != nil {
		return Result{}, err
	}
	return Result{Path: outPath, Archetype: source}, nil
}

// findArchetype tries <section>.md then default.md in the site and each
// theme, falling back to a built-in default.
func findArchetype(absSiteRoot string, cfg config.SiteConfig, section string) (string, string, error) {
	dirs := []string{filepath.Join(absSiteRoot, dirName)}
	themeDirs, err := themes.Dirs(absSiteRoot, cfg, dirName)
	if err != nil {
		return "", "", err
	}
	for i := len(themeDirs) - 1; i >= 0; i-- {
		dirs = append(dirs, themeDirs[i])
	}

	var names []string
	if section != "" {
		names = append(names, section+".md")
	}
	names = append(names, "default.md")
	for _, name := range names {
		for _, dir := range dirs {
			file := filepath.Join(dir, name)
			if err := safepath.RejectSymlink(file); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return "", "", err
			}
			data, err := os.ReadFile(file)
			if err == nil {
				return string(data), file, nil
			}
			if !os.IsNotExist(err) {
				return "", "", err
			}
		}
	}
	return defaultArchetype, "built-in default", nil
}

func titleFromName(name string) string {
	return content.TitleCase(strings.NewReplacer("-", " ", "_", " ").Replace(name))
}
