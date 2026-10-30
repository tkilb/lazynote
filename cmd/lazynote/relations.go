package main

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/links"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/picker"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

const addReferenceTitle = "Add reference"

// relationEdit is an add or remove of a reference waiting for the user: the
// note it applies to and the name involved.
type removal struct {
	rel, name string
}

// openReferencePicker lists the notes the shown note could reference: every
// note except itself and those it already references.
func (m model) openReferencePicker() (model, tea.Cmd) {
	rel := m.related.Subject()
	taken := map[string]bool{}
	for _, l := range m.links.References(rel) {
		taken[strings.ToLower(l.Name)] = true
	}
	var items []picker.Item
	for _, n := range m.links.Notes() {
		if n.Rel == rel || taken[strings.ToLower(n.Name)] {
			continue
		}
		items = append(items, picker.Item{Label: n.Name, Detail: folderOf(n.Rel), Value: n.Name})
	}
	if len(items) == 0 {
		return m.notify(popup.Info, "There are no other notes to reference."), nil
	}
	p, cmd := picker.New(addReferenceTitle, items)
	m.picker, m.pickerFor = &p, rel
	return m, cmd
}

// folderOf is the first folder of rel, the PARA tab the note lives in.
func folderOf(rel string) string {
	if i := strings.Index(rel, "/"); i > 0 {
		return rel[:i]
	}
	return ""
}

// pickerKey feeds a message to the open picker and applies its outcome.
func (m model) pickerKey(msg tea.Msg) (model, tea.Cmd) {
	p, cmd, out := m.picker.Update(msg)
	m.picker = &p
	switch out {
	case picker.Cancelled:
		m.picker = nil
	case picker.Chosen:
		item, _ := p.Chosen()
		m.picker = nil
		return m.editReference(m.pickerFor, fmt.Sprintf("add reference to %s", item.Value),
			func(fm *store.Frontmatter) { links.AddReference(fm, item.Value) }), nil
	}
	return m, cmd
}

// askRemoval asks before removing the reference name from the note at rel.
func (m model) askRemoval(rel, name string) model {
	m.removal = &removal{rel: rel, name: name}
	return m
}

// answerRemoval handles a key while the removal question is open.
func (m model) answerRemoval(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "enter":
		r := m.removal
		m.removal = nil
		return m.editReference(r.rel, "remove reference to "+r.name,
			func(fm *store.Frontmatter) { links.RemoveReference(fm, r.name) }), nil
	case "n", "esc":
		m.removal = nil
	}
	return m, nil
}

// removalPrompt words the removal question.
func removalPrompt(r *removal) string {
	return fmt.Sprintf("Remove the reference to %q from %s?", r.name, strings.TrimSuffix(filepath.Base(r.rel), filepath.Ext(r.rel)))
}

// editReference changes the properties of the note at rel, records the undo
// action and refreshes the panels. A failure is shown as an error popup.
func (m model) editReference(rel, what string, edit func(*store.Frontmatter)) model {
	act, err := m.store.Edit(rel, edit)
	if err != nil {
		return m.notify(popup.Error, capitalize(fmt.Sprintf("%s failed: %s.", what, err)))
	}
	m.history.Push(act)
	m.para = m.para.Reload()
	return m.refreshLinks()
}
