// Package links indexes the wikilinks in notes' related property: which
// notes a note references and which notes it is referenced by (backlinks).
// Names resolve as in Obsidian: by file name without .md, case-insensitive,
// ignoring a |alias and a #heading, and a duplicate name resolves to the
// shortest path. Only the related property counts; links in the body and
// shared tags are out of scope.
package links

import (
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/tkilb/lazynote/internal/store"
)

const (
	relatedKey = "related"
	uidKey     = "uid"
	noteExt    = ".md"
	linkOpen   = "[["
	linkClose  = "]]"
)

// Roots are the folders of a notebook that hold notes, the same five PARA
// shows as tabs. The index covers what is under them.
var Roots = []string{"inbox", "projects", "areas", "resources", "archives"}

// Link is one reference of a note.
type Link struct {
	// Name is the target as written, without any |alias or #heading.
	Name string
	// Rel is the note it resolves to, relative to the notebook root, or ""
	// when no note has that name.
	Rel string
}

// Target extracts the note name from a wikilink such as
// "[[projects/Alpha#Goals|the goals]]", which gives "Alpha". It reports false
// for text that is not a wikilink or names no note.
func Target(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if !strings.HasPrefix(s, linkOpen) || !strings.HasSuffix(s, linkClose) || len(s) < len(linkOpen)+len(linkClose) {
		return "", false
	}
	s = s[len(linkOpen) : len(s)-len(linkClose)]
	s, _, _ = strings.Cut(s, "|")
	s, _, _ = strings.Cut(s, "#")
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	s = strings.TrimSpace(s)
	if len(s) >= len(noteExt) && strings.EqualFold(s[len(s)-len(noteExt):], noteExt) {
		s = strings.TrimSpace(s[:len(s)-len(noteExt)])
	}
	return s, s != ""
}

// noteInfo is what the index keeps per note: the change time that tells
// whether to read it again, and the names it links to.
type noteInfo struct {
	mod     time.Time
	uid     string
	targets []string
}

// Index answers link queries for one notebook. Refresh brings it up to date.
type Index struct {
	notes        map[string]noteInfo // by rel
	byName       map[string]string   // lowercased name -> rel
	referencedBy map[string][]string // rel -> sorted rels linking to it
	byUID        map[string]string   // uid -> rel
}

// Build indexes every note in s.
func Build(s store.Store) (*Index, error) {
	x := &Index{notes: map[string]noteInfo{}}
	return x, x.Refresh(s)
}

// Refresh reads the notes added or changed since the last build and forgets
// the ones that are gone. A note that cannot be read is indexed without
// links, so one bad file never stops the rest.
func (x *Index) Refresh(s store.Store) error {
	var entries []store.Entry
	for _, root := range Roots {
		found, err := walk(s, root)
		if err != nil {
			return err
		}
		entries = append(entries, found...)
	}
	next := make(map[string]noteInfo, len(entries))
	for _, e := range entries {
		if old, ok := x.notes[e.Rel]; ok && old.mod.Equal(e.ModTime) {
			next[e.Rel] = old
			continue
		}
		info := noteInfo{mod: e.ModTime}
		if n, err := s.ReadNote(e.Rel); err == nil {
			info.targets = relatedTargets(n.Frontmatter)
			info.uid = uidOf(n.Frontmatter)
		}
		next[e.Rel] = info
	}
	x.notes = next
	x.resolveAll()
	return nil
}

// walk lists every note under dir, depth first.
func walk(s store.Store, dir string) ([]store.Entry, error) {
	entries, err := s.List(dir, store.ListOptions{})
	if err != nil {
		return nil, err
	}
	var notes []store.Entry
	for _, e := range entries {
		if !e.IsDir {
			notes = append(notes, e)
			continue
		}
		sub, err := walk(s, e.Rel)
		if err != nil {
			return nil, err
		}
		notes = append(notes, sub...)
	}
	return notes, nil
}

// resolveAll rebuilds the name table and the backlinks from the notes.
func (x *Index) resolveAll() {
	x.byName = make(map[string]string, len(x.notes))
	x.byUID = make(map[string]string, len(x.notes))
	for rel, info := range x.notes {
		if cur, ok := x.byUID[info.uid]; info.uid != "" && (!ok || shorter(rel, cur)) {
			x.byUID[info.uid] = rel
		}
	}
	for rel := range x.notes {
		key := strings.ToLower(nameOf(rel))
		if cur, ok := x.byName[key]; !ok || shorter(rel, cur) {
			x.byName[key] = rel
		}
	}
	sets := map[string]map[string]bool{}
	for rel, info := range x.notes {
		for _, name := range info.targets {
			to, ok := x.byName[strings.ToLower(name)]
			if !ok || to == rel {
				continue
			}
			if sets[to] == nil {
				sets[to] = map[string]bool{}
			}
			sets[to][rel] = true
		}
	}
	x.referencedBy = make(map[string][]string, len(sets))
	for to, set := range sets {
		rels := make([]string, 0, len(set))
		for rel := range set {
			rels = append(rels, rel)
		}
		sort.Strings(rels)
		x.referencedBy[to] = rels
	}
}

