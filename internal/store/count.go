package store

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CountNotes counts the .md notes in the five PARA folders of the notebook at
// root, hidden folders and files left out and symlinks never followed. A
// notebook that cannot be read counts as zero.
func CountNotes(root string) int {
	total := 0
	for _, tab := range tabRoots {
		dir := filepath.Join(root, tab)
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return fs.SkipDir
			case p == dir:
				if d.Type()&os.ModeSymlink != 0 {
					return fs.SkipDir
				}
				return nil
			case strings.HasPrefix(d.Name(), "."):
				if d.IsDir() {
					return fs.SkipDir
				}
			case d.Type().IsRegular() && isNote(d.Name()):
				total++
			}
			return nil
		})
	}
	return total
}
