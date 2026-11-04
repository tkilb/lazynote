package loader

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadFile reads the regular file at path, which may be anywhere (a leading
// "~" is the home directory; a symlink the user typed is followed). It must be
// UTF-8 text no larger than MaxBytes and not blank. The file is never
// changed; a leading byte order mark is dropped from the text.
func ReadFile(path string) (Source, error) {
	path = expandHome(path)
	if path == "" || strings.ContainsRune(path, 0) {
		return Source{}, ErrNotFound
	}
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return Source{}, ErrNotFound
	case err != nil:
		return Source{}, ErrUnreadable
	case !info.Mode().IsRegular():
		return Source{}, ErrNotFile
	case info.Size() > MaxBytes:
		return Source{}, ErrTooLarge
	}
	f, err := os.Open(path)
	if err != nil {
		return Source{}, ErrUnreadable
	}
	defer f.Close()
	// Check the opened file, not only the path, in case it changed between.
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
		return Source{}, ErrNotFile
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return Source{}, ErrUnreadable
	}
	text, err := checkText(data)
	if err != nil {
		return Source{}, err
	}
	return Source{Kind: File, Name: fileName(path), Text: text}, nil
}

// fileName is the base name of path without its last extension.
func fileName(path string) string {
	base := filepath.Base(path)
	if name := strings.TrimSuffix(base, filepath.Ext(base)); name != "" {
		return name
	}
	return "file"
}

// expandHome replaces a leading "~" or "~/" with the home directory.
func expandHome(p string) string {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~"))
}
