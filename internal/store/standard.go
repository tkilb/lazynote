package store

import (
	"slices"

	"go.yaml.in/yaml/v3"
)

const (
	peopleKey  = "people"
	relatedKey = "related"
)

// standardHead is the order every note's properties start in, which is the
// order the Info panel shows them: uid first, then the title and the
// properties of Info's standard rows. Any other properties follow, in the
// order they were in, and related comes last.
var standardHead = []string{uidKey, titleKey, kindKey, sourceKey, dateKey, tagsKey, peopleKey}

// Standardize makes the properties standard: the standard ones that are
// missing are added (a uid, empty text for kind, source and date, empty
// lists for tags, people and related; the title is the caller's to set), and
// the order becomes standardHead, then the other properties as they were,
// then related. Values, comments and unknown properties are kept.
func (f *Frontmatter) Standardize() error {
	if f.blank(uidKey) {
		uid, err := NewUID()
		if err != nil {
			return err
		}
		f.SetString(uidKey, uid)
	}
	for _, key := range []string{kindKey, sourceKey, dateKey} {
		if f.Get(key) == nil {
			f.SetString(key, "")
		}
	}
	for _, key := range []string{tagsKey, peopleKey, relatedKey} {
		if f.Get(key) == nil {
			f.SetStringList(key, nil)
		}
	}
	f.reorder()
	return nil
}

// pair is a key node and its value node.
type pair struct{ key, value *yaml.Node }

// reorder puts the properties in standard order.
func (f *Frontmatter) reorder() {
	var pairs []pair
	for i := 0; i+1 < len(f.node.Content); i += 2 {
		pairs = append(pairs, pair{f.node.Content[i], f.node.Content[i+1]})
	}
	take := func(key string) (pair, bool) {
		for i, p := range pairs {
			if p.key.Kind == yaml.ScalarNode && p.key.Value == key {
				pairs = slices.Delete(pairs, i, i+1)
				return p, true
			}
		}
		return pair{}, false
	}
	var head, tail []pair
	for _, key := range standardHead {
		if p, ok := take(key); ok {
			head = append(head, p)
		}
	}
	if p, ok := take(relatedKey); ok {
		tail = append(tail, p)
	}
	f.node.Content = f.node.Content[:0]
	for _, p := range slices.Concat(head, pairs, tail) {
		f.node.Content = append(f.node.Content, p.key, p.value)
	}
}

// isStandard reports whether the properties are already standard for a note
// named name: every standard property is there, the title is the file name,
// and the order is standard. Standardize would change nothing about it.
func (f *Frontmatter) isStandard(name string) bool {
	if f == nil || f.blank(uidKey) {
		return false
	}
	for _, key := range append(slices.Clone(standardHead), relatedKey) {
		if f.Get(key) == nil {
			return false
		}
	}
	if t := f.Get(titleKey); t.Tag != "!!str" || t.Value != noteTitle(name) {
		return false
	}
	var keys []string
	for _, p := range f.Properties() {
		keys = append(keys, p.Key)
	}
	return slices.Equal(keys[:len(standardHead)], standardHead) && keys[len(keys)-1] == relatedKey
}

// Conform makes the note at rel standard on disk, as it is when lazynote
// writes it: the standard properties added, the title set to the file name,
// and the order standard. It reports whether it wrote anything; a note that
// is standard already is left as it is, and so is one whose properties
// block cannot be read. It is not an undoable action.
func (m *Markdown) Conform(rel string) (bool, error) {
	note, err := m.ReadNote(rel)
	if err != nil || note.FrontmatterErr != nil {
		return false, nil // reading reports it, and an unreadable block stays as written
	}
	if note.Frontmatter.isStandard(note.Name) {
		return false, nil
	}
	if _, err := m.Edit(rel, func(*Frontmatter) {}); err != nil {
		return false, err
	}
	return true, nil
}
