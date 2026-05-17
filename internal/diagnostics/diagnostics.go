package diagnostics

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/markdown"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

type Issue struct {
	Path    string
	Message string
}

type Error struct {
	Issues []Issue
}

func (e Error) Error() string {
	var b strings.Builder
	b.WriteString("content diagnostics failed")
	for _, issue := range e.Issues {
		b.WriteString("\n- ")
		if issue.Path != "" {
			b.WriteString(issue.Path)
			b.WriteString(": ")
		}
		b.WriteString(issue.Message)
	}
	return b.String()
}

func Check(siteRoot string, cfg config.SiteConfig, pages []content.Page, sections []content.Section, lookup markdown.PathLookup) error {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root: %w", err)
	}
	contentRoot, err := safepath.Join(absSiteRoot, cfg.ContentDir)
	if err != nil {
		return fmt.Errorf("resolve content dir: %w", err)
	}
	staticRoot, err := safepath.Join(absSiteRoot, cfg.StaticDir)
	if err != nil {
		return fmt.Errorf("resolve static dir: %w", err)
	}

	checker := checker{
		contentRoot: contentRoot,
		staticRoot:  staticRoot,
		lookup:      lookup,
	}
	for _, page := range pages {
		if page.Draft && !cfg.Drafts {
			continue
		}
		checker.checkDocument(page.RelativePath, page.SourcePath, page.BodyMarkdown)
	}
	for _, section := range sections {
		if section.Draft && !cfg.Drafts {
			continue
		}
		checker.checkDocument(section.RelativePath, section.SourcePath, section.BodyMarkdown)
	}

	if len(checker.issues) == 0 {
		return nil
	}
	slices.SortFunc(checker.issues, func(a, b Issue) int {
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		return strings.Compare(a.Message, b.Message)
	})
	return Error{Issues: checker.issues}
}

type checker struct {
	contentRoot string
	staticRoot  string
	lookup      markdown.PathLookup
	issues      []Issue
}

func (c *checker) checkDocument(relativePath, sourcePath, body string) {
	if strings.TrimSpace(body) == "" || strings.TrimSpace(sourcePath) == "" {
		return
	}

	engine := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote))
	doc := engine.Parser().Parse(text.NewReader([]byte(body)))
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Link:
			c.checkInternalReference(relativePath, string(n.Destination))
		case *ast.Image:
			destination := string(n.Destination)
			c.checkInternalReference(relativePath, destination)
			c.checkImageAsset(relativePath, sourcePath, destination)
		}
		return ast.WalkContinue, nil
	})
}

func (c *checker) checkInternalReference(relativePath, destination string) {
	if !strings.HasPrefix(strings.TrimSpace(destination), "@/") {
		return
	}
	withoutPrefix := strings.TrimPrefix(strings.TrimSpace(destination), "@/")
	target, _ := markdown.SplitReferenceSuffix(withoutPrefix)
	if _, ok := c.lookup[target]; ok {
		return
	}
	withoutExt := strings.TrimSuffix(target, ".md")
	if _, ok := c.lookup[withoutExt]; ok {
		return
	}
	c.add(relativePath, fmt.Sprintf("broken internal link %q", destination))
}

func (c *checker) checkImageAsset(relativePath, sourcePath, destination string) {
	target := strings.TrimSpace(destination)
	if target == "" || strings.HasPrefix(target, "@/") || isExternalReference(target) || strings.HasPrefix(target, "#") {
		return
	}
	target, _ = markdown.SplitReferenceSuffix(target)
	if target == "" {
		return
	}

	var path string
	if strings.HasPrefix(target, "/") {
		path = filepath.Join(c.staticRoot, strings.TrimPrefix(target, "/"))
	} else {
		path = filepath.Join(filepath.Dir(sourcePath), filepath.FromSlash(target))
	}
	if pathEscapes(path, c.contentRoot, c.staticRoot) {
		c.add(relativePath, fmt.Sprintf("image path escapes site content/static roots: %q", destination))
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.add(relativePath, fmt.Sprintf("missing image asset %q", destination))
			return
		}
		c.add(relativePath, fmt.Sprintf("cannot inspect image asset %q: %v", destination, err))
		return
	}
	if info.IsDir() {
		c.add(relativePath, fmt.Sprintf("image asset points to a directory: %q", destination))
	}
}

func (c *checker) add(path, message string) {
	c.issues = append(c.issues, Issue{Path: path, Message: message})
}

func isExternalReference(value string) bool {
	if strings.HasPrefix(value, "//") {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != ""
}

func pathEscapes(path string, roots ...string) bool {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return true
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		cleanRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(cleanRoot, cleanPath)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return false
		}
	}
	return true
}
