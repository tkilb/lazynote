package para

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/ui/kindstyle"
	"github.com/tkilb/lazynote/internal/ui/panel"
)

// panelNumber is the key that focuses this panel.
const panelNumber = '2'

const (
	badgeWidth = 4
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

// View renders the panel into outerWidth x outerHeight, border included.
func (m Model) View(outerWidth, outerHeight int, focused bool) string {
	width, height := panel.InnerSize(outerWidth, outerHeight)
	t := m.tabs[m.active]

	tabs := make([]panel.Tab, len(tabDefs))
	for i, d := range tabDefs {
		tabs[i] = panel.Tab{Label: d.label, Active: i == m.active}
	}

	left := "/" + filepath.ToSlash(t.dir)
	if t.search != "" {
		left += "  search: " + t.search
	}
	footer := panel.Footer{Left: left, Right: status(t, m.clip)}
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
		lines = append(lines, renderRow(t.rows[i], width, i == t.cursor && focused, m.markOf(t, t.rows[i]), t.search))
	}
	return strings.Join(lines, "\n")
}

// renderRow draws one row: a marker column, a kind badge (blank for
// directories), the name, and for directories an item count right-aligned.
// cursor highlights the row; mk draws the left-edge bar; query, when the row
// matches it, highlights the matching text in the name.
func renderRow(r row, width int, cursor bool, mk mark, query string) string {
	if !matches(r.entry, query) {
		query = ""
	}
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
	} else {
		badge, badgeColor = kindstyle.Badge(r.kind)
	}

	nameWidth := max(width-badgeWidth-1-len(right)-1, 1)
	name = ansi.Truncate(name, nameWidth, "…")
	gap := strings.Repeat(" ", max(width-badgeWidth-1-ansi.StringWidth(name)-len(right), 0))

	if cursor {
		return selStyle.Foreground(markerColors[mk]).Render(bar) +
			selStyle.Render(badge+" ") + highlight(name, query, selStyle, true) + selStyle.Render(gap+right)
	}
	nameStyle := lipgloss.NewStyle()
	if r.entry.IsDir {
		nameStyle = dirStyle
	}
	return markerStyle.Render(bar) + lipgloss.NewStyle().Foreground(badgeColor).Render(badge) + " " +
		highlight(name, query, nameStyle, false) + gap + dimStyle.Render(right)
}

// Search highlight backgrounds, as in lazytask: teal on the cursor row so
// the row n and N jump to stands out, yellow on every other matching row.
var (
	matchOnCursor  = lipgloss.Color("6")
	matchElsewhere = lipgloss.Color("3")
)

// highlight renders s with base, drawing each case-insensitive occurrence of
// query with a highlight background. An empty query renders s unchanged.
func highlight(s, query string, base lipgloss.Style, onCursor bool) string {
	lowerS, lowerQ := strings.ToLower(s), strings.ToLower(query)
	if query == "" || len(lowerS) != len(s) || len(lowerQ) != len(query) {
		return base.Render(s)
	}
	bg := matchElsewhere
	if onCursor {
		bg = matchOnCursor
	}
	match := base.Background(bg).Foreground(lipgloss.Color("0")).Bold(false)

	var b strings.Builder
	for i := 0; i < len(s); {
		idx := strings.Index(lowerS[i:], lowerQ)
		if idx < 0 {
			b.WriteString(base.Render(s[i:]))
			break
		}
		idx += i
		if idx > i {
			b.WriteString(base.Render(s[i:idx]))
		}
		b.WriteString(match.Render(s[idx : idx+len(query)]))
		i = idx + len(query)
	}
	return b.String()
}
