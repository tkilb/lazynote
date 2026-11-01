// Package body implements the Body panel: a read-only plain-text view of the
// selected note, with its frontmatter hidden and long lines wrapped to the
// panel width. A cursor line keeps your place, and whole lines can be
// selected, as in PARA (space picks a line, v and V select or unselect a
// range), for extracting them into a new note. A wrapped line is one line.
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
	// cursorStyle is the cursor line, as the cursor row in a list panel;
	// pickStyle a selected line; both together use the teal of the search
	// match under the cursor.
	cursorStyle = lipgloss.NewStyle().Bold(true).Background(panel.SelectedRowBackground)
	pickStyle   = lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipgloss.Color("0"))
	bothStyle   = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
)

// visualMode says whether a range is being selected, unselected, or neither.
type visualMode int

const (
	visualOff visualMode = iota
	visualSelect
	visualUnselect
)

// visualState anchors a visual-mode range. While it is active, the selection
// is base plus (or minus) every line between anchor and the cursor.
type visualState struct {
	mode   visualMode
	anchor int
	base   map[int]bool
}

// Model is the Body panel state. Methods return updated copies.
type Model struct {
	// rel identifies the note shown, so switching notes resets the scroll
	// while reloading the same note keeps it.
	rel  string
	text string // sanitized, unwrapped
	err  error

	// srcLines are the lines of text as written. lines are those wrapped to
	// the width, srcOf maps each wrapped row to its source line, and
	// firstRow and lastRow give the rows each source line covers.
	srcLines          []string
	lines             []string
	srcOf             []int
	firstRow, lastRow []int

	offset int // index of the first visible row
	cursor int // the cursor's source line
	// picked holds the selected source lines when no range is open. It is
	// replaced, never changed in place.
	picked map[int]bool
	visual visualState
	// yanked is how many lines the last yank copied, shown in the footer
	// until the next key press.
	yanked int
	// places remembers the cursor line of each note shown this session, so
	// coming back to a note puts you where you were.
	places map[string]int

	width, height int
}

// New returns an empty panel.
func New() Model {
	return Model{height: defaultHeight, places: map[string]int{}}
}

// Resize sets the content area size, re-wrapping when the width changed.
func (m Model) Resize(width, height int) Model {
	rewrap := width != m.width
	m.width, m.height = width, max(height, 1)
	if rewrap {
		m = m.wrap()
	}
	return m.showCursor()
}

// Load shows text for the note at rel, without the blank lines around it.
// Reloading the note already shown keeps the cursor (and the selection, while
// the text is unchanged); another note starts where you left it last time, or
// at the top.
func (m Model) Load(rel, text string) Model {
	text = textsafe.Sanitize(strings.Trim(text, "\r\n"))
	if rel != m.rel {
		m = m.remember()
		m.cursor, m.offset = m.places[rel], 0
		m.picked, m.visual = nil, visualState{}
	} else if text != m.text {
		m.picked, m.visual = nil, visualState{}
	}
	m.rel, m.err, m.text = rel, nil, text
	m = m.wrap()
	m.cursor = min(m.cursor, max(len(m.srcLines)-1, 0))
	return m.showCursor()
}

// Fail shows err in place of the note at rel.
func (m Model) Fail(rel string, err error) Model {
	m = m.remember()
	m = m.reset()
	m.rel, m.err = rel, err
	return m
}

// Clear empties the panel.
func (m Model) Clear() Model {
	m = m.remember()
	return m.reset()
}

// Renamed moves the remembered place of the note at oldRel to newRel, so a
// note keeps your place when it is renamed. Call it before the panel loads
// the note under its new name.
func (m Model) Renamed(oldRel, newRel string) Model {
	m = m.remember()
	if place, ok := m.places[oldRel]; ok {
		m.places[newRel] = place
		delete(m.places, oldRel)
	}
	if m.rel == oldRel {
		m.rel = newRel
	}
	return m
}

// remember keeps the cursor of the note shown for later.
func (m Model) remember() Model {
	if m.rel != "" && m.err == nil {
		m.places[m.rel] = m.cursor
	}
	return m
}

// reset drops everything shown, keeping the size and the remembered places.
func (m Model) reset() Model {
	return Model{width: m.width, height: m.height, places: m.places}
}

// Rel returns the note shown, or "" when the panel is empty.
func (m Model) Rel() string { return m.rel }

