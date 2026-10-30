package main

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/panel"
	"github.com/tkilb/lazynote/internal/ui/para"
	"github.com/tkilb/lazynote/internal/ui/statusbar"
	"github.com/tkilb/lazynote/internal/undo"
)

// panelID identifies a panel. Its value is the number key that focuses it,
// so Body is 0 and Notebook is 1, matching the titles in the grid.
type panelID int

const (
	panelBody panelID = iota
	panelNotebook
	panelPARA
	panelRelated
	panelInfo
)

var panelNames = map[panelID]string{
	panelBody:     "Body",
	panelNotebook: "Notebook",
	panelPARA:     "PARA",
	panelRelated:  "Related",
	panelInfo:     "Info",
}

// focusCycle is the Tab/Shift+Tab order: the left column top to bottom,
// then the right column top to bottom.
var focusCycle = []panelID{panelNotebook, panelPARA, panelRelated, panelInfo, panelBody}

func (p panelID) next() panelID { return focusCycle[(p.cycleIndex()+1)%len(focusCycle)] }

func (p panelID) prev() panelID {
	return focusCycle[(p.cycleIndex()-1+len(focusCycle))%len(focusCycle)]
}

func (p panelID) cycleIndex() int {
	for i, c := range focusCycle {
		if c == p {
			return i
		}
	}
	return 0
}

// globalBindings are always shown in the status bar, after any bindings of
// the focused panel.
var globalBindings = []statusbar.Binding{
	{Key: "0-4", Label: "focus panel"},
	{Key: "tab", Label: "next panel"},
	{Key: "u", Label: "undo"},
	{Key: "ctrl+z", Label: "suspend"},
	{Key: "q", Label: "quit"},
}

var paraBindings = []statusbar.Binding{
	{Key: "j/k", Label: "move"},
	{Key: "h/l", Label: "up/open"},
	{Key: "[ ]", Label: "tab"},
	{Key: "space", Label: "select"},
	{Key: "y/x/p", Label: "yank/cut/paste"},
	{Key: "d/D", Label: "trash/delete"},
	{Key: "a", Label: "new"},
	{Key: "v", Label: "visual"},
	{Key: ".", Label: "hidden"},
	{Key: "enter", Label: "open"},
}

var bodyBindings = []statusbar.Binding{
	{Key: "esc/h", Label: "back to PARA"},
}

type model struct {
	focus         panelID
	para          para.Model
	width, height int
	// history is shared by pointer: it holds the session's undo/redo stack.
	history *undo.Stack
	// message is a one-line result shown in place of the key hints until the
	// next key press.
	message message
	// pending is a question waiting for y or n; while set, it takes every key.
	pending *para.Confirm
	// input is a one-line text prompt; while set, it takes every key.
	input *input
}

// input is an open text prompt.
type input struct {
	label string
	op    para.InputOp
	field textinput.Model
}

// inputCharLimit bounds what the prompt accepts.
const inputCharLimit = 255

// message is a transient status line.
type message struct {
	text  string
	isErr bool
}

var (
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	confirmStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	dangerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
)

func initialModel(s store.Store) model {
	return model{focus: panelPARA, para: para.New(s), history: &undo.Stack{}}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		g := computeGrid(m.width, m.height-1)
		_, rows := panel.InnerSize(g.leftWidth, g.paraHeight)
		m.para = m.para.Resize(rows)
	case tea.ResumeMsg:
		// Returning from ctrl+z: repaint everything on the alternate screen.
		return m, tea.ClearScreen
	case tea.KeyMsg:
		return m.updateKey(msg)
	default:
		// Cursor blink and similar messages belong to the open prompt.
		if m.input != nil {
			var cmd tea.Cmd
			m.input.field, cmd = m.input.field.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	m.message = message{}
	if key != "ctrl+c" {
		if m.pending != nil {
			return m.answer(key)
		}
		if m.input != nil {
			return m.typeKey(msg)
		}
	}
	var cmd tea.Cmd
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "ctrl+z":
		return m, tea.Suspend
	case "u":
		m = m.runHistory("Undid", "Undo", m.history.Undo)
	case "ctrl+r":
		m = m.runHistory("Redid", "Redo", m.history.Redo)
	case "tab":
		m.focus = m.focus.next()
	case "shift+tab":
		m.focus = m.focus.prev()
	case "0", "1", "2", "3", "4":
		m.focus = panelID(key[0] - '0')
	case "esc", "left", "h":
		switch m.focus {
		case panelBody:
			m.focus = panelPARA
		case panelPARA:
			m, cmd = m.paraKey(msg)
		}
	case "enter", "right", "l":
		if m.focus != panelPARA {
			break
		}
		if _, ok := m.para.SelectedNote(); ok {
			m.focus = panelBody
		} else {
			m, cmd = m.paraKey(msg)
		}
	default:
		if m.focus == panelPARA {
			m, cmd = m.paraKey(msg)
		}
	}
	return m, cmd
}

