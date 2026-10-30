// Package related implements the Related panel: the notes the selected note
// links to (its related property) and the notes that link to it, as two
// groups. Entries that match no file are shown dimmed as unresolved.
package related

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/links"
	"github.com/tkilb/lazynote/internal/ui/panel"
)

const (
	referencesTitle   = "References"
	referencedByTitle = "Referenced by"
	unresolved        = "unresolved"
	noteExt           = ".md"
	indent            = "  "
)

var (
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	selStyle    = lipgloss.NewStyle().Bold(true).Background(panel.SelectedRowBackground)
)

// line is one screen line: a group header or an entry.
type line struct {
	header bool
	// name is what is shown; rel is the note it resolves to, "" when none.
	name, rel string
	// where is the note's folder, shown dim at the right.
	where string
}

// Effect is what a key press asks the root model to do.
type Effect struct {
	// Jump, when set, is the note to select in PARA.
	Jump string
	// Info is a notice to show as a popup.
	Info string
}

// Model is the Related panel state. Methods return updated copies.
type Model struct {
	subject string // what is shown, to keep the cursor on refresh
	lines   []line
	cursor  int // index into lines, always on an entry when there is one
}

// New returns an empty panel.
func New() Model { return Model{} }

// Clear empties the panel.
func (m Model) Clear() Model { return Model{} }

// Show lists the references and the referenced-by notes of the note at rel.
// The cursor stays where it was when rel is already shown.
func (m Model) Show(rel string, references []links.Link, referencedBy []string) Model {
	var lines []line
	if len(references) > 0 {
		lines = append(lines, line{header: true, name: referencesTitle})
		for _, l := range references {
			e := line{name: l.Name, rel: l.Rel}
			if l.Rel != "" {
				e.where = folder(l.Rel)
			} else {
				e.where = unresolved
			}
			lines = append(lines, e)
		}
	}
	if len(referencedBy) > 0 {
		lines = append(lines, line{header: true, name: referencedByTitle})
		for _, r := range referencedBy {
			lines = append(lines, line{name: nameOf(r), rel: r, where: folder(r)})
		}
	}
	cursor := 0
	if rel == m.subject {
		cursor = m.cursor
	}
	m.subject, m.lines = rel, lines
	m.cursor = m.snap(cursor, 1)
	return m
}

// folder is the first folder of rel, the PARA tab it lives in.
func folder(rel string) string {
	if i := strings.Index(rel, "/"); i > 0 {
		return rel[:i]
	}
	return ""
}

// nameOf is rel's file name without its extension.
func nameOf(rel string) string {
	name := rel[strings.LastIndex(rel, "/")+1:]
	if len(name) >= len(noteExt) && strings.EqualFold(name[len(name)-len(noteExt):], noteExt) {
		return name[:len(name)-len(noteExt)]
	}
	return name
}

// snap moves i onto the nearest entry, looking in direction dir first and
// then the other way; it returns 0 when there are no entries.
func (m Model) snap(i, dir int) int {
	i = min(max(i, 0), max(len(m.lines)-1, 0))
	for _, d := range []int{dir, -dir} {
		for j := i; j >= 0 && j < len(m.lines); j += d {
			if !m.lines[j].header {
				return j
			}
		}
	}
	return 0
}

// move steps the cursor to the next entry in direction dir, staying put at
// the ends.
func (m Model) move(dir int) Model {
	for j := m.cursor + dir; j >= 0 && j < len(m.lines); j += dir {
		if !m.lines[j].header {
			m.cursor = j
			break
		}
	}
	return m
}

// Update handles a key press while the panel has focus.
func (m Model) Update(msg tea.KeyMsg) (Model, Effect) {
	switch msg.String() {
	case "up":
		return m.move(-1), Effect{}
	case "down":
		return m.move(1), Effect{}
	case "enter":
		if len(m.lines) == 0 || m.lines[m.cursor].header {
			return m, Effect{}
		}
		l := m.lines[m.cursor]
		if l.rel == "" {
			return m, Effect{Info: l.name + " is unresolved: no note has that name."}
		}
		return m, Effect{Jump: l.rel}
	}
	return m, Effect{}
}

// View renders the panel into outerWidth x outerHeight, border included.
func (m Model) View(outerWidth, outerHeight int, focused bool) string {
	width, height := panel.InnerSize(outerWidth, outerHeight)
	var body string
	if len(m.lines) == 0 {
		body = dimStyle.Render("(none)")
	} else {
		start, end := panel.ScrollWindow(m.cursor, len(m.lines), height)
		rendered := make([]string, 0, end-start)
		for i := start; i < end; i++ {
			rendered = append(rendered, renderLine(m.lines[i], width, i == m.cursor && focused))
		}
		body = strings.Join(rendered, "\n")
	}
	return panel.Frame("3 Related", body, width, height, focused)
}

// renderLine draws one line: a header, or an entry with its folder (or
// "unresolved") right-aligned.
func renderLine(l line, width int, cursor bool) string {
	if l.header {
		return headerStyle.Render(ansi.Truncate(l.name, width, "…"))
	}
	right := l.where
	name := ansi.Truncate(indent+l.name, max(width-ansi.StringWidth(right)-1, 1), "…")
	gap := strings.Repeat(" ", max(width-ansi.StringWidth(name)-ansi.StringWidth(right), 0))
	if cursor {
		return selStyle.Render(name + gap + right)
	}
	if l.rel == "" {
		return dimStyle.Render(name + gap + right)
	}
	return name + gap + dimStyle.Render(right)
}
