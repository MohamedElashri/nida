package frontmatter

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"
)

var knownFields = map[string]bool{
	"title":           true,
	"description":     true,
	"date":            true,
	"updated":         true,
	"draft":           true,
	"weight":          true,
	"slug":            true,
	"template":        true,
	"page_template":   true,
	"sort_by":         true,
	"transparent":     true,
	"generate_feeds":  true,
	"paginate_by":     true,
	"paginate_path":   true,
	"extra":           true,
}

func decodeMetadata(values map[string]any) (Metadata, error) {
	meta := Metadata{
		Title:          stringValue(values["title"]),
		Description:    stringValue(values["description"]),
		Date:           timeValue(values["date"]),
		Updated:        timeValue(values["updated"]),
		Draft:          boolValue(values["draft"]),
		Weight:         intValue(values["weight"]),
		Slug:           stringValue(values["slug"]),
		Template:       stringValue(values["template"]),
		PageTemplate:   stringValue(values["page_template"]),
		SortBy:         stringValue(values["sort_by"]),
		Transparent:    boolValue(values["transparent"]),
		GenerateFeeds:   boolValue(values["generate_feeds"]),
		PaginateBy:     intValue(values["paginate_by"]),
		PaginatePath:   stringValue(values["paginate_path"]),
	}

	if extra, ok := mapValue(values["extra"]); ok {
		meta.Extra = extra
	}

	for key, value := range values {
		if knownFields[key] {
			continue
		}
		if meta.Extra == nil {
			meta.Extra = make(map[string]any)
		}
		meta.Extra[key] = value
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
