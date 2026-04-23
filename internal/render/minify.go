package render

import (
	"fmt"
	"regexp"
	"strings"
)

var betweenTagsWhitespaceRe = regexp.MustCompile(`>\s+<`)
var preBlockRe = regexp.MustCompile(`(?s)<pre[^>]*>.*?</pre>`)

func minifyHTML(input string) string {
	input = strings.TrimSpace(input)

	// Extract <pre> blocks to preserve whitespace in code
	preBlocks := preBlockRe.FindAllString(input, -1)
	placeholders := make([]string, len(preBlocks))
	for i, block := range preBlocks {
		placeholder := fmt.Sprintf("\x00PRE%d\x00", i)
		placeholders[i] = placeholder
		input = strings.Replace(input, block, placeholder, 1)
	}

	// Minify whitespace between tags
	input = betweenTagsWhitespaceRe.ReplaceAllString(input, "><")

	// Restore <pre> blocks
	for i, placeholder := range placeholders {
		input = strings.Replace(input, placeholder, preBlocks[i], 1)
	}

	if input == "" {
		return ""
	}
	return input + "\n"
}