// paraKey sends a key to the PARA panel and applies what it asks for.
func (m model) paraKey(msg tea.KeyMsg) (model, tea.Cmd) {
	var eff para.Effect
	m.para, eff = m.para.Update(msg)
	return m.apply(eff)
}

// apply records an Effect from a panel: the undo action, the message, or
// the question to ask.
func (m model) apply(eff para.Effect) (model, tea.Cmd) {
	if eff.Action != nil {
		m.history.Push(*eff.Action)
	}
	var cmd tea.Cmd
	switch {
	case eff.Prompt != nil:
		m.input, cmd = openInput(eff.Prompt)
	case eff.Confirm != nil:
		m.pending = eff.Confirm
	case eff.Err != nil:
		m.message = message{text: capitalize(eff.Err.Error()), isErr: true}
	case eff.Info != "":
		m.message = message{text: eff.Info}
	}
	return m, cmd
}

// openInput starts a text prompt and returns the command that blinks its
// cursor.
func openInput(p *para.Prompt) (*input, tea.Cmd) {
	field := textinput.New()
	field.Prompt = ""
	field.CharLimit = inputCharLimit
	field.Focus()
	return &input{label: p.Label, op: p.Op, field: field}, textinput.Blink
}

// typeKey feeds a key to the open prompt: enter submits, esc cancels.
func (m model) typeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input = nil
		m.message = message{text: "Cancelled"}
		return m, nil
	case "enter":
		op, text := m.input.op, m.input.field.Value()
		m.input = nil
		var eff para.Effect
		m.para, eff = m.para.ApplyInput(op, text)
		return m.apply(eff)
	}
	var cmd tea.Cmd
	m.input.field, cmd = m.input.field.Update(msg)
	return m, cmd
}

// answer handles a key while a question is pending: y or enter accepts, n
// or esc declines, anything else is ignored.
func (m model) answer(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "enter":
		op := m.pending.Op
		m.pending = nil
		var eff para.Effect
		m.para, eff = m.para.ApplyConfirm(op)
		return m.apply(eff)
	case "n", "esc":
		m.pending = nil
		m.message = message{text: "Cancelled"}
	}
	return m, nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// runHistory runs an undo or redo, reports the result in the status line,
// and reloads the listing because files may have moved.
func (m model) runHistory(done, failed string, run func() (string, error)) model {
	desc, err := run()
	switch {
	case errors.Is(err, undo.ErrNothingToUndo), errors.Is(err, undo.ErrNothingToRedo):
		m.message = message{text: capitalize(err.Error())}
	case err != nil:
		m.message = message{text: failed + " failed (" + desc + "): " + err.Error(), isErr: true}
	default:
		m.message = message{text: done + ": " + desc}
	}
	m.para = m.para.Reload()
	return m
}

func (m model) statusLine() string {
	switch {
	case m.input != nil:
		return m.input.label + m.input.field.View()
	case m.pending != nil && m.pending.Danger:
		return dangerStyle.Render("(y/n) " + m.pending.Prompt)
	case m.pending != nil:
		return confirmStyle.Render("(y/n) " + m.pending.Prompt)
	case m.message.isErr:
		return errorStyle.Render(m.message.text)
	case m.message.text != "":
		return infoStyle.Render(m.message.text)
	}
	return statusbar.Render(m.bindings())
}

func (m model) bindings() []statusbar.Binding {
	var local []statusbar.Binding
	switch m.focus {
	case panelPARA:
		local = paraBindings
	case panelBody:
		local = bodyBindings
	}
	return append(append([]statusbar.Binding(nil), local...), globalBindings...)
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	g := computeGrid(m.width, m.height-1) // one row for the status bar
	render := func(id panelID, w, h int) string {
		if id == panelPARA {
			return m.para.View(w, h, m.focus == id)
		}
		title := string(rune('0'+id)) + " " + panelNames[id]
		return panel.Render(title, "", w, h, m.focus == id)
	}

	left := lipgloss.JoinVertical(lipgloss.Left,
		render(panelNotebook, g.leftWidth, g.notebookHeight),
		render(panelPARA, g.leftWidth, g.paraHeight),
		render(panelRelated, g.leftWidth, g.relatedHeight),
	)
	right := lipgloss.JoinVertical(lipgloss.Left,
		render(panelInfo, g.rightWidth, g.infoHeight),
		render(panelBody, g.rightWidth, g.bodyHeight),
	)
	grid := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, grid, ansi.Truncate(m.statusLine(), m.width, "…"))
}
