package store

import (
	"strings"

	"go.yaml.in/yaml/v3"
)

const noteExt = ".md"

// Note is a parsed markdown note.
type Note struct {
	// Name is the file name, including the .md extension.
	Name string
	// Title is the title property when present, otherwise Name without .md.
	Title string
	// Frontmatter is nil when the note has no properties block or the block
	// is invalid (see FrontmatterErr).
	Frontmatter *Frontmatter
	// FrontmatterErr is set when a properties block exists but cannot be
	// used. The block is still removed from Body and left untouched on disk.
	FrontmatterErr error
	// Body is the note text after the frontmatter, unmodified.
	Body string
}

// ParseNote splits data into frontmatter and body and derives the title.
func ParseNote(name string, data []byte) Note {
	n := Note{Name: name}
	raw, body, ok := splitFrontmatter(data)
	n.Body = string(body)
	if ok {
		n.Frontmatter, n.FrontmatterErr = ParseFrontmatter(raw)
	}
	n.Title = noteTitle(name, n.Frontmatter)
	return n
}

// noteTitle follows Obsidian, which has no title key: the title property
// when it is a non-empty scalar, otherwise the file name without .md.
func noteTitle(name string, fm *Frontmatter) string {
	if fm != nil {
		if v := fm.Get("title"); v != nil && v.Kind == yaml.ScalarNode {
			if t := strings.TrimSpace(v.Value); t != "" {
				return t
			}
		}
	}
	return strings.TrimSuffix(name, noteExt)
}

// isNote reports whether name is a markdown note file name.
func isNote(name string) bool {
	return strings.EqualFold(name[max(len(name)-len(noteExt), 0):], noteExt)
}
