package content

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/frontmatter"
)

const (
	TypePost    = "post"
	TypePage    = "page"
	TypeSection = "section"
)

type Item struct {
	SourcePath     string
	RelativePath   string
	Type           string
	SectionPath    string
	RawFrontMatter string
	BodyMarkdown   string
	BodyHTML       string
	Title          string
	Slug           string
	URL            string
	Description    string
	Date           time.Time
	ReadingTime    int
	Draft          bool
	Tags           []string
	Categories     []string
	Template       string
	PageTemplate   string
	PaginateBy     int
	Extra          map[string]any
}

func Discover(siteRoot string, cfg config.SiteConfig) ([]Item, error) {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve site root %q: %w", siteRoot, err)
	}

	contentRoot := filepath.Join(absSiteRoot, cfg.ContentDir)
	items, err := discoverAll(contentRoot, cfg)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(items, func(a, b Item) int {
		return strings.Compare(a.RelativePath, b.RelativePath)
	})

	return items, nil
}

func discoverAll(contentRoot string, cfg config.SiteConfig) ([]Item, error) {
	entries := make([]string, 0)

	err := filepath.WalkDir(contentRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == contentRoot && os.IsNotExist(walkErr) {
				return fs.SkipDir
			}
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			entries = append(entries, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover content under %q: %w", contentRoot, err)
	}

	slices.Sort(entries)

	items := make([]Item, 0, len(entries))
	for _, path := range entries {
		item, err := loadItem(contentRoot, path, classifyItem(contentRoot, path, cfg))
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func loadItem(contentRoot, sourcePath, itemType string) (Item, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return Item{}, fmt.Errorf("read content file %q: %w", sourcePath, err)
	}

	doc, err := frontmatter.Parse(data)
	if err != nil {
		return Item{}, fmt.Errorf("parse content file %q: %w", sourcePath, err)
	}

	relativePath, err := filepath.Rel(contentRoot, sourcePath)
	if err != nil {
		return Item{}, fmt.Errorf("compute relative path for %q: %w", sourcePath, err)
	}

	meta := normalizeMetadata(doc.Metadata)
	slug := meta.Slug
	if itemType == TypeSection {
		slug = DeriveSlug(filepath.Base(filepath.Dir(sourcePath)))
		if relativePath == "_index.md" {
			slug = ""
		}
	} else if slug == "" {
		slug = DeriveSlug(filepath.Base(sourcePath))
	}

	return Item{
		SourcePath:     sourcePath,
		RelativePath:   filepath.ToSlash(relativePath),
		Type:           itemType,
		SectionPath:    sectionPath(filepath.ToSlash(relativePath), itemType),
		RawFrontMatter: doc.RawFrontMatter,
		BodyMarkdown:   doc.BodyMarkdown,
		Title:          meta.Title,
		Slug:           slug,
		Description:    meta.Description,
		Date:           meta.Date,
		ReadingTime:    EstimateReadingTime(doc.BodyMarkdown),
		Draft:          meta.Draft,
		Tags:           meta.Tags,
		Categories:     meta.Categories,
		Template:       meta.Template,
		PageTemplate:   meta.PageTemplate,
		PaginateBy:     meta.PaginateBy,
		Extra:          meta.Extra,
	}, nil
}

func normalizeMetadata(meta frontmatter.Metadata) frontmatter.Metadata {
	meta.Title = strings.TrimSpace(meta.Title)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.Slug = DeriveSlug(meta.Slug)
	meta.Template = strings.TrimSpace(meta.Template)
	meta.PageTemplate = strings.TrimSpace(meta.PageTemplate)
	meta.Tags = normalizeStringList(meta.Tags)
	meta.Categories = normalizeStringList(meta.Categories)
	return meta
}

func classifyItem(contentRoot, sourcePath string, cfg config.SiteConfig) string {
	relativePath, err := filepath.Rel(contentRoot, sourcePath)
	if err != nil {
		return TypePost
	}

	normalized := filepath.ToSlash(relativePath)
	if filepath.Base(sourcePath) == "_index.md" {
		return TypeSection
	}
	if matchesContentDir(normalized, cfg.PagesDir) {
		return TypePage
	}
	return TypePost
}

func sectionPath(relativePath, itemType string) string {
	dir := filepath.ToSlash(filepath.Dir(relativePath))
	if dir == "." {
		dir = ""
	}
	if itemType == TypeSection {
		return dir
	}
	return dir
}

func matchesContentDir(relativePath, dir string) bool {
	dir = strings.Trim(filepath.ToSlash(dir), "/")
	if dir == "" {
		return false
	}
	return relativePath == dir || strings.HasPrefix(relativePath, dir+"/")
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func DeriveSlug(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, filepath.Ext(value))
	value = strings.ToLower(value)

	var b strings.Builder
	lastHyphen := false

	for _, r := range value {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastHyphen = false
		case r == '-' || r == '_' || r == ' ':
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		default:
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}
