package templates

import (
	"fmt"
	"image"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/route"
	"github.com/MohamedElashri/nida/internal/safepath"
	"github.com/MohamedElashri/nida/internal/themes"
)

func assetURLFunc(cfg config.SiteConfig) func(string) (string, error) {
	return func(value string) (string, error) {
		assetPath, suffix, err := splitAssetPath(value)
		if err != nil {
			return "", err
		}
		return joinBasePath(config.BasePath(cfg.BaseURL), assetPath) + suffix, nil
	}
}

func imageVariantFunc(cfg config.SiteConfig) func(string, int) (string, error) {
	assetURL := assetURLFunc(cfg)
	return func(value string, width int) (string, error) {
		if width <= 0 {
			return "", fmt.Errorf("image variant width must be greater than 0")
		}
		variant, err := imageVariantPath(value, width)
		if err != nil {
			return "", err
		}
		return assetURL(variant)
	}
}

func imageSrcsetFunc(siteRoot string, cfg config.SiteConfig) func(string) (string, error) {
	widthOf := staticImageWidths(siteRoot, cfg)
	return func(value string) (string, error) {
		return imageSrcset(value, cfg.Pipeline.Images.Widths, cfg, widthOf)
	}
}

func imagePresetSrcsetFunc(siteRoot string, cfg config.SiteConfig) func(string, string) (string, error) {
	widthOf := staticImageWidths(siteRoot, cfg)
	return func(name, value string) (string, error) {
		preset, ok := cfg.Pipeline.Images.Presets[strings.TrimSpace(name)]
		if !ok {
			return "", fmt.Errorf("unknown image preset %q", name)
		}
		return imageSrcset(value, preset.Widths, cfg, widthOf)
	}
}

func imagePresetSizesFunc(cfg config.SiteConfig) func(string) (string, error) {
	return func(name string) (string, error) {
		preset, ok := cfg.Pipeline.Images.Presets[strings.TrimSpace(name)]
		if !ok {
			return "", fmt.Errorf("unknown image preset %q", name)
		}
		return preset.Sizes, nil
	}
}

// imageSrcset lists the width variants of a static image. The pipeline
// only writes variants narrower than the original, so when the original's
// width is known, wider ones are left out and the original is the last
// candidate.
func imageSrcset(value string, widths []int, cfg config.SiteConfig, widthOf func(string) int) (string, error) {
	assetURL := assetURLFunc(cfg)
	uniqueWidths := uniquePositiveWidths(widths)
	if len(uniqueWidths) == 0 {
		return "", fmt.Errorf("image srcset needs at least one width")
	}
	original := 0
	if assetPath, _, err := splitAssetPath(value); err == nil {
		original = widthOf(assetPath)
	}

	candidates := make([]string, 0, len(uniqueWidths)+1)
	for _, width := range uniqueWidths {
		if original > 0 && width >= original {
			continue
		}
		variant, err := imageVariantPath(value, width)
		if err != nil {
			return "", err
		}
		u, err := assetURL(variant)
		if err != nil {
			return "", err
		}
		candidates = append(candidates, u+" "+strconv.Itoa(width)+"w")
	}
	if original > 0 {
		u, err := assetURL(value)
		if err != nil {
			return "", err
		}
		candidates = append(candidates, u+" "+strconv.Itoa(original)+"w")
	}
	return strings.Join(candidates, ", "), nil
}

// staticImageWidths returns a cached lookup of the pixel width of a static
// image, searching the site's static directory and then its themes', or 0
// when the image is missing or unreadable.
func staticImageWidths(siteRoot string, cfg config.SiteConfig) func(string) int {
	var roots []string
	if absSiteRoot, err := filepath.Abs(siteRoot); err == nil {
		if root, err := safepath.Join(absSiteRoot, cfg.StaticDir); err == nil {
			roots = append(roots, root)
		}
		if dirs, err := themes.Dirs(absSiteRoot, cfg, "static"); err == nil {
			for i := len(dirs) - 1; i >= 0; i-- {
				roots = append(roots, dirs[i])
			}
		}
	}
	var mu sync.Mutex
	widths := map[string]int{}
	return func(assetPath string) int {
		mu.Lock()
		defer mu.Unlock()
		if width, ok := widths[assetPath]; ok {
			return width
		}
		width := 0
		for _, root := range roots {
			file, err := safepath.Join(root, assetPath)
			if err != nil || safepath.RejectSymlink(file) != nil {
				continue
			}
			if f, err := os.Open(file); err == nil {
				if header, _, err := image.DecodeConfig(f); err == nil {
					width = header.Width
				}
				_ = f.Close()
				break
			}
		}
		widths[assetPath] = width
		return width
	}
}

func imageVariantPath(value string, width int) (string, error) {
	assetPath, suffix, err := splitAssetPath(value)
	if err != nil {
		return "", err
	}
	if suffix != "" {
		return "", fmt.Errorf("image variant paths must not include query strings or fragments")
	}
	ext := path.Ext(assetPath)
	if ext == "" {
		return "", fmt.Errorf("image path %q has no extension", value)
	}
	base := strings.TrimSuffix(assetPath, ext)
	return base + "." + strconv.Itoa(width) + "w" + ext, nil
}

func splitAssetPath(value string) (string, string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", "", fmt.Errorf("asset path must not be empty")
	}
	if strings.Contains(trimmed, "://") || strings.HasPrefix(trimmed, "//") {
		return "", "", fmt.Errorf("asset path must be site-relative, got %q", value)
	}

	cleanValue, suffix := route.SplitSuffix(trimmed)
	cleanValue = strings.TrimPrefix(cleanValue, "/")
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(cleanValue)))
	if cleaned == "." || cleaned == "" {
		return "", "", fmt.Errorf("asset path must not be empty")
	}
	if safepath.Escapes(cleaned) {
		return "", "", fmt.Errorf("asset path must stay inside the static directory")
	}
	return cleaned, suffix, nil
}

func joinBasePath(basePath, assetPath string) string {
	assetPath = strings.Trim(assetPath, "/")
	if basePath == "" {
		return "/" + assetPath
	}
	return strings.TrimSuffix(basePath, "/") + "/" + assetPath
}

func uniquePositiveWidths(widths []int) []int {
	seen := map[int]bool{}
	result := make([]int, 0, len(widths))
	for _, width := range widths {
		if width <= 0 || seen[width] {
			continue
		}
		seen[width] = true
		result = append(result, width)
	}
	slices.Sort(result)
	return result
}
