// Package info implements the Info panel: the kind and file name of the
// selected note followed by every frontmatter property except the uid, in
// file order, with nested values drawn generically. (The title has its own
// panel.) For a folder it shows the name and item count.
package info

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"go.yaml.in/yaml/v3"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/kindstyle"
	"github.com/tkilb/lazynote/internal/ui/panel"
)

const (
	// maxLabelWidth caps the label column; longer keys are shortened.
	maxLabelWidth = 14
	// defaultHeight is the page size until the first Resize.
	defaultHeight = 3
	// maxValueLines bounds how many lines one note's properties can take.
	maxValueLines = 400
	// maxDepth bounds how deep nested values and YAML aliases are followed.
	maxDepth = 8
	// uidKey is the stable id property, never shown.
	uidKey = "uid"
	// kindKey is shown in the highlighted row, not as a property.
	kindKey = "kind"
	// titleKey is not repeated as a property when it is the title shown.
	titleKey = "title"
	// labelGap separates the label column from the values.
	labelGap = 2
)

var (
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	plain      = lipgloss.NewStyle()
)

// row is one label with its value lines.
type row struct {
	label string
	value []string
	// style draws the value; highlight draws the whole row on a colored
	// background instead (the kind row).
	style     lipgloss.Style
	highlight lipgloss.Color
}

// line is one wrapped screen line.
type line struct {
	text  string
	style lipgloss.Style
}

// Model is the Info panel state. Methods return updated copies.
type Model struct {
	subject string // identifies what is shown, to keep the scroll on refresh
	rows    []row
	lines   []line
	offset  int

	width, height int
}

// New returns an empty panel.
func New() Model { return Model{height: defaultHeight} }

// Resize sets the content area size and re-lays out the rows.
func (m Model) Resize(width, height int) Model {
	m.width, m.height = width, max(height, 1)
	return m.layout()
}

// Clear empties the panel.
func (m Model) Clear() Model {
	return Model{width: m.width, height: m.height}
}

// ShowNote shows the note read from rel. The scroll is kept when rel is
// already shown and reset otherwise.
func (m Model) ShowNote(rel string, n store.Note) Model {
	rows := []row{
		{label: "Kind", value: []string{n.Kind.String()}, highlight: kindstyle.Color(n.Kind)},
		{label: "File", value: single(n.Name), style: plain},
	}
	if n.FrontmatterErr != nil {
		rows = append(rows, row{
			label: "Properties", value: []string{"invalid frontmatter, left unchanged"}, style: errorStyle,
		})
	}
	if n.Frontmatter != nil {
		budget := maxValueLines
		for _, p := range n.Frontmatter.Properties() {
			if p.Key == uidKey ||
				(p.Key == kindKey && n.Kind != store.KindUnset) ||
				(p.Key == titleKey && p.Value.Kind == yaml.ScalarNode && strings.TrimSpace(p.Value.Value) == n.Title) {
				continue
			}
			rows = append(rows, row{label: p.Key, value: valueLines(p.Value, 0, &budget), style: plain})
		}
	}
	return m.show(rel, rows)
}

// ShowDir shows a folder's name and the number of entries it lists.
func (m Model) ShowDir(rel, name string, items int) Model {
	return m.show(rel, []row{
		{label: "Folder", value: single(name), style: plain},
		{label: "Items", value: []string{fmt.Sprint(items)}, style: plain},
	})
}

// ShowError shows that the note at rel could not be read.
func (m Model) ShowError(rel, name string, err error) Model {
	return m.show(rel, []row{
		{label: "File", value: single(name), style: plain},
		{label: "Error", value: single(err.Error()), style: errorStyle},
	})
}

func (m Model) show(subject string, rows []row) Model {
	if subject != m.subject {
		m.offset = 0
	}
	m.subject, m.rows = subject, rows
	return m.layout()
}

// layout wraps every row to the panel width.
func (m Model) layout() Model {
	m.lines = nil
	if m.width > 0 {
		labelWidth := 0
		for _, r := range m.rows {
			labelWidth = max(labelWidth, min(ansi.StringWidth(r.label), maxLabelWidth))
		}
		valueWidth := max(m.width-labelWidth-labelGap, 1)
		for _, r := range m.rows {
			m.lines = append(m.lines, rowLines(r, labelWidth, valueWidth, m.width)...)
		}
	}
	m.offset = m.clamp(m.offset)
	return m
}

// rowLines renders one row: the label on its first line, continuation lines
// indented under the value.
func rowLines(r row, labelWidth, valueWidth, width int) []line {
	var wrapped []string
	for _, v := range r.value {
		wrapped = append(wrapped, strings.Split(ansi.Wrap(v, valueWidth, ""), "\n")...)
	}
	if len(wrapped) == 0 {
		wrapped = []string{""}
	}

	label := ansi.Truncate(r.label, labelWidth, "…")
	label += strings.Repeat(" ", max(labelWidth-ansi.StringWidth(label), 0))
	indent := strings.Repeat(" ", labelWidth)

	out := make([]line, len(wrapped))
	for i, v := range wrapped {
		head := indent
		if i == 0 {
			head = label
		}
		switch {
		case r.highlight != "":
			text := head + strings.Repeat(" ", labelGap) + v
			text += strings.Repeat(" ", max(width-ansi.StringWidth(text), 0))
			out[i] = line{text: text, style: lipgloss.NewStyle().Bold(true).
				Background(r.highlight).Foreground(lipgloss.Color("0"))}
		default:
			out[i] = line{text: labelStyle.Render(head) + strings.Repeat(" ", labelGap) + r.style.Render(v)}
		}
	}
	return out
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
	}
	return m
}

// View renders the panel into outerWidth x outerHeight, border included.
func (m Model) View(outerWidth, outerHeight int, focused bool) string {
	width, height := panel.InnerSize(outerWidth, outerHeight)
	end := min(m.offset+height, len(m.lines))
	rendered := make([]string, 0, end-m.offset)
	for _, l := range m.lines[min(m.offset, end):end] {
		rendered = append(rendered, l.style.Render(l.text))
	}

	var footer panel.Footer
	if len(m.lines) > height {
		footer.Right = fmt.Sprintf("%d/%d", m.offset+1, len(m.lines))
	}
	return panel.Frame("4 Info", strings.Join(rendered, "\n"), width, height, focused, footer)
}
