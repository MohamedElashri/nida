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

// CountWords counts runs of letters and digits, which works for scripts that
// separate words with spaces, including Arabic.
func CountWords(text string) int {
	return countWords(text)
}

func countWords(text string) int {
	return len(strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}))
}
