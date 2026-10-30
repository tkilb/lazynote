package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

const moveTitle = "Move to"

// moveDestination is a key of the move menu and the tab folder it names.
type moveDestination struct {
	key, folder, label string
}

// moveDestinations are the menu's choices. The keys are case-sensitive: a is
// Areas and A is Archives.
var moveDestinations = []moveDestination{
	{"i", "inbox", "Inbox"},
	{"p", "projects", "Projects"},
	{"a", "areas", "Areas"},
	{"r", "resources", "Resources"},
	{"A", "archives", "Archives"},
}

func moveMenu() []popup.MenuItem {
	items := make([]popup.MenuItem, len(moveDestinations))
	for i, d := range moveDestinations {
		items[i] = popup.MenuItem{Key: d.key, Label: d.label}
	}
	return items
}

// moveKey handles a key while the move menu is open: a destination key moves
// the note at once, the arrows highlight a row and enter moves to it, esc
// closes the menu, and other keys are ignored.
func (m model) moveKey(key string) (tea.Model, tea.Cmd) {
	pick := func(d moveDestination) (tea.Model, tea.Cmd) {
		rel := m.moving
		m.moving = ""
		return m.moveTo(rel, d), nil
	}
	switch key {
	case "esc":
		m.moving = ""
	case "up":
		m.moveCursor = max(m.moveCursor-1, 0)
	case "down":
		m.moveCursor = min(m.moveCursor+1, len(moveDestinations)-1)
	case "enter":
		return pick(moveDestinations[m.moveCursor])
	default:
		for _, d := range moveDestinations {
			if d.key == key {
				return pick(d)
			}
		}
	}
	return m, nil
}

// moveTo moves the note at rel to the root of the destination tab, as one
// undo entry. A note already there stays put.
func (m model) moveTo(rel string, d moveDestination) model {
	dst := store.MoveDestination(rel, d.folder)
	if dst == "" || dst == rel {
		return m.notify(popup.Info, fmt.Sprintf("Already in %s.", d.label))
	}
	act, err := m.store.Move(rel, dst)
	if err != nil {
		return m.notify(popup.Error, capitalize(fmt.Sprintf("move to %s failed: %s.", d.label, err)))
	}
	m.history.Push(act)
	m.para = m.para.Reload()
	return m.refreshLinks()
}
