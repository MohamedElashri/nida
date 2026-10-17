package markdown

import (
	"strconv"
	"strings"
)

// codeInfo is the parsed info string of a fenced code block, for example
//
//	```go {title="main.go", hl_lines="2 4-6", linenos=true, linenostart=10}
type codeInfo struct {
	Language    string
	Title       string
	HighlightLn [][2]int
	LineNumbers bool
	LineStart   int
}

func parseCodeInfo(info string) codeInfo {
	info = strings.TrimSpace(info)
	attrs := ""
	if open := strings.IndexByte(info, '{'); open >= 0 {
		if end := strings.LastIndexByte(info, '}'); end > open {
			attrs = info[open+1 : end]
			info = info[:open]
		}
	}

	parsed := codeInfo{}
	if fields := strings.Fields(info); len(fields) > 0 {
		parsed.Language = fields[0]
	}

	for key, value := range parseCodeAttrs(attrs) {
		switch key {
		case "title":
			parsed.Title = value
		case "hl_lines":
			parsed.HighlightLn = parseLineRanges(value)
		case "linenos":
			parsed.LineNumbers = value == "" || value == "true"
		case "linenostart":
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				parsed.LineStart = n
			}
		}
	}
	return parsed
}

// parseCodeAttrs reads comma- or space-separated key=value pairs. Values may
// be bare, double-quoted, or a [list]; a key without a value maps to "".
func parseCodeAttrs(attrs string) map[string]string {
	out := map[string]string{}
	i := 0
	for i < len(attrs) {
		for i < len(attrs) && (attrs[i] == ' ' || attrs[i] == ',' || attrs[i] == '\t') {
			i++
		}
		start := i
		for i < len(attrs) && attrs[i] != '=' && attrs[i] != ' ' && attrs[i] != ',' {
			i++
		}
		key := strings.TrimSpace(attrs[start:i])
		if key == "" {
			i++
			continue
		}
		if i >= len(attrs) || attrs[i] != '=' {
			out[key] = ""
			continue
		}
		i++ // skip '='
		var value string
		switch {
		case i < len(attrs) && attrs[i] == '"':
			end := strings.IndexByte(attrs[i+1:], '"')
			if end < 0 {
				value, i = attrs[i+1:], len(attrs)
			} else {
				value, i = attrs[i+1:i+1+end], i+2+end
			}
		case i < len(attrs) && attrs[i] == '[':
			end := strings.IndexByte(attrs[i:], ']')
			if end < 0 {
				value, i = attrs[i+1:], len(attrs)
			} else {
				value, i = attrs[i+1:i+end], i+end+1
			}
		default:
			start := i
			for i < len(attrs) && attrs[i] != ',' && attrs[i] != ' ' {
				i++
			}
			value = attrs[start:i]
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

// parseLineRanges accepts "2 4-6", "2,4-6", or a list body like `2, "4-6"`.
func parseLineRanges(value string) [][2]int {
	var ranges [][2]int
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ' ' || r == ',' }) {
		part = strings.Trim(part, `"'`)
		from, to, isRange := strings.Cut(part, "-")
		start, err := strconv.Atoi(from)
		if err != nil || start <= 0 {
			continue
		}
		end := start
		if isRange {
			if end, err = strconv.Atoi(to); err != nil || end < start {
				continue
			}
		}
		ranges = append(ranges, [2]int{start, end})
	}
	return ranges
}
