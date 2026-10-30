// Package store holds the notes backend. This file is the single choke
// point every file operation (create, rename, paste, move, trash, delete,
// import, add, retention purge) must use to turn a notebook-relative name
// into an absolute path that is guaranteed to stay inside the notebook.
package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Rejection errors. Messages are short and never echo the offending path,
// because they are shown in the status bar.
var (
	ErrEmptyName     = errors.New("empty name")
	ErrNulByte       = errors.New("name contains a NUL byte")
	ErrAbsolutePath  = errors.New("absolute paths are not allowed")
	ErrParentSegment = errors.New("\"..\" is not allowed")
	ErrEscapesRoot   = errors.New("path leaves the notebook")
	ErrUnresolvable  = errors.New("path cannot be resolved")
)

// ResolveInRoot joins rel onto root and returns the absolute, symlink-free
// path of the target, or an error when the target would fall outside root.
// The target does not need to exist: the deepest existing ancestor is
// resolved through symlinks and checked, and the remaining names are
// appended. A dangling symlink is rejected because its target is unknown.
//
// When the final component is itself a symlink that stays inside root, the
// result is the link's own path (parent resolved), not its target, so a
// delete or rename acts on the link. A link that resolves to root itself is
// rejected.
//
// The check and the later use are separate steps, so a writer inside the
// notebook can race them; callers that need more must open the result with
// O_NOFOLLOW or os.Root.
func ResolveInRoot(root, rel string) (string, error) {
	cleaned, err := cleanRel(rel)
	if err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", ErrUnresolvable
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", ErrUnresolvable
	}

	existing := filepath.Join(realRoot, cleaned)
	var missing []string
	for {
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			return "", ErrUnresolvable
		}
		missing = append([]string{filepath.Base(existing)}, missing...)
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", ErrEscapesRoot
		}
		existing = parent
	}

	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		// The path exists but cannot be resolved: a dangling symlink.
		return "", ErrEscapesRoot
	}
	if !within(realRoot, resolved) || (resolved == realRoot && len(missing) == 0) {
		return "", ErrEscapesRoot
	}
	if len(missing) == 0 && isSymlink(existing) {
		parent, err := filepath.EvalSymlinks(filepath.Dir(existing))
		if err != nil || !within(realRoot, parent) {
			return "", ErrEscapesRoot
		}
		return filepath.Join(parent, filepath.Base(existing)), nil
	}
	return filepath.Join(append([]string{resolved}, missing...)...), nil
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&fs.ModeSymlink != 0
}

// cleanRel validates the purely lexical rules and returns the cleaned name.
func cleanRel(rel string) (string, error) {
	switch {
	case rel == "":
		return "", ErrEmptyName
	case strings.ContainsRune(rel, 0):
		return "", ErrNulByte
	case filepath.IsAbs(rel):
		return "", ErrAbsolutePath
	}
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == ".." {
			return "", ErrParentSegment
		}
	}
	cleaned := filepath.Clean(rel)
	if cleaned == "." {
		return "", ErrEmptyName
	}
	return cleaned, nil
}

// within reports whether path is root or lies below it, comparing on a
// directory boundary so "/a/nb-evil" is not inside "/a/nb".
func within(root, path string) bool {
	if path == root {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(root, string(filepath.Separator))+string(filepath.Separator))
}
