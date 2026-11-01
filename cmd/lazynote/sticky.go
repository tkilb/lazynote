package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/state"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/popup"
	"github.com/tkilb/lazynote/internal/ui/stickies"
)

// noteUID reads the uid property of the note at rel, or "" when it has none.
func (m model) noteUID(rel string) (string, error) {
	note, err := m.store.ReadNote(rel)
	if err != nil {
		return "", err
	}
	if v := note.Frontmatter.Get("uid"); v != nil {
		return strings.TrimSpace(v.Value), nil
	}
	return "", nil
}

// toggleSticky pins the note under the PARA cursor, or unpins it when it is
// already pinned. A sticky follows the note's uid, so a note without one is
// given one first, as any edit does (and undo can take back).
func (m model) toggleSticky() model {
	entry, ok := m.para.SelectedNote()
	if !ok {
		return m.notify(popup.Info, "Select a note to pin it as a sticky.")
	}
	uid, err := m.noteUID(entry.Rel)
	if err != nil {
		return m.notify(popup.Warning, "Could not read the note: "+err.Error()+".")
	}
	if uid != "" && slices.Contains(m.stickyUIDs, uid) {
		return m.unpin(uid)
	}
	if len(m.stickyUIDs) >= state.MaxStickies {
		return m.notify(popup.Info, fmt.Sprintf("You can pin at most %d stickies. Unpin one first.", state.MaxStickies))
	}
	if uid == "" {
		m = m.editNote(entry.Rel, "pin", func(*store.Frontmatter) {})
		if uid, err = m.noteUID(entry.Rel); err != nil || uid == "" {
			return m
		}
	}
	m.stickyUIDs = append(slices.Clone(m.stickyUIDs), uid)
	return m.saveStickies().syncStickies()
}

// unpin removes the sticky with the given uid.
func (m model) unpin(uid string) model {
	m.stickyUIDs = slices.DeleteFunc(slices.Clone(m.stickyUIDs), func(u string) bool { return u == uid })
	return m.saveStickies().syncStickies()
}

// stickiesKey sends a key to the Stickies panel and applies what it asks for.
func (m model) stickiesKey(msg tea.KeyMsg) model {
	var eff stickies.Effect
	m.stickies, eff = m.stickies.Update(msg)
	switch {
	case eff.Info != "":
		return m.notify(popup.Info, eff.Info)
	case eff.Unpin != "":
		return m.unpin(eff.Unpin)
	case eff.Jump != "":
		return m.chooseFoundNote(eff.Jump)
	}
	return m
}

// syncStickies rebuilds the panel's rows from the uids and the link index, so
// a renamed or moved note shows under its new name and place. A note that
// cannot be found shows as missing.
func (m model) syncStickies() model {
	items := make([]stickies.Item, len(m.stickyUIDs))
	for i, uid := range m.stickyUIDs {
		it := stickies.Item{UID: uid, Name: "(missing note)"}
		if rel, ok := m.links.ByUID(uid); ok {
			it.Rel, it.Name = rel, noteName(rel)
			if dir := filepath.ToSlash(filepath.Dir(rel)); dir != "." {
				it.Where = dir
			}
		}
		items[i] = it
	}
	m.stickies = m.stickies.Set(items)
	return m
}

// saveStickies writes the stickies of the open notebook. It does nothing when
// persistence is off.
func (m model) saveStickies() model {
	if m.session.db == nil {
		return m
	}
	if err := m.session.db.SaveStickies(m.session.notebook, m.stickyUIDs); err != nil {
		return m.notify(popup.Warning, fmt.Sprintf("Could not save your stickies: %s.", err))
	}
	return m
}

// loadStickies reads the saved stickies of the open notebook.
func (m model) loadStickies() model {
	m.stickyUIDs = nil
	if m.session.db == nil {
		return m
	}
	uids, err := m.session.db.LoadStickies(m.session.notebook)
	if err != nil {
		return m.notify(popup.Warning, fmt.Sprintf("Could not restore your stickies: %s.", err))
	}
	m.stickyUIDs = uids
	return m
}
