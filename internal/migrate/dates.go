package migrate

import (
	"fmt"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// dateLayouts are the date formats found in the supported generators'
// front matter, most specific first. Nida itself reads RFC 3339 and a few
// offset-free forms, so dates in any other form are rewritten.
var dateLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999-0700",
	"2006-01-02T15:04:05-0700",
	"2006-01-02 15:04:05.999999999 -0700",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05 -07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05 MST",
	"2006-01-02 15:04 -0700",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
	"2006/01/02 15:04:05",
	"2006/01/02 15:04",
	"2006/01/02",
	time.RFC1123Z,
	time.RFC1123,
	"January 2, 2006",
	"Jan 2, 2006",
	"2 January 2006",
}

// ParseDate reads a front matter date in any of the forms the supported
// generators accept. dateOnly is true when the value has no time of day.
func ParseDate(value any) (t time.Time, dateOnly bool, err error) {
	switch v := value.(type) {
	case time.Time:
		return v, false, nil
	case toml.LocalDate:
		return v.AsTime(time.UTC), true, nil
	case toml.LocalDateTime:
		return v.AsTime(time.UTC), false, nil
	case string:
		s := strings.TrimSpace(v)
		for _, layout := range dateLayouts {
			if parsed, err := time.Parse(layout, s); err == nil {
				return parsed, !strings.Contains(layout, "15:"), nil
			}
		}
		return time.Time{}, false, fmt.Errorf("unrecognized date %q", v)
	}
	return time.Time{}, false, fmt.Errorf("unrecognized date %v", value)
}

// nidaReadsDate reports whether Nida parses a date value as it is.
func nidaReadsDate(value any) bool {
	switch v := value.(type) {
	case time.Time, toml.LocalDate, toml.LocalDateTime:
		return true
	case string:
		s := strings.TrimSpace(v)
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02", "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
			if _, err := time.Parse(layout, s); err == nil {
				return true
			}
		}
	}
	return false
}

// NormalizeDate rewrites a date key into a form Nida reads, if it is not
// already in one, and reports what it did. It returns false when the value
// cannot be parsed at all; the caller should report that.
func NormalizeDate(doc *Document, key string, report *Report, file string) bool {
	value, ok := doc.Get(key)
	if !ok || nidaReadsDate(value) {
		return true
	}
	t, dateOnly, err := ParseDate(value)
	if err != nil {
		report.Add(Manual, file, 0, "Front matter `%s` has a date Nida cannot read (%v). Rewrite it as `2006-01-02` or RFC 3339.", key, value)
		return false
	}
	_ = doc.Set(key, DateValue(doc, t, dateOnly))
	report.Add(Changed, file, 0, "Rewrote `%s` dates in RFC 3339 form, which Nida reads.", key)
	return true
}

// DateValue returns t in the form that suits the document's front matter:
// a TOML date or datetime, or an RFC 3339 string for YAML.
func DateValue(doc *Document, t time.Time, dateOnly bool) any {
	if doc.Format == TOML || doc.Format == NoFrontMatter {
		if dateOnly {
			return toml.LocalDate{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
		}
		return t
	}
	if dateOnly {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339)
}
