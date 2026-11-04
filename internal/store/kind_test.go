package store

import (
	"path/filepath"
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
	assert.Equal(t, []Kind{KindFact, KindRumor, KindProposal, KindFossil, KindIdea, KindDraft, KindLink}, got)
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

func TestKindMeaningIsDefinedForEveryKind(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range Kinds() {
		m := k.Meaning()
		assert.NotEmpty(t, m, k)
		assert.False(t, seen[m], "meanings are distinct: %s", k)
		seen[m] = true
	}
	assert.Empty(t, KindUnset.Meaning())
}

func TestSetKindThroughEditChangesOnlyTheKind(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	put(t, p, "---\nuid: u1\ntitle: n\nkind: Rumor\ntags: [x]\n---\nbody\n")

	act, err := NewMarkdown(root).Edit("inbox/n.md", func(fm *Frontmatter) { SetKind(fm, KindFact) })
	require.NoError(t, err)
	n := ParseNote("n.md", []byte(read(t, p)))
	assert.Equal(t, KindFact, n.Kind)
	assert.Equal(t, "u1", n.Frontmatter.Get("uid").Value)
	assert.Equal(t, "body\n", n.Body)
	assert.NotNil(t, n.Frontmatter.Get("tags"))

	require.NoError(t, act.Undo())
	assert.Equal(t, KindRumor, ParseNote("n.md", []byte(read(t, p))).Kind, "undo restores the previous kind")
}

func TestSetKindOnANoteWithoutOne(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	put(t, p, "plain\n")
	_, err := NewMarkdown(root).Edit("inbox/n.md", func(fm *Frontmatter) { SetKind(fm, KindIdea) })
	require.NoError(t, err)
	assert.Equal(t, KindIdea, ParseNote("n.md", []byte(read(t, p))).Kind)
}

func TestWebKind(t *testing.T) {
	assert.Equal(t, KindLink, ParseKind(" Link "))
	assert.Contains(t, Kinds(), KindLink)
	assert.NotEmpty(t, KindLink.Meaning())
	assert.Len(t, Kinds(), 7)
}
