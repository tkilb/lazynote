// Package noteform implements the Quick Add form: a Title, a multi-line Body,
// a Kind chooser and a Tags field, each in its own bordered box with the
// field name in the border and the focused box highlighted, as in lazytask's
// add-task form. Tab and Shift+Tab move between the fields (not the arrows,
// which the body's text area needs for its cursor). The body scrolls within a
// fixed height, and Left and Right change the kind.
package noteform

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/kindstyle"
	"github.com/tkilb/lazynote/internal/ui/panel"
)

// Field identifies one of the form's inputs, in tab order.
type Field int

const (
	// FieldTitle is the single-line title.
	FieldTitle Field = iota
	// FieldBody is the multi-line body.
	FieldBody
	// FieldKind is the kind chooser.
	FieldKind
	// FieldTags is the single-line tag list.
	FieldTags

	fieldCount
)

const (
	minWidth     = 40
	maxWidth     = 70
	screenMargin = 8 // columns and rows kept free around the form
	lineBoxRows  = 3 // a one-line box: its line plus the border
	boxChrome    = 2 // a box's top and bottom border
	lineBoxes    = 3 // the title, kind and tags boxes
	minBodyRows  = 3 // visible body lines, at the smallest
	maxBodyRows  = 8 // and at the largest
	titleLimit   = 120
	tagsLimit    = 200
	bodyLimit    = 20000

	titlePlaceholder = "required"
	bodyPlaceholder  = "optional; enter starts a new line"
	tagsPlaceholder  = "optional; comma or space separated"
	noKindMeaning    = "No kind yet"
)

// hints are the key hints for each focused field, shown on the bottom border
// of the last box.
var hints = [fieldCount]string{
	FieldTitle: "(tab) next  (enter) save  (esc) cancel",
	FieldBody:  "(tab) next  (ctrl+s) save  (esc) cancel",
	FieldKind:  "(←/→) change  (enter) save  (esc) cancel",
	FieldTags:  "(tab) next  (enter) save  (esc) cancel",
}

// Values are what the user entered.
type Values struct {
	Title, Body string
	// Kind is KindUnset when none was chosen.
	Kind store.Kind
	// Tags is the tags text as typed, to be parsed by the caller.
	Tags string
}

// Model is the form state. Methods return updated copies.
type Model struct {
	name    string // heads the title box, e.g. "Quick Add"
	title   textinput.Model
	body    textarea.Model
	tags    textinput.Model
	kinds   []store.Kind // unset first, then the six kinds
	kindIdx int
	focused Field
	width   int
	// bodyRows is how many body lines show at once.
	bodyRows int
}

// New returns an empty form named name ("Quick Add"), sized for a
// screenWidth x screenHeight terminal, with the title focused. The command
// blinks the cursor.
func New(name string, screenWidth, screenHeight int) (Model, tea.Cmd) {
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
	// all fields look alike.
	flat := lipgloss.NewStyle()
	body.FocusedStyle.CursorLine = flat
	body.FocusedStyle.CursorLineNumber = flat

	tags := textinput.New()
	tags.Prompt = ""
	tags.Placeholder = tagsPlaceholder
	tags.CharLimit = tagsLimit

	m := Model{
		name: name, title: title, body: body, tags: tags,
		kinds: append([]store.Kind{store.KindUnset}, store.Kinds()...),
	}
	m = m.Resize(screenWidth, screenHeight)
	return m.focus(FieldTitle)
}

// WithValues fills the form with v and puts the keyboard on f, for showing a
// form again after a mistake without losing what was typed.
func (m Model) WithValues(v Values, f Field) (Model, tea.Cmd) {
	m.title.SetValue(v.Title)
	m.body.SetValue(v.Body)
	m.tags.SetValue(v.Tags)
	for i, k := range m.kinds {
		if k == v.Kind {
			m.kindIdx = i
		}
	}
	return m.focus(f)
}

