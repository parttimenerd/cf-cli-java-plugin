package utils

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	"sort"
	"strings"
)

// FuzzySearch returns up to `max` words from `words` that are closest in
// Levenshtein distance to `needle`.
func FuzzySearch(needle string, words []string, max int) []string {
	type match struct {
		distance int
		word     string
	}

	matches := make([]match, 0, len(words))
	for _, w := range words {
		matches = append(matches, match{
			distance: fuzzy.LevenshteinDistance(needle, w),
			word:     w,
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].distance < matches[j].distance
	})

	if max > len(matches) {
		max = len(matches)
	}

	results := make([]string, 0, max)
	for i := range max {
		results = append(results, matches[i].word)
	}

	return results
}

// "x, y, or z"
func JoinWithOr(a []string) string {
	if len(a) == 0 {
		return ""
	}
	if len(a) == 1 {
		return a[0]
	}
	return strings.Join(a[:len(a)-1], ", ") + ", or " + a[len(a)-1]
}
