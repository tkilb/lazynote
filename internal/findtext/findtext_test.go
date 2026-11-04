package findtext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkilb/lazynote/internal/store"
)

func put(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
}

func TestLinesMatchWhatContentShows(t *testing.T) {
	root := t.TempDir()
	// Content trims leading and trailing newlines, so line 0 is "first".
	put(t, root, "inbox/a.md", "---\nuid: u\ntitle: a\n---\n\n\nfirst\n\n  second  \nthird\n\n")
	res := Lines(store.NewMarkdown(root), []string{"inbox/a.md"})

	assert.False(t, res.Truncated)
	assert.Equal(t, []Line{
		{Rel: "inbox/a.md", Index: 0, Text: "first"},
		{Rel: "inbox/a.md", Index: 2, Text: "second"},
		{Rel: "inbox/a.md", Index: 3, Text: "third"},
	}, res.Lines, "blank lines are skipped but the index counts them")
}

func TestLinesSkipFrontmatterAndControlCharacters(t *testing.T) {
	root := t.TempDir()
	put(t, root, "inbox/a.md", "---\nuid: secret-uid\n---\nclean\x1b[31m text\n")
	res := Lines(store.NewMarkdown(root), []string{"inbox/a.md"})
	require.Len(t, res.Lines, 1)
	assert.Equal(t, "clean[31m text", res.Lines[0].Text)
}

func TestLinesCutLongLinesAtACharacterBoundary(t *testing.T) {
	root := t.TempDir()
	put(t, root, "inbox/a.md", strings.Repeat("é", maxLineRunes*2)+"\n")
	res := Lines(store.NewMarkdown(root), []string{"inbox/a.md"})
	require.Len(t, res.Lines, 1)
	assert.Equal(t, maxLineRunes, len([]rune(res.Lines[0].Text)))
}

func TestLinesStopAtTheLimit(t *testing.T) {
	root := t.TempDir()
	put(t, root, "inbox/a.md", strings.Repeat("x\n", maxLines+5))
	put(t, root, "inbox/b.md", "later\n")
	res := Lines(store.NewMarkdown(root), []string{"inbox/a.md", "inbox/b.md"})
	assert.True(t, res.Truncated)
	assert.Len(t, res.Lines, maxLines)
}

func TestLinesSkipUnreadableNotes(t *testing.T) {
	root := t.TempDir()
	put(t, root, "inbox/ok.md", "fine\n")
	res := Lines(store.NewMarkdown(root), []string{"inbox/missing.md", "inbox/ok.md"})
	require.Len(t, res.Lines, 1)
	assert.Equal(t, "fine", res.Lines[0].Text)
}
