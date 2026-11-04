package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// useKinds installs specs for one test and restores the built-ins after.
func useKinds(t *testing.T, specs ...KindSpec) {
	t.Helper()
	t.Cleanup(ResetKinds)
	require.NoError(t, SetKinds(specs))
}

func TestSetKindsFollowsTheConfigOrderBuiltInsIncluded(t *testing.T) {
	useKinds(t,
		KindSpec{Name: "recipe"},
		KindSpec{Name: "link"},
		KindSpec{Name: "fact"},
	)
	assert.Equal(t,
		[]Kind{"recipe", KindLink, KindFact, KindRumor, KindProposal, KindFossil, KindIdea, KindDraft},
		Kinds(), "listed kinds come first in config order, then the built-ins not listed, in their own order")
}

func TestSetKindsPutsAddedKindsInConfigOrderBeforeUnlistedBuiltIns(t *testing.T) {
	useKinds(t,
		KindSpec{Name: "recipe", Color: "#ff8800", Meaning: "Something to cook", Badge: "cook"},
		KindSpec{Name: "Quote"},
	)
	got := Kinds()
	assert.Equal(t, []Kind{"recipe", "quote", KindFact, KindRumor, KindProposal, KindFossil, KindIdea, KindDraft, KindLink}, got)

	assert.Equal(t, Kind("recipe"), ParseKind("  RECIPE "), "matching ignores case and space")
	assert.Equal(t, "Something to cook", Kind("recipe").Meaning())
	badge, color := KindStyle("recipe")
	assert.Equal(t, "cook", badge)
	assert.Equal(t, "#ff8800", color)
	badge, color = KindStyle("quote")
	assert.Empty(t, badge, "no badge given: the interface derives one")
	assert.Empty(t, color)
}

func TestSetKindsOverridesABuiltIn(t *testing.T) {
	useKinds(t, KindSpec{Name: "fact", Color: "#2ea043", Meaning: "Verified"})
	assert.Equal(t, KindFact, Kinds()[0])
	assert.Len(t, Kinds(), 7, "an override adds no kind")
	assert.Equal(t, "Verified", KindFact.Meaning())
	assert.Equal(t, "Unconfirmed information", KindRumor.Meaning(), "the others keep their meaning")
	_, color := KindStyle(KindFact)
	assert.Equal(t, "#2ea043", color)
	_, color = KindStyle(KindRumor)
	assert.Empty(t, color)
}

func TestUnknownKindsStayUnsetAndResetRestoresTheBuiltIns(t *testing.T) {
	assert.Equal(t, KindUnset, ParseKind("recipe"))
	require.NoError(t, SetKinds([]KindSpec{{Name: "recipe"}}))
	assert.Equal(t, Kind("recipe"), ParseKind("recipe"))
	ResetKinds()
	assert.Equal(t, KindUnset, ParseKind("recipe"), "a note with a removed kind reads as unset")
	assert.Len(t, Kinds(), 7)
}

func TestKindOfReadsACustomKind(t *testing.T) {
	useKinds(t, KindSpec{Name: "recipe"})
	n := ParseNote("a.md", []byte("---\nkind: recipe\n---\nbody\n"))
	assert.Equal(t, Kind("recipe"), n.Kind)
}

func TestValidateKindSpecsRejectsBadEntries(t *testing.T) {
	tests := []struct {
		name  string
		specs []KindSpec
		want  string
	}{
		{"no name", []KindSpec{{Color: "#112233"}}, "entry 1"},
		{"blank name", []KindSpec{{Name: "   "}}, "entry 1"},
		{"name with a space", []KindSpec{{Name: "my kind"}}, `"my kind"`},
		{"name starts with a digit", []KindSpec{{Name: "1st"}}, `"1st"`},
		{"name with a symbol", []KindSpec{{Name: "a/b"}}, `"a/b"`},
		{"name too long", []KindSpec{{Name: strings.Repeat("a", maxKindName+1)}}, "too long"},
		{"unset is reserved", []KindSpec{{Name: "Unset"}}, "unset"},
		{"duplicate ignoring case", []KindSpec{{Name: "recipe"}, {Name: "RECIPE"}}, "twice"},
		{"duplicate built-in", []KindSpec{{Name: "fact"}, {Name: "fact"}}, "twice"},
		{"color not hex", []KindSpec{{Name: "a", Color: "green"}}, "color"},
		{"short hex", []KindSpec{{Name: "a", Color: "#123"}}, "color"},
		{"hex without hash", []KindSpec{{Name: "a", Color: "ff8800"}}, "color"},
		{"hex too long", []KindSpec{{Name: "a", Color: "#ff880011"}}, "color"},
		{"not a hex digit", []KindSpec{{Name: "a", Color: "#gg8800"}}, "color"},
		{"badge too long", []KindSpec{{Name: "a", Badge: "toolong"}}, "badge"},
		{"badge with a control character", []KindSpec{{Name: "a", Badge: "a\x1b"}}, "badge"},
		{"meaning too long", []KindSpec{{Name: "a", Meaning: strings.Repeat("x", maxKindMeaning+1)}}, "meaning"},
		{"meaning with a newline", []KindSpec{{Name: "a", Meaning: "two\nlines"}}, "meaning"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateKindSpecs(tt.specs)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
			assert.Error(t, SetKinds(tt.specs), "nothing is installed")
			assert.Len(t, Kinds(), 7, "the built-ins are untouched after a refusal")
		})
	}
}

func TestValidateKindSpecsBoundsTheNumberOfKinds(t *testing.T) {
	var specs []KindSpec
	for i := 0; i < maxCustomKinds+1; i++ {
		specs = append(specs, KindSpec{Name: "k" + strings.Repeat("a", i%5) + string(rune('a'+i/5)) + string(rune('a'+i%26))})
	}
	_, err := ValidateKindSpecs(specs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most")
}

func TestValidateKindSpecsCleansWhatItAccepts(t *testing.T) {
	got, err := ValidateKindSpecs([]KindSpec{{Name: " Recipe ", Color: "#FF8800", Meaning: "  Cook it  ", Badge: " cook "}})
	require.NoError(t, err)
	assert.Equal(t, []KindSpec{{Name: "recipe", Color: "#ff8800", Meaning: "Cook it", Badge: "cook"}}, got)
}
