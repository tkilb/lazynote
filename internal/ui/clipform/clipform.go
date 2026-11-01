// Package clipform implements the web clip form: a Title and a URL, each in
// its own bordered box with the field name in the border and the focused box
// highlighted, as in the Quick Add form. Tab, Shift+Tab and the arrows move
// between the two boxes. The host saves on enter and cancels on esc.
package clipform

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/ui/panel"
)

// Field identifies one of the form's inputs, in tab order.
type Field int

const (
	// FieldTitle is the optional title.
	FieldTitle Field = iota
	// FieldURL is the web address.
	FieldURL

	fieldCount
)

const (
	minWidth     = 40
	maxWidth     = 70
	screenMargin = 8
	titleLimit   = 120
	urlLimit     = 2048

	titlePlaceholder = "optional; the page's own title when blank"
	urlPlaceholder   = "required; an http or https address"
	hint             = "(tab/↑↓) move  (enter) save  (esc) cancel"
)

// Values are what the user entered.
type Values struct {
	Title, URL string
}

// Model is the form state. Methods return updated copies.
type Model struct {
	title, url textinput.Model
	focused    Field
	width      int
}

// New returns a form with url filled in, sized for a screenWidth x
// screenHeight terminal, with the URL focused. The command blinks the cursor.
func New(url string, screenWidth, screenHeight int) (Model, tea.Cmd) {
	title := textinput.New()
	title.Prompt = ""
	title.Placeholder = titlePlaceholder
	title.CharLimit = titleLimit

	addr := textinput.New()
	addr.Prompt = ""
	addr.Placeholder = urlPlaceholder
	addr.CharLimit = urlLimit
	addr.SetValue(url)

	m := Model{title: title, url: addr}.Resize(screenWidth, screenHeight)
	return m.focus(FieldURL)
}

// WithValues fills the form with v and puts the keyboard on f, for showing a
// form again after a mistake without losing what was typed.
func (m Model) WithValues(v Values, f Field) (Model, tea.Cmd) {
	m.title.SetValue(v.Title)
	m.url.SetValue(v.URL)
	return m.focus(f)
}

// Resize fits the form to a screenWidth x screenHeight terminal.
func (m Model) Resize(screenWidth, _ int) Model {
	m.width = min(max(screenWidth-screenMargin, minWidth), maxWidth)
	inner, _ := panel.InnerSize(m.width, 1)
	m.title.Width = inner
	m.url.Width = inner
	return m
}

// Values returns what has been entered.
func (m Model) Values() Values {
	return Values{Title: m.title.Value(), URL: m.url.Value()}
}

// FocusedField reports which field has the keyboard.
func (m Model) FocusedField() Field { return m.focused }

// focus moves the keyboard to f.
func (m Model) focus(f Field) (Model, tea.Cmd) {
	m.focused = f
	m.title.Blur()
	m.url.Blur()
	if f == FieldTitle {
		return m, m.title.Focus()
	}
	return m, m.url.Focus()
}

// Update handles a message: tab and shift+tab move between the boxes (and
// wrap), up and down move too (and stop at the ends), and everything else
// goes to the focused box. It does not interpret enter or esc, which the host
// uses to save and cancel.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab":
			return m.focus((m.focused + 1) % fieldCount)
		case "shift+tab":
			return m.focus((m.focused + fieldCount - 1) % fieldCount)
		case "up":
			return m.focus(FieldTitle)
		case "down":
			return m.focus(FieldURL)
		}
	}
	var cmd tea.Cmd
	if m.focused == FieldTitle {
		m.title, cmd = m.title.Update(msg)
	} else {
		m.url, cmd = m.url.Update(msg)
	}
	return m, cmd
}

// Blink forwards a non-key message, such as a cursor blink, to the boxes.
func (m Model) Blink(msg tea.Msg) (Model, tea.Cmd) {
	var c1, c2 tea.Cmd
	m.title, c1 = m.title.Update(msg)
	m.url, c2 = m.url.Update(msg)
	return m, tea.Batch(c1, c2)
}

// View draws the two boxes, one above the other.
func (m Model) View() string {
	inner, _ := panel.InnerSize(m.width, 1)
	titleBox := panel.Frame("Clip: Title", m.title.View(), inner, 1, m.focused == FieldTitle)
	urlBox := panel.Frame("URL", m.url.View(), inner, 1, m.focused == FieldURL,
		panel.Footer{Right: hint, Tint: true})
	return strings.Join([]string{titleBox, urlBox}, "\n")
}
