package links

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/undo"
)

// linkBreakers are characters that cannot be part of a note name inside a
// wikilink: they separate its alias, heading and brackets.
const linkBreakers = "|#[]"

// ErrUnlinkable reports a new name that other notes could not link to.
var ErrUnlinkable = errors.New("name cannot contain | # [ or ] while other notes link to it")

// RenameResult reports what Rename could not do completely.
type RenameResult struct {
	// Updated counts the notes whose related property was rewritten.
	Updated int
	// Skipped counts referencing notes left as they are because their
	// properties are invalid or the note is too large to edit.
	Skipped int
	// TitleKept is set when the renamed note's own title could not be
	// updated for the same reasons.
	TitleKept bool
}

// Rename renames the note or folder at rel to the typed name. For a note it
// also sets the title property to the new file name and rewrites the related
// entries of every note that references it, so the links keep working
// (Obsidian's behavior). The whole rename is one undo entry. A note that
// cannot be edited (invalid properties, too large) is skipped and counted in
// the result; any other failure undoes what was already done.
//
// x must be up to date: it says which notes reference the note.
func Rename(s store.Store, x *Index, rel string, isDir bool, typed string) (undo.Action, RenameResult, error) {
	var res RenameResult
	name, err := store.CleanRenamedName(typed, isDir)
	if err != nil {
		return undo.Action{}, res, err
	}
	dst := filepath.Join(filepath.Dir(rel), name)
	oldName, newName := nameOf(rel), nameOf(name)
	var referrers []string
	if !isDir {
		referrers = x.ReferencedBy(rel)
		if len(referrers) > 0 && strings.ContainsAny(newName, linkBreakers) {
			return undo.Action{}, res, ErrUnlinkable
		}
	}

	var done []undo.Action
	fail := func(err error) (undo.Action, RenameResult, error) {
		if len(done) > 0 {
			if rbErr := undo.Group("rollback", done...).Undo(); rbErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
			}
		}
		return undo.Action{}, RenameResult{}, err
	}

	move, err := s.Move(rel, dst)
	if err != nil {
		return undo.Action{}, res, err
	}
	done = append(done, move)
	if isDir {
		return undo.Group("rename "+filepath.Base(rel), done...), res, nil
	}

	// Editing with no changes still sets the title and adds a missing uid.
	switch act, err := s.Edit(dst, func(*store.Frontmatter) {}); {
	case err == nil:
		done = append(done, act)
	case skippable(err):
		res.TitleKept = true
	default:
		return fail(err)
	}
	for _, ref := range referrers {
		act, err := s.Edit(ref, func(fm *store.Frontmatter) { RenameReference(fm, oldName, newName) })
		switch {
		case err == nil:
			done = append(done, act)
			res.Updated++
		case skippable(err):
			res.Skipped++
		default:
			return fail(err)
		}
	}
	return undo.Group("rename "+filepath.Base(rel), done...), res, nil
}

// skippable reports an Edit failure that only means this note cannot be
// edited, not that something went wrong.
func skippable(err error) bool {
	return errors.Is(err, store.ErrFrontmatterInvalid) || errors.Is(err, store.ErrTooLarge)
}

// RenameReference rewrites every entry of the related property of fm that
// names oldName so it names newName, keeping any folder prefix, .md, #heading
// and |alias. It reports whether anything changed.
func RenameReference(fm *store.Frontmatter, oldName, newName string) bool {
	node := fm.Get(relatedKey)
	if node == nil {
		return false
	}
	if node.Kind == yaml.ScalarNode {
		return retargetScalar(node, oldName, newName)
	}
	if node.Kind != yaml.SequenceNode {
		return false
	}
	changed := false
	for i, item := range node.Content {
		switch {
		case item.Kind == yaml.ScalarNode:
			changed = retargetScalar(item, oldName, newName) || changed
		case entryNames(item, oldName):
			// An unquoted [[Name]] that YAML read as a nested list becomes a
			// quoted string.
			raw := linkOpen + item.Content[0].Value + linkClose
			node.Content[i] = quoted(retarget(raw, newName))
			changed = true
		}
	}
	return changed
}

func retargetScalar(n *yaml.Node, oldName, newName string) bool {
	if !entryNames(n, oldName) {
		return false
	}
	n.Value, n.Tag, n.Style = retarget(n.Value, newName), "!!str", yaml.DoubleQuotedStyle
	return true
}

func quoted(text string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: text}
}

// retarget returns the wikilink raw with its note name replaced by newName,
// keeping the folder prefix, a .md extension, the #heading and the |alias.
// raw must already have passed Target.
func retarget(raw, newName string) string {
	s := strings.TrimSpace(raw)
	inner := s[len(linkOpen) : len(s)-len(linkClose)]
	target, alias, hasAlias := strings.Cut(inner, "|")
	path, heading, hasHeading := strings.Cut(target, "#")
	prefix := ""
	if i := strings.LastIndex(path, "/"); i >= 0 {
		prefix, path = path[:i+1], path[i+1:]
	}
	ext := ""
	if len(path) >= len(noteExt) && strings.EqualFold(path[len(path)-len(noteExt):], noteExt) {
		ext = path[len(path)-len(noteExt):]
	}
	out := prefix + newName + ext
	if hasHeading {
		out += "#" + heading
	}
	if hasAlias {
		out += "|" + alias
	}
	return linkOpen + out + linkClose
}