// shorter reports whether a is the better resolution than b: the shorter
// path, then the alphabetically first.
func shorter(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// nameOf is a note's file name without its .md extension.
func nameOf(rel string) string {
	name := rel[strings.LastIndex(rel, "/")+1:]
	if len(name) >= len(noteExt) && strings.EqualFold(name[len(name)-len(noteExt):], noteExt) {
		return name[:len(name)-len(noteExt)]
	}
	return name
}

// References returns the links of the note at rel, in the order written, one
// per distinct name.
func (x *Index) References(rel string) []Link {
	if x == nil {
		return nil
	}
	info, ok := x.notes[rel]
	if !ok {
		return nil
	}
	var out []Link
	for _, name := range info.targets {
		out = append(out, Link{Name: name, Rel: x.byName[strings.ToLower(name)]})
	}
	return out
}

// ReferencedBy returns the notes whose related property links to the note at
// rel, sorted, excluding the note itself.
func (x *Index) ReferencedBy(rel string) []string {
	if x == nil {
		return nil
	}
	return x.referencedBy[rel]
}

// relatedTargets reads the note names out of the related property: a list of
// wikilink strings, or one string. An unquoted [[Name]] parses as a nested
// list in YAML and is accepted too. Anything else in the list is ignored.
func relatedTargets(fm *store.Frontmatter) []string {
	if fm == nil {
		return nil
	}
	node := fm.Get(relatedKey)
	if node == nil {
		return nil
	}
	var raws []string
	switch node.Kind {
	case yaml.ScalarNode:
		raws = append(raws, node.Value)
	case yaml.SequenceNode:
		for _, item := range node.Content {
			switch {
			case item.Kind == yaml.ScalarNode:
				raws = append(raws, item.Value)
			case item.Kind == yaml.SequenceNode && len(item.Content) == 1 && item.Content[0].Kind == yaml.ScalarNode:
				raws = append(raws, linkOpen+item.Content[0].Value+linkClose)
			}
		}
	}
	var names []string
	seen := map[string]bool{}
	for _, raw := range raws {
		name, ok := Target(raw)
		if key := strings.ToLower(name); ok && !seen[key] {
			seen[key] = true
			names = append(names, name)
		}
	}
	return names
}

// Note is a note a link can name.
type Note struct {
	// Name is the file name without .md.
	Name string
	// Rel is the note's path relative to the notebook root.
	Rel string
}

// ByUID returns the path of the note whose uid property is uid. A uid that
// two notes share (a copy made outside lazynote) goes to the one with the
// shortest path.
func (x *Index) ByUID(uid string) (string, bool) {
	if x == nil || uid == "" {
		return "", false
	}
	rel, ok := x.byUID[uid]
	return rel, ok
}

// uidOf reads the uid property of fm as plain text, or "" when it has none.
func uidOf(fm *store.Frontmatter) string {
	if fm == nil {
		return ""
	}
	v := fm.Get(uidKey)
	if v == nil || v.Kind != yaml.ScalarNode {
		return ""
	}
	return strings.TrimSpace(v.Value)
}

// Rels returns the path of every note in the index, sorted by file name
// ignoring case and then by path. Unlike Notes it keeps notes that share a
// name.
func (x *Index) Rels() []string {
	if x == nil {
		return nil
	}
	rels := make([]string, 0, len(x.notes))
	for rel := range x.notes {
		rels = append(rels, rel)
	}
	sort.Slice(rels, func(i, j int) bool {
		a, b := strings.ToLower(nameOf(rels[i])), strings.ToLower(nameOf(rels[j]))
		if a != b {
			return a < b
		}
		return rels[i] < rels[j]
	})
	return rels
}

// Notes returns every note a link can name, one per distinct name (the one a
// link to that name resolves to), sorted by name ignoring case.
func (x *Index) Notes() []Note {
	if x == nil {
		return nil
	}
	notes := make([]Note, 0, len(x.byName))
	for _, rel := range x.byName {
		notes = append(notes, Note{Name: nameOf(rel), Rel: rel})
	}
	sort.Slice(notes, func(i, j int) bool {
		a, b := strings.ToLower(notes[i].Name), strings.ToLower(notes[j].Name)
		if a != b {
			return a < b
		}
		return notes[i].Rel < notes[j].Rel
	})
	return notes
}
