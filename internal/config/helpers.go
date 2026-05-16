package config

import (
	"net/url"
	"path"
	"strings"
)

func DocumentDirection(language string) string {
	primary := strings.ToLower(strings.TrimSpace(language))
	if primary == "" {
		return "ltr"
	}

	if index := strings.IndexAny(primary, "-_"); index >= 0 {
		primary = primary[:index]
	}

	switch primary {
	case "ar", "fa", "he", "ur", "ps", "sd", "ug", "yi":
		return "rtl"
	default:
		return "ltr"
	}
}

func BasePath(baseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return ""
	}

	clean := path.Clean("/" + strings.Trim(parsed.EscapedPath(), "/"))
	if clean == "/" || clean == "." {
		return ""
	}
	return clean
}
