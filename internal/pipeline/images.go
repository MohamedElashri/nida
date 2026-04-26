package pipeline

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"golang.org/x/image/draw"
)

func processImage(srcPath, relPath, outputRoot string, cfg config.SiteConfig, manifest Manifest) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}

	srcImg, format, err := image.Decode(strings.NewReader(string(data)))
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
		dstPath := filepath.Join(outputRoot, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return fmt.Errorf("write original: %w", err)
		}
	}

	for _, targetWidth := range cfg.Pipeline.Images.Widths {
		if targetWidth >= origWidth {
			continue
		}

		targetHeight := int(float64(origHeight) * float64(targetWidth) / float64(origWidth))
		if targetHeight < 1 {
			targetHeight = 1
		}

		resized := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
		draw.BiLinear.Scale(resized, resized.Bounds(), srcImg, bounds, draw.Over, nil)

		widthSuffix := strconv.Itoa(targetWidth) + "w"
		resizedRel := base + "." + widthSuffix + ext
		resizedPath := filepath.Join(outputRoot, filepath.FromSlash(resizedRel))

		if err := os.MkdirAll(filepath.Dir(resizedPath), 0o755); err != nil {
			return fmt.Errorf("create dir for resized: %w", err)
		}

		outFile, err := os.Create(resizedPath)
		if err != nil {
			return fmt.Errorf("create resized file: %w", err)
		}

		quality := cfg.Pipeline.Images.Quality
		if quality <= 0 || quality > 100 {
			quality = 85
		}

		switch format {
		case "jpeg":
			err = jpeg.Encode(outFile, resized, &jpeg.Options{Quality: quality})
		case "png":
			err = png.Encode(outFile, resized)
		case "gif":
			err = gif.Encode(outFile, resized, nil)
		default:
			outFile.Close()
			return fmt.Errorf("unsupported image format %q", format)
		}
		outFile.Close()
		if err != nil {
			return fmt.Errorf("encode resized image: %w", err)
		}

		if cfg.Pipeline.Fingerprint {
			fpPath, err := fingerprintFile(resizedPath, resizedRel, outputRoot)
			if err != nil {
				return fmt.Errorf("fingerprint resized: %w", err)
			}
			os.Remove(resizedPath)
			manifestKey := base + "." + widthSuffix + ext
			manifest[manifestKey] = fpPath
		}
	}

	return nil
}
