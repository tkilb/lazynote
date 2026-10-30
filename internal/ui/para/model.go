// Package para implements the PARA panel: five tabs (Inbox, Projects,
// Areas, Resources, Archives), each listing the notes under its folder in
// the notebook.
package para

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/store"
)

type tabDef struct {
	label string
	dir   string // relative to the notebook root
}

var tabDefs = []tabDef{
	{"Inbox", "inbox"},
	{"Projects", "projects"},
	{"Areas", "areas"},
	{"Resources", "resources"},
	{"Archives", "archives"},
}

type row struct {
	entry store.Entry
	kind  store.Kind // notes only
}

// tabState is one tab's listing. A tab loads when it is first shown and
// remembers its cursor while another tab is active.
type tabState struct {
	loaded bool
	rows   []row
	cursor int
	err    error
}

// Model is the PARA panel state. Methods return updated copies.
type Model struct {
	store  store.Store
	active int
	tabs   [5]tabState
}

// New returns a panel over s with the first tab loaded.
func New(s store.Store) Model {
	return Model{store: s}.load(0)
}

// Reload re-reads the active tab from disk and marks the others stale.
func (m Model) Reload() Model {
	for i, t := range m.tabs {
		m.tabs[i] = tabState{cursor: t.cursor}
	}
	return m.load(m.active)
}

// load reads tab i's listing and each note's kind.
func (m Model) load(i int) Model {
	entries, err := m.store.List(tabDefs[i].dir, store.ListOptions{})
	rows := make([]row, len(entries))
	for j, e := range entries {
		rows[j] = row{entry: e, kind: store.KindUnset}
		if !e.IsDir {
			if n, err := m.store.ReadNote(e.Rel); err == nil {
				rows[j].kind = n.Kind
			}
		}
	}
	t := m.tabs[i]
	t.loaded, t.rows, t.err = true, rows, err
	t.cursor = min(t.cursor, max(len(rows)-1, 0))
	m.tabs[i] = t
	return m
}

// Update handles a key press while the panel has focus.
func (m Model) Update(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "up", "k":
		m.tabs[m.active].cursor = max(m.tabs[m.active].cursor-1, 0)
	case "down", "j":
		t := &m.tabs[m.active]
		t.cursor = min(t.cursor+1, max(len(t.rows)-1, 0))
	case "[":
		return m.switchTab(m.active - 1)
	case "]":
		return m.switchTab(m.active + 1)
	}
	return m
}

func (m Model) switchTab(i int) Model {
	m.active = (i + len(tabDefs)) % len(tabDefs)
	if !m.tabs[m.active].loaded {
		return m.load(m.active)
	}
	return m
}

// SelectedNote returns the entry under the cursor when it is a note.
func (m Model) SelectedNote() (store.Entry, bool) {
	t := m.tabs[m.active]
	if t.cursor >= len(t.rows) || t.rows[t.cursor].entry.IsDir {
		return store.Entry{}, false
	}
	return t.rows[t.cursor].entry, true
}
