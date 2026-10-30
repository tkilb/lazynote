// Package body implements the Body panel: a read-only plain-text view of the
// selected note, with its frontmatter hidden, long lines wrapped to the
// panel width, and vertical scrolling. It can also select whole lines of the
// note (a wrapped line is one line), for extracting them into a new note.
package body

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/textsafe"
	"github.com/tkilb/lazynote/internal/ui/panel"
)

const (
	// defaultHeight is the page size until the first Resize.
	defaultHeight = 10
)

var (
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	// selStyle marks the selected lines; cursorStyle the line the selection
	// extends from, in the teal lazynote uses for the search match under the
	// cursor.
	selStyle    = lipgloss.NewStyle().Bold(true).Background(panel.SelectedRowBackground)
	cursorStyle = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
)

// selection is a range of source lines, anchored where it started and
// extended by moving cursor.
type selection struct {
	anchor, cursor int
}

func (s selection) bounds() (lo, hi int) {
	return min(s.anchor, s.cursor), max(s.anchor, s.cursor)
}

// Model is the Body panel state. Methods return updated copies.
type Model struct {
	// rel identifies the note shown, so switching notes resets the scroll
	// while reloading the same note keeps it.
	rel    string
	text   string // sanitized, unwrapped
	err    error
	lines  []string // text wrapped to width
	offset int      // index of the first visible line
	// srcLines are the lines of text as written, and srcOf maps each wrapped
	// row to the source line it came from.
	srcLines []string
	srcOf    []int
	// sel is the line selection, nil when not selecting.
	sel *selection

	width, height int
}

// New returns an empty panel.
func New() Model {
	return Model{height: defaultHeight}
}

// Resize sets the content area size, re-wrapping when the width changed.
func (m Model) Resize(width, height int) Model {
	rewrap := width != m.width
	m.width, m.height = width, max(height, 1)
	if rewrap {
		m = m.wrap()
	}
	m.offset = m.clamp(m.offset)
	return m
}

// Load shows text for the note at rel, without the blank lines around it.
// The scroll position is kept when rel is the note already shown, and reset
// to the top otherwise.
func (m Model) Load(rel, text string) Model {
	if rel != m.rel {
		m.offset = 0
	}
	text = textsafe.Sanitize(strings.Trim(text, "\r\n"))
	if rel != m.rel || text != m.text {
		m.sel = nil
	}
	m.rel, m.err = rel, nil
	m.text = text
	m = m.wrap()
	m.offset = m.clamp(m.offset)
	return m
}

// Fail shows err in place of the note at rel.
func (m Model) Fail(rel string, err error) Model {
	m.rel, m.err = rel, err
	m.text, m.lines, m.offset = "", nil, 0
	m.srcLines, m.srcOf, m.sel = nil, nil, nil
	return m
}

// Clear empties the panel.
func (m Model) Clear() Model {
	return Model{width: m.width, height: m.height}
}

// Rel returns the note shown, or "" when the panel is empty.
func (m Model) Rel() string { return m.rel }

// wrap lays text out for the current width.
func (m Model) wrap() Model {
	m.lines, m.srcLines, m.srcOf = nil, nil, nil
	if m.text == "" || m.width <= 0 {
		return m
	}
	m.srcLines = strings.Split(m.text, "\n")
	for i, line := range m.srcLines {
		for _, row := range strings.Split(ansi.Wrap(line, m.width, ""), "\n") {
			m.lines = append(m.lines, row)
			m.srcOf = append(m.srcOf, i)
		}
	}
	return m
}

// Selecting reports whether a line selection is open.
func (m Model) Selecting() bool { return m.sel != nil }

// Cancel closes the line selection.
func (m Model) Cancel() Model {
	m.sel = nil
	return m
}

// SelectedText returns the selected source lines joined by line breaks, as
// shown (control characters already removed), or "" when not selecting.
func (m Model) SelectedText() string {
	if m.sel == nil {
		return ""
	}
	lo, hi := m.sel.bounds()
	return strings.Join(m.srcLines[lo:hi+1], "\n")
}

