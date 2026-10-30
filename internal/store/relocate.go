package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tkilb/lazynote/internal/undo"
)

// Errors returned by write operations. Like the others they never echo
// paths.
var (
	ErrExists      = errors.New("already exists")
	ErrChanged     = errors.New("changed on disk since the action")
	ErrUnsupported = errors.New("cannot handle this kind of item")
	ErrIntoItself  = errors.New("cannot move or copy a folder into itself")
)

const (
	dirPerm                = 0o755
	filePerm               = 0o644
	ownerWrite fs.FileMode = 0o200
)

// snapshot is what undo compares to notice an outside change. Directories
// record only that they are directories, since their contents may change
// legitimately.
type snapshot struct {
	isDir   bool
	size    int64
	modNano int64
}

func takeSnapshot(path string) (snapshot, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return snapshot{}, err
	}
	if info.IsDir() {
		return snapshot{isDir: true}, nil
	}
	return snapshot{size: info.Size(), modNano: info.ModTime().UnixNano()}, nil
}

// relocation is one item that lives at a before the action and at b after
// it. Every reversible write is a relocation: a move goes src to dst, a
// trash goes the item to a slot under .lazynote/trash, and a create or copy
// is the reverse (its undo goes the new item to a slot). Nothing is ever
// deleted by an undo, so nothing is lost.
type relocation struct {
	a, b string
	want snapshot
	// madeForward and madeBackward are the directories each direction had to
	// create; they are removed again, when empty, on the opposite move.
	madeForward, madeBackward []string
}

// forward moves the item from a to b.
func (r *relocation) forward() error {
	made, err := moveItem(r.a, r.b, r.want)
	if err != nil {
		return err
	}
	r.madeForward = made
	removeEmpty(r.madeBackward)
	r.madeBackward = nil
	return nil
}

// backward moves the item from b to a.
func (r *relocation) backward() error {
	made, err := moveItem(r.b, r.a, r.want)
	if err != nil {
		return err
	}
	r.madeBackward = made
	removeEmpty(r.madeForward)
	r.madeForward = nil
	return nil
}

func (r *relocation) action(description string) undo.Action {
	return undo.Action{Description: description, Undo: r.backward, Redo: r.forward}
}

// moveItem renames from to to, first checking that from still matches want
// and that to is free, and creating the missing parents of to. It returns
// the directories it created.
func moveItem(from, to string, want snapshot) ([]string, error) {
	got, err := takeSnapshot(from)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, ErrChanged
	case err != nil:
		return nil, ErrUnreadable
	case got != want:
		return nil, ErrChanged
	}
	if err := mustNotExist(to); err != nil {
		return nil, err
	}
	if want.isDir && isInside(from, to) {
		return nil, ErrIntoItself
	}
	made, err := mkdirAllTracked(filepath.Dir(to))
	if err != nil {
		return nil, err
	}
	if err := renameNoReplace(from, to, want); err != nil {
		removeEmpty(made)
		return nil, err
	}
	return made, nil
}

// renameNoReplace moves from to to without ever replacing an existing item,
// even one created after the caller's check: os.Rename silently replaces a
// file at the target. A regular file is hard-linked to the new name, which
// fails when the name is taken, and the old name is then removed. Anything
// else, and filesystems without hard links, use os.Rename.
func renameNoReplace(from, to string, want snapshot) error {
	if want.isDir {
		return renameOrFail(from, to)
	}
	switch err := os.Link(from, to); {
	case err == nil:
		if err := os.Remove(from); err != nil {
			_ = os.Remove(to)
			return ErrUnreadable
		}
		return nil
	case errors.Is(err, fs.ErrExist):
		return ErrExists
	}
	return renameOrFail(from, to)
}

func renameOrFail(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return ErrUnreadable
	}
	return nil
}

// isInside reports whether to is dir itself or lies below it. Besides the
// path comparison it compares files with os.SameFile, so a differently
// cased spelling on a case-insensitive filesystem (Dir and dir/sub) is
// still caught.
func isInside(dir, to string) bool {
	if within(dir, to) {
		return true
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return false
	}
	for p := to; ; p = filepath.Dir(p) {
		if info, err := os.Stat(p); err == nil && os.SameFile(dirInfo, info) {
			return true
		}
		if filepath.Dir(p) == p {
			return false
		}
	}
}

// mustNotExist returns ErrExists when something is at path.
func mustNotExist(path string) error {
	_, err := os.Lstat(path)
	switch {
	case err == nil:
		return ErrExists
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, errNotDir):
		return nil
	}
	return ErrUnreadable
}

// mkdirAllTracked is os.MkdirAll that reports which directories it created,
// shallowest first, so a failed or undone operation can remove them again.
func mkdirAllTracked(dir string) ([]string, error) {
	var missing []string
	for p := dir; ; p = filepath.Dir(p) {
		_, err := os.Lstat(p)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, ErrUnreadable
		}
		missing = append([]string{p}, missing...)
		if filepath.Dir(p) == p {
			break
		}
	}
	var made []string
	for _, p := range missing {
		if err := os.Mkdir(p, dirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
			removeEmpty(made)
			return nil, ErrUnreadable
		}
		made = append(made, p)
	}
	return made, nil
}

// removeEmpty removes dirs deepest first, skipping any that is not empty.
func removeEmpty(dirs []string) {
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i])
	}
}
