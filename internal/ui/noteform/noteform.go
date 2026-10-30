// Package noteform implements the quick-add form: a Title field and a
// multi-line Body field, each in its own bordered box with the field name in
// the border and the focused box highlighted, as in lazytask's add-task form.
// Tab and Shift+Tab move between the fields (not the arrows, which the body's
// text area needs for its cursor). The body scrolls within a fixed height.
package noteform

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tkilb/lazynote/internal/ui/panel"
)

// Field identifies one of the form's inputs.
type Field int

const (
	// FieldTitle is the single-line title.
	FieldTitle Field = iota
	// FieldBody is the multi-line body.
	FieldBody

	fieldCount
)

const (
	minWidth         = 40
	maxWidth         = 70
	screenMargin     = 8 // columns and rows kept free around the form
	titleRows        = 3 // the title box: one line plus its border
	boxChrome        = 2 // a box's top and bottom border
	minBodyRows      = 3 // visible body lines, at the smallest
	maxBodyRows      = 8 // and at the largest
	titleLimit       = 120
	bodyLimit        = 20000
	titleHint        = "(tab) body  (enter) save  (esc) cancel"
	bodyHint         = "(tab) title  (ctrl+s) save  (esc) cancel"
	titlePlaceholder = "required"
	bodyPlaceholder  = "optional; enter starts a new line"
)

// Model is the form state. Methods return updated copies.
type Model struct {
	label   string // names the kind, e.g. "Fact", in the title box
	title   textinput.Model
	body    textarea.Model
	focused Field
	width   int
	// bodyRows is how many body lines show at once.
	bodyRows int
}

// New returns an empty form for a note of the kind named label ("Fact"),
// sized for a screenWidth x screenHeight terminal, with the title focused.
// The command blinks the cursor.
func New(label string, screenWidth, screenHeight int) (Model, tea.Cmd) {
	title := textinput.New()
	title.Prompt = ""
	title.Placeholder = titlePlaceholder
	title.CharLimit = titleLimit

	body := textarea.New()
	body.Prompt = ""
	body.ShowLineNumbers = false
	body.Placeholder = bodyPlaceholder
	body.CharLimit = bodyLimit
	// The text area highlights its cursor line by default; keep it flat so
	// both fields look alike.
	flat := lipgloss.NewStyle()
	body.FocusedStyle.CursorLine = flat
	body.FocusedStyle.CursorLineNumber = flat

	m := Model{label: label, title: title, body: body}
	m = m.Resize(screenWidth, screenHeight)
	return m, m.title.Focus()
}

// Resize fits the form to a screenWidth x screenHeight terminal. The body
// area keeps its text and scroll position.
func (m Model) Resize(screenWidth, screenHeight int) Model {
	m.width = min(max(screenWidth-screenMargin, minWidth), maxWidth)
	m.bodyRows = min(max(screenHeight-screenMargin-titleRows-boxChrome, minBodyRows), maxBodyRows)
	inner, _ := panel.InnerSize(m.width, 1)
	m.title.Width = inner
	m.body.SetWidth(inner)
	m.body.SetHeight(m.bodyRows)
	return m
}

// Title returns the title text, untrimmed.
func (m Model) Title() string { return m.title.Value() }

// Body returns the body text, untrimmed, with its line breaks.
func (m Model) Body() string { return m.body.Value() }

// FocusedField reports which field has the keyboard.
func (m Model) FocusedField() Field { return m.focused }

// focus moves the keyboard to f.
func (m Model) focus(f Field) (Model, tea.Cmd) {
	m.focused = f
	m.title.Blur()
	m.body.Blur()
	if f == FieldTitle {
		return m, m.title.Focus()
	}
	return m, m.body.Focus()
}

// Update handles a message: tab and shift+tab move between the fields, and
// everything else goes to the focused one. It does not interpret enter, esc
// or ctrl+s, which the host uses to save and cancel.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab":
			return m.focus((m.focused + 1) % fieldCount)
		case "shift+tab":
			return m.focus((m.focused + fieldCount - 1) % fieldCount)
		}
	}
	var cmd tea.Cmd
	if m.focused == FieldTitle {
		m.title, cmd = m.title.Update(msg)
	} else {
		m.body, cmd = m.body.Update(msg)
		m.body = followCursor(m.body)
	}
	return m, cmd
}

// followCursor scrolls the text area to its cursor right away. The text
// area scrolls during Update but only loads new text into its viewport when
// it renders, so without this the cursor can sit one line below the visible
// rows until the next key press. Rendering loads the text, and an empty
// update then scrolls with it.
func followCursor(t textarea.Model) textarea.Model {
	_ = t.View()
	t, _ = t.Update(nil)
	return t
}

// Blink forwards a non-key message, such as a cursor blink, to both fields.
func (m Model) Blink(msg tea.Msg) (Model, tea.Cmd) {
	var c1, c2 tea.Cmd
	m.title, c1 = m.title.Update(msg)
	m.body, c2 = m.body.Update(msg)
	return m, tea.Batch(c1, c2)
}

// View draws the two boxes, one above the other.
func (m Model) View() string {
	inner, _ := panel.InnerSize(m.width, 1)
	titleBox := panel.Frame(m.label+" title", m.title.View(), inner, 1, m.focused == FieldTitle)

	hint := titleHint
	if m.focused == FieldBody {
		hint = bodyHint
	}
	footer := panel.Footer{Right: hint, Tint: true}
	if total := m.body.LineCount(); total > m.bodyRows {
		footer.Left = fmt.Sprintf("%d/%d", m.body.Line()+1, total)
	}
	bodyBox := panel.Frame("Body", m.body.View(), inner, m.bodyRows, m.focused == FieldBody, footer)
	return strings.Join([]string{titleBox, bodyBox}, "\n")
}
