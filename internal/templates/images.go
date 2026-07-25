package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/safepath"
	"golang.org/x/image/draw"
)

const (
	maxResizeDimension = 8192
	maxResizePixels    = 60_000_000
	maxImageBytes      = 50 << 20
)

func resizeImageFunc(siteRoot string) func(string, int, int, string) string {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		absSiteRoot = siteRoot
	}
	return func(srcPath string, width, height int, format string) string {
		if !validResizeDimensions(width, height) {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		candidateRoots := []string{
			absSiteRoot,
			filepath.Join(absSiteRoot, "static"),
			filepath.Join(absSiteRoot, "content"),
		}

		var sourceFile string
		for _, root := range candidateRoots {
			c, err := safepath.Join(root, srcPath)
			if err != nil {
				continue
			}
			if err := safepath.EnsureNoSymlinkPath(absSiteRoot, c); err != nil {
				continue
			}
			if err := safepath.RejectSymlink(c); err == nil {
				sourceFile = c
				break
			}
		}
		if sourceFile == "" {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		// Compute output filename
		hashInput := fmt.Sprintf("%s|%d|%d|%s", srcPath, width, height, format)
		hash := sha256.Sum256([]byte(hashInput))
		hashStr := hex.EncodeToString(hash[:])[:16]

		ext := strings.ToLower(format)
		if ext == "" {
			ext = filepath.Ext(sourceFile)
			if ext == "" {
				ext = ".jpg"
			}
		} else if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}

		baseName := strings.TrimSuffix(filepath.Base(sourceFile), filepath.Ext(sourceFile))
		outName := fmt.Sprintf("%s.%s%s", baseName, hashStr, ext)
		outDir := filepath.Join(absSiteRoot, "static", "processed_images")
		outPath := filepath.Join(outDir, outName)
		outURL := "/processed_images/" + filepath.ToSlash(outName)
		if err := safepath.EnsureNoSymlinkPath(absSiteRoot, outPath); err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		// If already processed, return existing URL
		if _, err := os.Stat(outPath); err == nil {
			return outURL
		}

		// Open and decode source image
		f, err := os.Open(sourceFile)
		if err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}
		defer func() { _ = f.Close() }()

		if info, err := f.Stat(); err != nil || info.Size() > maxImageBytes {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}
		cfg, _, err := image.DecodeConfig(f)
		if err != nil || !validResizeDimensions(cfg.Width, cfg.Height) {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}
		if _, err := f.Seek(0, 0); err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		srcImg, _, err := image.Decode(f)
		if err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		// Resize
		dstRect := image.Rect(0, 0, width, height)
		dstImg := image.NewRGBA(dstRect)
		draw.CatmullRom.Scale(dstImg, dstRect, srcImg, srcImg.Bounds(), draw.Over, nil)

		// Ensure output directory exists
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}
		defer func() { _ = outFile.Close() }()

		// Encode based on format
		switch ext {
		case ".png":
			err = png.Encode(outFile, dstImg)
		case ".gif":
			err = gif.Encode(outFile, dstImg, nil)
		default:
			err = jpeg.Encode(outFile, dstImg, &jpeg.Options{Quality: 85})
		}
		if err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		return outURL
	}
}

func validResizeDimensions(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	if width > maxResizeDimension || height > maxResizeDimension {
		return false
	}
	return width <= maxResizePixels/height
}
