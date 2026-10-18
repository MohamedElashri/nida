package migrate

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// pelicanSlugger reproduces Pelican's slugify: transliterate to ASCII
// (unidecode), apply SLUG_REGEX_SUBSTITUTIONS, then lowercase.
type pelicanSlugger struct {
	subs         []pelicanSub
	useUnicode   bool
	preserveCase bool
}

type pelicanSub struct {
	re   *regexp.Regexp
	repl string
}

// Pelican's default SLUG_REGEX_SUBSTITUTIONS.
var defaultPelicanSubs = []pelicanSub{
	{regexp.MustCompile(`[^\w\s-]`), ""},
	{regexp.MustCompile(`\A\s*`), ""},
	{regexp.MustCompile(`\s*\z`), ""},
	{regexp.MustCompile(`[-\s]+`), "-"},
}

// unicodePelicanSubs are the default substitutions with Python's Unicode
// \w, for SLUGIFY_USE_UNICODE; Go's \w is ASCII only.
var unicodePelicanSubs = []pelicanSub{
	{regexp.MustCompile(`[^\p{L}\p{N}_\s-]`), ""},
	{regexp.MustCompile(`\A\s*`), ""},
	{regexp.MustCompile(`\s*\z`), ""},
	{regexp.MustCompile(`[-\s]+`), "-"},
}

// slugify returns the slug and whether it is exact. It is not exact when
// the text has characters this transliteration table does not cover,
// which unidecode would romanize.
func (s pelicanSlugger) slugify(text string) (string, bool) {
	exact := true
	if !s.useUnicode {
		text, exact = transliterate(text)
	}
	subs := s.subs
	if subs == nil {
		subs = defaultPelicanSubs
	}
	for _, sub := range subs {
		text = sub.re.ReplaceAllString(text, sub.repl)
	}
	if !s.preserveCase {
		text = strings.ToLower(text)
	}
	return text, exact
}

// transliterate approximates unidecode for Latin, Greek, and Cyrillic.
func transliterate(text string) (string, bool) {
	var b strings.Builder
	exact := true
	for _, r := range text {
		switch {
		case r < 0x80:
			b.WriteRune(r)
		case translit[r] != "":
			b.WriteString(translit[r])
		case unicode.Is(unicode.Mn, r):
			// combining marks vanish
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			// unidecode maps most punctuation to ASCII punctuation, which
			// the default substitutions then drop anyway.
			if s, ok := punctTranslit[r]; ok {
				b.WriteString(s)
			}
		default:
			exact = false
		}
	}
	return b.String(), exact
}

var punctTranslit = map[rune]string{
	'–': "-", '—': "--", '‘': "'", '’': "'", '“': `"`, '”': `"`, '…': "...", '«': "<<", '»': ">>",
}

var translit = func() map[rune]string {
	m := map[rune]string{}
	add := func(chars, ascii string) {
		parts := strings.Split(ascii, " ")
		for i, r := range []rune(chars) {
			m[r] = parts[i]
		}
	}
	add("ÀÁÂÃÄÅàáâãäåĀāĂăĄą", "A A A A A A a a a a a a A a A a A a")
	add("ÇçĆćĈĉĊċČč", "C c C c C c C c C c")
	add("ĎďĐđÐð", "D d D d D d")
	add("ÈÉÊËèéêëĒēĔĕĖėĘęĚě", "E E E E e e e e E e E e E e E e E e")
	add("ĜĝĞğĠġĢģ", "G g G g G g G g")
	add("ĤĥĦħ", "H h H h")
	add("ÌÍÎÏìíîïĨĩĪīĬĭĮįİı", "I I I I i i i i I i I i I i I i I i")
	add("ĴĵĶķĹĺĻļĽľĿŀŁł", "J j K k L l L l L l L l L l")
	add("ÑñŃńŅņŇňŉŊŋ", "N n N n N n N n 'n N n")
	add("ÒÓÔÕÖØòóôõöøŌōŎŏŐő", "O O O O O O o o o o o o O o O o O o")
	add("ŔŕŖŗŘřŚśŜŝŞşŠšȘș", "R r R r R r S s S s S s S s S s")
	add("ŢţŤťŦŧȚț", "T t T t T t T t")
	add("ÙÚÛÜùúûüŨũŪūŬŭŮůŰűŲų", "U U U U u u u u U u U u U u U u U u U u")
	add("ŴŵÝýÿŶŷŸŹźŻżŽž", "W w Y y y Y y Y Z z Z z Z z")
	add("ÆæŒœßÞþ", "AE ae OE oe ss Th th")
	add("ΑΒΓΔΕΖΗΘΙΚΛΜΝΞΟΠΡΣΤΥΦΧΨΩ", "A B G D E Z E Th I K L M N Ks O P R S T Y Ph Kh Ps O")
	add("αβγδεζηθικλμνξοπρσςτυφχψω", "a b g d e z e th i k l m n x o p r s s t u ph kh ps o")
	add("άέήίόύώΆΈΉΊΌΎΏ", "a e e i o u o A E E I O U O")
	add("АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ", "A B V G D E Io Zh Z I I K L M N O P R S T U F Kh Ts Ch Sh Shch ' Y ' E Iu Ia")
	add("абвгдеёжзийклмнопрстуфхцчшщъыьэюя", "a b v g d e io zh z i i k l m n o p r s t u f kh ts ch sh shch ' y ' e iu ia")
	return m
}()

