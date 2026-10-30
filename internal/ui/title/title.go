// Package title implements the Title panel: one bordered line showing the
// title of the selected note, led by its kind badge. It is display only: it
// has no number key, is not part of the focus cycle, and never draws as
// focused.
package title

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/textsafe"
	"github.com/tkilb/lazynote/internal/ui/kindstyle"
	"github.com/tkilb/lazynote/internal/ui/panel"
)

// panelTitle is the label in the top border.
const panelTitle = "Title"

var textStyle = lipgloss.NewStyle().Bold(true)

// Model is the Title panel state. Methods return updated copies.
type Model struct {
	text string
	// badge is the kind badge shown before the text, empty for none.
	badge string
	color lipgloss.Color
}

// New returns an empty panel.
func New() Model { return Model{} }

// Set shows text, which may come from an untrusted note, on one line, with
// no badge (for a folder, an unreadable note, or nothing selected).
func (m Model) Set(text string) Model {
	return Model{text: strings.TrimSpace(textsafe.SingleLine(text))}
}

// SetNote shows a note's title led by the badge for its kind.
func (m Model) SetNote(kind store.Kind, title string) Model {
	m = m.Set(title)
	m.badge, m.color = kindstyle.Badge(kind)
	return m
}

// View renders the panel into outerWidth x outerHeight, border included.
func (m Model) View(outerWidth, outerHeight int) string {
	width, height := panel.InnerSize(outerWidth, outerHeight)

	var lead string
	textWidth := width
	if m.badge != "" {
		pill := lipgloss.NewStyle().Bold(true).Background(m.color).Foreground(lipgloss.Color("0")).
			Render(" " + m.badge + " ")
		lead = pill + " "
		textWidth -= ansi.StringWidth(lead)
	}
	body := lead + textStyle.Render(ansi.Truncate(m.text, max(textWidth, 1), "…"))
	return panel.Frame(panelTitle, body, width, height, false)
}
