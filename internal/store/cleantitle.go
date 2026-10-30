package store

import (
	"strings"
	"unicode"
)

const (
	titleMaxRunes = 80
	// unsafeChars are removed from a title: unsafe in a file name, or unable
	// to appear in a wikilink.
	unsafeChars = `/\:*?"<>|#[]^`
)

// CleanTitle turns text into a title that is safe as a file name and as a
// link target: characters that are unsafe in either are removed, runs of
// white space collapse to one space, leading and trailing dots and spaces go,
// and it is cut at titleMaxRunes. Case and punctuation are kept. It may be
// empty.
func CleanTitle(text string) string {
	var words []string
	for _, w := range strings.Fields(text) {
		w = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) || strings.ContainsRune(unsafeChars, r) {
				return -1
			}
			return r
		}, w)
		if w != "" {
			words = append(words, w)
		}
	}
	title := strings.Trim(strings.Join(words, " "), ". ")
	if r := []rune(title); len(r) > titleMaxRunes {
		title = strings.TrimRight(string(r[:titleMaxRunes]), ". ")
	}
	return title
}
