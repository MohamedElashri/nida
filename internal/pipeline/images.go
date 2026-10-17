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

const (
	maxPipelineImageDimension = 8192
	maxPipelineImagePixels    = 60_000_000
	maxPipelineImageBytes     = 50 << 20
)

func processImage(srcPath, relPath, outputRoot string, cfg config.SiteConfig, manifest Manifest) error {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("stat image: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse image symlink %q", srcPath)
	}
	if info.Size() > maxPipelineImageBytes {
		return fmt.Errorf("image exceeds maximum size of %d bytes", maxPipelineImageBytes)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}

	cfgImg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image config: %w", err)
	}
	if !validPipelineImageDimensions(cfgImg.Width, cfgImg.Height) {
		return fmt.Errorf("image dimensions %dx%d exceed limits", cfgImg.Width, cfgImg.Height)
	}

	srcImg, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	bounds := srcImg.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

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

	for _, targetWidth := range imageTargetWidths(cfg.Pipeline.Images) {
		if targetWidth <= 0 || targetWidth > maxPipelineImageDimension {
			return fmt.Errorf("invalid image target width %d", targetWidth)
		}
		if targetWidth >= origWidth {
			continue
		}

		targetHeight := int(float64(origHeight) * float64(targetWidth) / float64(origWidth))
		if targetHeight < 1 {
			targetHeight = 1
		}

		resized := scaleImage(srcImg, targetWidth, targetHeight)

		widthSuffix := strconv.Itoa(targetWidth) + "w"
		resizedRel := base + "." + widthSuffix + ext
		resizedPath, err := safepath.Join(outputRoot, resizedRel)
		if err != nil {
			return fmt.Errorf("resolve resized image path: %w", err)
		}
		if err := safepath.EnsureNoSymlinkPath(outputRoot, resizedPath); err != nil {
			return fmt.Errorf("check resized image path: %w", err)
		}

		if err := encodeImage(resizedPath, resized, format, cfg.Pipeline.Images.Quality); err != nil {
			return err
		}

		if cfg.Pipeline.Fingerprint {
			fpPath, err := fingerprintFile(resizedPath, resizedRel, outputRoot)
			if err != nil {
				return fmt.Errorf("fingerprint resized: %w", err)
			}
			_ = os.Remove(resizedPath)
			manifestKey := base + "." + widthSuffix + ext
			manifest[manifestKey] = fpPath
		}
	}

	return nil
}

func scaleImage(src image.Image, width, height int) image.Image {
	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.BiLinear.Scale(resized, resized.Bounds(), src, src.Bounds(), draw.Over, nil)
	return resized
}

func encodeImage(path string, img image.Image, format string, quality int) error {
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
	if !info.Mode().IsRegular() || info.Size() > maxPipelineImageBytes {
		return nil
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}
	header, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || !validPipelineImageDimensions(header.Width, header.Height) {
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
	base := strings.TrimSuffix(name, ext)
	for _, width := range imageTargetWidths(cfg.Pipeline.Images) {
		if width <= 0 || width >= header.Width || width > maxPipelineImageDimension {
			continue
		}
		height := max(1, int(float64(header.Height)*float64(width)/float64(header.Width)))
		dst, err := safepath.Join(dstDir, base+"."+strconv.Itoa(width)+"w"+ext)
		if err != nil {
			return err
		}
		if err := safepath.EnsureNoSymlinkPath(dstDir, dst); err != nil {
			return err
		}
		if err := encodeImage(dst, scaleImage(src, width, height), format, cfg.Pipeline.Images.Quality); err != nil {
			return err
		}
	}
	return nil
}

// Widths returns the variant widths the pipeline generates, sorted.
func Widths(cfg config.SiteConfig) []int {
	return imageTargetWidths(cfg.Pipeline.Images)
}

func validPipelineImageDimensions(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	if width > maxPipelineImageDimension || height > maxPipelineImageDimension {
		return false
	}
	return width <= maxPipelineImagePixels/height
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
