package templates

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
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

func imageSrcsetFunc(cfg config.SiteConfig) func(string) (string, error) {
	return func(value string) (string, error) {
		return imageSrcset(value, cfg.Pipeline.Images.Widths, cfg)
	}
}

func imagePresetSrcsetFunc(cfg config.SiteConfig) func(string, string) (string, error) {
	return func(name, value string) (string, error) {
		preset, ok := cfg.Pipeline.Images.Presets[strings.TrimSpace(name)]
		if !ok {
			return "", fmt.Errorf("unknown image preset %q", name)
		}
		return imageSrcset(value, preset.Widths, cfg)
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

func imageSrcset(value string, widths []int, cfg config.SiteConfig) (string, error) {
	assetURL := assetURLFunc(cfg)
	uniqueWidths := uniquePositiveWidths(widths)
	if len(uniqueWidths) == 0 {
		return "", fmt.Errorf("image srcset needs at least one width")
	}

	candidates := make([]string, 0, len(uniqueWidths))
	for _, width := range uniqueWidths {
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
	return strings.Join(candidates, ", "), nil
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

	suffixIndex := len(trimmed)
	for _, marker := range []string{"?", "#"} {
		if index := strings.Index(trimmed, marker); index >= 0 && index < suffixIndex {
			suffixIndex = index
		}
	}
	cleanValue := trimmed[:suffixIndex]
	suffix := trimmed[suffixIndex:]
	cleanValue = strings.TrimPrefix(cleanValue, "/")
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(cleanValue)))
	if cleaned == "." || cleaned == "" {
		return "", "", fmt.Errorf("asset path must not be empty")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || filepath.IsAbs(cleaned) {
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
