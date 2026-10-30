// Package picker implements a fuzzy picker popup: a text field above a list
// of items that narrows as you type. Matching uses github.com/sahilm/fuzzy
// (ranked for names, in memory, no network). It has no lazytask counterpart:
// lazytask's pickers reuse its list panels inside popups.
package picker

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"

	"github.com/tkilb/lazynote/internal/ui/panel"
)

const (
	minBoxWidth  = 30
	maxBoxWidth  = 60
	maxListRows  = 10
	minListRows  = 3
	boxMargin    = 8 // columns and rows kept free around the box
	chromeRows   = 4 // border (2), field and gap
	charLimit    = 100
	noMatchLabel = "(no matches)"
	hint         = "enter select  esc cancel"
)

var (
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selStyle   = lipgloss.NewStyle().Bold(true).Background(panel.SelectedRowBackground)
	matchColor = lipgloss.Color("3")
)

// Item is one choice. Label is matched against the query and shown; Detail
// is shown dim at the right; Value is what the caller gets back.
type Item struct {
	Label, Detail, Value string
}

// Outcome says how a key press ended the picker, if it did.
type Outcome int

const (
	// Pending means the picker is still open.
	Pending Outcome = iota
	// Chosen means enter picked the highlighted item.
	Chosen
	// Cancelled means esc closed the picker.
	Cancelled
)

// match is an item that fits the query, with the byte offsets of its
// matched characters in Label.
type match struct {
	item int
	at   map[int]bool
}

// Model is the picker state. Methods return updated copies.
type Model struct {
	title   string
	items   []Item
	field   textinput.Model
	matches []match
	cursor  int
}

// New returns a picker over items, listed in the given order until the user
// types. The command blinks the field's cursor.
func New(title string, items []Item) (Model, tea.Cmd) {
	field := textinput.New()
	field.CharLimit = charLimit
	field.Focus()
	m := Model{title: title, items: items, field: field}
	return m.refilter(), textinput.Blink
}

// WithCursor starts the highlight on the i-th listed item (before the user
// types anything), clamped to the list.
func (m Model) WithCursor(i int) Model {
	m.cursor = min(max(i, 0), max(len(m.matches)-1, 0))
	return m
}

// refilter recomputes the matches for the current text and puts the cursor
// on the best one.
func (m Model) refilter() Model {
	m.cursor = 0
	query := m.field.Value()
	if query == "" {
		m.matches = make([]match, len(m.items))
		for i := range m.items {
			m.matches[i] = match{item: i}
		}
		return m
	}
	labels := make([]string, len(m.items))
	for i, it := range m.items {
		labels[i] = it.Label
	}
	found := fuzzy.Find(query, labels)
	m.matches = make([]match, len(found))
	for i, f := range found {
		at := make(map[int]bool, len(f.MatchedIndexes))
		for _, b := range f.MatchedIndexes {
			at[b] = true
		}
		m.matches[i] = match{item: f.Index, at: at}
	}
	return m
}

// Chosen returns the highlighted item, if there is one.
func (m Model) Chosen() (Item, bool) {
	if m.cursor >= len(m.matches) {
		return Item{}, false
	}
	return m.items[m.matches[m.cursor].item], true
}

// Update handles a message while the picker is open: up and down move,
// enter picks, esc cancels, and everything else edits the query.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd, Outcome) {
	key, isKey := msg.(tea.KeyMsg)
	if isKey {
		switch key.String() {
		case "esc":
			return m, nil, Cancelled
		case "enter":
			if _, ok := m.Chosen(); ok {
				return m, nil, Chosen
			}
			return m, nil, Pending
		case "up":
			m.cursor = max(m.cursor-1, 0)
			return m, nil, Pending
		case "down":
			m.cursor = min(m.cursor+1, max(len(m.matches)-1, 0))
			return m, nil, Pending
		}
	}
	before := m.field.Value()
	var cmd tea.Cmd
	m.field, cmd = m.field.Update(msg)
	if m.field.Value() != before {
		m = m.refilter()
	}
	return m, cmd, Pending
}

// View renders the picker as a bordered box sized for a screenWidth x
// screenHeight terminal; draw it with popup.Overlay.
func (m Model) View(screenWidth, screenHeight int) string {
	boxWidth := min(max(screenWidth-boxMargin, minBoxWidth), maxBoxWidth)
	inner := boxWidth - 2
	rows := min(max(screenHeight-boxMargin-chromeRows, minListRows), maxListRows)
	rows = min(rows, max(len(m.matches), 1))

	m.field.Width = max(inner-lipgloss.Width(m.field.Prompt)-1, 1)
	lines := []string{m.field.View()}
	if len(m.matches) == 0 {
		lines = append(lines, dimStyle.Render(noMatchLabel))
	} else {
		start, end := panel.ScrollWindow(m.cursor, len(m.matches), rows)
		for i := start; i < end; i++ {
			lines = append(lines, m.row(m.matches[i], inner, i == m.cursor))
		}
	}
	footer := panel.Footer{Left: hint, Tint: true}
	if len(m.matches) > 0 {
		footer.Right = strings.Join([]string{itoa(m.cursor + 1), itoa(len(m.matches))}, "/")
	}
	return panel.Frame(m.title, strings.Join(lines, "\n"), inner, len(lines), true, footer)
}

// row draws one item: its label with the matched characters picked out, and
// its detail right-aligned.
func (m Model) row(mt match, width int, cursor bool) string {
	it := m.items[mt.item]
	detail := ansi.Truncate(it.Detail, width/2, "…")
	label := ansi.Truncate(it.Label, max(width-lipgloss.Width(detail)-1, 1), "…")
	gap := strings.Repeat(" ", max(width-lipgloss.Width(label)-lipgloss.Width(detail), 0))

	base := lipgloss.NewStyle()
	if cursor {
		base = selStyle
	}
	hit := base.Foreground(matchColor).Underline(true)
	var b strings.Builder
	for i, r := range label {
		if mt.at[i] {
			b.WriteString(hit.Render(string(r)))
		} else {
			b.WriteString(base.Render(string(r)))
		}
	}
	if cursor {
		return b.String() + selStyle.Render(gap+detail)
	}
	return b.String() + gap + dimStyle.Render(detail)
}

func itoa(n int) string { return strconv.Itoa(n) }
