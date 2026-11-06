package store

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/tkilb/para/internal/undo"
)

// Edit implements Store.
func (m *Markdown) Edit(rel string, edit func(*Frontmatter)) (undo.Action, error) {
	path, err := ResolveInRoot(m.root, rel)
	if err != nil {
		return undo.Action{}, err
	}
	if !isNote(filepath.Base(path)) {
		return undo.Action{}, ErrUnsupported
	}
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return undo.Action{}, ErrNotFound
	case err != nil:
		return undo.Action{}, ErrUnreadable
	case !info.Mode().IsRegular():
		return undo.Action{}, ErrUnsupported
	case info.Size() > maxNoteBytes:
		return undo.Action{}, ErrTooLarge
	}
	before, err := os.ReadFile(path)
	if err != nil {
		return undo.Action{}, ErrUnreadable
	}
	after, err := editedNote(before, filepath.Base(path), edit)
	if err != nil {
		return undo.Action{}, err
	}
	pre := snapshot{size: info.Size(), modNano: info.ModTime().UnixNano()}
	e := &editedFile{path: path, before: before, after: after, perm: info.Mode().Perm(), pre: pre}
	if err := e.write(after, pre, nil); err != nil {
		return undo.Action{}, err
	}
	e.post, err = takeSnapshot(path)
	if err != nil {
		return undo.Action{}, ErrUnreadable
	}
	return undo.Action{
		Description: "edit " + filepath.Base(path),
		Undo:        func() error { return e.write(e.before, e.post, &e.pre) },
		Redo:        func() error { return e.write(e.after, e.pre, &e.post) },
	}, nil
}

// editedNote applies edit to the properties of data and returns the new
// note: the body and any byte order mark are kept, and a uid and the title
// are added when missing.
func editedNote(data []byte, name string, edit func(*Frontmatter)) ([]byte, error) {
	bom := bytes.HasPrefix(data, utf8BOM)
	raw, body, ok := splitFrontmatter(data)
	fm := NewFrontmatter()
	if ok {
		var err error
		if fm, err = ParseFrontmatter(raw); err != nil {
			return nil, ErrFrontmatterInvalid
		}
	}
	edit(fm)
	if v := fm.Get(uidKey); v == nil || v.Value == "" {
		uid, err := NewUID()
		if err != nil {
			return nil, err
		}
		fm.SetString(uidKey, uid)
	}
	fm.SetString(titleKey, noteTitle(name))
	return assembleNote(fm, body, bom)
}

// editedFile is one note rewritten by Edit. pre and post are what the file
// looked like before and after the edit; each write requires the file to be
// in the state it expects and puts it in the other one, modification time
// included. That keeps stacked edits of one note consistent: undoing the
// later edit returns the file to exactly what the earlier edit left.
type editedFile struct {
	path          string
	before, after []byte
	perm          fs.FileMode
	pre, post     snapshot
}

// write replaces the file with data, first checking that it still matches
// want. When restore is set, the modification time becomes that of the state
// the file moves to; the first write passes nil and keeps the new time, so
// anything watching modification times sees the change.
func (e *editedFile) write(data []byte, want snapshot, restore *snapshot) error {
	got, err := takeSnapshot(e.path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ErrChanged
	case err != nil:
		return ErrUnreadable
	case got != want:
		return ErrChanged
	}
	if err := rewriteFile(e.path, data, e.perm); err != nil {
		return ErrUnreadable
	}
	if restore != nil {
		// Best effort: when this fails the next undo or redo of the note sees
		// a different time and refuses, reporting it, instead of overwriting.
		mod := time.Unix(0, restore.modNano)
		_ = os.Chtimes(e.path, mod, mod)
	}
	return nil
}
