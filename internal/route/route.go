// Package route validates and builds site URLs.
package route

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// ValidateComponent checks a single URL path segment such as a slug.
func ValidateComponent(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	if containsUnsafeRouteRune(value) {
		return fmt.Errorf("%s contains unsafe characters", field)
	}
	if strings.ContainsAny(value, `/\?#`) {
		return fmt.Errorf("%s must be a single URL path component", field)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("%s must not be %q", field, value)
	}
	return nil
}

// Normalize cleans a route and rejects unsafe characters and segments.
func Normalize(field, route string) (string, error) {
	route = strings.TrimSpace(route)
	if route == "" {
		return "", fmt.Errorf("%s must not be empty", field)
	}
	if containsUnsafeRouteRune(route) {
		return "", fmt.Errorf("%s contains unsafe characters", field)
	}
	if strings.ContainsAny(route, `\?#`) {
		return "", fmt.Errorf("%s must be a URL path without query, fragment, or backslash", field)
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}

	trimmed := strings.Trim(route, "/")
	if trimmed != "" {
		for _, segment := range strings.Split(trimmed, "/") {
			if segment == "" || segment == "." || segment == ".." {
				return "", fmt.Errorf("%s contains unsafe path segment %q", field, segment)
			}
		}
	}

	trailingSlash := strings.HasSuffix(route, "/")
	route = path.Clean(route)
	if route == "." {
		route = "/"
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	if trailingSlash && route != "/" {
		route += "/"
	}
	return route, nil
}

// Page normalizes a page or alias route. A route whose last segment has a
// file extension, such as /2024/05/01/post.html, names that exact file and
// keeps no trailing slash; any other route gets one and is written as
// index.html inside it.
func Page(field, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%s must not be empty", field)
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	if !IsFile(raw) && !strings.HasSuffix(raw, "/") {
		raw += "/"
	}
	return Normalize(field, raw)
}

// IsFile reports whether a route names a file, like /feed.xml or
// /post.html, rather than a directory.
func IsFile(route string) bool {
	if strings.HasSuffix(route, "/") {
		return false
	}
	return path.Ext(path.Base(route)) != ""
}

// Dir returns the directory a route's relative links resolve against:
// the route itself for /post/, and the parent for /post.html.
func Dir(route string) string {
	if !IsFile(route) {
		return route
	}
	dir := path.Dir(route)
	if dir == "/" {
		return dir
	}
	return dir + "/"
}

func containsUnsafeRouteRune(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

// CanonicalURL joins a route onto baseURL, keeping a trailing slash.
func CanonicalURL(baseURL, route string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return route
	}
	base.Path = path.Join(base.Path, route)
	if strings.HasSuffix(route, "/") && !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	return base.String()
}
