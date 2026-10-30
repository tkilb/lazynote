package para

import (
	"path/filepath"
	"strings"
)

// Reveal selects the note at rel (relative to the notebook root): it
// switches to the tab holding it, enters its directory and puts the cursor
// on it. It reports false, changing nothing, when rel is not inside a tab
// folder. Anything tied to the directory it leaves (selection, search) is
// dropped, as when leaving it by hand.
func (m Model) Reveal(rel string) (Model, bool) {
	if !filepath.IsLocal(rel) {
		return m, false
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	for i, d := range tabDefs {
		inside, ok := strings.CutPrefix(rel, d.dir+"/")
		if !ok {
			continue
		}
		dir, name := filepath.Split(inside)
		t := m.tabs[i]
		t.dir = filepath.Clean(strings.TrimSuffix(dir, "/"))
		if t.dir == "." {
			t.dir = ""
		}
		t.keep = name
		t.selected, t.visual, t.search = nil, visualState{}, ""
		m.tabs[i] = t
		m.active = i
		return m.load(i), true
	}
	return m, false
}
