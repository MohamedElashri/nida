package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CleanRelative(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	return filepath.Clean(value)
}

func ValidateRelative(field, value string, allowDot bool) error {
	value = CleanRelative(value)
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	if filepath.IsAbs(value) {
		return fmt.Errorf("%s must be relative", field)
	}
	if value == "." && !allowDot {
		return fmt.Errorf("%s must not resolve to the current directory", field)
	}
	if value == ".." || strings.HasPrefix(value, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s must not escape its root", field)
	}
	return nil
}

func Join(root, relative string) (string, error) {
	if strings.TrimSpace(relative) == "" {
		return "", fmt.Errorf("relative path is required")
	}
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("absolute path %q is not allowed", relative)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}

	cleanRel := filepath.Clean(filepath.FromSlash(relative))
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes root %q", relative, absRoot)
	}

	joined := filepath.Join(absRoot, cleanRel)
	relToRoot, err := filepath.Rel(absRoot, joined)
	if err != nil {
		return "", fmt.Errorf("check path %q under %q: %w", joined, absRoot, err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) || filepath.IsAbs(relToRoot) {
		return "", fmt.Errorf("path %q escapes root %q", relative, absRoot)
	}

	return joined, nil
}

func RejectSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse symlink path %q", path)
	}
	return nil
}

func EnsureNoSymlinkPath(root, path string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root %q: %w", root, err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", path, err)
	}

	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return fmt.Errorf("check path %q under %q: %w", absPath, absRoot, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path %q escapes root %q", absPath, absRoot)
	}
	if err := RejectSymlink(absRoot); err != nil && !os.IsNotExist(err) {
		return err
	}
	if rel == "." {
		return nil
	}

	current := absRoot
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		if err := RejectSymlink(current); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
	}
	return nil
}
