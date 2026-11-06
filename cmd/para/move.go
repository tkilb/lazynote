package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/para/internal/store"
	"github.com/tkilb/para/internal/ui/popup"
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

// openMoveMenu asks where to move the note at rel.
func (m model) openMoveMenu(rel string) model {
	dests := moveDestinations
	if _, daily := store.DailyDate(rel); daily {
		// A daily note is scratch space: digest it, then archive it.
		dests = moveDestinations[len(moveDestinations)-1:]
	}
	items := make([]popup.MenuItem, len(dests))
	for i, d := range dests {
		items[i] = popup.MenuItem{Key: d.key, Label: d.label}
	}
	return m.openMenu(moveTitle, items, func(m model, key string) model {
		for _, d := range dests {
			if d.key == key {
				return m.moveTo(rel, d)
			}
		}
		return m
	})
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

// menu is a small box of choices: the next key press picks one, or the
// arrows highlight a row and enter picks it.
type menu struct {
	title  string
	items  []popup.MenuItem
	cursor int
	// pick runs with the key of the chosen item.
	pick func(model, string) model
}

func (m model) openMenu(title string, items []popup.MenuItem, pick func(model, string) model) model {
	m.menu = &menu{title: title, items: items, pick: pick}
	return m
}

// menuKey handles a key while a menu is open: an item's key picks it, the
// arrows highlight and enter picks, esc closes, other keys are ignored.
func (m model) menuKey(key string) (tea.Model, tea.Cmd) {
	mn := m.menu
	choose := func(k string) (tea.Model, tea.Cmd) {
		m.menu = nil
		return mn.pick(m, k), nil
	}
	switch key {
	case "esc":
		m.menu = nil
	case "up":
		mn.cursor = max(mn.cursor-1, 0)
	case "down":
		mn.cursor = min(mn.cursor+1, len(mn.items)-1)
	case "enter":
		return choose(mn.items[mn.cursor].Key)
	default:
		for _, it := range mn.items {
			if it.Key == key {
				return choose(key)
			}
		}
	}
	return m, nil
}
