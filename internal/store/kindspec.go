package store

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	// maxKindName bounds the length of a kind's name.
	maxKindName = 20
	// maxKindMeaning bounds the length of a kind's meaning, in characters.
	maxKindMeaning = 60
	// maxKindBadge bounds a badge, in characters.
	maxKindBadge = 4
	// maxCustomKinds bounds how many kinds can be added to the built-in ones.
	maxCustomKinds = 20
	// hexColorLen is the length of "#rrggbb".
	hexColorLen = 7
)

// KindSpec describes a kind as the config file gives it. An entry whose Name
// is a built-in kind overrides what is set; any other name adds a kind. Empty
// fields keep the built-in value, or for a new kind leave it to the interface.
type KindSpec struct {
	// Name is the lowercase name written to the kind property.
	Name string
	// Meaning is the short description shown when picking the kind.
	Meaning string
	// Badge is the short text (at most four characters) shown on a row.
	Badge string
	// Color is "#rrggbb", or empty.
	Color string
}

// The registry is the built-in kinds plus what the config adds or overrides.
// It is set once at startup (and by tests), and read everywhere else.
var (
	kindMu    sync.RWMutex
	kindList  = builtinKinds()
	kindSpecs = map[Kind]KindSpec{}
)

func builtinKinds() []Kind {
	return []Kind{KindFact, KindRumor, KindProposal, KindFossil, KindIdea, KindDraft, KindWeb}
}

// ResetKinds goes back to the built-in kinds.
func ResetKinds() {
	kindMu.Lock()
	defer kindMu.Unlock()
	kindList, kindSpecs = builtinKinds(), map[Kind]KindSpec{}
}

// SetKinds makes the kinds the built-in ones plus what specs add or override
// (see KindSpec). The kinds in specs come first, in the order given, then the
// built-in kinds not in specs, in their own order. Nothing changes when a
// spec is not valid.
func SetKinds(specs []KindSpec) error {
	clean, err := ValidateKindSpecs(specs)
	if err != nil {
		return err
	}
	list, byName := make([]Kind, 0, len(clean)+len(builtinKinds())), make(map[Kind]KindSpec, len(clean))
	for _, s := range clean {
		k := Kind(s.Name)
		byName[k] = s
		list = append(list, k)
	}
	for _, b := range builtinKinds() {
		if _, listed := byName[b]; !listed {
			list = append(list, b)
		}
	}
	kindMu.Lock()
	defer kindMu.Unlock()
	kindList, kindSpecs = list, byName
	return nil
}

func isBuiltin(k Kind) bool {
	for _, b := range builtinKinds() {
		if b == k {
			return true
		}
	}
	return false
}

// KindStyle returns the badge and color the config gave k, either of which is
// empty when it did not. The interface fills in the defaults.
func KindStyle(k Kind) (badge, color string) {
	kindMu.RLock()
	defer kindMu.RUnlock()
	s := kindSpecs[k]
	return s.Badge, s.Color
}

// ValidateKindSpecs checks specs and returns them cleaned: names lowercase,
// white space trimmed, colors lowercase. A name is a lowercase letter followed
// by letters, digits and hyphens, at most 20 long, not "unset" and not used
// twice; a color is "#rrggbb"; a badge is at most four characters; a meaning
// at most 60; at most 20 entries may add a new kind. The error names the entry.
func ValidateKindSpecs(specs []KindSpec) ([]KindSpec, error) {
	out := make([]KindSpec, 0, len(specs))
	seen := map[string]bool{}
	added := 0
	for i, s := range specs {
		s.Name = strings.ToLower(strings.TrimSpace(s.Name))
		s.Meaning = strings.TrimSpace(s.Meaning)
		s.Badge = strings.TrimSpace(s.Badge)
		s.Color = strings.ToLower(strings.TrimSpace(s.Color))
		where := fmt.Sprintf("kinds entry %d", i+1)
		if err := checkKindSpec(s, where); err != nil {
			return nil, err
		}
		if seen[s.Name] {
			return nil, fmt.Errorf("%s: the kind %q is listed twice", where, s.Name)
		}
		seen[s.Name] = true
		if !isBuiltin(Kind(s.Name)) {
			if added++; added > maxCustomKinds {
				return nil, fmt.Errorf("kinds: at most %d kinds can be added to the built-in ones", maxCustomKinds)
			}
		}
		out = append(out, s)
	}
	return out, nil
}

// checkKindSpec checks one cleaned entry. where names it in the error.
func checkKindSpec(s KindSpec, where string) error {
	switch {
	case s.Name == "":
		return fmt.Errorf("%s: a name is needed", where)
	case len(s.Name) > maxKindName:
		return fmt.Errorf("%s (%q): the name is too long (at most %d characters)", where, s.Name, maxKindName)
	case !validKindName(s.Name):
		return fmt.Errorf("%s (%q): a name starts with a letter and holds only letters, digits and hyphens", where, s.Name)
	case s.Name == string(KindUnset):
		return fmt.Errorf("%s: the name unset is reserved", where)
	case s.Color != "" && !validHexColor(s.Color):
		return fmt.Errorf("%s (%q): the color must look like #2ea043", where, s.Name)
	case utf8.RuneCountInString(s.Badge) > maxKindBadge || hasControl(s.Badge):
		return fmt.Errorf("%s (%q): the badge is at most %d printable characters", where, s.Name, maxKindBadge)
	case utf8.RuneCountInString(s.Meaning) > maxKindMeaning || hasControl(s.Meaning):
		return fmt.Errorf("%s (%q): the meaning is one line of at most %d characters", where, s.Name, maxKindMeaning)
	}
	return nil
}

func validKindName(name string) bool {
	for i, r := range name {
		letter := r >= 'a' && r <= 'z'
		if i == 0 && !letter {
			return false
		}
		if !letter && !(r >= '0' && r <= '9') && r != '-' {
			return false
		}
	}
	return true
}

func validHexColor(c string) bool {
	if len(c) != hexColorLen || c[0] != '#' {
		return false
	}
	for _, r := range c[1:] {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func hasControl(s string) bool {
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}
