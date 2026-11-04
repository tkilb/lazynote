package store

import "strings"

const noteExt = ".md"

// Note is a parsed markdown note.
type Note struct {
	// Name is the file name, including the .md extension.
	Name string
	// Title is always Name without the .md extension. A title property in
	// the frontmatter never overrides it.
	Title string
	// Frontmatter is nil when the note has no properties block or the block
	// is invalid (see FrontmatterErr).
	Frontmatter *Frontmatter
	// FrontmatterErr is set when a properties block exists but cannot be
	// used. The block is still removed from Body and left untouched on disk.
	FrontmatterErr error
	// HasFrontmatter is set when the note starts with a closed properties
	// block, valid or not. RawFrontmatter is that block as written, without
	// its --- lines and the final line break.
	HasFrontmatter bool
	RawFrontmatter string
	// Kind is the kind property, or KindUnset.
	Kind Kind
	// Body is the note text after the frontmatter, unmodified.
	Body string
}

// ParseNote splits data into frontmatter and body and derives the title.
func ParseNote(name string, data []byte) Note {
	n := Note{Name: name}
	raw, body, ok := splitFrontmatter(data)
	n.Body = string(body)
	if ok {
		n.HasFrontmatter = true
		n.RawFrontmatter = strings.TrimRight(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
		n.Frontmatter, n.FrontmatterErr = ParseFrontmatter(raw)
	}
	n.Title = noteTitle(name)
	n.Kind = KindOf(n.Frontmatter)
	return n
}

// noteTitle is the file name without its .md extension (in any case). The
// title always reflects the file name, so renaming a note renames its title
// and the two can never disagree.
func noteTitle(name string) string {
	if isNote(name) {
		return name[:len(name)-len(noteExt)]
	}
	return name
}

// isNote reports whether name is a markdown note file name.
func isNote(name string) bool {
	return strings.EqualFold(name[max(len(name)-len(noteExt), 0):], noteExt)
}
