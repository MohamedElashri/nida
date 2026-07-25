package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/safepath"
)

func fingerprintFile(srcPath, relPath, outputRoot string) (string, error) {
	if err := safepath.RejectSymlink(srcPath); err != nil {
		return "", fmt.Errorf("check source file: %w", err)
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])[:8]

	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	fpRelPath := base + "." + hashStr + ext

	dstPath, err := safepath.Join(outputRoot, fpRelPath)
	if err != nil {
		return "", fmt.Errorf("resolve fingerprint path: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(outputRoot, dstPath); err != nil {
		return "", fmt.Errorf("check fingerprint path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fpRelPath, nil
}

func writeManifest(outputRoot string, manifest Manifest) error {
	if len(manifest) == 0 {
		return nil
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	manifestPath, err := safepath.Join(outputRoot, "manifest.json")
	if err != nil {
		return fmt.Errorf("resolve manifest path: %w", err)
	}
	if err := safepath.EnsureNoSymlinkPath(outputRoot, manifestPath); err != nil {
		return fmt.Errorf("check manifest path: %w", err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	if err := safepath.RejectSymlink(src); err != nil {
		return fmt.Errorf("check src: %w", err)
	}
	if err := safepath.RejectSymlink(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check dst: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return nil
}