// pelicanFormat evaluates a Pelican URL or SAVE_AS pattern, which uses
// Python str.format: {slug}, {category}, {date:%Y}, and so on. It returns
// an error naming any field it cannot fill.
func pelicanFormat(pattern string, fields map[string]string, date time.Time) (string, error) {
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '{' && i+1 < len(pattern) && pattern[i+1] == '{' {
			b.WriteByte('{')
			i++
			continue
		}
		if c == '}' && i+1 < len(pattern) && pattern[i+1] == '}' {
			b.WriteByte('}')
			i++
			continue
		}
		if c != '{' {
			b.WriteByte(c)
			continue
		}
		end := strings.IndexByte(pattern[i:], '}')
		if end < 0 {
			return "", fmt.Errorf("unclosed { in %q", pattern)
		}
		field := pattern[i+1 : i+end]
		i += end
		name, spec, _ := strings.Cut(field, ":")
		if name == "date" || name == "modified" {
			if date.IsZero() {
				return "", fmt.Errorf("the page has no date for {%s}", field)
			}
			if spec == "" {
				b.WriteString(date.Format("2006-01-02 15:04:05"))
			} else {
				b.WriteString(strftime(spec, date))
			}
			continue
		}
		value, ok := fields[name]
		if !ok {
			return "", fmt.Errorf("unknown field {%s}", field)
		}
		b.WriteString(value)
	}
	return b.String(), nil
}

// strftime formats t with Python strftime directives, in English.
func strftime(spec string, t time.Time) string {
	var b strings.Builder
	for i := 0; i < len(spec); i++ {
		if spec[i] != '%' || i+1 >= len(spec) {
			b.WriteByte(spec[i])
			continue
		}
		i++
		switch spec[i] {
		case 'Y':
			b.WriteString(t.Format("2006"))
		case 'y':
			b.WriteString(t.Format("06"))
		case 'm':
			b.WriteString(t.Format("01"))
		case 'd':
			b.WriteString(t.Format("02"))
		case 'e':
			b.WriteString(t.Format("_2"))
		case 'b', 'h':
			b.WriteString(t.Format("Jan"))
		case 'B':
			b.WriteString(t.Format("January"))
		case 'a':
			b.WriteString(t.Format("Mon"))
		case 'A':
			b.WriteString(t.Format("Monday"))
		case 'H':
			b.WriteString(t.Format("15"))
		case 'I':
			b.WriteString(t.Format("03"))
		case 'M':
			b.WriteString(t.Format("04"))
		case 'S':
			b.WriteString(t.Format("05"))
		case 'p':
			b.WriteString(t.Format("PM"))
		case 'j':
			fmt.Fprintf(&b, "%03d", t.YearDay())
		case 'U', 'W':
			_, week := t.ISOWeek()
			fmt.Fprintf(&b, "%02d", week)
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(spec[i])
		}
	}
	return b.String()
}

// pelicanPermalink turns a SAVE_AS pattern into a Nida permalink pattern
// if it uses only fields Nida has: {slug}, {date:%Y}, {date:%m},
// {date:%d}, and {category} (as Nida's {categories}).
func pelicanPermalink(saveAs string, withCategories bool) (string, bool) {
	replacer := strings.NewReplacer(
		"{slug}", "{slug}",
		"{date:%Y}", "{year}",
		"{date:%m}", "{month}",
		"{date:%d}", "{day}",
	)
	out := replacer.Replace(saveAs)
	if withCategories {
		out = strings.ReplaceAll(out, "{category}", "{categories}")
	}
	for _, field := range regexp.MustCompile(`\{[^}]*\}`).FindAllString(out, -1) {
		switch field {
		case "{slug}", "{year}", "{month}", "{day}", "{categories}":
		default:
			return "", false
		}
	}
	if !strings.Contains(out, "{slug}") {
		return "", false
	}
	out = strings.TrimSuffix(out, "index.html")
	return "/" + strings.TrimPrefix(out, "/"), true
}
