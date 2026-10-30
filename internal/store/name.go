package store

import (
	"errors"
	"strings"
)

// ErrBadName reports a typed name that cannot be a single file or folder
// name.
var ErrBadName = errors.New("invalid name")

// CleanNewName validates a name typed into the create prompt. A trailing
// "/" makes it a folder; any other "/" is rejected, as are empty names,
// "." and "..", and NUL bytes. A note name gets ".md" unless it already
// ends with it. The returned name never has a trailing slash.
func CleanNewName(input string) (name string, isDir bool, err error) {
	name = strings.TrimSpace(input)
	if strings.ContainsRune(name, 0) {
		return "", false, ErrNulByte
	}
	if trimmed, ok := strings.CutSuffix(name, "/"); ok {
		name, isDir = strings.TrimSpace(trimmed), true
	}
	switch {
	case name == "":
		return "", false, ErrEmptyName
	case name == "." || name == "..", strings.Contains(name, "/"):
		return "", false, ErrBadName
	}
	if !isDir && !isNote(name) {
		name += noteExt
	}
	return name, isDir, nil
}
