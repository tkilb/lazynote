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
	// markerStyle colors the left-edge bar of a selected row, as Yazi does.
	markerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

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

	footer := panel.Footer{Left: "/" + filepath.ToSlash(t.dir), Right: status(t)}
	return panel.FrameTabs(panelNumber, tabs, m.body(t, width, height, focused), width, height, focused, footer)
}

// status builds the bottom-right text: the visual-mode flag, the selection
// count and the cursor position, e.g. "VISUAL 3 sel 4/9".
func status(t tabState) string {
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
		lines = append(lines, renderRow(t.rows[i], width, i == t.cursor && focused, t.selected[t.rows[i].entry.Name]))
	}
	return strings.Join(lines, "\n")
}

// renderRow draws one row: a marker column, a kind badge (blank for
// directories), the name, and for directories an item count right-aligned.
// cursor highlights the row; marked draws the selection bar.
func renderRow(r row, width int, cursor, marked bool) string {
	bar := " "
	if marked {
		bar = markerBar
	}
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
		return selStyle.Foreground(markerStyle.GetForeground()).Render(bar) +
			selStyle.Render(ansi.Truncate(badge+" "+name+gap+right, width, ""))
	}
	nameStyle := lipgloss.NewStyle()
	if r.entry.IsDir {
		nameStyle = dirStyle
	}
	return markerStyle.Render(bar) + lipgloss.NewStyle().Foreground(badgeColor).Render(badge) + " " +
		nameStyle.Render(name) + gap + dimStyle.Render(right)
}
