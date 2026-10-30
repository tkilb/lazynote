package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKind(t *testing.T) {
	tests := []struct {
		in   string
		want Kind
	}{
		{"fact", KindFact},
		{"rumor", KindRumor},
		{"proposal", KindProposal},
		{"fossil", KindFossil},
		{"idea", KindIdea},
		{"draft", KindDraft},
		{"FACT", KindFact},
		{"Rumor", KindRumor},
		{"  idea\t", KindIdea},
		{"", KindUnset},
		{"unset", KindUnset},
		{"facts", KindUnset},
		{"fact idea", KindUnset},
		{"note", KindUnset},
		{"fa ct", KindUnset},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseKind(tt.in))
		})
	}
}

func TestKinds(t *testing.T) {
	got := Kinds()
	assert.Equal(t, []Kind{KindFact, KindRumor, KindProposal, KindFossil, KindIdea, KindDraft}, got)
	assert.NotContains(t, got, KindUnset)

	got[0] = KindDraft
	assert.Equal(t, KindFact, Kinds()[0], "callers cannot modify the shared list")
}

func TestKindString(t *testing.T) {
	assert.Equal(t, "fact", KindFact.String())
	assert.Equal(t, "unset", KindUnset.String())
}

func TestKindOf(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want Kind
	}{
		{"valid", "kind: fact\n", KindFact},
		{"case-insensitive", "kind: Proposal\n", KindProposal},
		{"quoted", "kind: \"draft\"\n", KindDraft},
		{"absent", "tags: [a]\n", KindUnset},
		{"empty value", "kind:\n", KindUnset},
		{"unrecognized", "kind: story\n", KindUnset},
		{"list value", "kind: [fact]\n", KindUnset},
		{"mapping value", "kind:\n  fact: 1\n", KindUnset},
		{"number value", "kind: 5\n", KindUnset},
		{"bool value", "kind: true\n", KindUnset},
		{"key is case-sensitive", "Kind: fact\n", KindUnset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, err := ParseFrontmatter([]byte(tt.yaml))
			require.NoError(t, err)
			assert.Equal(t, tt.want, KindOf(fm))
		})
	}

	t.Run("nil frontmatter", func(t *testing.T) {
		assert.Equal(t, KindUnset, KindOf(nil))
	})
}

func TestKindIsNeverRewrittenOnRead(t *testing.T) {
	fm, err := ParseFrontmatter([]byte("kind: Story\nother: 1\n"))
	require.NoError(t, err)
	require.Equal(t, KindUnset, KindOf(fm))

	out, err := fm.Marshal()
	require.NoError(t, err)
	assert.Contains(t, string(out), "kind: Story", "an unrecognized value stays as written")
}

func TestParseNoteKind(t *testing.T) {
	assert.Equal(t, KindRumor, ParseNote("x.md", []byte("---\nkind: RUMOR\n---\nb")).Kind)
	assert.Equal(t, KindUnset, ParseNote("x.md", []byte("---\nkind: nope\n---\nb")).Kind)
	assert.Equal(t, KindUnset, ParseNote("x.md", []byte("no frontmatter")).Kind)
	assert.Equal(t, KindUnset, ParseNote("x.md", []byte("---\n- a\n---\nb")).Kind, "invalid frontmatter")
}
