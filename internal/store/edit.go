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
	path, info, before, err := m.readForEdit(rel)
	if err != nil {
		return undo.Action{}, err
	}
	after, err := editedNote(before, filepath.Base(path), edit)
	if err != nil {
		return undo.Action{}, err
	}
	return rewriteNote("edit", path, info, before, after)
}

// readForEdit resolves rel, checks that it is a regular note of an editable
// size and returns its path, file info and bytes.
func (m *Markdown) readForEdit(rel string) (string, fs.FileInfo, []byte, error) {
	path, err := ResolveInRoot(m.root, rel)
	if err != nil {
		return "", nil, nil, err
	}
	if !isNote(filepath.Base(path)) {
		return "", nil, nil, ErrUnsupported
	}
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", nil, nil, ErrNotFound
	case err != nil:
		return "", nil, nil, ErrUnreadable
	case !info.Mode().IsRegular():
		return "", nil, nil, ErrUnsupported
	case info.Size() > maxNoteBytes:
		return "", nil, nil, ErrTooLarge
	}
	before, err := os.ReadFile(path)
	if err != nil {
		return "", nil, nil, ErrUnreadable
	}
	return path, info, before, nil
}

// rewriteNote replaces the note at path, whose bytes were before, with after
// and returns the undo action, described as verb and the file name.
func rewriteNote(verb, path string, info fs.FileInfo, before, after []byte) (undo.Action, error) {
	pre := snapshot{size: info.Size(), modNano: info.ModTime().UnixNano()}
	e := &editedFile{path: path, before: before, after: after, perm: info.Mode().Perm(), pre: pre}
	if err := e.write(after, pre, nil); err != nil {
		return undo.Action{}, err
	}
	var err error
	if e.post, err = takeSnapshot(path); err != nil {
		return undo.Action{}, ErrUnreadable
	}
	return undo.Action{
		Description: verb + " " + filepath.Base(path),
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
