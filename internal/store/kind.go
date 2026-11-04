package store

import (
	"strings"

	"go.yaml.in/yaml/v3"
)

// kindKey is the frontmatter property that holds a note's kind.
const kindKey = "kind"

// Kind classifies a note, such as how far it can be trusted. The set is the
// built-in kinds plus any the config adds (see SetKinds); a missing or
// unrecognized value reads as KindUnset and is never rewritten.
type Kind string

const (
	// KindUnset means the note has no valid kind property.
	KindUnset Kind = "unset"
	// KindFact is accurate information.
	KindFact Kind = "fact"
	// KindRumor is unconfirmed information.
	KindRumor Kind = "rumor"
	// KindProposal is documented desired outcomes; it may not reflect reality.
	KindProposal Kind = "proposal"
	// KindFossil is or was true, but may have drifted due to time.
	KindFossil Kind = "fossil"
	// KindIdea is a loose thought that has not graduated to a proposal.
	KindIdea Kind = "idea"
	// KindDraft is a forming idea that is incomplete and still being revised.
	KindDraft Kind = "draft"
	// KindLink is a saved link to a web page; its address is the source property.
	KindLink Kind = "link"
)

// Kinds returns the assignable kinds (not KindUnset) in display order: the
// ones the config lists, in its order, then the built-in ones it does not
// list (see SetKinds).
func Kinds() []Kind {
	kindMu.RLock()
	defer kindMu.RUnlock()
	return append([]Kind(nil), kindList...)
}

// ParseKind maps s to a kind, ignoring case and surrounding space. Anything
// that is not a configured kind is KindUnset.
func ParseKind(s string) Kind {
	k := Kind(strings.ToLower(strings.TrimSpace(s)))
	kindMu.RLock()
	defer kindMu.RUnlock()
	for _, valid := range kindList {
		if k == valid {
			return k
		}
	}
	return KindUnset
}

// String returns the lowercase kind name.
func (k Kind) String() string { return string(k) }

// KindOf reads the kind property of fm. A nil fm, an absent key, or a value
// that is not a plain string reads as KindUnset.
func KindOf(fm *Frontmatter) Kind {
	if fm == nil {
		return KindUnset
	}
	v := fm.Get(kindKey)
	if v == nil || v.Kind != yaml.ScalarNode || v.Tag != "!!str" {
		return KindUnset
	}
	return ParseKind(v.Value)
}

// Meaning is a short description of the kind, shown when picking one: the
// config's, else the built-in one. It is empty for KindUnset and for an added
// kind that has none.
func (k Kind) Meaning() string {
	kindMu.RLock()
	m := kindSpecs[k].Meaning
	kindMu.RUnlock()
	if m != "" {
		return m
	}
	switch k {
	case KindFact:
		return "Accurate information"
	case KindRumor:
		return "Unconfirmed information"
	case KindProposal:
		return "Desired outcome, may not be real"
	case KindFossil:
		return "Was true, may have drifted"
	case KindIdea:
		return "Loose thought"
	case KindDraft:
		return "Incomplete, still revising"
	case KindLink:
		return "Saved link to a web page (o opens it)"
	}
	return ""
}

// SetKind sets the kind property of fm to k, keeping its place among the
// properties when it already has one.
func SetKind(fm *Frontmatter, k Kind) {
	fm.SetString(kindKey, k.String())
}
