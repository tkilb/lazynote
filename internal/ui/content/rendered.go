package content

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// Rendered reports whether the note is drawn as Markdown (the reading view).
// It has a cursor row that keeps your place, but no selection.
func (m Model) Rendered() bool {
	return !m.plain && !m.kicked && m.listing == nil && m.err == nil && m.rendered != nil
}

// LeaveRendered switches to the plain view for the note shown, with the
// cursor on the source line the rendered cursor row came from. It is what a
// selection key does, and does nothing in the plain view.
func (m Model) LeaveRendered() Model {
	if !m.Rendered() {
		return m
	}
	m = m.fromRendered()
	m.kicked = true
	return m.showCursor()
}

// fromRendered copies the rendered cursor's place into the source cursor, so
// the place survives a change of view, width or text. It does nothing in the
// plain view, where the source cursor is the place.
func (m Model) fromRendered() Model {
	if m.Rendered() && len(m.srcLines) > 0 {
		m.cursor = min(m.srcForRow(m.rcursor), len(m.srcLines)-1)
	}
	return m
}

// maxRenderedOffset is the largest first row of the rendered view.
func (m Model) maxRenderedOffset() int {
	return max(len(m.rendered)-m.height, 0)
}

// moveRendered puts the rendered cursor on row i, clamped, and scrolls to it.
func (m Model) moveRendered(i int) Model {
	m.rcursor = min(max(i, 0), max(len(m.rendered)-1, 0))
	return m.showRendered()
}

// showRendered keeps the rendered cursor in view.
func (m Model) showRendered() Model {
	if len(m.rendered) == 0 {
		m.roffset, m.rcursor = 0, 0
		return m
	}
	m.rcursor = min(max(m.rcursor, 0), len(m.rendered)-1)
	switch {
	case m.rcursor < m.roffset:
		m.roffset = m.rcursor
	case m.rcursor >= m.roffset+m.height:
		m.roffset = m.rcursor - m.height + 1
	}
	m.roffset = min(max(m.roffset, 0), m.maxRenderedOffset())
	return m
}

// renderedRow draws rendered row i; the cursor row is highlighted, in plain
// text, while the panel has focus.
func (m Model) renderedRow(i, width int, focused bool) string {
	if !focused || i != m.rcursor {
		return m.rendered[i]
	}
	text := ansi.Strip(m.rendered[i])
	return cursorStyle.Render(text + strings.Repeat(" ", max(width-ansi.StringWidth(text), 0)))
}

// blankRow reports whether rendered row i has nothing on it.
func (m Model) blankRow(i int) bool {
	return strings.TrimSpace(ansi.Strip(m.rendered[i])) == ""
}

// blankLine reports whether source line i has nothing on it.
func (m Model) blankLine(i int) bool {
	return strings.TrimSpace(m.srcLines[i]) == ""
}

// braceDir is the direction of { (back) or } (forward).
func braceDir(key string) int {
	if key == "{" {
		return -1
	}
	return 1
}

// paragraph returns where { (dir -1) or } (dir 1) takes the cursor from cur
// among n lines, as in vim: past any blank lines, then to the blank line
// after the paragraph (or the first or last line when there is none).
func paragraph(cur, n, dir int, blank func(int) bool) int {
	if n == 0 {
		return 0
	}
	i := min(max(cur, 0), n-1)
	for i+dir >= 0 && i+dir < n && blank(i) {
		i += dir
	}
	for i+dir >= 0 && i+dir < n && !blank(i) {
		i += dir
	}
	return i
}

// updateRendered handles a key in the rendered view: the arrows, pages, g and
// G, and { and } move the cursor row (the view follows), m goes to the plain
// view, and a key that picks lines leaves the rendered view and then does its
// work in the plain one.
func (m Model) updateRendered(msg tea.KeyMsg) Model {
	switch key := msg.String(); key {
	case "up":
		m = m.moveRendered(m.rcursor - 1)
	case "down":
		m = m.moveRendered(m.rcursor + 1)
	case "ctrl+u":
		m = m.moveRendered(m.rcursor - m.halfPage())
	case "ctrl+d":
		m = m.moveRendered(m.rcursor + m.halfPage())
	case "pgup":
		m = m.moveRendered(m.rcursor - m.height)
	case "pgdown":
		m = m.moveRendered(m.rcursor + m.height)
	case "g", "home":
		m = m.moveRendered(0)
	case "G", "end":
		m = m.moveRendered(len(m.rendered) - 1)
	case "{", "}":
		m = m.moveRendered(paragraph(m.rcursor, len(m.rendered), braceDir(key), m.blankRow))
	case "m":
		m = m.LeaveRendered()
		m.plain = true
	case " ", "v", "V", "ctrl+a":
		return m.LeaveRendered().Update(msg)
	}
	return m
}
