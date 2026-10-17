package site

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	_ "golang.org/x/image/webp"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/content"
	"github.com/MohamedElashri/nida/internal/markdown"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/themes"
)

// imageResolver gives Markdown images their intrinsic size and, when the
// image pipeline is enabled, a srcset of the generated width variants.
type imageResolver struct {
	cfg         config.SiteConfig
	basePath    string
	contentRoot string
	staticRoots []string

	mu    sync.Mutex
	sizes map[string][2]int
}

func newImageResolver(siteRoot string, cfg config.SiteConfig) *imageResolver {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return nil
	}
	r := &imageResolver{
		cfg:      cfg,
		basePath: config.BasePath(cfg.BaseURL),
		sizes:    map[string][2]int{},
	}
	if root, err := safepath.Join(absSiteRoot, cfg.ContentDir); err == nil {
		r.contentRoot = root
	}
	if root, err := safepath.Join(absSiteRoot, cfg.StaticDir); err == nil {
		r.staticRoots = append(r.staticRoots, root)
	}
	// Static lookup mirrors asset copying: site first, then themes, child
	// before parent.
	if dirs, err := themes.Dirs(absSiteRoot, cfg, "static"); err == nil {
		for i := len(dirs) - 1; i >= 0; i-- {
			r.staticRoots = append(r.staticRoots, dirs[i])
		}
	}
	return r
}

func (r *imageResolver) Image(src string, page *content.Page) (markdown.ImageInfo, bool) {
	parsed, err := url.Parse(src)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.Path == "" {
		return markdown.ImageInfo{}, false
	}
	urlPath := parsed.Path

	var file string
	switch {
	case strings.HasPrefix(urlPath, "/"):
		rel := strings.TrimPrefix(strings.TrimPrefix(urlPath, r.basePath), "/")
		for _, root := range r.staticRoots {
			if candidate, err := safepath.Join(root, rel); err == nil && isRegularFile(candidate) {
				file = candidate
				break
			}
		}
	case page != nil && page.IsBundle && r.contentRoot != "":
		if candidate, err := safepath.Join(r.contentRoot, path.Join(page.BundleDir, urlPath)); err == nil && isRegularFile(candidate) {
			file = candidate
		}
	}
	if file == "" {
		return markdown.ImageInfo{}, false
	}

	width, height, ok := r.dimensions(file)
	if !ok {
		return markdown.ImageInfo{}, false
	}
	info := markdown.ImageInfo{Width: width, Height: height}
	if r.cfg.Pipeline.Images.Enabled && isEncodable(file) {
		// Static images are resized by the asset pipeline and bundle images
		// when they are copied; both use the name.<width>w.ext convention.
		info.Srcset, info.Sizes = r.srcset(urlPath, width)
	}
	return info, true
}

// srcset lists the pipeline variants narrower than the original, which are
// the ones the pipeline actually writes, followed by the original.
func (r *imageResolver) srcset(urlPath string, width int) (string, string) {
	widths := r.cfg.Pipeline.Images.Widths
	sizes := ""
	if preset, ok := r.cfg.Pipeline.Images.Presets["content"]; ok {
		widths, sizes = preset.Widths, preset.Sizes
	}
	widths = slices.Clone(widths)
	slices.Sort(widths)
	widths = slices.Compact(widths)

	ext := path.Ext(urlPath)
	base := strings.TrimSuffix(urlPath, ext)
	var candidates []string
	for _, w := range widths {
		if w > 0 && w < width {
			candidates = append(candidates, base+"."+strconv.Itoa(w)+"w"+ext+" "+strconv.Itoa(w)+"w")
		}
	}
	if len(candidates) == 0 {
		return "", ""
	}
	candidates = append(candidates, urlPath+" "+strconv.Itoa(width)+"w")
	return strings.Join(candidates, ", "), sizes
}

func (r *imageResolver) dimensions(file string) (int, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if size, ok := r.sizes[file]; ok {
		return size[0], size[1], size[0] > 0
	}
	width, height := 0, 0
	if f, err := os.Open(file); err == nil {
		if cfg, _, err := image.DecodeConfig(f); err == nil {
			width, height = cfg.Width, cfg.Height
		}
		_ = f.Close()
	}
	r.sizes[file] = [2]int{width, height}
	return width, height, width > 0
}

// isEncodable reports whether bundle resizing can write variants for file.
func isEncodable(file string) bool {
	switch strings.ToLower(path.Ext(file)) {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	}
	return false
}

func isRegularFile(file string) bool {
	info, err := os.Lstat(file)
	return err == nil && info.Mode().IsRegular()
}
