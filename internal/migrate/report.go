package migrate

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Severity says how much attention a report note needs.
type Severity int

const (
	// Changed notes something the migrator converted on its own.
	Changed Severity = iota
	// Review notes a conversion that probably works but is worth a look.
	Review
	// Manual notes something left as it was that must be fixed by hand.
	Manual
	// Unsupported notes a feature Nida has no equivalent for.
	Unsupported
)

var severityTitles = map[Severity]string{
	Changed:     "Changed automatically",
	Review:      "Needs review",
	Manual:      "Needs manual work",
	Unsupported: "Not supported by Nida",
}

var severityHelp = map[Severity]string{
	Changed:     "The migrator converted these. They are listed so you know what changed.",
	Review:      "These were converted, but the result may not match the old site exactly.",
	Manual:      "These were left in place. `nida check` fails on most of them until they are fixed.",
	Unsupported: "Nida has no equivalent for these, so they were dropped or kept only for reference.",
}

func (s Severity) String() string {
	switch s {
	case Changed:
		return "changed"
	case Review:
		return "review"
	case Manual:
		return "manual"
	case Unsupported:
		return "unsupported"
	}
	return "unknown"
}

// Note is one entry in the migration report. File is a source-relative
// path in forward-slash form, empty for site-wide notes; Line is 1-based,
// or 0 when the note is about a whole file.
type Note struct {
	Severity Severity
	File     string
	Line     int
	Message  string
}

// Report collects notes while a site is planned and migrated.
type Report struct {
	Source string
	From   string
	Dest   string
	Notes  []Note
	// Stats counts what was written.
	Pages, Sections, Copied, Templates int
	// URLsKept counts pages whose new URL matches the old one; URLsMoved
	// counts those given a path or alias to match it.
	URLsKept, URLsMoved int
	// Templates records which template set the site ended up with.
	TemplateSet string
	// Check is the result of building the migrated site, or empty if
	// the build was not run.
	Check string
}

// Add records a note. Identical notes are kept once.
func (r *Report) Add(sev Severity, file string, line int, format string, args ...any) {
	note := Note{Severity: sev, File: file, Line: line, Message: fmt.Sprintf(format, args...)}
	if slices.Contains(r.Notes, note) {
		return
	}
	r.Notes = append(r.Notes, note)
}

// Count returns how many notes have the given severity.
func (r *Report) Count(sev Severity) int {
	n := 0
	for _, note := range r.Notes {
		if note.Severity == sev {
			n++
		}
	}
	return n
}

// Markdown renders the report as the MIGRATION.md file written into the
// migrated site. Notes with the same message are grouped, so a rename
// applied to two hundred files is one entry with a file list.
func (r *Report) Markdown() string {
	var b strings.Builder
	b.WriteString("# Migration report\n\n")
	fmt.Fprintf(&b, "Migrated from %s by `nida migrate`.\n\n", r.From)
	fmt.Fprintf(&b, "- Source: `%s`\n", r.Source)
	fmt.Fprintf(&b, "- Pages: %d, sections: %d, files copied: %d\n", r.Pages, r.Sections, r.Copied)
	fmt.Fprintf(&b, "- URLs: %d unchanged, %d kept with a path or alias\n", r.URLsKept, r.URLsMoved)
	if r.TemplateSet != "" {
		fmt.Fprintf(&b, "- Templates: %s\n", r.TemplateSet)
	}
	if r.Check != "" {
		fmt.Fprintf(&b, "- `nida check`: %s\n", r.Check)
	}
	b.WriteString("\n| Severity | Notes |\n| --- | --- |\n")
	for _, sev := range []Severity{Manual, Unsupported, Review, Changed} {
		fmt.Fprintf(&b, "| %s | %d |\n", severityTitles[sev], r.Count(sev))
	}
	b.WriteString("\nWork through the sections from the top. When `nida check` passes, compare the old and new `sitemap.xml` files to confirm every URL still exists.\n")

	for _, sev := range []Severity{Manual, Unsupported, Review, Changed} {
		groups := r.groups(sev)
		if len(groups) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n%s\n\n", severityTitles[sev], severityHelp[sev])
		for _, g := range groups {
			fmt.Fprintf(&b, "- %s", g.message)
			switch {
			case len(g.places) == 0:
				b.WriteString("\n")
			case len(g.places) == 1:
				fmt.Fprintf(&b, " (`%s`)\n", g.places[0])
			default:
				fmt.Fprintf(&b, " (%d places)\n", len(g.places))
				for _, place := range g.places {
					fmt.Fprintf(&b, "  - `%s`\n", place)
				}
			}
		}
	}
	return b.String()
}

type noteGroup struct {
	message string
	places  []string
}

func (r *Report) groups(sev Severity) []noteGroup {
	var notes []Note
	for _, note := range r.Notes {
		if note.Severity == sev {
			notes = append(notes, note)
		}
	}
	slices.SortStableFunc(notes, func(a, b Note) int {
		return cmp.Or(strings.Compare(a.File, b.File), cmp.Compare(a.Line, b.Line))
	})
	index := map[string]int{}
	var groups []noteGroup
	for _, note := range notes {
		i, ok := index[note.Message]
		if !ok {
			i = len(groups)
			index[note.Message] = i
			groups = append(groups, noteGroup{message: note.Message})
		}
		if place := note.place(); place != "" {
			groups[i].places = append(groups[i].places, place)
		}
	}
	return groups
}

func (n Note) place() string {
	if n.File == "" {
		return ""
	}
	if n.Line > 0 {
		return fmt.Sprintf("%s:%d", n.File, n.Line)
	}
	return n.File
}