// startSelection begins selecting at the first source line in view.
func (m Model) startSelection() Model {
	if len(m.srcOf) == 0 {
		return m
	}
	line := m.srcOf[min(m.offset, len(m.srcOf)-1)]
	m.sel = &selection{anchor: line, cursor: line}
	return m
}

// moveCursor moves the selection's cursor by delta source lines and scrolls
// to keep its line in view (its first row, when it wraps past the height).
func (m Model) moveCursor(delta int) Model {
	s := *m.sel
	s.cursor = min(max(s.cursor+delta, 0), len(m.srcLines)-1)
	m.sel = &s
	first, last := -1, -1
	for i, src := range m.srcOf {
		if src == s.cursor {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	switch {
	case first < m.offset:
		m.offset = first
	case last >= m.offset+m.height:
		m.offset = max(last-m.height+1, first)
	}
	m.offset = m.clamp(m.offset)
	return m
}

// clamp keeps offset within the scrollable range.
func (m Model) clamp(offset int) int {
	return min(max(offset, 0), max(len(m.lines)-m.height, 0))
}

// Update handles a key press while the panel has focus.
func (m Model) Update(msg tea.KeyMsg) Model {
	if m.sel != nil {
		return m.updateSelecting(msg)
	}
	switch msg.String() {
	case "v":
		return m.startSelection()
	case "up":
		m.offset = m.clamp(m.offset - 1)
	case "down":
		m.offset = m.clamp(m.offset + 1)
	case "ctrl+u":
		m.offset = m.clamp(m.offset - max(m.height/2, 1))
	case "ctrl+d":
		m.offset = m.clamp(m.offset + max(m.height/2, 1))
	}
	return m
}

// updateSelecting handles a key press while a line selection is open: the
// arrows and half-page keys move its cursor, v and esc close it.
func (m Model) updateSelecting(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "up":
		return m.moveCursor(-1)
	case "down":
		return m.moveCursor(1)
	case "ctrl+u":
		return m.moveCursor(-max(m.height/2, 1))
	case "ctrl+d":
		return m.moveCursor(max(m.height/2, 1))
	case "v", "esc":
		return m.Cancel()
	}
	return m
}

// View renders the panel into outerWidth x outerHeight, border included.
func (m Model) View(outerWidth, outerHeight int, focused bool) string {
	width, height := panel.InnerSize(outerWidth, outerHeight)

	var footer panel.Footer
	var content string
	switch {
	case m.err != nil:
		content = errorStyle.Render(ansi.Truncate(m.err.Error(), width, "…"))
	case m.rel != "" && len(m.lines) == 0:
		content = dimStyle.Render("(empty note)")
	default:
		end := min(m.offset+height, len(m.lines))
		rows := make([]string, 0, end-min(m.offset, end))
		for i := min(m.offset, end); i < end; i++ {
			rows = append(rows, m.row(i, width))
		}
		content = strings.Join(rows, "\n")
		if len(m.lines) > height {
			footer.Right = fmt.Sprintf("%d/%d", m.offset+1, len(m.lines))
		}
		if m.sel != nil {
			lo, hi := m.sel.bounds()
			footer.Left = fmt.Sprintf("VISUAL %d lines", hi-lo+1)
		}
	}
	return panel.Frame("0 Body", content, width, height, focused, footer)
}

// row draws wrapped row i, highlighted when its source line is selected.
func (m Model) row(i, width int) string {
	if m.sel == nil {
		return m.lines[i]
	}
	lo, hi := m.sel.bounds()
	src := m.srcOf[i]
	if src < lo || src > hi {
		return m.lines[i]
	}
	text := m.lines[i] + strings.Repeat(" ", max(width-ansi.StringWidth(m.lines[i]), 0))
	if src == m.sel.cursor {
		return cursorStyle.Render(text)
	}
	return selStyle.Render(text)
}
