package para

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/panel"
)

// panelNumber is the key that focuses this panel.
const panelNumber = '2'

const (
	badgeWidth = 4
	// unsetBadge stands in for a note whose kind is unset.
	unsetBadge = "----"
)

var (
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dirStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	selStyle   = lipgloss.NewStyle().Bold(true).Background(panel.SelectedRowBackground)
	// The left-edge bar of a marked row is colored as in Yazi.
	markerColors = map[mark]lipgloss.Color{
		markSelected: lipgloss.Color("3"), // yellow
		markYank:     lipgloss.Color("2"), // green
		markCut:      lipgloss.Color("1"), // red
	}
)

// mark says why a row has a left-edge bar.
type mark int

const (
	markNone mark = iota
	markSelected
	markYank
	markCut
)

// markOf picks a row's marker. The yank buffer outranks the selection,
// because it is what a paste acts on.
func (m Model) markOf(t tabState, r row) mark {
	if m.clip.set[r.entry.Rel] {
		if m.clip.mode == clipCut {
			return markCut
		}
		return markYank
	}
	if t.selected[r.entry.Name] {
		return markSelected
	}
	return markNone
}

const (
	// markerBar is drawn in the first column of a selected row.
	markerBar   = "▎"
	markerWidth = 1
)

// badges gives each kind a four-letter label and a color class.
var badges = map[store.Kind]struct {
	text  string
	color lipgloss.Color
}{
	store.KindFact:     {"fact", lipgloss.Color("2")},
	store.KindRumor:    {"rumr", lipgloss.Color("3")},
	store.KindProposal: {"prop", lipgloss.Color("4")},
	store.KindFossil:   {"foss", lipgloss.Color("8")},
	store.KindIdea:     {"idea", lipgloss.Color("6")},
	store.KindDraft:    {"draf", lipgloss.Color("5")},
}

// View renders the panel into outerWidth x outerHeight, border included.
func (m Model) View(outerWidth, outerHeight int, focused bool) string {
	width, height := panel.InnerSize(outerWidth, outerHeight)
	t := m.tabs[m.active]

	tabs := make([]panel.Tab, len(tabDefs))
	for i, d := range tabDefs {
		tabs[i] = panel.Tab{Label: d.label, Active: i == m.active}
	}

	footer := panel.Footer{Left: "/" + filepath.ToSlash(t.dir), Right: status(t, m.clip)}
	return panel.FrameTabs(panelNumber, tabs, m.body(t, width, height, focused), width, height, focused, footer)
}

// status builds the bottom-right text: the visual-mode flag, the selection
// count and the cursor position, e.g. "VISUAL 3 sel 4/9".
func status(t tabState, c clip) string {
	var parts []string
	switch t.visual.mode {
	case visualSelect:
		parts = append(parts, "VISUAL")
	case visualUnselect:
		parts = append(parts, "UNSELECT")
	}
	if n := len(t.selected); n > 0 {
		parts = append(parts, fmt.Sprintf("%d sel", n))
	}
	switch c.mode {
	case clipYank:
		parts = append(parts, fmt.Sprintf("%d yanked", len(c.rels)))
	case clipCut:
		parts = append(parts, fmt.Sprintf("%d cut", len(c.rels)))
	}
	if len(t.rows) > 0 {
		parts = append(parts, fmt.Sprintf("%d/%d", t.cursor+1, len(t.rows)))
	}
	return strings.Join(parts, " ")
}

func (m Model) body(t tabState, width, height int, focused bool) string {
	switch {
	case t.err != nil:
		return errorStyle.Render(ansi.Truncate(t.err.Error(), width, "…"))
	case len(t.rows) == 0:
		return dimStyle.Render("(empty)")
	}

	start, end := panel.ScrollWindow(t.cursor, len(t.rows), height)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, renderRow(t.rows[i], width, i == t.cursor && focused, m.markOf(t, t.rows[i])))
	}
	return strings.Join(lines, "\n")
}

// renderRow draws one row: a marker column, a kind badge (blank for
// directories), the name, and for directories an item count right-aligned.
// cursor highlights the row; mk draws the left-edge bar.
func renderRow(r row, width int, cursor bool, mk mark) string {
	bar := " "
	if mk != markNone {
		bar = markerBar
	}
	markerStyle := lipgloss.NewStyle().Foreground(markerColors[mk])
	width -= markerWidth

	badge, badgeColor := strings.Repeat(" ", badgeWidth), lipgloss.Color("")
	name := strings.TrimSuffix(r.entry.Name, ".md")
	right := ""
	if r.entry.IsDir {
		name = r.entry.Name + "/"
		right = fmt.Sprint(r.entry.ItemCount)
	} else if b, ok := badges[r.kind]; ok {
		badge, badgeColor = b.text, b.color
	} else {
		badge, badgeColor = unsetBadge, lipgloss.Color("245")
	}

	nameWidth := max(width-badgeWidth-1-len(right)-1, 1)
	name = ansi.Truncate(name, nameWidth, "…")
	gap := strings.Repeat(" ", max(width-badgeWidth-1-ansi.StringWidth(name)-len(right), 0))

	if cursor {
		return selStyle.Foreground(markerColors[mk]).Render(bar) +
			selStyle.Render(ansi.Truncate(badge+" "+name+gap+right, width, ""))
	}
	nameStyle := lipgloss.NewStyle()
	if r.entry.IsDir {
		nameStyle = dirStyle
	}
	return markerStyle.Render(bar) + lipgloss.NewStyle().Foreground(badgeColor).Render(badge) + " " +
		nameStyle.Render(name) + gap + dimStyle.Render(right)
}
