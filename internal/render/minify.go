package render

import (
	"regexp"
	"strings"
)

var betweenTagsWhitespaceRe = regexp.MustCompile(`>\s+<`)

func minifyHTML(input string) string {
	input = strings.TrimSpace(input)
	input = betweenTagsWhitespaceRe.ReplaceAllString(input, "><")
	if input == "" {
		return ""
	}
	return input + "\n"
}
