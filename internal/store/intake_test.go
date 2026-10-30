package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntakeNote(t *testing.T) {
	data, err := IntakeNote("call mum.md", KindIdea, "quick", "2026-10-30", "call mum\n", nil)
	require.NoError(t, err)
	n := ParseNote("call mum.md", data)
	require.NoError(t, n.FrontmatterErr)
	assert.Equal(t, KindIdea, n.Kind)
	assert.Equal(t, "quick", n.Frontmatter.Get("source").Value)
	assert.Equal(t, "2026-10-30", n.Frontmatter.Get("date").Value)
	assert.Equal(t, "call mum", n.Frontmatter.Get("title").Value)
	assert.NotEmpty(t, n.Frontmatter.Get("uid").Value)
	assert.NotNil(t, n.Frontmatter.Get("tags"), "every intake note has a tags property")
	assert.Empty(t, n.Frontmatter.Get("tags").Content)
	assert.Equal(t, "call mum\n", n.Body)
}

func TestIntakeNoteQuotesTextSoItStaysAString(t *testing.T) {
	data, err := IntakeNote("n.md", KindFact, "2026-01-01", "2026-10-30", "x\n", nil)
	require.NoError(t, err)
	n := ParseNote("n.md", data)
	assert.Equal(t, "2026-01-01", n.Frontmatter.Get("source").Value)
	assert.Equal(t, "!!str", n.Frontmatter.Get("source").Tag, "a date-looking source is not turned into a timestamp")
	assert.Equal(t, "!!timestamp", n.Frontmatter.Get("date").Tag, "a date is written unquoted, so Obsidian reads it as a date")
}

func TestImportedNote(t *testing.T) {
	tests := []struct {
		name, in string
		wantSrc  string
		wantDate string
		wantKind Kind
	}{
		{"adds source and date", "---\nkind: fact\n---\nbody\n", "import", "2026-10-30", KindFact},
		{"keeps an existing source and date", "---\nsource: web\ndate: 2020-01-01\n---\nb\n", "web", "2020-01-01", KindUnset},
		{"no frontmatter", "just text\n", "import", "2026-10-30", KindUnset},
		{"blank source counts as absent", "---\nsource: \"\"\n---\n", "import", "2026-10-30", KindUnset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := ImportedNote([]byte(tt.in), "New.md", "2026-10-30")
			require.NoError(t, err)
			n := ParseNote("New.md", out)
			require.NoError(t, n.FrontmatterErr)
			assert.Equal(t, tt.wantSrc, n.Frontmatter.Get("source").Value)
			assert.Equal(t, tt.wantDate, n.Frontmatter.Get("date").Value)
			assert.Equal(t, tt.wantKind, n.Kind, "kind is left as it is")
			assert.Equal(t, "New", n.Frontmatter.Get("title").Value)
			assert.NotEmpty(t, n.Frontmatter.Get("uid").Value)
		})
	}
}

func TestImportedNoteGetsAFreshUIDAndKeepsTheRest(t *testing.T) {
	in := "---\nuid: old\ntitle: Old\ntags: [a]\nowner: me\n---\nbody here\n"
	out, err := ImportedNote([]byte(in), "New.md", "2026-10-30")
	require.NoError(t, err)
	n := ParseNote("New.md", out)
	assert.NotEqual(t, "old", n.Frontmatter.Get("uid").Value)
	assert.Equal(t, "me", n.Frontmatter.Get("owner").Value)
	assert.NotNil(t, n.Frontmatter.Get("tags"))
	assert.Equal(t, "body here\n", n.Body)
}

func TestImportedNoteRefusesInvalidFrontmatter(t *testing.T) {
	_, err := ImportedNote([]byte("---\nkind: [oops\n---\nbody\n"), "New.md", "2026-10-30")
	assert.ErrorIs(t, err, ErrFrontmatterInvalid)
}

func TestIntakeNoteTagsAndUnsetKind(t *testing.T) {
	data, err := IntakeNote("n.md", KindUnset, "quick", "2026-10-30", "", []string{"work", "ideas/new"})
	require.NoError(t, err)
	n := ParseNote("n.md", data)
	require.NoError(t, n.FrontmatterErr)
	assert.Nil(t, n.Frontmatter.Get("kind"), "an unset kind writes no kind property")
	assert.Equal(t, KindUnset, n.Kind)
	tags := n.Frontmatter.Get("tags")
	require.NotNil(t, tags)
	require.Len(t, tags.Content, 2)
	assert.Equal(t, "work", tags.Content[0].Value)
	assert.Equal(t, "ideas/new", tags.Content[1].Value)
	assert.Empty(t, n.Body)
}
