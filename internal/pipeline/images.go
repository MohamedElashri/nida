package pipeline

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/safepath"
	"golang.org/x/image/draw"
)

// Limits that keep image processing from exhausting memory.
const (
	MaxImageDimension = 8192
	MaxImagePixels    = 60_000_000
	MaxImageBytes     = 50 << 20
)

func processImage(srcPath, relPath, outputRoot string, cfg config.SiteConfig, manifest Manifest) error {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("stat image: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse image symlink %q", srcPath)
	}
	if info.Size() > MaxImageBytes {
		return fmt.Errorf("image exceeds maximum size of %d bytes", MaxImageBytes)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}

	cfgImg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image config: %w", err)
	}
	if !ValidDimensions(cfgImg.Width, cfgImg.Height) {
		return fmt.Errorf("image dimensions %dx%d exceed limits", cfgImg.Width, cfgImg.Height)
	}

	srcImg, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)

	if cfg.Pipeline.Fingerprint {
		origFp, err := fingerprintFile(srcPath, relPath, outputRoot)
		if err != nil {
			return fmt.Errorf("fingerprint original: %w", err)
		}
		manifest[relPath] = origFp
	} else {
		dstPath, err := safepath.Join(outputRoot, relPath)
		if err != nil {
			return fmt.Errorf("resolve original image path: %w", err)
		}
		if err := safepath.EnsureNoSymlinkPath(outputRoot, dstPath); err != nil {
			return fmt.Errorf("check original image path: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return fmt.Errorf("write original: %w", err)
		}
	}

	return writeVariants(srcImg, format, outputRoot, base, ext, cfg, func(rel, path string) error {
		if !cfg.Pipeline.Fingerprint {
			return nil
		}
		fpPath, err := fingerprintFile(path, rel, outputRoot)
		if err != nil {
			return fmt.Errorf("fingerprint resized: %w", err)
		}
		_ = os.Remove(path)
		manifest[rel] = fpPath
		return nil
	})
}

// ScaleImage resizes src to exactly width by height.
func ScaleImage(src image.Image, width, height int) image.Image {
	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(resized, resized.Bounds(), src, src.Bounds(), draw.Over, nil)
	return resized
}

// EncodeImage writes img to path as "jpeg", "png", or "gif".
func EncodeImage(path string, img image.Image, format string, quality int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for resized: %w", err)
	}
	outFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create resized file: %w", err)
	}
	if quality <= 0 || quality > 100 {
		quality = 85
	}
	switch format {
	case "jpeg":
		err = jpeg.Encode(outFile, img, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(outFile, img)
	case "gif":
		err = gif.Encode(outFile, img, nil)
	default:
		_ = outFile.Close()
		return fmt.Errorf("unsupported image format %q", format)
	}
	if closeErr := outFile.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("encode resized image: %w", err)
	}
	return nil
}

// ResizeVariants writes name.<width>w.ext variants of a bundle image into
// dstDir for each configured width smaller than the original.
func ResizeVariants(srcPath, dstDir string, cfg config.SiteConfig) error {
	if !IsImageFile(srcPath) {
		return nil
	}
	info, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("stat image: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > MaxImageBytes {
		return nil
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}
	header, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || !ValidDimensions(header.Width, header.Height) {
		// Not a decodable image (or too large); copy as-is without variants.
		return nil
	}
	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png" && format != "gif") {
		// WebP and other formats can be read but not encoded.
		return nil
	}

	name := filepath.Base(srcPath)
	ext := filepath.Ext(name)
	return writeVariants(src, format, dstDir, strings.TrimSuffix(name, ext), ext, cfg, nil)
}

// writeVariants encodes base.<width>w.ext under root for each configured
// width smaller than src, calling written with the relative and full path.
func writeVariants(src image.Image, format, root, base, ext string, cfg config.SiteConfig, written func(rel, path string) error) error {
	bounds := src.Bounds()
	for _, width := range imageTargetWidths(cfg.Pipeline.Images) {
		if width <= 0 || width > MaxImageDimension {
			return fmt.Errorf("invalid image target width %d", width)
		}
		if width >= bounds.Dx() {
			continue
		}
		height := max(1, bounds.Dy()*width/bounds.Dx())
		rel := base + "." + strconv.Itoa(width) + "w" + ext
		path, err := safepath.Join(root, rel)
		if err != nil {
			return fmt.Errorf("resolve resized image path: %w", err)
		}
		if err := safepath.EnsureNoSymlinkPath(root, path); err != nil {
			return fmt.Errorf("check resized image path: %w", err)
		}
		if err := EncodeImage(path, ScaleImage(src, width, height), format, cfg.Pipeline.Images.Quality); err != nil {
			return err
		}
		if written != nil {
			if err := written(rel, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func ValidDimensions(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	if width > MaxImageDimension || height > MaxImageDimension {
		return false
	}
	return width <= MaxImagePixels/height
}

func imageTargetWidths(cfg config.ImageConfig) []int {
	seen := map[int]bool{}
	widths := make([]int, 0, len(cfg.Widths))
	for _, width := range cfg.Widths {
		if seen[width] {
			continue
		}
		seen[width] = true
		widths = append(widths, width)
	}
	for _, preset := range cfg.Presets {
		for _, width := range preset.Widths {
			if seen[width] {
				continue
			}
			seen[width] = true
			widths = append(widths, width)
		}
	}
	slices.Sort(widths)
	return widths
}
