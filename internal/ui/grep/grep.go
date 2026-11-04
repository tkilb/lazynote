// Package grep implements the text search popup: a query box and a list of
// matching lines on the left, and a preview of the note around the chosen
// line on the right, as the Snacks grep picker in Neovim does. The matching
// is exact words, ignoring case (see internal/search).
package grep

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/findtext"
	"github.com/tkilb/lazynote/internal/search"
)

const (
	// maxHits bounds the lines listed; the count still says how many matched.
	maxHits   = 500
	charLimit = 100
)

// Outcome says how a key press ended the popup, if it did.
type Outcome int

const (
	// Pending means the popup is still open.
	Pending Outcome = iota
	// Chosen means enter picked the highlighted line.
	Chosen
	// Cancelled means esc closed the popup.
	Cancelled
)

// Loader returns the lines of the note at rel, as the Content panel shows
// them, for the preview.
type Loader func(rel string) []string

// Model is the popup state. Methods return updated copies.
type Model struct {
	field  textinput.Model
	lines  []findtext.Line
	capped bool // the lines were cut at findtext.MaxLines
	load   Loader

	res    search.Result
	cursor int

	// preview is the note of the highlighted line; scroll moves the view of
	// it away from the line, in lines.
	previewRel string
	preview    []string
	scroll     int

	screenW, screenH int
}

// New returns the popup over lines, which capped says were cut short. load
// reads a note for the preview. The command blinks the query's cursor.
func New(lines []findtext.Line, capped bool, load Loader) (Model, tea.Cmd) {
	field := textinput.New()
	field.CharLimit = charLimit
	field.Prompt = "> "
	field.PromptStyle = promptStyle
	field.Focus()
	m := Model{field: field, lines: lines, capped: capped, load: load}
	return m, textinput.Blink
}

// WithScreen tells the popup how big the terminal is, so paging and the
// preview scroll move by what the panes show.
func (m Model) WithScreen(w, h int) Model {
	m.screenW, m.screenH = w, h
	return m
}

// Chosen returns the highlighted line, if there is one.
func (m Model) Chosen() (search.Hit, bool) {
	if m.cursor < 0 || m.cursor >= len(m.res.Hits) {
		return search.Hit{}, false
	}
	return m.res.Hits[m.cursor], true
}

// Update handles a message: the arrows move through the lines, page up and
// down move by a page, ctrl+d and ctrl+u scroll the preview, enter chooses,
// esc cancels, and anything else edits the query.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd, Outcome) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			return m, nil, Cancelled
		case "enter":
			if _, ok := m.Chosen(); ok {
				return m, nil, Chosen
			}
			return m, nil, Pending
		case "up", "ctrl+p":
			return m.move(-1), nil, Pending
		case "down", "ctrl+n":
			return m.move(1), nil, Pending
		case "pgup":
			return m.move(-m.pageRows()), nil, Pending
		case "pgdown":
			return m.move(m.pageRows()), nil, Pending
		case "ctrl+u":
			return m.scrollPreview(-m.pageRows() / 2), nil, Pending
		case "ctrl+d":
			return m.scrollPreview(m.pageRows() / 2), nil, Pending
		}
	}
	before := m.field.Value()
	var cmd tea.Cmd
	m.field, cmd = m.field.Update(msg)
	if m.field.Value() != before {
		m.res = search.Find(m.lines, m.field.Value(), maxHits)
		m.cursor, m.scroll = 0, 0
		m = m.syncPreview()
	}
	return m, cmd, Pending
}

// move puts the highlight delta lines away, clamped.
func (m Model) move(delta int) Model {
	m.cursor = min(max(m.cursor+delta, 0), max(len(m.res.Hits)-1, 0))
	m.scroll = 0
	return m.syncPreview()
}

// scrollPreview moves the preview delta lines away from the highlighted line.
func (m Model) scrollPreview(delta int) Model {
	m.scroll += delta
	return m
}

// syncPreview loads the note of the highlighted line when it changed.
func (m Model) syncPreview() Model {
	h, ok := m.Chosen()
	if !ok {
		m.previewRel, m.preview = "", nil
		return m
	}
	if h.Rel != m.previewRel {
		m.previewRel, m.preview = h.Rel, m.load(h.Rel)
	}
	return m
}
