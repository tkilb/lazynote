package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/kindstyle"
	"github.com/tkilb/lazynote/internal/ui/para"
	"github.com/tkilb/lazynote/internal/ui/picker"
)

const setKindTitle = "Set kind"

// openKindPicker lists the six kinds with their meanings, the highlight on
// the note's current one.
func (m model) openKindPicker(req *para.KindRequest) (model, tea.Cmd) {
	var items []picker.Item
	current := 0
	for i, k := range store.Kinds() {
		items = append(items, picker.Item{Label: k.String(), Detail: k.Meaning(), Value: k.String(), Color: kindstyle.Color(k)})
		if k == req.Current {
			current = i
		}
	}
	p, cmd := picker.New(setKindTitle, items)
	p = p.WithCursor(current)
	m.picker, m.pickerFor, m.pickerOp, m.pickerKind = &p, req.Rel, pickKind, req.Current
	return m, cmd
}

// setKind writes the kind property of the note at rel, as one undo entry.
// Choosing the kind the note already has changes nothing.
func (m model) setKind(rel string, k store.Kind) model {
	if k == m.pickerKind {
		return m
	}
	return m.editNote(rel, "set kind to "+k.String(), func(fm *store.Frontmatter) { store.SetKind(fm, k) })
}
