package info

import (
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tkilb/para/internal/textsafe"
)

const (
	// ellipsis marks output cut short by a limit.
	ellipsis = "…"
	// indent is the nested-value indentation.
	indent = "  "
)

// single sanitizes text that must stay on one line.
func single(s string) []string { return []string{textsafe.SingleLine(s)} }

// valueLines draws a YAML value generically as lines of text:
//   - a scalar is its text, line by line;
//   - a list of scalars is one comma-separated line, any other list is
//     one "- " item per line;
//   - a mapping is "key: value" lines, with a multi-line value indented
//     under "key:";
//   - an alias draws what it points to.
//
// budget counts the lines still allowed, and depth bounds recursion, so a
// huge or self-referencing value cannot make the panel unusable. Everything
// is sanitized: property values are untrusted text.
func valueLines(n *yaml.Node, depth int, budget *int) []string {
	if n == nil || *budget <= 0 || depth > maxDepth {
		return []string{ellipsis}
	}
	switch n.Kind {
	case yaml.AliasNode:
		return valueLines(n.Alias, depth+1, budget)
	case yaml.ScalarNode:
		return scalarLines(n, budget)
	case yaml.SequenceNode:
		return sequenceLines(n, depth, budget)
	case yaml.MappingNode:
		return mappingLines(n, depth, budget)
	}
	return []string{""}
}

func scalarLines(n *yaml.Node, budget *int) []string {
	lines := strings.Split(strings.TrimRight(textsafe.Sanitize(n.Value), "\n"), "\n")
	if len(lines) > *budget {
		lines = append(lines[:max(*budget, 0)], ellipsis)
	}
	*budget -= len(lines)
	return lines
}

func sequenceLines(n *yaml.Node, depth int, budget *int) []string {
	if len(n.Content) == 0 {
		return []string{""}
	}
	if items, ok := scalarItems(n); ok {
		*budget--
		return []string{strings.Join(items, ", ")}
	}
	var out []string
	for _, item := range n.Content {
		lines := valueLines(item, depth+1, budget)
		for i, l := range lines {
			if i == 0 {
				out = append(out, "- "+l)
			} else {
				out = append(out, indent+l)
			}
		}
		if *budget <= 0 {
			break
		}
	}
	return out
}

// scalarItems returns the texts of a list whose items are all single-line
// scalars (following aliases), for drawing on one line.
func scalarItems(n *yaml.Node) ([]string, bool) {
	items := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		item = resolve(item)
		if item.Kind != yaml.ScalarNode || strings.Contains(item.Value, "\n") {
			return nil, false
		}
		items = append(items, textsafe.SingleLine(item.Value))
	}
	return items, true
}

// resolve follows aliases to the node they point to.
func resolve(n *yaml.Node) *yaml.Node {
	for depth := 0; n.Kind == yaml.AliasNode && n.Alias != nil && depth < maxDepth; depth++ {
		n = n.Alias
	}
	return n
}

func mappingLines(n *yaml.Node, depth int, budget *int) []string {
	if len(n.Content) == 0 {
		return []string{""}
	}
	var out []string
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := textsafe.SingleLine(n.Content[i].Value)
		if key == "" {
			key = "?"
		}
		child := n.Content[i+1]
		lines := valueLines(child, depth+1, budget)
		switch {
		case len(lines) == 1 && lines[0] == "":
			out = append(out, key+":")
		case len(lines) == 1 && resolve(child).Kind != yaml.MappingNode:
			out = append(out, key+": "+lines[0])
		default:
			out = append(out, key+":")
			for _, l := range lines {
				out = append(out, indent+l)
			}
		}
		if *budget <= 0 {
			break
		}
	}
	return out
}
