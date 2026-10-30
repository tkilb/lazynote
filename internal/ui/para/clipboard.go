package para

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/undo"
)

type clipMode int

const (
	clipNone clipMode = iota
	clipYank
	clipCut
)

// clip is the yank/cut buffer. It belongs to the whole panel, so it
// survives directory and tab changes, and is replaced, never mutated.
type clip struct {
	mode clipMode
	// rels are notebook-relative paths, in list order.
	rels []string
	set  map[string]bool
}

func newClip(mode clipMode, entries []store.Entry) clip {
	c := clip{mode: mode, set: make(map[string]bool, len(entries))}
	for _, e := range entries {
		c.rels = append(c.rels, e.Rel)
		c.set[e.Rel] = true
	}
	return c
}

// targets returns the selected rows' entries, or the hovered one when
// nothing is selected.
func (m Model) targets() []store.Entry {
	t := m.tabs[m.active]
	var out []store.Entry
	for _, r := range t.rows {
		if t.selected[r.entry.Name] {
			out = append(out, r.entry)
		}
	}
	if len(out) == 0 && t.cursor < len(t.rows) {
		out = append(out, t.rows[t.cursor].entry)
	}
	return out
}

// setClip fills the buffer from the targets, replacing what was there, and
// clears the selection.
func (m Model) setClip(mode clipMode) (Model, Effect) {
	entries := m.targets()
	if len(entries) == 0 {
		return m, Effect{}
	}
	m.clip = newClip(mode, entries)
	t := &m.tabs[m.active]
	t.selected, t.visual = nil, visualState{}
	verb := map[clipMode]string{clipYank: "Yanked", clipCut: "Cut"}[mode]
	return m, Effect{Info: verb + " " + describe(len(entries), entries[0].Name)}
}

// cancelClip empties the buffer.
func (m Model) cancelClip() (Model, Effect) {
	if m.clip.mode == clipNone {
		return m, Effect{}
	}
	m.clip = clip{}
	return m, Effect{Info: "Cleared the yank buffer"}
}

// currentDir is the notebook-relative path of the directory on screen.
func (m Model) currentDir() string {
	return filepath.Join(tabDefs[m.active].dir, m.tabs[m.active].dir)
}

// pasteItem is one planned paste.
type pasteItem struct {
	src, dst string
	collides bool
}

// pastePlan is the buffer resolved against the current directory.
type pastePlan struct {
	items   []pasteItem
	missing int // sources that vanished and are dropped
	skipped int // cut items already in this directory
}

func (p pastePlan) collisions() int {
	n := 0
	for _, it := range p.items {
		if it.collides {
			n++
		}
	}
	return n
}

func (m Model) planPaste() (pastePlan, error) {
	var plan pastePlan
	dir := m.currentDir()
	for _, src := range m.clip.rels {
		ok, err := m.store.Exists(src)
		if err != nil {
			return plan, err
		}
		if !ok {
			plan.missing++
			continue
		}
		dst := filepath.Join(dir, filepath.Base(src))
		if m.clip.mode == clipCut && dst == src {
			plan.skipped++
			continue
		}
		collides, err := m.store.Exists(dst)
		if err != nil {
			return plan, err
		}
		plan.items = append(plan.items, pasteItem{src: src, dst: dst, collides: collides})
	}
	return plan, nil
}

// paste puts the buffer into the current directory. Without overwrite a
// name collision cancels the whole paste. With overwrite and collisions it
// asks first, unless confirmed; replaced items go to the trash, and the
// whole paste is one undo entry.
func (m Model) paste(overwrite, confirmed bool) (Model, Effect) {
	if m.clip.mode == clipNone {
		return m, Effect{Info: "Nothing to paste"}
	}
	plan, err := m.planPaste()
	if err != nil {
		return m, Effect{Err: fmt.Errorf("paste: %w", err)}
	}

	if n := plan.collisions(); n > 0 {
		switch {
		case !overwrite:
			return m, Effect{Err: fmt.Errorf("paste cancelled: %s already exist here (P overwrites)", count(n))}
		case !confirmed:
			return m, Effect{Confirm: &Confirm{
				Prompt: fmt.Sprintf("Overwrite %s? Replaced items go to the trash.", count(n)),
				Op:     OpOverwritePaste,
			}}
		}
	}

	actions, err := m.execute(plan)
	if err != nil {
		return m, Effect{Err: fmt.Errorf("paste failed: %w", err)}
	}
	m.clip = clip{}
	m = m.Reload()
	return m, pasteResult(plan, actions)
}

// execute runs the plan, trashing replaced items first. On the first error
// it undoes what it already did, so a failed paste changes nothing.
func (m Model) execute(plan pastePlan) ([]undo.Action, error) {
	var done []undo.Action
	fail := func(err error) ([]undo.Action, error) {
		if len(done) > 0 {
			if rbErr := undo.Group("rollback", done...).Undo(); rbErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
			}
		}
		return nil, err
	}
	for _, it := range plan.items {
		if it.collides {
			a, err := m.store.Trash(it.dst)
			if err != nil {
				return fail(err)
			}
			done = append(done, a)
		}
		op := m.store.Copy
		if m.clip.mode == clipCut {
			op = m.store.Move
		}
		a, err := op(it.src, it.dst)
		if err != nil {
			return fail(err)
		}
		done = append(done, a)
	}
	return done, nil
}

func pasteResult(plan pastePlan, actions []undo.Action) Effect {
	var note string
	if plan.missing > 0 {
		note = fmt.Sprintf(" (%d no longer exist, dropped)", plan.missing)
	}
	if len(plan.items) == 0 {
		if plan.skipped > 0 {
			return Effect{Info: "Already in this directory"}
		}
		return Effect{Info: "Nothing pasted" + note}
	}
	name := filepath.Base(plan.items[0].dst)
	group := undo.Group("paste "+describe(len(plan.items), name), actions...)
	return Effect{Action: &group, Info: "Pasted " + describe(len(plan.items), name) + note}
}

// ApplyConfirm carries out an accepted Confirm.
func (m Model) ApplyConfirm(op ConfirmOp) (Model, Effect) {
	switch op {
	case OpOverwritePaste:
		return m.paste(true, true)
	case OpTrash:
		return m.trash()
	case OpDelete:
		return m.deletePermanently()
	}
	return m, Effect{}
}

// describe names one item, or counts several.
func describe(n int, first string) string {
	if n == 1 {
		return first
	}
	return count(n)
}

func count(n int) string {
	if n == 1 {
		return "1 item"
	}
	return fmt.Sprintf("%d items", n)
}
