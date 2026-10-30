package links

import (
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tkilb/lazynote/internal/store"
)

// AddReference adds [[name]] to the related property of fm, creating the
// property or turning a single string into a list as needed. It reports
// false, changing nothing, when the note already references name (ignoring
// case, any |alias and any #heading). Entries it does not own stay as they
// are.
func AddReference(fm *store.Frontmatter, name string) bool {
	node := fm.Get(relatedKey)
	if node != nil && referencesName(node, name) {
		return false
	}
	link := linkOpen + name + linkClose
	switch {
	case node == nil:
		fm.SetStringList(relatedKey, []string{link})
	case node.Kind == yaml.SequenceNode:
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: link})
	case node.Kind == yaml.ScalarNode:
		fm.SetStringList(relatedKey, []string{node.Value, link})
	default:
		return false
	}
	return true
}

// RemoveReference removes every entry of the related property of fm that
// names name (ignoring case, any |alias and any #heading), and drops the
// property when nothing is left. It reports whether anything was removed.
// Entries it does not own stay as they are.
func RemoveReference(fm *store.Frontmatter, name string) bool {
	node := fm.Get(relatedKey)
	if node == nil {
		return false
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if !entryNames(node, name) {
			return false
		}
		return fm.Delete(relatedKey)
	case yaml.SequenceNode:
		kept := node.Content[:0:0]
		for _, item := range node.Content {
			if !entryNames(item, name) {
				kept = append(kept, item)
			}
		}
		if len(kept) == len(node.Content) {
			return false
		}
		if len(kept) == 0 {
			return fm.Delete(relatedKey)
		}
		node.Content = kept
		return true
	}
	return false
}

// referencesName reports whether the related value already names name.
func referencesName(node *yaml.Node, name string) bool {
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			if entryNames(item, name) {
				return true
			}
		}
		return false
	}
	return entryNames(node, name)
}

// entryNames reports whether one related entry (a wikilink string, or an
// unquoted [[Name]] that YAML read as a nested list) names name.
func entryNames(item *yaml.Node, name string) bool {
	raw := ""
	switch {
	case item.Kind == yaml.ScalarNode:
		raw = item.Value
	case item.Kind == yaml.SequenceNode && len(item.Content) == 1 && item.Content[0].Kind == yaml.ScalarNode:
		raw = linkOpen + item.Content[0].Value + linkClose
	default:
		return false
	}
	target, ok := Target(raw)
	return ok && strings.EqualFold(target, name)
}
