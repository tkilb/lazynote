// Package textsafe makes untrusted text safe to draw in a terminal.
package textsafe

import (
	"strings"
	"unicode"
)

// TabWidth is how many spaces a tab expands to.
const TabWidth = 4

// Sanitize returns s with carriage returns dropped, tabs turned into spaces
// and every other control character removed, including escape and the C1
// range, so text from a note can never send terminal escape sequences.
// Newlines are kept.
func Sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(strings.Repeat(" ", TabWidth))
		case r == '\r', unicode.IsControl(r):
			// dropped
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SingleLine is Sanitize for text that must stay on one line: newlines
// become spaces.
func SingleLine(s string) string {
	return strings.ReplaceAll(Sanitize(s), "\n", " ")
}
