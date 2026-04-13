package frontmatter

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"
)

func decodeMetadata(values map[string]any) (Metadata, error) {
	meta := Metadata{
		Title:        stringValue(values["title"]),
		Date:         timeValue(values["date"]),
		Draft:        boolValue(values["draft"]),
		Tags:         stringSliceValue(values["tags"]),
		Categories:   stringSliceValue(values["categories"]),
		Description:  stringValue(values["description"]),
		Slug:         stringValue(values["slug"]),
		Template:     stringValue(values["template"]),
		PageTemplate: stringValue(values["page_template"]),
		PaginateBy:   intValue(values["paginate_by"]),
	}

	if extra, ok := mapValue(values["extra"]); ok {
		meta.Extra = extra
	}

	if _, ok := values["date"]; ok && meta.Date.IsZero() {
		return Metadata{}, fmt.Errorf("unsupported date value %v", values["date"])
	}

	return meta, nil
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func boolValue(value any) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

func intValue(value any) int {
	switch v := value.(type) {
	case int64:
		return int(v)
	case int32:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func stringSliceValue(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]string); ok {
			return append([]string(nil), typed...)
		}
		return nil
	}

	items := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			items = append(items, strings.TrimSpace(s))
		}
	}
	return items
}

func timeValue(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case string:
		return parseTimeString(v)
	case fmt.Stringer:
		return parseTimeString(v.String())
	default:
		rv := reflect.ValueOf(value)
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			if method := rv.MethodByName("AsTime"); method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 {
				if out, ok := method.Call(nil)[0].Interface().(time.Time); ok {
					return out
				}
			}
		}
		return time.Time{}
	}
}

func parseTimeString(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05.999999999",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}

	return time.Time{}
}

func mapValue(value any) (map[string]any, bool) {
	switch v := value.(type) {
	case map[string]any:
		return maps.Clone(v), true
	default:
		return nil, false
	}
}
