package config

import "strings"

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
