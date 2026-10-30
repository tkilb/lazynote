package para

import (
	"path/filepath"

	"github.com/tkilb/lazynote/internal/state"
)

// Snapshot returns the position to remember: the active tab and, for each
// tab, its current directory and the name under the cursor.
func (m Model) Snapshot() state.Notebook {
	nb := state.Notebook{ActiveTab: m.active}
	for i, t := range m.tabs {
		nb.Tabs[i] = state.Tab{Dir: t.dir, Name: t.wantedName()}
	}
	return nb
}

// Restore puts the panel back at a saved position. Saved text is not
// trusted: a directory that is not a plain relative path inside the tab, or
// a name that is not a single file name, is dropped. A directory that no
// longer exists is handled by the next load, which climbs to the nearest one
// that does.
func (m Model) Restore(nb state.Notebook) Model {
	for i := range m.tabs {
		t := nb.Tabs[i]
		m.tabs[i] = tabState{dir: localDir(t.Dir), keep: localName(t.Name)}
	}
	m.active = min(max(nb.ActiveTab, 0), len(m.tabs)-1)
	return m.load(m.active)
}

// localDir returns dir when it is a relative path that stays inside its
// root, and the root ("") otherwise.
func localDir(dir string) string {
	if dir != "" && filepath.IsLocal(dir) {
		return filepath.Clean(dir)
	}
	return ""
}

// localName returns name when it is a single file name, and "" otherwise.
func localName(name string) string {
	if name != "" && filepath.IsLocal(name) && filepath.Base(name) == name {
		return name
	}
	return ""
}
