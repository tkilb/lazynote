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

// Get returns the value node for key, or nil when the key is absent.
func (f *Frontmatter) Get(key string) *yaml.Node {
	for i := 0; i+1 < len(f.node.Content); i += 2 {
		if k := f.node.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			return f.node.Content[i+1]
		}
	}
	return nil
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
