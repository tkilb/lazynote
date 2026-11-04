package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keysOf returns the property names of a note's data, in file order.
func keysOf(t *testing.T, data []byte) []string {
	t.Helper()
	n := ParseNote("x.md", data)
	require.NoError(t, n.FrontmatterErr)
	require.NotNil(t, n.Frontmatter)
	var keys []string
	for _, p := range n.Frontmatter.Properties() {
		keys = append(keys, p.Key)
	}
	return keys
}

var standardKeys = []string{"uid", "title", "kind", "source", "date", "tags", "people"}

func TestEditedNoteIsStandard(t *testing.T) {
	tests := []struct {
		name, in string
		want     []string
	}{
		{"no frontmatter", "text\n", append(append([]string(nil), standardKeys...), "related")},
		{"out of order, others kept in place, related last",
			"---\nrelated: [\"[[A]]\"]\nzeta: 1\ntags: [a]\nalpha: 2\nuid: abc\nkind: fact\n---\nbody\n",
			append(append([]string(nil), standardKeys...), "zeta", "alpha", "related")},
		{"already standard", "---\nuid: abc\ntitle: x\nkind: fact\nsource: s\ndate: 2026-01-02\ntags: []\npeople: []\nk: v\nrelated: []\n---\nb\n",
			append(append([]string(nil), standardKeys...), "k", "related")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := editedNote([]byte(tt.in), "x.md", func(*Frontmatter) {})
			require.NoError(t, err)
			assert.Equal(t, tt.want, keysOf(t, out))
			assert.Equal(t, ParseNote("x.md", []byte(tt.in)).Body, ParseNote("x.md", out).Body, "the body is untouched")
		})
	}
}

func TestEditedNoteKeepsValuesAndAddsEmptyOnes(t *testing.T) {
	in := "---\nuid: keep-me\nkind: fact\nzeta: [1, 2]\n---\nbody\n"
	out, err := editedNote([]byte(in), "x.md", func(*Frontmatter) {})
	require.NoError(t, err)
	n := ParseNote("x.md", out)
	assert.Equal(t, "keep-me", n.Frontmatter.Get("uid").Value, "an existing uid is never replaced")
	assert.Equal(t, KindFact, n.Kind)
	assert.Equal(t, "", n.Frontmatter.Get("source").Value, "a missing property is added empty")
	assert.Equal(t, "", n.Frontmatter.Get("date").Value)
	assert.Empty(t, n.Frontmatter.Get("tags").Content)
	assert.Len(t, n.Frontmatter.Get("zeta").Content, 2, "unknown properties round-trip")
}

func TestCreatedNotesAreStandard(t *testing.T) {
	blank, err := BlankNote("x.md")
	require.NoError(t, err)
	intake, err := IntakeNote("x.md", KindUnset, "adhoc", "2026-11-03", "body", []string{"t"})
	require.NoError(t, err)
	extracted, err := ExtractedNote("x.md", KindIdea, "adhoc", "2026-11-03", "body", []string{"[[O]]"})
	require.NoError(t, err)
	imported, err := ImportedNote([]byte("---\nz: 1\nrelated: []\ntags: [a]\n---\nb"), "x.md", "2026-11-03")
	require.NoError(t, err)
	want := append(append([]string(nil), standardKeys...), "related")
	for name, data := range map[string][]byte{"blank": blank, "intake": intake, "extracted": extracted} {
		assert.Equal(t, want, keysOf(t, data), name)
	}
	assert.Equal(t, append(append([]string(nil), standardKeys...), "z", "related"), keysOf(t, imported), "imported")
}

func writeNote(t *testing.T, root, rel, content string) string {
	t.Helper()
	p := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestConformRewritesOnlyWhatIsNotStandard(t *testing.T) {
	root := t.TempDir()
	s := NewMarkdown(root)
	p := writeNote(t, root, "inbox/a.md", "---\ntags: [a]\nuid: u1\n---\nbody text\n")

	changed, err := s.Conform("inbox/a.md")
	require.NoError(t, err)
	assert.True(t, changed)
	data, _ := os.ReadFile(p)
	assert.Equal(t, append(append([]string(nil), standardKeys...), "related"), keysOf(t, data))
	n := ParseNote("a.md", data)
	assert.Equal(t, "body text\n", n.Body)
	assert.Equal(t, "u1", n.Frontmatter.Get("uid").Value)
	assert.Equal(t, "a", n.Frontmatter.Get("title").Value, "the title is the file name")

	// A second look changes nothing, not even the modification time.
	old := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(p, old, old))
	changed, err = s.Conform("inbox/a.md")
	require.NoError(t, err)
	assert.False(t, changed)
	info, _ := os.Stat(p)
	assert.True(t, info.ModTime().Equal(old.Truncate(0)) || info.ModTime().Unix() == old.Unix(), "an untouched note is not rewritten")
}

func TestConformAddsABlockToANoteWithNone(t *testing.T) {
	root := t.TempDir()
	p := writeNote(t, root, "inbox/plain.md", "# Heading\n\ntext\n")
	changed, err := NewMarkdown(root).Conform("inbox/plain.md")
	require.NoError(t, err)
	assert.True(t, changed)
	data, _ := os.ReadFile(p)
	assert.Equal(t, "# Heading\n\ntext\n", ParseNote("plain.md", data).Body)
	assert.Equal(t, append(append([]string(nil), standardKeys...), "related"), keysOf(t, data))
}

func TestConformLeavesAnInvalidBlockAlone(t *testing.T) {
	root := t.TempDir()
	in := "---\nkey: [oops\n---\nbody\n"
	p := writeNote(t, root, "inbox/bad.md", in)
	changed, err := NewMarkdown(root).Conform("inbox/bad.md")
	require.NoError(t, err)
	assert.False(t, changed)
	data, _ := os.ReadFile(p)
	assert.Equal(t, in, string(data))
}

func TestConformFixesATitleThatDisagreesWithTheName(t *testing.T) {
	root := t.TempDir()
	p := writeNote(t, root, "inbox/real.md", "---\nuid: u\ntitle: Old\nkind: \"\"\nsource: \"\"\ndate: \"\"\ntags: []\npeople: []\nrelated: []\n---\n")
	changed, err := NewMarkdown(root).Conform("inbox/real.md")
	require.NoError(t, err)
	assert.True(t, changed)
	data, _ := os.ReadFile(p)
	assert.Equal(t, "real", ParseNote("real.md", data).Frontmatter.Get("title").Value)
}
