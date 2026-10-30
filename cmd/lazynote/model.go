package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/panel"
	"github.com/tkilb/lazynote/internal/ui/para"
	"github.com/tkilb/lazynote/internal/ui/popup"
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
	{Key: "ctrl+l", Label: "refresh"},
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
	{Key: "/", Label: "search"},
	{Key: "n/N", Label: "next/prev match"},
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
	// popups queues notifications; the front one blocks input until a key
	// dismisses it.
	popups popup.Model
	// pending is a yes/no question waiting for an answer.
	pending *para.Confirm
	// input is an open one-line text prompt.
	input *input
}

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
	case tea.FocusMsg:
		// Files may have changed while the terminal was in the background.
		// A pending question was worded for the rows on screen, so it waits.
		if m.pending == nil {
			m.para = m.para.Reload()
		}
		return m, nil
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
	if key != "ctrl+c" {
		switch {
		case m.popups.Active():
			m.popups = m.popups.Dismiss()
			return m, nil
		case m.pending != nil:
			return m.answer(key)
		case m.input != nil:
			return m.typeKey(msg)
		}
	}
	var cmd tea.Cmd
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "ctrl+z":
		return m, tea.Suspend
	case "ctrl+l":
		m.para = m.para.Reload()
		cmd = tea.ClearScreen
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
	view := lipgloss.JoinVertical(lipgloss.Left, grid, ansi.Truncate(m.statusLine(), m.width, "…"))
	return m.overlayModals(view)
}
