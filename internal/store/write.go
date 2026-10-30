package store

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tkilb/lazynote/internal/undo"
)

// trashRel is the notebook-relative folder that holds trashed items. Each
// trashed item gets its own slot folder so names never collide.
const trashRel = ".lazynote/trash"

const slotIDBytes = 8

// Create writes a new file at rel (relative to the root) containing data,
// creating missing folders. It never overwrites. Undo moves the file to the
// trash.
func (m *Markdown) Create(rel string, data []byte) (undo.Action, error) {
	path, err := ResolveInRoot(m.root, rel)
	if err != nil {
		return undo.Action{}, err
	}
	if err := mustNotExist(path); err != nil {
		return undo.Action{}, err
	}
	slot, err := m.newSlot(filepath.Base(path))
	if err != nil {
		return undo.Action{}, err
	}
	made, err := mkdirAllTracked(filepath.Dir(path))
	if err != nil {
		return undo.Action{}, err
	}
	if err := writeNewFile(path, data); err != nil {
		removeEmpty(made)
		return undo.Action{}, err
	}
	act, err := placed(path, slot, made, "create "+filepath.Base(path))
	if err != nil {
		_ = os.Remove(path)
		removeEmpty(made)
		return undo.Action{}, err
	}
	return act, nil
}

// MakeDir creates a new directory at rel, with missing parents. Undo
// removes it again, and refuses when it is no longer empty.
func (m *Markdown) MakeDir(rel string) (undo.Action, error) {
	path, err := ResolveInRoot(m.root, rel)
	if err != nil {
		return undo.Action{}, err
	}
	if err := mustNotExist(path); err != nil {
		return undo.Action{}, err
	}
	made, err := mkdirAllTracked(path)
	if err != nil {
		return undo.Action{}, err
	}
	return undo.Action{
		Description: "create folder " + filepath.Base(path),
		Undo: func() error {
			info, err := os.Lstat(path)
			switch {
			case errors.Is(err, fs.ErrNotExist):
				// Already gone: the folder is not there to remove.
			case err != nil || !info.IsDir():
				return ErrChanged
			default:
				// os.Remove only removes an empty folder.
				if err := os.Remove(path); err != nil {
					return ErrChanged
				}
			}
			removeEmpty(made[:max(len(made)-1, 0)])
			return nil
		},
		Redo: func() error {
			if err := mustNotExist(path); err != nil {
				return err
			}
			again, err := mkdirAllTracked(path)
			if err != nil {
				return err
			}
			made = again
			return nil
		},
	}, nil
}

// Move renames the item at src to dst, never overwriting. Undo moves it
// back.
func (m *Markdown) Move(src, dst string) (undo.Action, error) {
	from, to, err := m.resolvePair(src, dst)
	if err != nil {
		return undo.Action{}, err
	}
	want, err := m.existing(from)
	if err != nil {
		return undo.Action{}, err
	}
	r := &relocation{a: from, b: to, want: want}
	if err := r.forward(); err != nil {
		return undo.Action{}, err
	}
	return r.action("move " + filepath.Base(from)), nil
}

// Trash moves the item at rel into .lazynote/trash. Undo restores it, and
// refuses when its original path is taken or the item changed.
func (m *Markdown) Trash(rel string) (undo.Action, error) {
	from, err := ResolveInRoot(m.root, rel)
	if err != nil {
		return undo.Action{}, err
	}
	want, err := m.existing(from)
	if err != nil {
		return undo.Action{}, err
	}
	slot, err := m.newSlot(filepath.Base(from))
	if err != nil {
		return undo.Action{}, err
	}
	r := &relocation{a: from, b: slot, want: want}
	if err := r.forward(); err != nil {
		return undo.Action{}, err
	}
	return r.action("trash " + filepath.Base(from)), nil
}

// Copy copies the file or folder at src to dst, never overwriting. Folders
// copy recursively with everything inside, attachments included; symlinks
// and other special files inside are skipped. Undo moves the copy to the
// trash.
func (m *Markdown) Copy(src, dst string) (undo.Action, error) {
	from, to, err := m.resolvePair(src, dst)
	if err != nil {
		return undo.Action{}, err
	}
	want, err := m.existing(from)
	if err != nil {
		return undo.Action{}, err
	}
	if err := mustNotExist(to); err != nil {
		return undo.Action{}, err
	}
	if want.isDir && isInside(from, to) {
		return undo.Action{}, ErrIntoItself
	}
	slot, err := m.newSlot(filepath.Base(to))
	if err != nil {
		return undo.Action{}, err
	}
	made, err := mkdirAllTracked(filepath.Dir(to))
	if err != nil {
		return undo.Action{}, err
	}
	var c copier
	if err := c.copy(from, to); err != nil {
		c.cleanup()
		removeEmpty(made)
		return undo.Action{}, err
	}
	if !want.isDir && isNote(to) && filepath.Base(to) != filepath.Base(from) {
		// The copy has a new name, so its title property must follow it.
		if err := retitle(to); err != nil {
			c.cleanup()
			removeEmpty(made)
			return undo.Action{}, err
		}
	}
	act, err := placed(to, slot, made, "copy "+filepath.Base(from))
	if err != nil {
		c.cleanup()
		removeEmpty(made)
		return undo.Action{}, err
	}
	return act, nil
}

