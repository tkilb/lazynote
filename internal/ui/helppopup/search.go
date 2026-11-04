package helppopup

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Search is the state of the popup's "/" search. It is a value: methods
// return updated copies.
type Search struct {
	// Typing is set while the prompt is open; Input is what has been typed.
	Typing bool
	Input  string
	// Query is the active search (set by enter, kept for n and N), and
	// Current the body line of the match the view is on, or -1 for none.
	Query   string
	Current int
}

// NoSearch is the state with no prompt and no query.
var NoSearch = Search{Current: -1}

// Active reports whether a query is set.
func (s Search) Active() bool { return s.Query != "" }

// Highlight backgrounds, as in PARA: teal on the current match, yellow on
// the others.
var (
	matchCurrent = lipgloss.Color("6")
	matchOther   = lipgloss.Color("3")
)

// Matches returns the body lines (as numbered for scrolling) whose key or
// description contains query, ignoring case. Section headers never match.
func Matches(sections []Section, query string) []int {
	q := strings.ToLower(query)
	if q == "" {
		return nil
	}
	var found []int
	line := 0
	for i, sec := range sections {
		if i > 0 {
			line++ // the blank separator
		}
		line++ // the header
		for _, b := range sec.Bindings {
			if strings.Contains(strings.ToLower(b.Key), q) || strings.Contains(strings.ToLower(b.Label), q) {
				found = append(found, line)
			}
			line++
		}
	}
	return found
}

// Step returns the match after (dir 1) or before (dir -1) from, wrapping
// around, or -1 when there are none. A from that is not itself a match (such
// as -1, or the top line of the view) steps to the nearest match in that
// direction, and dir 0 takes the first match at or after from.
func Step(matches []int, from, dir int) int {
	if len(matches) == 0 {
		return -1
	}
	switch {
	case dir > 0:
		for _, l := range matches {
			if l > from {
				return l
			}
		}
		return matches[0]
	case dir < 0:
		for i := len(matches) - 1; i >= 0; i-- {
			if matches[i] < from {
				return matches[i]
			}
		}
		return matches[len(matches)-1]
	}
	for _, l := range matches {
		if l >= from {
			return l
		}
	}
	return matches[0]
}

// highlight draws text in base, with each case-insensitive occurrence of
// query on a highlight background. An empty query draws text as it is.
func highlight(text, query string, base lipgloss.Style, current bool) string {
	lower, q := strings.ToLower(text), strings.ToLower(query)
	if q == "" || len(lower) != len(text) || len(q) != len(query) {
		return base.Render(text)
	}
	bg := matchOther
	if current {
		bg = matchCurrent
	}
	match := base.Background(bg).Foreground(lipgloss.Color("0")).Bold(false)
	var b strings.Builder
	for i := 0; i < len(text); {
		idx := strings.Index(lower[i:], q)
		if idx < 0 {
			b.WriteString(base.Render(text[i:]))
			break
		}
		idx += i
		if idx > i {
			b.WriteString(base.Render(text[i:idx]))
		}
		b.WriteString(match.Render(text[idx : idx+len(q)]))
		i = idx + len(q)
	}
	return b.String()
}
