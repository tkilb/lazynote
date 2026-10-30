package store

import (
	"errors"
	"syscall"
	"time"

	"github.com/tkilb/lazynote/internal/undo"
)

// Errors returned by Store implementations. Like the path-safety errors,
// they never echo paths because they are shown in the status bar.
var (
	ErrNotFound   = errors.New("not found")
	ErrUnreadable = errors.New("cannot read")
	ErrTooLarge   = errors.New("note is too large")
)

// errNotDir is what the OS reports for a path below a regular file.
var errNotDir = syscall.ENOTDIR

// maxNoteBytes bounds how much of a single note is read into memory.
const maxNoteBytes = 10 << 20

// Entry is one row of a directory listing.
type Entry struct {
	Name string
	// Rel is the path relative to the notebook root.
	Rel     string
	IsDir   bool
	ModTime time.Time
	// ItemCount is the number of entries the directory would list under the
	// same options; it is zero for notes.
	ItemCount int
}

// ListOptions controls which entries List returns.
type ListOptions struct {
	// ShowHidden includes names starting with ".". The .obsidian directory
	// is never listed.
	ShowHidden bool
}

// Store is the notes backend: a notebook of markdown files.
type Store interface {
	// List returns the directories and .md notes in dir (relative to the
	// notebook root): directories first, then notes, each sorted by name. A
	// missing directory lists as empty.
	List(dir string, opts ListOptions) ([]Entry, error)
	// ReadNote reads and parses the note at rel (relative to the root).
	ReadNote(rel string) (Note, error)

	// Inspect counts what is inside a directory, listed or not.
	Inspect(rel string) (DirInfo, error)
	// Delete permanently removes a file or directory tree. Not undoable.
	Delete(rel string) error
	// Path returns the absolute path of rel (relative to the root), after the
	// same path-safety checks as every other operation.
	Path(rel string) (string, error)
	// Exists reports whether anything is at rel (relative to the root).
	Exists(rel string) (bool, error)

	// The write operations take names relative to the notebook root, never
	// overwrite, create missing folders on demand, and return one undo
	// action for the whole operation.

	// Create writes a new file containing data.
	Create(rel string, data []byte) (undo.Action, error)
	// MakeDir creates a new directory.
	MakeDir(rel string) (undo.Action, error)
	// Move renames a file or directory.
	Move(src, dst string) (undo.Action, error)
	// Copy copies a file or a directory tree, giving every note in the copy
	// a fresh uid and a title equal to its file name.
	Copy(src, dst string) (undo.Action, CopyResult, error)
	// Trash moves a file or directory into .lazynote/trash.
	Trash(rel string) (undo.Action, error)
}
