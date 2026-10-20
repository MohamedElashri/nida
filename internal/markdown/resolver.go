package markdown

import (
	"strings"

	"github.com/MohamedElashri/nida/internal/route"
)

// PathLookup maps content relative paths (e.g. "posts/hello.md") to resolved URLs (e.g. "/posts/hello/").
type PathLookup map[string]string

// resolveInternalPath resolves a Zola-style @/ path to a URL using the provided lookup.
// If the path does not start with @/, or if the path is not found, the original path is returned.
func resolveInternalPath(path string, lookup PathLookup) string {
	if !strings.HasPrefix(path, "@/") {
		return path
	}

	relative, suffix := route.SplitSuffix(strings.TrimPrefix(path, "@/"))

	if url, ok := lookup[relative]; ok {
		return url + suffix
	}

	withoutExt := strings.TrimSuffix(relative, ".md")
	if url, ok := lookup[withoutExt]; ok {
		return url + suffix
	}

	return path
}

// resolveWithBasePath resolves an @/ path and prefixes the base_url path.
func resolveWithBasePath(destination string, lookup PathLookup, basePath string) string {
	resolved := resolveInternalPath(destination, lookup)
	if resolved == destination || !strings.HasPrefix(destination, "@/") {
		return resolved
	}
	return basePath + resolved
}
