package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/MohamedElashri/nida/internal/pipeline"
	"github.com/MohamedElashri/nida/internal/safepath"
)

var resizeMu sync.Mutex

func resizeImageFunc(siteRoot string, cfg config.SiteConfig) func(string, int, int, string) string {
	absSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		absSiteRoot = siteRoot
	}
	return func(srcPath string, width, height int, format string) string {
		// Pages render concurrently and may resize the same image; the lock
		// keeps two writers from producing the same output file at once.
		resizeMu.Lock()
		defer resizeMu.Unlock()
		if !pipeline.ValidDimensions(width, height) {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		candidateRoots := []string{
			absSiteRoot,
			filepath.Join(absSiteRoot, cfg.StaticDir),
			filepath.Join(absSiteRoot, cfg.ContentDir),
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
		outDir := filepath.Join(absSiteRoot, cfg.StaticDir, "processed_images")
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

		if info, err := f.Stat(); err != nil || info.Size() > pipeline.MaxImageBytes {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}
		header, _, err := image.DecodeConfig(f)
		if err != nil || !pipeline.ValidDimensions(header.Width, header.Height) {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}
		if _, err := f.Seek(0, 0); err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		srcImg, _, err := image.Decode(f)
		if err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		encoding := "jpeg"
		switch ext {
		case ".png":
			encoding = "png"
		case ".gif":
			encoding = "gif"
		}
		if err := pipeline.EncodeImage(outPath, pipeline.ScaleImage(srcImg, width, height), encoding, cfg.Pipeline.Images.Quality); err != nil {
			return "/" + strings.TrimPrefix(srcPath, "/")
		}

		return outURL
	}
}
