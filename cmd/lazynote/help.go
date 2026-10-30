package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/ui/helppopup"
	"github.com/tkilb/lazynote/internal/ui/statusbar"
)

// helpGlobalBindings are the keys that work whatever panel has focus. Panel
// switching is movement, so it is under Navigation, as in lazygit.
var helpGlobalBindings = []statusbar.Binding{
	{Key: "F/R/i", Label: "quick fact, rumor or idea (form: tab, ctrl+s saves)"},
	{Key: "I", Label: "import a file (transcript or note)"},
	{Key: "u", Label: "undo"},
	{Key: "ctrl+r", Label: "redo"},
	{Key: "ctrl+l", Label: "refresh from disk"},
	{Key: "ctrl+z", Label: "suspend"},
	{Key: "?", Label: "help"},
	{Key: "q", Label: "quit"},
}

var helpNavigationBindings = []statusbar.Binding{
	{Key: "0-4", Label: "jump to panel"},
	{Key: "tab/shift+tab", Label: "next/prev panel"},
}

// helpParaBindings is the full PARA list; the status bar shows a trimmed one.
var helpParaBindings = []statusbar.Binding{
	{Key: "↑/↓", Label: "move"},
	{Key: "←", Label: "go to the parent directory"},
	{Key: "→/enter", Label: "open a directory or note"},
	{Key: "gg/G", Label: "first/last entry"},
	{Key: "ctrl+u/d", Label: "half page up/down"},
	{Key: "ctrl+b/f", Label: "page up/down"},
	{Key: "[ ]", Label: "previous/next tab"},
	{Key: "space", Label: "select and move down"},
	{Key: "v/V", Label: "visual select/unselect"},
	{Key: "ctrl+a", Label: "select all"},
	{Key: "esc", Label: "leave visual mode, search, selection"},
	{Key: "y/x", Label: "yank/cut"},
	{Key: "Y/X", Label: "cancel yank/cut"},
	{Key: "p/P", Label: "paste/paste over"},
	{Key: "d/D", Label: "trash/delete permanently"},
	{Key: "a", Label: "new note (end with / for folder)"},
	{Key: "r", Label: "rename (updates references)"},
	{Key: "K", Label: "set the note's kind"},
	{Key: "m", Label: "move the note to a tab (keys i p a r A, or arrows)"},
	{Key: "e", Label: "edit in $EDITOR"},
	{Key: "/", Label: "search names"},
	{Key: "n/N", Label: "next/previous match"},
	{Key: ".", Label: "show hidden files"},
}

var helpRelatedBindings = []statusbar.Binding{
	{Key: "↑/↓", Label: "move"},
	{Key: "enter", Label: "jump to the note"},
	{Key: "l", Label: "add a reference (pick a note)"},
	{Key: "d", Label: "remove the selected reference"},
	{Key: "esc/←", Label: "back to PARA"},
}

var helpBodyBindings = []statusbar.Binding{
	{Key: "↑/↓", Label: "scroll"},
	{Key: "ctrl+d/u", Label: "half page down/up"},
	{Key: "esc/←", Label: "back to PARA"},
}

var helpInfoBindings = []statusbar.Binding{
	{Key: "↑/↓", Label: "scroll"},
	{Key: "esc/←", Label: "back to PARA"},
}

// helpSections returns the help popup's content for the focused panel: its
// own keys (Local), the keys that work everywhere (Global) and panel
// movement (Navigation). A panel with no keys of its own has no Local
// section.
func (m model) helpSections() []helppopup.Section {
	sections := make([]helppopup.Section, 0, 3)
	if local := m.helpLocalBindings(); len(local) > 0 {
		sections = append(sections, helppopup.Section{Title: "Local", Bindings: local})
	}
	return append(sections,
		helppopup.Section{Title: "Global", Bindings: helpGlobalBindings},
		helppopup.Section{Title: "Navigation", Bindings: helpNavigationBindings},
	)
}

func (m model) helpLocalBindings() []statusbar.Binding {
	switch m.focus {
	case panelPARA:
		return helpParaBindings
	case panelRelated:
		return helpRelatedBindings
	case panelBody:
		return helpBodyBindings
	case panelInfo:
		return helpInfoBindings
	}
	return nil
}

// updateHelp handles a key while the help popup is open: the up and down arrows
// scroll it, esc and ? close it, and other keys are ignored.
func (m model) updateHelp(msg tea.KeyMsg) model {
	switch msg.String() {
	case "up":
		m.helpScroll = max(m.helpScroll-1, 0)
	case "down":
		m.helpScroll = min(m.helpScroll+1, helppopup.MaxScroll(m.helpSections(), m.height))
	case "esc", "?":
		m.showingHelp, m.helpScroll = false, 0
	}
	return m
}
