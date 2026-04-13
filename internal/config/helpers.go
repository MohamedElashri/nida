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

func MainSections(cfg SiteConfig) []string {
	values, ok := cfg.Extra["main_sections"]
	if !ok {
		return nil
	}

	switch raw := values.(type) {
	case []any:
		sections := make([]string, 0, len(raw))
		for _, item := range raw {
			if value, ok := item.(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					sections = append(sections, value)
				}
			}
		}
		return sections
	case []string:
		sections := make([]string, 0, len(raw))
		for _, value := range raw {
			value = strings.TrimSpace(value)
			if value != "" {
				sections = append(sections, value)
			}
		}
		return sections
	default:
		return nil
	}
}