// Resize fits the form to a screenWidth x screenHeight terminal. The body
// area keeps its text and scroll position.
func (m Model) Resize(screenWidth, screenHeight int) Model {
	m.width = min(max(screenWidth-screenMargin, minWidth), maxWidth)
	fixed := lineBoxes*lineBoxRows + boxChrome
	m.bodyRows = min(max(screenHeight-screenMargin-fixed, minBodyRows), maxBodyRows)
	inner, _ := panel.InnerSize(m.width, 1)
	m.title.Width = inner
	m.tags.Width = inner
	m.body.SetWidth(inner)
	m.body.SetHeight(m.bodyRows)
	return m
}

// Values returns what has been entered.
func (m Model) Values() Values {
	return Values{Title: m.title.Value(), Body: m.body.Value(), Kind: m.kinds[m.kindIdx], Tags: m.tags.Value()}
}

// FocusedField reports which field has the keyboard.
func (m Model) FocusedField() Field { return m.focused }

// focus moves the keyboard to f.
func (m Model) focus(f Field) (Model, tea.Cmd) {
	m.focused = f
	m.title.Blur()
	m.body.Blur()
	m.tags.Blur()
	var cmd tea.Cmd
	switch f {
	case FieldTitle:
		cmd = m.title.Focus()
	case FieldBody:
		cmd = m.body.Focus()
	case FieldTags:
		cmd = m.tags.Focus()
	}
	return m, cmd
}

// Update handles a message: tab and shift+tab move between the fields,
// Left and Right change the kind while it has the focus, and everything else
// goes to the focused field. It does not interpret enter, esc or ctrl+s,
// which the host uses to save and cancel.
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
	switch m.focused {
	case FieldTitle:
		m.title, cmd = m.title.Update(msg)
	case FieldBody:
		m.body, cmd = m.body.Update(msg)
		m.body = followCursor(m.body)
	case FieldKind:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "left":
				m.kindIdx = (m.kindIdx + len(m.kinds) - 1) % len(m.kinds)
			case "right":
				m.kindIdx = (m.kindIdx + 1) % len(m.kinds)
			}
		}
	case FieldTags:
		m.tags, cmd = m.tags.Update(msg)
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

// Blink forwards a non-key message, such as a cursor blink, to the fields.
func (m Model) Blink(msg tea.Msg) (Model, tea.Cmd) {
	var c1, c2, c3 tea.Cmd
	m.title, c1 = m.title.Update(msg)
	m.body, c2 = m.body.Update(msg)
	m.tags, c3 = m.tags.Update(msg)
	return m, tea.Batch(c1, c2, c3)
}

// View draws the boxes, one above the other.
func (m Model) View() string {
	inner, _ := panel.InnerSize(m.width, 1)
	titleBox := panel.Frame(m.name+": Title", m.title.View(), inner, 1, m.focused == FieldTitle)

	bodyFooter := panel.Footer{}
	if total := m.body.LineCount(); total > m.bodyRows {
		bodyFooter.Left = fmt.Sprintf("%d/%d", m.body.Line()+1, total)
	}
	bodyBox := panel.Frame("Body", m.body.View(), inner, m.bodyRows, m.focused == FieldBody, bodyFooter)

	kindBox := panel.Frame("Kind", m.kindLine(), inner, 1, m.focused == FieldKind)

	tagsFooter := panel.Footer{Right: hints[m.focused], Tint: true}
	tagsBox := panel.Frame("Tags", m.tags.View(), inner, 1, m.focused == FieldTags, tagsFooter)
	return strings.Join([]string{titleBox, bodyBox, kindBox, tagsBox}, "\n")
}

// kindLine shows the chosen kind in its color between arrows, with what it
// means.
func (m Model) kindLine() string {
	k := m.kinds[m.kindIdx]
	meaning := k.Meaning()
	if k == store.KindUnset {
		meaning = noKindMeaning
	}
	name := lipgloss.NewStyle().Bold(true).Foreground(kindstyle.Color(k)).Render(k.String())
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	return dim.Render("‹ ") + name + dim.Render(" ›  "+meaning)
}
