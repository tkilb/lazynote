package store

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetTitle(t *testing.T) {
	t.Run("adds a missing title keeping everything else", func(t *testing.T) {
		in := "---\nuid: u1\n# keep me\ntags: [a, b]\nkind: fact\n---\n\n# Body\nline\n"
		out, err := SetTitle([]byte(in), "Name")
		require.NoError(t, err)
		n := ParseNote("Name.md", out)
		require.NoError(t, n.FrontmatterErr)
		assert.Equal(t, "Name", n.Frontmatter.Get("title").Value)
		assert.Equal(t, "u1", n.Frontmatter.Get("uid").Value)
		assert.Equal(t, KindFact, n.Kind)
		assert.Contains(t, string(out), "# keep me")
		assert.Equal(t, "\n# Body\nline\n", n.Body, "the body is byte for byte unchanged")
		var keys []string
		for _, p := range n.Frontmatter.Properties() {
			keys = append(keys, p.Key)
		}
		assert.Equal(t, []string{"uid", "tags", "kind", "title"}, keys)
	})

	t.Run("replaces a different title in place", func(t *testing.T) {
		out, err := SetTitle([]byte("---\ntitle: Old\nuid: u\n---\nbody"), "New")
		require.NoError(t, err)
		n := ParseNote("x.md", out)
		assert.Equal(t, "New", n.Frontmatter.Get("title").Value)
		assert.Equal(t, "title", n.Frontmatter.Properties()[0].Key)
		assert.Equal(t, "body", n.Body)
	})

	t.Run("an equal title leaves the bytes untouched", func(t *testing.T) {
		in := []byte("---\nuid:   spaced\ntitle:   Same\n---\nbody\n")
		out, err := SetTitle(in, "Same")
		require.NoError(t, err)
		assert.Equal(t, string(in), string(out), "no reformatting when nothing changes")
	})

	t.Run("a note without frontmatter gets a block with only the title", func(t *testing.T) {
		out, err := SetTitle([]byte("# Just text\n"), "Name")
		require.NoError(t, err)
		n := ParseNote("Name.md", out)
		require.NotNil(t, n.Frontmatter)
		assert.Equal(t, "Name", n.Frontmatter.Get("title").Value)
		assert.Equal(t, "# Just text\n", n.Body)
	})

	t.Run("a non-string title is replaced by a string", func(t *testing.T) {
		out, err := SetTitle([]byte("---\ntitle: 2026\n---\n"), "2026")
		require.NoError(t, err)
		v := ParseNote("x.md", out).Frontmatter.Get("title")
		assert.Equal(t, "!!str", v.Tag)
		assert.Equal(t, "2026", v.Value)
	})

	t.Run("invalid frontmatter is refused", func(t *testing.T) {
		_, err := SetTitle([]byte("---\nkind: [oops\n---\nbody"), "x")
		assert.ErrorIs(t, err, ErrFrontmatterInvalid)
		_, err = SetTitle([]byte("---\n- list\n---\nbody"), "x")
		assert.ErrorIs(t, err, ErrFrontmatterInvalid)
	})

	t.Run("a byte order mark is kept", func(t *testing.T) {
		out, err := SetTitle([]byte("\xef\xbb\xbf---\nuid: u\n---\nbody"), "N")
		require.NoError(t, err)
		assert.True(t, len(out) > 3 && string(out[:3]) == "\xef\xbb\xbf")
		assert.Equal(t, "N", ParseNote("N.md", out).Frontmatter.Get("title").Value)
	})

	t.Run("an unterminated block is plain text, so a block is added", func(t *testing.T) {
		out, err := SetTitle([]byte("---\nnot closed\n"), "N")
		require.NoError(t, err)
		n := ParseNote("N.md", out)
		assert.Equal(t, "N", n.Frontmatter.Get("title").Value)
		assert.Equal(t, "---\nnot closed\n", n.Body)
	})
}

func TestCopyUpdatesTheTitleOfARenamedCopy(t *testing.T) {
	root, _ := newRoot(t)
	src := filepath.Join(root, "inbox", "a.md")
	put(t, src, "---\nuid: u1\ntitle: a\ntags: [x]\n---\nbody\n")
	s := NewMarkdown(root)

	_, err := s.Copy("inbox/a.md", "inbox/a_1.md")
	require.NoError(t, err)

	n := ParseNote("a_1.md", []byte(read(t, filepath.Join(root, "inbox", "a_1.md"))))
	assert.Equal(t, "a_1", n.Frontmatter.Get("title").Value, "the copy's title follows its new name")
	assert.Equal(t, "u1", n.Frontmatter.Get("uid").Value)
	assert.Equal(t, "body\n", n.Body)
	assert.Equal(t, "---\nuid: u1\ntitle: a\ntags: [x]\n---\nbody\n", read(t, src), "the source is untouched")
}

func TestCopyKeepsBytesWhenTheNameIsUnchanged(t *testing.T) {
	root, _ := newRoot(t)
	content := "---\nuid:   u1\ntitle:   other\n---\nbody\n"
	put(t, filepath.Join(root, "inbox", "a.md"), content)

	_, err := NewMarkdown(root).Copy("inbox/a.md", "areas/a.md")
	require.NoError(t, err)
	assert.Equal(t, content, read(t, filepath.Join(root, "areas", "a.md")), "same name: a plain byte-for-byte copy")
}

func TestCopyLeavesInvalidFrontmatterAlone(t *testing.T) {
	root, _ := newRoot(t)
	content := "---\nkind: [oops\n---\nbody\n"
	put(t, filepath.Join(root, "inbox", "a.md"), content)

	_, err := NewMarkdown(root).Copy("inbox/a.md", "inbox/a_1.md")
	require.NoError(t, err)
	assert.Equal(t, content, read(t, filepath.Join(root, "inbox", "a_1.md")))
}

func TestCopyOfAFolderRetitlesNothingInside(t *testing.T) {
	root, _ := newRoot(t)
	content := "---\ntitle: n\n---\nx\n"
	put(t, filepath.Join(root, "projects", "p", "n.md"), content)

	_, err := NewMarkdown(root).Copy("projects/p", "projects/p_1")
	require.NoError(t, err)
	assert.Equal(t, content, read(t, filepath.Join(root, "projects", "p_1", "n.md")), "inner notes keep their names, so their titles stay")
}

func TestCopyUndoRestoresNothingBrokenAfterRetitle(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "a.md"), "---\ntitle: a\n---\nb\n")
	act, err := NewMarkdown(root).Copy("inbox/a.md", "inbox/a_1.md")
	require.NoError(t, err)

	require.NoError(t, act.Undo())
	assert.False(t, exists(filepath.Join(root, "inbox", "a_1.md")))
	require.NoError(t, act.Redo())
	got := ParseNote("a_1.md", []byte(read(t, filepath.Join(root, "inbox", "a_1.md"))))
	assert.Equal(t, "a_1", got.Frontmatter.Get("title").Value, "redo brings back the retitled copy")
}
