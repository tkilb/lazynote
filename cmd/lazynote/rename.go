package main

import (
	"fmt"
	"path/filepath"

	"github.com/tkilb/lazynote/internal/links"
	"github.com/tkilb/lazynote/internal/ui/para"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

// startRename carries out a rename asked for by the PARA panel. When other
// notes reference the note it first asks, saying how many it will update.
func (m model) startRename(req *para.RenameRequest) model {
	if n := len(m.links.ReferencedBy(req.Rel)); !req.IsDir && n > 0 {
		prompt := fmt.Sprintf("Rename %s to %s? This updates %s that reference it.",
			filepath.Base(req.Rel), req.Name, noteCount(n))
		return m.ask(prompt, func(m model) model { return m.doRename(req) })
	}
	return m.doRename(req)
}

// doRename renames, records one undo entry, moves the cursor to the renamed
// item and reports anything that could not be updated.
func (m model) doRename(req *para.RenameRequest) model {
	act, res, err := links.Rename(m.store, m.links, req.Rel, req.IsDir, req.Typed)
	if err != nil {
		return m.notify(popup.Error, capitalize(fmt.Sprintf("rename failed: %s.", err)))
	}
	m.history.Push(act)
	m.para = m.para.Reload()
	if p, ok := m.para.Reveal(filepath.Join(filepath.Dir(req.Rel), req.Name)); ok {
		m.para = p
	}
	m = m.refreshLinks()
	if res.TitleKept {
		m = m.notify(popup.Warning, "The note's properties could not be edited, so its title property was left as it was.")
	}
	if res.Skipped > 0 {
		m = m.notify(popup.Warning, fmt.Sprintf("%s could not be updated and still link to the old name.", noteCount(res.Skipped)))
	}
	return m
}

// noteCount words a number of notes.
func noteCount(n int) string {
	if n == 1 {
		return "1 note"
	}
	return fmt.Sprintf("%d notes", n)
}
