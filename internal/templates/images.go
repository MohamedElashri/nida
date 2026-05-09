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
	"strconv"
	"strings"

	"github.com/chai2010/webp"
	"golang.org/x/image/draw"
)

func resizeImageFunc(siteRoot string) func(string, int, int, string) string {
	return func(srcPath string, width, height int, format string) string {
		// Resolve source path
		candidates := []string{
			filepath.Join(siteRoot, filepath.FromSlash(srcPath)),
			filepath.Join(siteRoot, "static", filepath.FromSlash(srcPath)),
			filepath.Join(siteRoot, "content", filepath.FromSlash(srcPath)),
		}

		var sourceFile string
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				sourceFile = c
				break
			}
		}
		if sourceFile == "" {
			return "/" + srcPath
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
		outDir := filepath.Join(siteRoot, "static", "processed_images")
		outPath := filepath.Join(outDir, outName)
		outURL := "/processed_images/" + filepath.ToSlash(outName)

		// If already processed, return existing URL
		if _, err := os.Stat(outPath); err == nil {
			return outURL
		}

		// Open and decode source image
		f, err := os.Open(sourceFile)
		if err != nil {
			return "/" + srcPath
		}
		defer f.Close()

		srcImg, _, err := image.Decode(f)
		if err != nil {
			return "/" + srcPath
		}

		// Resize
		dstRect := image.Rect(0, 0, width, height)
		dstImg := image.NewRGBA(dstRect)
		draw.CatmullRom.Scale(dstImg, dstRect, srcImg, srcImg.Bounds(), draw.Over, nil)

		// Ensure output directory exists
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return "/" + srcPath
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return "/" + srcPath
		}
		defer outFile.Close()

		// Encode based on format
		switch ext {
		case ".webp":
			err = webp.Encode(outFile, dstImg, &webp.Options{Quality: 85})
		case ".png":
			err = png.Encode(outFile, dstImg)
		case ".gif":
			err = gif.Encode(outFile, dstImg, nil)
		default:
			err = jpeg.Encode(outFile, dstImg, &jpeg.Options{Quality: 85})
		}
		if err != nil {
			return "/" + srcPath
		}

		return outURL
	}
}

func intValue(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
