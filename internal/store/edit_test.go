package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setTags(fm *Frontmatter) { fm.SetStringList("tags", []string{"a", "b"}) }

func TestEditChangesOnlyTheNamedPropertyAndAddsIdentity(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	put(t, p, "---\nkind: fact\n# keep me\nowner: me\n---\nbody\n")

	_, err := NewMarkdown(root).Edit("inbox/n.md", setTags)
	require.NoError(t, err)

	n := ParseNote("n.md", []byte(read(t, p)))
	require.NoError(t, n.FrontmatterErr)
	assert.Equal(t, "fact", n.Frontmatter.Get("kind").Value)
	assert.Equal(t, "me", n.Frontmatter.Get("owner").Value)
	assert.Len(t, n.Frontmatter.Get("tags").Content, 2)
	assert.NotEmpty(t, n.Frontmatter.Get("uid").Value, "a note without a uid gets one on its first write")
	assert.Equal(t, "n", n.Frontmatter.Get("title").Value)
	assert.Equal(t, "body\n", n.Body)
	keys := []string{}
	for _, pr := range n.Frontmatter.Properties() {
		keys = append(keys, pr.Key)
	}
	assert.Equal(t, []string{"uid", "title", "kind", "source", "date", "tags", "people", "owner", "related"}, keys, "standard order, other properties kept")
}

func TestEditKeepsAnExistingUID(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	put(t, p, "---\nuid: keep-me\ntitle: n\n---\n")
	_, err := NewMarkdown(root).Edit("inbox/n.md", setTags)
	require.NoError(t, err)
	assert.Equal(t, "keep-me", ParseNote("n.md", []byte(read(t, p))).Frontmatter.Get("uid").Value)
}

func TestEditCreatesAPropertiesBlockAndKeepsTheBOM(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "plain.md"), "just text\n")
	put(t, filepath.Join(root, "inbox", "bom.md"), "\xef\xbb\xbf---\nkind: fact\n---\nx\n")
	s := NewMarkdown(root)
	_, err := s.Edit("inbox/plain.md", setTags)
	require.NoError(t, err)
	_, err = s.Edit("inbox/bom.md", setTags)
	require.NoError(t, err)

	plain := ParseNote("plain.md", []byte(read(t, filepath.Join(root, "inbox", "plain.md"))))
	assert.Equal(t, "just text\n", plain.Body)
	assert.NotNil(t, plain.Frontmatter.Get("tags"))
	assert.True(t, len(read(t, filepath.Join(root, "inbox", "bom.md"))) > 3 && read(t, filepath.Join(root, "inbox", "bom.md"))[:3] == "\xef\xbb\xbf")
}

func TestEditRefusals(t *testing.T) {
	root, _ := newRoot(t)
	invalid := "---\nkind: [oops\n---\nbody\n"
	put(t, filepath.Join(root, "inbox", "bad.md"), invalid)
	put(t, filepath.Join(root, "inbox", "a.txt"), "x")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "inbox", "dir.md"), 0o755))
	big := filepath.Join(root, "inbox", "big.md")
	put(t, big, "")
	require.NoError(t, os.Truncate(big, maxNoteBytes+1))
	s := NewMarkdown(root)

	_, err := s.Edit("inbox/bad.md", setTags)
	assert.ErrorIs(t, err, ErrFrontmatterInvalid)
	assert.Equal(t, invalid, read(t, filepath.Join(root, "inbox", "bad.md")), "an invalid block is left exactly as it is")
	_, err = s.Edit("inbox/missing.md", setTags)
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = s.Edit("inbox/a.txt", setTags)
	assert.ErrorIs(t, err, ErrUnsupported)
	_, err = s.Edit("inbox/dir.md", setTags)
	assert.ErrorIs(t, err, ErrUnsupported)
	_, err = s.Edit("inbox/big.md", setTags)
	assert.ErrorIs(t, err, ErrTooLarge)
	for _, bad := range []string{"../x.md", "/etc/x.md", ""} {
		_, err = s.Edit(bad, setTags)
		assert.Error(t, err, bad)
	}
}

func TestEditUndoRestoresTheExactBytesAndRedoReapplies(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	orig := "---\nkind:    fact\n---\nbody   \n"
	put(t, p, orig)
	act, err := NewMarkdown(root).Edit("inbox/n.md", setTags)
	require.NoError(t, err)
	edited := read(t, p)

	require.NoError(t, act.Undo())
	assert.Equal(t, orig, read(t, p), "byte for byte, formatting included")
	require.NoError(t, act.Redo())
	assert.Equal(t, edited, read(t, p), "redo brings back the same uid")
}

func TestEditUndoAndRedoRefuseAfterAnOutsideEdit(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	put(t, p, "---\nkind: fact\n---\n")
	act, err := NewMarkdown(root).Edit("inbox/n.md", setTags)
	require.NoError(t, err)

	bump(t, p, "someone else was here\n")
	assert.ErrorIs(t, act.Undo(), ErrChanged)
	assert.Equal(t, "someone else was here\n", read(t, p), "their edit is kept")

	require.NoError(t, os.Remove(p))
	assert.ErrorIs(t, act.Undo(), ErrChanged)
}

func TestStackedEditsOfOneNoteUndoAndRedoInOrder(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	orig := "---\nkind: fact\n---\nbody\n"
	put(t, p, orig)
	s := NewMarkdown(root)

	first, err := s.Edit("inbox/n.md", func(fm *Frontmatter) { fm.SetStringList("tags", []string{"a"}) })
	require.NoError(t, err)
	afterFirst := read(t, p)
	second, err := s.Edit("inbox/n.md", func(fm *Frontmatter) { fm.SetStringList("tags", []string{"a", "b"}) })
	require.NoError(t, err)
	afterSecond := read(t, p)

	require.NoError(t, second.Undo())
	assert.Equal(t, afterFirst, read(t, p))
	require.NoError(t, first.Undo())
	assert.Equal(t, orig, read(t, p), "the earlier edit still undoes after the later one was undone")
	require.NoError(t, first.Redo())
	assert.Equal(t, afterFirst, read(t, p))
	require.NoError(t, second.Redo())
	assert.Equal(t, afterSecond, read(t, p))
}

func TestEditChangesTheModificationTimeSoWatchersSeeIt(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	put(t, p, "---\nkind: fact\n---\n")
	old := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(p, old, old))

	_, err := NewMarkdown(root).Edit("inbox/n.md", setTags)
	require.NoError(t, err)
	info, err := os.Stat(p)
	require.NoError(t, err)
	assert.True(t, info.ModTime().After(old.Add(time.Minute)), "the edit must not keep the old time")
}
