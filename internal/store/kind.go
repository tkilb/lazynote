package store

import (
	"strings"

	"go.yaml.in/yaml/v3"
)

// kindKey is the frontmatter property that holds a note's kind.
const kindKey = "kind"

// Kind classifies how far a note can be trusted. The set is closed; a
// missing or unrecognized value reads as KindUnset and is never rewritten.
type Kind string

const (
	// KindUnset means the note has no valid kind property.
	KindUnset Kind = "unset"
	// KindFact is confirmed information.
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
)

// kinds lists every assignable kind in display order.
var kinds = []Kind{KindFact, KindRumor, KindProposal, KindFossil, KindIdea, KindDraft}

// Kinds returns the six assignable kinds (not KindUnset) in display order.
func Kinds() []Kind {
	return append([]Kind(nil), kinds...)
}

// ParseKind maps s to a kind, ignoring case and surrounding space. Anything
// that is not one of the six kinds is KindUnset.
func ParseKind(s string) Kind {
	k := Kind(strings.ToLower(strings.TrimSpace(s)))
	for _, valid := range kinds {
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

// Meaning is a short description of the kind, shown when picking one. It is
// empty for KindUnset.
func (k Kind) Meaning() string {
	switch k {
	case KindFact:
		return "Confirmed information"
	case KindRumor:
		return "Unconfirmed information"
	case KindProposal:
		return "Desired outcomes; may not reflect reality"
	case KindFossil:
		return "Was true, but may have drifted over time"
	case KindIdea:
		return "A loose thought, not yet a proposal"
	case KindDraft:
		return "Incomplete and still being revised"
	}
	return ""
}

// SetKind sets the kind property of fm to k, keeping its place among the
// properties when it already has one.
func SetKind(fm *Frontmatter, k Kind) {
	fm.SetString(kindKey, k.String())
}