// wrap lays text out for the current width.
func (m Model) wrap() Model {
	m.lines, m.srcLines, m.srcOf, m.firstRow, m.lastRow = nil, nil, nil, nil, nil
	if m.text == "" || m.width <= 0 {
		return m
	}
	m.srcLines = strings.Split(m.text, "\n")
	for i, line := range m.srcLines {
		m.firstRow = append(m.firstRow, len(m.lines))
		for _, row := range strings.Split(ansi.Wrap(line, m.width, ""), "\n") {
			m.lines = append(m.lines, row)
			m.srcOf = append(m.srcOf, i)
		}
		m.lastRow = append(m.lastRow, len(m.lines)-1)
	}
	return m
}

// clamp keeps offset within the scrollable range.
func (m Model) clamp(offset int) int {
	return min(max(offset, 0), max(len(m.lines)-m.height, 0))
}

// showCursor scrolls so the cursor line is in view (its first row, when it is
// taller than the panel).
func (m Model) showCursor() Model {
	if len(m.firstRow) == 0 {
		m.offset = 0
		return m
	}
	first, last := m.firstRow[m.cursor], m.lastRow[m.cursor]
	switch {
	case first < m.offset:
		m.offset = first
	case last >= m.offset+m.height:
		m.offset = max(last-m.height+1, first)
	}
	m.offset = m.clamp(m.offset)
	return m
}

// moveTo puts the cursor on source line i, clamped, and scrolls to it.
func (m Model) moveTo(i int) Model {
	if len(m.srcLines) == 0 {
		return m
	}
	m.cursor = min(max(i, 0), len(m.srcLines)-1)
	return m.showCursor()
}

// moveRows moves the cursor to the line shown delta rows away from its first
// row, for page movement.
func (m Model) moveRows(delta int) Model {
	if len(m.firstRow) == 0 {
		return m
	}
	row := min(max(m.firstRow[m.cursor]+delta, 0), len(m.srcOf)-1)
	return m.moveTo(m.srcOf[row])
}

// isSelected reports whether source line i is selected, counting an open
// range.
func (m Model) isSelected(i int) bool {
	if m.visual.mode == visualOff {
		return m.picked[i]
	}
	lo, hi := min(m.visual.anchor, m.cursor), max(m.visual.anchor, m.cursor)
	if i >= lo && i <= hi {
		return m.visual.mode == visualSelect
	}
	return m.visual.base[i]
}

// selection returns the selected source lines in order.
func (m Model) selection() []int {
	var out []int
	for i := range m.srcLines {
		if m.isSelected(i) {
			out = append(out, i)
		}
	}
	return out
}

// commit makes the open range part of the selection and closes it.
func (m Model) commit() Model {
	if m.visual.mode == visualOff {
		return m
	}
	picked := map[int]bool{}
	for _, i := range m.selection() {
		picked[i] = true
	}
	m.picked, m.visual = picked, visualState{}
	return m
}

// HasSelection reports whether any line is selected.
func (m Model) HasSelection() bool { return len(m.selection()) > 0 }

// Active reports whether Escape has something to close: an open range or a
// selection.
func (m Model) Active() bool { return m.visual.mode != visualOff || len(m.picked) > 0 }

// Escape closes an open range keeping what it selected, and otherwise clears
// the selection.
func (m Model) Escape() Model {
	if m.visual.mode != visualOff {
		return m.commit()
	}
	return m.ClearSelection()
}

// ClearSelection drops the selection and any open range.
func (m Model) ClearSelection() Model {
	m.picked, m.visual = nil, visualState{}
	return m
}

// SelectedText returns the selected lines in file order, as shown (control
// characters already removed). Lines that were not next to each other are
// separated by a blank line, so the gaps stay visible. It is "" when nothing
// is selected.
func (m Model) SelectedText() string {
	var groups []string
	var run []string
	prev := -2
	for _, i := range m.selection() {
		if i != prev+1 && len(run) > 0 {
			groups = append(groups, strings.Join(run, "\n"))
			run = nil
		}
		run = append(run, m.srcLines[i])
		prev = i
	}
	if len(run) > 0 {
		groups = append(groups, strings.Join(run, "\n"))
	}
	return strings.Join(groups, "\n\n")
}

// groupCount is the number of runs of adjacent selected lines.
func (m Model) groupCount() int {
	groups, prev := 0, -2
	for _, i := range m.selection() {
		if i != prev+1 {
			groups++
		}
		prev = i
	}
	return groups
}

