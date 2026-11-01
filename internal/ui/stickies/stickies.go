// Package stickies implements the Stickies panel: the notes you pinned. It is
// one row (how many there are) until it has focus, and then lists them
// one per row, as lazytask's Suggested panel does. The arrows choose, enter
// jumps to the note and d unpins it.
package stickies

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/ui/panel"
)

const (
	// MaxRows is the most rows the list takes while the panel has focus.
	MaxRows = 5
	// UnfocusedRows is the row count while it does not.
	UnfocusedRows = 1

	empty   = "(press s on a note to pin it)"
	missing = "missing"
	hint    = "enter jump  d unpin"
)

var (
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selStyle = lipgloss.NewStyle().Bold(true).Background(panel.SelectedRowBackground)
)

// Rows is how many content rows the panel takes for n stickies.
func Rows(focused bool, n int) int {
	if !focused {
		return UnfocusedRows
	}
	return min(max(n, 1), MaxRows)
}

// Item is one pinned note.
type Item struct {
	// UID identifies the note; it survives renames and moves.
	UID string
	// Name is the note's file name without .md.
	Name string
	// Rel is the note's path relative to the notebook root, or "" when it
	// cannot be found any more.
	Rel string
	// Where is the note's folder, shown dim.
	Where string
}

// Effect is what a key press asks the root model to do.
type Effect struct {
	// Jump is the note to select in PARA.
	Jump string
	// Unpin is the uid of the sticky to remove.
	Unpin string
	// Info is a notice to show as a popup.
	Info string
}

// Model is the panel state. Methods return updated copies.
type Model struct {
	items  []Item
	cursor int
}

// New returns an empty panel.
func New() Model { return Model{} }

// Len is the number of stickies.
func (m Model) Len() int { return len(m.items) }

// Set replaces the stickies, keeping the cursor on the same note when it is
// still there.
func (m Model) Set(items []Item) Model {
	var keep string
	if m.cursor < len(m.items) {
		keep = m.items[m.cursor].UID
	}
	m.items = items
	for i, it := range items {
		if it.UID == keep {
			m.cursor = i
			return m
		}
	}
	m.cursor = min(m.cursor, max(len(items)-1, 0))
	return m
}

// Update handles a key: up and down choose, enter jumps, d unpins.
func (m Model) Update(msg tea.KeyMsg) (Model, Effect) {
	if len(m.items) == 0 {
		return m, Effect{}
	}
	switch msg.String() {
	case "up":
		m.cursor = max(m.cursor-1, 0)
	case "down":
		m.cursor = min(m.cursor+1, len(m.items)-1)
	case "enter":
		it := m.items[m.cursor]
		if it.Rel == "" {
			return m, Effect{Info: "That note cannot be found any more. Press d to remove the sticky."}
		}
		return m, Effect{Jump: it.Rel}
	case "d":
		return m, Effect{Unpin: m.items[m.cursor].UID}
	}
	return m, Effect{}
}

// RowAt returns the index of the sticky drawn at content row y while the
// panel has focus (0 is the first row under the top border).
func (m Model) RowAt(y, outerHeight int) (int, bool) {
	_, height := panel.InnerSize(0, outerHeight)
	return panel.RowAt(y, m.cursor, len(m.items), height)
}

// SelectRow puts the cursor on sticky i.
func (m Model) SelectRow(i int) Model {
	if i >= 0 && i < len(m.items) {
		m.cursor = i
	}
	return m
}

// View renders the panel into outerWidth x outerHeight, border included.
func (m Model) View(outerWidth, outerHeight int, focused bool) string {
	width, height := panel.InnerSize(outerWidth, outerHeight)
	var footer panel.Footer
	var body string
	switch {
	case len(m.items) == 0:
		body = dimStyle.Render(empty)
	case !focused:
		body = m.count()
	default:
		body = m.list(width, height)
		footer = panel.Footer{Left: hint, Right: fmt.Sprintf("%d/%d", m.cursor+1, len(m.items)), Tint: true}
	}
	return panel.Frame("1 Stickies", body, width, height, focused, footer)
}

// count says how many stickies there are, such as "5 stickies".
func (m Model) count() string {
	if len(m.items) == 1 {
		return "1 sticky"
	}
	return fmt.Sprintf("%d stickies", len(m.items))
}

// list draws one sticky per row with the chosen one highlighted.
func (m Model) list(width, height int) string {
	start, end := panel.ScrollWindow(m.cursor, len(m.items), height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, row(m.items[i], width, i == m.cursor))
	}
	return strings.Join(rows, "\n")
}

// row draws a sticky's name and, dim at the right, its folder (or "missing").
func row(it Item, width int, cursor bool) string {
	right := it.Where
	if it.Rel == "" {
		right = missing
	}
	rightWidth := ansi.StringWidth(right)
	name := ansi.Truncate(it.Name, max(width-rightWidth-1, 1), "…")
	gap := strings.Repeat(" ", max(width-ansi.StringWidth(name)-rightWidth, 0))
	switch {
	case cursor:
		return selStyle.Render(name + gap + right)
	case it.Rel == "":
		return dimStyle.Render(name + gap + right)
	}
	return name + gap + dimStyle.Render(right)
}
