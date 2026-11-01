// Package findtext prepares the lines of the notes for a search inside note
// text. A line keeps the number it has in the Body panel, so a hit can open
// the note at the right place. Nothing is written and nothing leaves memory.
package findtext

import (
	"strings"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/textsafe"
)

const (
	// maxLines bounds the lines kept, so a huge notebook cannot use all the
	// memory or make the search slow.
	maxLines = 100000
	// maxLineRunes cuts a long line; the rest is not searched.
	maxLineRunes = 160
)

// MaxLines is the number of lines Lines keeps at most.
const MaxLines = maxLines

// Line is one non-blank line of a note.
type Line struct {
	// Rel is the note's path relative to the notebook root.
	Rel string
	// Index is the line's position in the note as the Body panel shows it
	// (properties hidden, surrounding newlines trimmed), counting blank lines.
	Index int
	// Text is the line without its surrounding white space, cut to a
	// bounded length.
	Text string
}

// Result is what Lines found.
type Result struct {
	Lines []Line
	// Truncated is set when the limit was reached and notes were left out.
	Truncated bool
}

// Lines reads the notes at rels from s and returns their non-blank lines in
// order, up to maxLines. A note that cannot be read is skipped. Control
// characters are removed, as the Body panel does, so the text is safe to draw.
func Lines(s store.Store, rels []string) Result {
	var res Result
	for _, rel := range rels {
		n, err := s.ReadNote(rel)
		if err != nil {
			continue
		}
		text := textsafe.Sanitize(strings.Trim(n.Body, "\r\n"))
		for i, raw := range strings.Split(text, "\n") {
			line := strings.TrimSpace(raw)
			if line == "" {
				continue
			}
			if len(res.Lines) >= maxLines {
				res.Truncated = true
				return res
			}
			res.Lines = append(res.Lines, Line{Rel: rel, Index: i, Text: cut(line)})
		}
	}
	return res
}

// cut limits line to maxLineRunes characters.
func cut(line string) string {
	if len(line) <= maxLineRunes {
		return line
	}
	r := []rune(line)
	if len(r) <= maxLineRunes {
		return line
	}
	return string(r[:maxLineRunes])
}
