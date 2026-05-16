package site

import (
	"fmt"
	"path"
	"strings"
)

func ValidatePathComponent(field, value string) error {
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

func NormalizeRoute(field, route string) (string, error) {
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

func containsUnsafeRouteRune(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}
