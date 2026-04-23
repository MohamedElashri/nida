package content

import (
	"math"
	"strings"
	"unicode"
)

const wordsPerMinute = 200

func EstimateReadingTime(markdown string) int {
	words := countWords(markdown)
	if words == 0 {
		return 0
	}
	return int(math.Ceil(float64(words) / wordsPerMinute))
}

func countWords(text string) int {
	return len(strings.FieldsFunc(text, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r))
	}))
}
