package migrate

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Jekyll's URL rules, from jekyll/url.rb, jekyll/drops/url_drop.rb,
// jekyll/utils.rb, and jekyll/page.rb (Jekyll 4.3).

var (
	jekyllSlugDefault = regexp.MustCompile(`[^\p{M}\p{L}\p{Nd}]+`)
	jekyllSlugPretty  = regexp.MustCompile(`[^\p{M}\p{L}\p{Nd}._~!$&'()+,;=@]+`)
	jekyllSlugASCII   = regexp.MustCompile(`[^A-Za-z0-9]+`)
	jekyllSlugRaw     = regexp.MustCompile(`\s+`)
	jekyllPlaceholder = regexp.MustCompile(`:([a-z_]+)`)
)

// jekyllSlugify ports Jekyll's Utils.slugify for the raw, default, pretty,
// and ascii modes.
func jekyllSlugify(s, mode string, cased bool) string {
	var re *regexp.Regexp
	switch mode {
	case "raw":
		re = jekyllSlugRaw
	case "pretty":
		re = jekyllSlugPretty
	case "ascii":
		re = jekyllSlugASCII
	case "default", "":
		re = jekyllSlugDefault
	default:
		if cased {
			return s
		}
		return strings.ToLower(s)
	}
	slug := re.ReplaceAllString(s, "-")
	slug = strings.TrimPrefix(slug, "-")
	slug = strings.TrimSuffix(slug, "-")
	if !cased {
		slug = strings.ToLower(slug)
	}
	return slug
}

// jekyllTitleize ports Utils.titleize_slug: "my-post" becomes "My Post".
func jekyllTitleize(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// jekyllStyles are the built-in permalink styles for posts.
var jekyllStyles = map[string]string{
	"date":     "/:categories/:year/:month/:day/:title:output_ext",
	"pretty":   "/:categories/:year/:month/:day/:title/",
	"ordinal":  "/:categories/:year/:y_day/:title:output_ext",
	"weekdate": "/:categories/:year/W:week/:short_day/:title:output_ext",
	"none":     "/:categories/:title:output_ext",
}

// jekyllPostTemplate returns the posts URL template for a permalink
// setting, which is a style name or a pattern.
func jekyllPostTemplate(permalink string) string {
	if permalink == "" {
		permalink = "date"
	}
	if t, ok := jekyllStyles[permalink]; ok {
		return t
	}
	return permalink
}

// jekyllAddSuffix ports Utils.add_permalink_suffix, which gives pages and
// collections a trailing slash or :output_ext to match the site style.
func jekyllAddSuffix(template, permalink string) string {
	switch permalink {
	case "pretty":
		return template + "/"
	case "", "date", "ordinal", "none", "weekdate":
		return template + ":output_ext"
	}
	if strings.HasSuffix(permalink, "/") {
		template += "/"
	}
	if strings.HasSuffix(permalink, ":output_ext") {
		template += ":output_ext"
	}
	return template
}

// jekyllURL expands placeholders in a template and cleans the result the
// way URL#sanitize_url does. Unknown placeholders are left as text.
func jekyllURL(template string, values map[string]string) string {
	out := jekyllPlaceholder.ReplaceAllStringFunc(template, func(m string) string {
		if v, ok := values[m[1:]]; ok {
			return v
		}
		return m
	})
	out = "/" + out
	out = strings.ReplaceAll(out, "..", "/")
	out = strings.ReplaceAll(out, "./", "")
	for strings.Contains(out, "//") {
		out = strings.ReplaceAll(out, "//", "/")
	}
	return out
}

// jekyllServedURL is the URL a reader uses for a generated URL. Jekyll
// writes a URL without a trailing slash or extension as that name plus
// the output extension, so /about is the file /about.html.
func jekyllServedURL(u, outputExt string) string {
	if strings.HasSuffix(u, "/") || path.Ext(path.Base(u)) != "" {
		return u
	}
	return u + outputExt
}

// jekyllDateValues returns the date placeholders for t.
func jekyllDateValues(t time.Time, values map[string]string) {
	year, week := t.ISOWeek()
	values["year"] = t.Format("2006")
	values["month"] = t.Format("01")
	values["day"] = t.Format("02")
	values["hour"] = t.Format("15")
	values["minute"] = t.Format("04")
	values["second"] = t.Format("05")
	values["i_day"] = strconv.Itoa(t.Day())
	values["i_month"] = strconv.Itoa(int(t.Month()))
	values["short_month"] = t.Format("Jan")
	values["long_month"] = t.Format("January")
	values["short_day"] = t.Format("Mon")
	values["long_day"] = t.Format("Monday")
	values["short_year"] = t.Format("06")
	values["y_day"] = fmt.Sprintf("%03d", t.YearDay())
	values["w_year"] = strconv.Itoa(year)
	values["week"] = fmt.Sprintf("%02d", week)
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	values["w_day"] = strconv.Itoa(wd)
}

// jekyllCategoriesPath joins categories the way UrlDrop#categories does:
// lowercased, de-duplicated, and separated by slashes.
func jekyllCategoriesPath(categories []string) string {
	seen := map[string]bool{}
	var parts []string
	for _, c := range categories {
		c = strings.ToLower(c)
		if !seen[c] {
			seen[c] = true
			parts = append(parts, c)
		}
	}
	return strings.Join(parts, "/")
}

// nidaPattern translates a Jekyll posts or collection template into a
// Nida permalink pattern. slugToken names the Jekyll placeholder that
// becomes {slug}, so each page's slug can be set to its value. ok is false
// when the template uses something Nida cannot express; pages then keep
// their URLs through `path`. categories is the Nida replacement for
// :categories: "" drops it, "{categories}" uses the taxonomy.
func nidaPattern(template, collection, categories string) (pattern, slugToken string, ok bool) {
	ok = true
	pattern = jekyllPlaceholder.ReplaceAllStringFunc(template, func(m string) string {
		switch name := m[1:]; name {
		case "year", "month", "day":
			return "{" + name + "}"
		case "output_ext":
			return ".html"
		case "collection":
			return collection
		case "categories":
			return categories
		case "title", "slug", "name":
			if slugToken != "" && slugToken != name {
				ok = false
			}
			slugToken = name
			return "{slug}"
		}
		ok = false
		return m
	})
	if !ok || slugToken == "" {
		return "", "", false
	}
	pattern = "/" + pattern
	for strings.Contains(pattern, "//") {
		pattern = strings.ReplaceAll(pattern, "//", "/")
	}
	return pattern, slugToken, true
}