// retitle sets the title property of the note at path to its file name. A
// note whose properties block cannot be parsed, or that is too large to
// read as a note, is left exactly as it is.
func retitle(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return ErrUnreadable
	}
	if info.Size() > maxNoteBytes {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ErrUnreadable
	}
	out, err := SetTitle(data, noteTitle(filepath.Base(path)))
	switch {
	case errors.Is(err, ErrFrontmatterInvalid):
		return nil
	case err != nil:
		return ErrUnreadable
	case bytes.Equal(out, data):
		return nil
	}
	if err := os.WriteFile(path, out, info.Mode().Perm()); err != nil {
		return ErrUnreadable
	}
	return nil
}

// placed builds the undo action for an item just written at path: undo
// moves it to slot, redo moves it back.
func placed(path, slot string, made []string, description string) (undo.Action, error) {
	want, err := takeSnapshot(path)
	if err != nil {
		return undo.Action{}, ErrUnreadable
	}
	r := &relocation{a: slot, b: path, want: want, madeForward: made}
	return r.action(description), nil
}

// resolvePair resolves two root-relative names.
func (m *Markdown) resolvePair(src, dst string) (from, to string, err error) {
	if from, err = ResolveInRoot(m.root, src); err != nil {
		return "", "", err
	}
	if to, err = ResolveInRoot(m.root, dst); err != nil {
		return "", "", err
	}
	return from, to, nil
}

// existing snapshots path, reporting ErrNotFound when it is absent.
func (m *Markdown) existing(path string) (snapshot, error) {
	s, err := takeSnapshot(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return snapshot{}, ErrNotFound
	case err != nil:
		return snapshot{}, ErrUnreadable
	}
	return s, nil
}

// newSlot returns a fresh, not yet existing path for name inside the trash.
func (m *Markdown) newSlot(name string) (string, error) {
	var id [slotIDBytes]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", ErrUnreadable
	}
	return ResolveInRoot(m.root, filepath.Join(trashRel, hex.EncodeToString(id[:]), name))
}

// writeNewFile creates path exclusively and writes data to it.
func writeNewFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrExists
		}
		return ErrUnreadable
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(path)
		return ErrUnreadable
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return ErrUnreadable
	}
	return nil
}

// copier copies a file or directory tree and remembers everything it
// created, so a failure removes only its own output and never touches
// anything another process put there.
type copier struct {
	created []string
}

// copy copies a regular file or a directory tree from to to. The top item
// must be a file or directory; nested symlinks and special files are
// skipped.
func (c *copier) copy(from, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return ErrUnreadable
	}
	switch {
	case info.IsDir():
		return c.copyDir(from, to, info.Mode().Perm())
	case info.Mode().IsRegular():
		return c.copyFile(from, to, info.Mode().Perm())
	}
	return ErrUnsupported
}

func (c *copier) copyDir(from, to string, perm fs.FileMode) error {
	// Created with normal permissions so the children can be written even
	// when the source folder is read-only; the source mode is applied last.
	if err := os.Mkdir(to, dirPerm); err != nil {
		return ErrUnreadable
	}
	c.created = append(c.created, to)
	entries, err := os.ReadDir(from)
	if err != nil {
		return ErrUnreadable
	}
	for _, e := range entries {
		if t := e.Type(); !t.IsDir() && !t.IsRegular() {
			continue
		}
		if err := c.copy(filepath.Join(from, e.Name()), filepath.Join(to, e.Name())); err != nil {
			return err
		}
	}
	if err := os.Chmod(to, perm); err != nil {
		return ErrUnreadable
	}
	return nil
}

func (c *copier) copyFile(from, to string, perm fs.FileMode) error {
	in, err := os.Open(from)
	if err != nil {
		return ErrUnreadable
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return ErrUnreadable
	}
	c.created = append(c.created, to)
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return ErrUnreadable
	}
	if err := out.Close(); err != nil {
		return ErrUnreadable
	}
	return nil
}

// cleanup removes what copy created, newest first. A folder only goes when
// it is empty, so a file someone else added meanwhile is left alone.
func (c *copier) cleanup() {
	for i := len(c.created) - 1; i >= 0; i-- {
		if info, err := os.Lstat(c.created[i]); err == nil && info.IsDir() {
			_ = os.Chmod(c.created[i], dirPerm)
		}
		_ = os.Remove(c.created[i])
	}
}
