package markdown

import (
	"strings"
)

// PathLookup maps content relative paths (e.g. "posts/hello.md") to resolved URLs (e.g. "/posts/hello/").
type PathLookup map[string]string

// ResolveInternalPath resolves a Zola-style @/ path to a URL using the provided lookup.
// If the path does not start with @/, or if the path is not found, the original path is returned.
func ResolveInternalPath(path string, lookup PathLookup) string {
	if !strings.HasPrefix(path, "@/") {
		return path
	}

	relative := strings.TrimPrefix(path, "@/")

	if url, ok := lookup[relative]; ok {
		return url
	}

	withoutExt := strings.TrimSuffix(relative, ".md")
	if url, ok := lookup[withoutExt]; ok {
		return url
	}

	return path
}
