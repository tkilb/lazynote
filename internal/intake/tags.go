package intake

import (
	"errors"
	"strings"
	"unicode"
)

// ErrBadTag reports a tag with characters Obsidian does not allow in one.
var ErrBadTag = errors.New("tags may only contain letters, digits, _ - and /")

// ParseTags splits text typed into a tags box into tag names: names are
// separated by commas and/or white space, each may start with # (dropped),
// empty names and duplicates are dropped, and the order is kept. A tag may
// contain only letters, digits, underscores, hyphens and slashes, as in
// Obsidian, and must not be all digits; otherwise it returns ErrBadTag.// error context by ErrBadTag.
func ParseTags(text string) ([]string, error) {
	var tags []string
	seen := map[string]bool{}
	for _, f := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
		name := strings.TrimPrefix(f, "#")
		if name == "" {
			continue
		}
		if !validTag(name) {
			return nil, ErrBadTag
		}
		if key := strings.ToLower(name); !seen[key] {
			seen[key] = true
			tags = append(tags, name)
		}
	}
	return tags, nil
}

func validTag(name string) bool {
	allDigits := true
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			allDigits = false
		case unicode.IsDigit(r):
		case r == '_' || r == '-' || r == '/':
			allDigits = false
		default:
			return false
		}
	}
	return !allDigits
}
