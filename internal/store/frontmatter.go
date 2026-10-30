package store

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	fmDelimiter = "---"
	fmIndent    = 2
)

var utf8BOM = []byte("\xef\xbb\xbf")

// ErrFrontmatterNotMapping reports a frontmatter block that is valid YAML
// but not a key/value mapping.
var ErrFrontmatterNotMapping = errors.New("frontmatter is not a mapping")

// Frontmatter is a note's YAML properties block. It keeps the parsed
// yaml.Node tree rather than a fixed struct, so unknown keys, key order and
// comments survive a parse and Marshal round trip. Marshal normalizes
// formatting (indentation, quoting), not content.
type Frontmatter struct {
	node *yaml.Node // always a MappingNode
}

// ParseFrontmatter parses raw YAML (without the --- delimiters).
func ParseFrontmatter(raw []byte) (*Frontmatter, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	if doc.Kind == 0 { // empty or comment-free blank input
		return &Frontmatter{node: &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}}, nil
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, ErrFrontmatterNotMapping
	}
	return &Frontmatter{node: doc.Content[0]}, nil
}

// NewFrontmatter returns an empty properties block.
func NewFrontmatter() *Frontmatter {
	return &Frontmatter{node: &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}}
}

// SetString sets key to a string value, replacing it in place when the key
// exists and appending it otherwise, so key order is kept. The value is
// always a YAML string, quoted when it would otherwise read as something
// else (a number, a boolean, a mapping).
func (f *Frontmatter) SetString(key, value string) {
	val := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	for i := 0; i+1 < len(f.node.Content); i += 2 {
		if k := f.node.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			// Keep the comments the old value carried.
			old := f.node.Content[i+1]
			val.HeadComment, val.LineComment, val.FootComment = old.HeadComment, old.LineComment, old.FootComment
			f.node.Content[i+1] = val
			return
		}
	}
	f.node.Content = append(f.node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
}

// SetStringList sets key to a list of strings, in place when the key exists
// and appended otherwise. Every item is a double-quoted YAML string, so text
// such as [[Link]] is never read as a nested list. The comments the old
// value carried are kept.
func (f *Frontmatter) SetStringList(key string, items []string) {
	val := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, item := range items {
		val.Content = append(val.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: item})
	}
	for i := 0; i+1 < len(f.node.Content); i += 2 {
		if k := f.node.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			old := f.node.Content[i+1]
			val.HeadComment, val.LineComment, val.FootComment = old.HeadComment, old.LineComment, old.FootComment
			f.node.Content[i+1] = val
			return
		}
	}
	f.node.Content = append(f.node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
}

// Delete removes key and its value, reporting whether it was there.
func (f *Frontmatter) Delete(key string) bool {
	for i := 0; i+1 < len(f.node.Content); i += 2 {
		if k := f.node.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			f.node.Content = append(f.node.Content[:i], f.node.Content[i+2:]...)
			return true
		}
	}
	return false
}

// Get returns the value node for key, or nil when the key is absent.
func (f *Frontmatter) Get(key string) *yaml.Node {
	for i := 0; i+1 < len(f.node.Content); i += 2 {
		if k := f.node.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			return f.node.Content[i+1]
		}
	}
	return nil
}

// Property is one key and value of the properties block.
type Property struct {
	Key   string
	Value *yaml.Node
}

// Properties returns every property in file order. Keys that are not plain
// scalars are named by their text, or "?" when they have none.
func (f *Frontmatter) Properties() []Property {
	props := make([]Property, 0, len(f.node.Content)/2)
	for i := 0; i+1 < len(f.node.Content); i += 2 {
		key := f.node.Content[i].Value
		if key == "" {
			key = "?"
		}
		props = append(props, Property{Key: key, Value: f.node.Content[i+1]})
	}
	return props
}

// Marshal encodes the properties as YAML (without delimiters). An empty
// mapping encodes as no bytes.
func (f *Frontmatter) Marshal() ([]byte, error) {
	if len(f.node.Content) == 0 {
		return nil, nil
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(fmIndent)
	if err := enc.Encode(f.node); err != nil {
		return nil, fmt.Errorf("encode frontmatter: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encode frontmatter: %w", err)
	}
	return buf.Bytes(), nil
}

// splitFrontmatter separates a leading "---" delimited block from the body.
// It reports ok=false, with body as the whole input, when the file does not
// start with a delimiter line or the block is never closed.
func splitFrontmatter(data []byte) (raw, body []byte, ok bool) {
	data = bytes.TrimPrefix(data, utf8BOM)
	rest, found := cutLine(data, func(l string) bool { return l == fmDelimiter })
	if !found {
		return nil, data, false
	}
	var block []byte
	for len(rest) > 0 {
		line, next := nextLine(rest)
		if strings.TrimRight(string(line), "\r") == fmDelimiter {
			return block, next, true
		}
		block = append(block, rest[:len(rest)-len(next)]...)
		rest = next
	}
	return nil, data, false
}

// cutLine checks that the first line of data satisfies match (ignoring a
// trailing \r) and returns what follows it.
func cutLine(data []byte, match func(string) bool) ([]byte, bool) {
	line, next := nextLine(data)
	if len(data) == 0 || !match(strings.TrimRight(string(line), "\r")) {
		return data, false
	}
	return next, true
}

// nextLine returns the first line of data (without its \n) and the rest.
func nextLine(data []byte) (line, rest []byte) {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return data[:i], data[i+1:]
	}
	return data, nil
}
