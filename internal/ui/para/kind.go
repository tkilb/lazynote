package para

import "github.com/tkilb/lazynote/internal/store"

// KindRequest asks to choose a new kind for the note at Rel, which has
// Current now.
type KindRequest struct {
	Rel     string
	Current store.Kind
}

// askKind asks the root model to open the kind picker for the hovered note.
// A folder has no kind, so it does nothing.
func (m Model) askKind() (Model, Effect) {
	t := m.tabs[m.active]
	if t.cursor >= len(t.rows) || t.rows[t.cursor].entry.IsDir {
		return m, Effect{}
	}
	r := t.rows[t.cursor]
	return m, Effect{SetKind: &KindRequest{Rel: r.entry.Rel, Current: r.kind}}
}
