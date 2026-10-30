// Package body implements the Body panel: a read-only plain-text view of the
// selected note, with its frontmatter hidden, long lines wrapped to the
// panel width, and vertical scrolling.
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
)

// Model is the Body panel state. Methods return updated copies.
type Model struct {
	// rel identifies the note shown, so switching notes resets the scroll
	// while reloading the same note keeps it.
	rel    string
	text   string // sanitized, unwrapped
	err    error
	lines  []string // text wrapped to width
	offset int      // index of the first visible line

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
	m.rel, m.err = rel, nil
	m.text = textsafe.Sanitize(strings.Trim(text, "\r\n"))
	m = m.wrap()
	m.offset = m.clamp(m.offset)
	return m
}

// Fail shows err in place of the note at rel.
func (m Model) Fail(rel string, err error) Model {
	m.rel, m.err = rel, err
	m.text, m.lines, m.offset = "", nil, 0
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
	m.lines = nil
	if m.text == "" || m.width <= 0 {
		return m
	}
	for _, line := range strings.Split(m.text, "\n") {
		wrapped := ansi.Wrap(line, m.width, "")
		m.lines = append(m.lines, strings.Split(wrapped, "\n")...)
	}
	return m
}

// clamp keeps offset within the scrollable range.
func (m Model) clamp(offset int) int {
	return min(max(offset, 0), max(len(m.lines)-m.height, 0))
}

// Update handles a key press while the panel has focus.
func (m Model) Update(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "up", "k":
		m.offset = m.clamp(m.offset - 1)
	case "down", "j":
		m.offset = m.clamp(m.offset + 1)
	case "ctrl+u":
		m.offset = m.clamp(m.offset - max(m.height/2, 1))
	case "ctrl+d":
		m.offset = m.clamp(m.offset + max(m.height/2, 1))
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
		content = strings.Join(m.lines[min(m.offset, end):end], "\n")
		if len(m.lines) > height {
			footer.Right = fmt.Sprintf("%d/%d", m.offset+1, len(m.lines))
		}
	}
	return panel.Frame("0 Body", content, width, height, focused, footer)
}
