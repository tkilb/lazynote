// Package search finds text in the lines of the notes, as grep -i does: every
// word typed must appear on the line, as exact text, ignoring case. It works
// on lines already read into memory (see internal/findtext), so each key
// press searches without touching the disk.
package search

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tkilb/para/internal/findtext"
)

// Hit is one line that matched.
type Hit struct {
	// Rel is the note's path and Index the line's number in it (see
	// findtext.Line).
	Rel   string
	Index int
	Text  string
	// Col is the 1-based column, in characters, of the first match.
	Col int
	// Spans are the byte ranges of Text that matched, in order and apart.
	Spans [][2]int
}

// Result is what Find found.
type Result struct {
	// Hits are the matching lines, best first, at most the limit.
	Hits []Hit
	// Total is how many lines matched, however many are in Hits.
	Total     int
	Truncated bool
}

// Find returns the lines that contain every word of query, ignoring case. A
// line in a note whose name contains every word comes before the others;
// otherwise the order of lines is kept. At most limit hits are returned, but
// Total counts every match. Nothing typed finds nothing.
func Find(lines []findtext.Line, query string, limit int) Result {
	words := strings.Fields(query)
	if len(words) == 0 || limit <= 0 {
		return Result{}
	}
	var res Result
	var named, rest []Hit
	byName := map[string]bool{}
	for _, l := range lines {
		spans, ok := match(l.Text, words)
		if !ok {
			continue
		}
		res.Total++
		isNamed, seen := byName[l.Rel]
		if !seen {
			isNamed = allIn(noteName(l.Rel), words)
			byName[l.Rel] = isNamed
		}
		switch {
		case isNamed && len(named) < limit:
			named = append(named, hit(l, spans))
		case !isNamed && len(rest) < limit:
			rest = append(rest, hit(l, spans))
		}
	}
	res.Hits = append(named, rest...)
	if len(res.Hits) > limit {
		res.Hits = res.Hits[:limit]
	}
	res.Truncated = res.Total > len(res.Hits)
	return res
}

func hit(l findtext.Line, spans [][2]int) Hit {
	return Hit{
		Rel: l.Rel, Index: l.Index, Text: l.Text, Spans: spans,
		Col: utf8.RuneCountInString(l.Text[:spans[0][0]]) + 1,
	}
}

// noteName is a note's file name without its extension.
func noteName(rel string) string {
	return strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
}

// allIn reports whether every word is in s, ignoring case.
func allIn(s string, words []string) bool {
	for _, w := range words {
		if start, _ := index(s, w, 0); start < 0 {
			return false
		}
	}
	return true
}

// match returns the spans of text covered by the words, or false when a word
// is missing.
func match(text string, words []string) ([][2]int, bool) {
	var spans [][2]int
	for _, w := range words {
		from, found := 0, false
		for {
			start, end := index(text, w, from)
			if start < 0 {
				break
			}
			found = true
			spans = append(spans, [2]int{start, end})
			from = end
		}
		if !found {
			return nil, false
		}
	}
	return merge(spans), true
}

// merge sorts spans and joins any that overlap or touch.
func merge(spans [][2]int) [][2]int {
	sort.Slice(spans, func(i, j int) bool { return spans[i][0] < spans[j][0] })
	out := spans[:1]
	for _, s := range spans[1:] {
		last := &out[len(out)-1]
		if s[0] <= last[1] {
			last[1] = max(last[1], s[1])
			continue
		}
		out = append(out, s)
	}
	return out
}

// index finds word in s at or after byte from, ignoring case, and returns
// its byte range, or -1 when it is not there. It compares character by
// character, so offsets stay right for text that changes length when lowered.
func index(s, word string, from int) (start, end int) {
	wr := []rune(word)
	for i := from; i < len(s); {
		if e, ok := matchAt(s, i, wr); ok {
			return i, e
		}
		_, w := utf8.DecodeRuneInString(s[i:])
		i += w
	}
	return -1, -1
}

// matchAt reports whether the runes of word are at byte i of s, ignoring
// case, and where the match ends.
func matchAt(s string, i int, word []rune) (int, bool) {
	for _, w := range word {
		if i >= len(s) {
			return 0, false
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r != w && unicode.ToLower(r) != unicode.ToLower(w) && unicode.ToUpper(r) != unicode.ToUpper(w) {
			return 0, false
		}
		i += size
	}
	return i, true
}