// YankText returns what a yank copies and how many lines that is: the
// selection, or the cursor line when nothing is selected.
func (m Model) YankText() (text string, lines int) {
	if m.HasSelection() {
		return m.SelectedText(), len(m.selection())
	}
	if len(m.srcLines) == 0 {
		return "", 0
	}
	return m.srcLines[m.cursor], 1
}

// Yanked records that n lines were copied: the selection is cleared, as a
// yank does in PARA, and the footer says so until the next key press.
func (m Model) Yanked(n int) Model {
	m = m.ClearSelection()
	m.yanked = n
	return m
}

// toggleVisual opens a range in mode, or closes it when it is already open in
// that mode (the other mode switches).
func (m Model) toggleVisual(mode visualMode) Model {
	switch m.visual.mode {
	case mode:
		return m.commit()
	case visualOff:
		m.visual = visualState{mode: mode, anchor: m.cursor, base: m.picked}
	default:
		m.visual.mode = mode
	}
	return m
}

// Update handles a key press while the panel has focus.
func (m Model) Update(msg tea.KeyMsg) Model {
	m.yanked = 0
	switch msg.String() {
	case "up":
		return m.moveTo(m.cursor - 1)
	case "down":
		return m.moveTo(m.cursor + 1)
	case "ctrl+u":
		return m.moveRows(-max(m.height/2, 1))
	case "ctrl+d":
		return m.moveRows(max(m.height/2, 1))
	case "g", "home":
		return m.moveTo(0)
	case "G", "end":
		return m.moveTo(len(m.srcLines) - 1)
	case " ":
		if m.visual.mode != visualOff || len(m.srcLines) == 0 {
			return m
		}
		picked := make(map[int]bool, len(m.picked)+1)
		for i := range m.picked {
			picked[i] = true
		}
		if picked[m.cursor] {
			delete(picked, m.cursor)
		} else {
			picked[m.cursor] = true
		}
		m.picked = picked
		return m.moveTo(m.cursor + 1)
	case "v":
		return m.toggleVisual(visualSelect)
	case "V":
		return m.toggleVisual(visualUnselect)
	case "ctrl+a":
		picked := make(map[int]bool, len(m.srcLines))
		for i := range m.srcLines {
			picked[i] = true
		}
		m.picked, m.visual = picked, visualState{}
		return m
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
			rows = append(rows, m.row(i, width, focused))
		}
		content = strings.Join(rows, "\n")
		if len(m.lines) > height {
			footer.Right = fmt.Sprintf("%d/%d", m.cursor+1, len(m.srcLines))
		}
		footer.Left = m.status()
	}
	return panel.Frame("0 Body", content, width, height, focused, footer)
}

// status is the bottom-left text: the range mode and the selection size.
func (m Model) status() string {
	var parts []string
	if m.yanked > 0 {
		parts = append(parts, fmt.Sprintf("%d yanked", m.yanked))
	}
	switch m.visual.mode {
	case visualSelect:
		parts = append(parts, "VISUAL")
	case visualUnselect:
		parts = append(parts, "UNSELECT")
	}
	if n := len(m.selection()); n > 0 {
		sel := fmt.Sprintf("%d sel", n)
		if g := m.groupCount(); g > 1 {
			sel += fmt.Sprintf(" in %d groups", g)
		}
		parts = append(parts, sel)
	}
	return strings.Join(parts, " ")
}

// row draws wrapped row i: the cursor line is highlighted while the panel has
// focus, and selected lines always are.
func (m Model) row(i, width int, focused bool) string {
	src := m.srcOf[i]
	cursor, selected := focused && src == m.cursor, m.isSelected(src)
	if !cursor && !selected {
		return m.lines[i]
	}
	text := m.lines[i] + strings.Repeat(" ", max(width-ansi.StringWidth(m.lines[i]), 0))
	switch {
	case cursor && selected:
		return bothStyle.Render(text)
	case cursor:
		return cursorStyle.Render(text)
	}
	return pickStyle.Render(text)
}

// RowAt returns the source line drawn at content row y (0 is the first row
// under the top border). It reports false past the last row.
func (m Model) RowAt(y int) (int, bool) {
	row := m.offset + y
	if y < 0 || y >= m.height || row >= len(m.srcOf) {
		return 0, false
	}
	return m.srcOf[row], true
}

// SelectRow puts the cursor on source line i, as moving with the arrow keys
// does.
func (m Model) SelectRow(i int) Model {
	m.yanked = 0
	return m.moveTo(i)
}
