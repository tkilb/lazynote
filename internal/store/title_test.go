package store

import (
	"bytes"
	"fmt"
	"os"
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
		assert.Equal(t, []string{"uid", "title", "kind", "source", "date", "tags", "people", "related"}, keys)
	})

	t.Run("replaces a different title in place", func(t *testing.T) {
		out, err := SetTitle([]byte("---\ntitle: Old\nuid: u\n---\nbody"), "New")
		require.NoError(t, err)
		n := ParseNote("x.md", out)
		assert.Equal(t, "New", n.Frontmatter.Get("title").Value)
		assert.Equal(t, "uid", n.Frontmatter.Properties()[0].Key, "uid first")
		assert.Equal(t, "title", n.Frontmatter.Properties()[1].Key)
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

	_, _, err := s.Copy("inbox/a.md", "inbox/a_1.md")
	require.NoError(t, err)

	n := ParseNote("a_1.md", []byte(read(t, filepath.Join(root, "inbox", "a_1.md"))))
	assert.Equal(t, "a_1", n.Frontmatter.Get("title").Value, "the copy's title follows its new name")
	assert.NotEqual(t, "u1", n.Frontmatter.Get("uid").Value, "the copy gets a fresh uid")
	assert.Equal(t, "body\n", n.Body)
	assert.Equal(t, "---\nuid: u1\ntitle: a\ntags: [x]\n---\nbody\n", read(t, src), "the source is untouched")
}

func TestCopyWithTheSameNameStillGetsAFreshUIDAndTitle(t *testing.T) {
	root, _ := newRoot(t)
	content := "---\nuid:   u1\ntitle:   other\n---\nbody\n"
	put(t, filepath.Join(root, "inbox", "a.md"), content)

	_, _, err := NewMarkdown(root).Copy("inbox/a.md", "areas/a.md")
	require.NoError(t, err)
	n := ParseNote("a.md", []byte(read(t, filepath.Join(root, "areas", "a.md"))))
	assert.NotEqual(t, "u1", n.Frontmatter.Get("uid").Value)
	assert.Equal(t, "a", n.Frontmatter.Get("title").Value)
	assert.Equal(t, content, read(t, filepath.Join(root, "inbox", "a.md")), "the source is untouched")
}

func TestCopyLeavesInvalidFrontmatterAlone(t *testing.T) {
	root, _ := newRoot(t)
	content := "---\nkind: [oops\n---\nbody\n"
	put(t, filepath.Join(root, "inbox", "a.md"), content)

	_, res, err := NewMarkdown(root).Copy("inbox/a.md", "inbox/a_1.md")
	require.NoError(t, err)
	assert.Equal(t, content, read(t, filepath.Join(root, "inbox", "a_1.md")))
	assert.Equal(t, 1, res.KeptUID, "the caller is told the copy kept its source's uid")
}

func TestCopyOfAFolderRenewsEveryNoteInside(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "projects", "p", "n.md"), "---\nuid: u1\ntitle: n\n---\nx\n")
	put(t, filepath.Join(root, "projects", "p", "sub", "m.MD"), "plain\n")
	put(t, filepath.Join(root, "projects", "p", "pic.png"), "png")

	_, _, err := NewMarkdown(root).Copy("projects/p", "projects/p_1")
	require.NoError(t, err)
	n := ParseNote("n.md", []byte(read(t, filepath.Join(root, "projects", "p_1", "n.md"))))
	assert.NotEqual(t, "u1", n.Frontmatter.Get("uid").Value)
	assert.Equal(t, "n", n.Frontmatter.Get("title").Value, "inner notes keep their names")
	m := ParseNote("m.MD", []byte(read(t, filepath.Join(root, "projects", "p_1", "sub", "m.MD"))))
	assert.Equal(t, "m", m.Frontmatter.Get("title").Value, "a note with no frontmatter gets a block")
	assert.NotEmpty(t, m.Frontmatter.Get("uid").Value)
	assert.Equal(t, "plain\n", m.Body)
	assert.Equal(t, "png", read(t, filepath.Join(root, "projects", "p_1", "pic.png")), "other files are verbatim")
}

func TestCopiesNeverShareAUID(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "a.md"), "---\nuid: u1\n---\nb\n")
	s := NewMarkdown(root)
	seen := map[string]bool{"u1": true}
	for i := 0; i < 50; i++ {
		dst := fmt.Sprintf("areas/c%d.md", i)
		_, _, err := s.Copy("inbox/a.md", dst)
		require.NoError(t, err)
		uid := ParseNote("c.md", []byte(read(t, filepath.Join(root, dst)))).Frontmatter.Get("uid").Value
		require.False(t, seen[uid], "uid %q repeated", uid)
		seen[uid] = true
	}
}

func TestCopyUndoAndRedoKeepTheSameNewUID(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "a.md"), "---\nuid: u1\n---\nb\n")
	act, _, err := NewMarkdown(root).Copy("inbox/a.md", "inbox/a_1.md")
	require.NoError(t, err)
	dst := filepath.Join(root, "inbox", "a_1.md")
	before := read(t, dst)

	require.NoError(t, act.Undo())
	require.NoError(t, act.Redo())
	assert.Equal(t, before, read(t, dst))
}

func TestCopyOfAReadOnlyNoteStillGetsAFreshUID(t *testing.T) {
	root, _ := newRoot(t)
	src := filepath.Join(root, "inbox", "a.md")
	put(t, src, "---\nuid: u1\n---\nb\n")
	require.NoError(t, os.Chmod(src, 0o444))

	_, res, err := NewMarkdown(root).Copy("inbox/a.md", "areas/a.md")
	require.NoError(t, err)
	assert.Zero(t, res.KeptUID)
	dst := filepath.Join(root, "areas", "a.md")
	assert.NotEqual(t, "u1", ParseNote("a.md", []byte(read(t, dst))).Frontmatter.Get("uid").Value)
	info, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o444), info.Mode().Perm(), "the source mode is kept")
}

func TestRenewIdentityKeepsTheBOM(t *testing.T) {
	for _, in := range []string{"\xef\xbb\xbf---\nuid: u1\n---\nb\n", "\xef\xbb\xbfplain\n"} {
		out, err := RenewIdentity([]byte(in), "n")
		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(out, utf8BOM), "BOM kept for %q", in)
		n := ParseNote("n.md", out)
		assert.Equal(t, "n", n.Frontmatter.Get("title").Value)
		assert.NotEqual(t, "u1", n.Frontmatter.Get("uid").Value)
	}
}

func TestCopyOfAFolderCountsOnlyTheNotesItCouldNotRewrite(t *testing.T) {
	root, _ := newRoot(t)
	bad := "---\nkind: [oops\n---\nbody\n"
	put(t, filepath.Join(root, "projects", "p", "bad.md"), bad)
	put(t, filepath.Join(root, "projects", "p", "ok.md"), "---\nuid: u1\n---\nx\n")
	good := read(t, filepath.Join(root, "projects", "p", "ok.md"))

	act, res, err := NewMarkdown(root).Copy("projects/p", "projects/p_1")
	require.NoError(t, err)
	assert.Equal(t, 1, res.KeptUID)
	assert.Equal(t, bad, read(t, filepath.Join(root, "projects", "p_1", "bad.md")))
	okCopy := filepath.Join(root, "projects", "p_1", "ok.md")
	assert.NotEqual(t, "u1", ParseNote("ok.md", []byte(read(t, okCopy))).Frontmatter.Get("uid").Value)

	before := read(t, okCopy)
	require.NoError(t, act.Undo())
	require.NoError(t, act.Redo())
	assert.Equal(t, before, read(t, okCopy), "redo of a folder copy keeps the new uids")
	assert.Equal(t, good, read(t, filepath.Join(root, "projects", "p", "ok.md")), "the source folder is untouched")
	assert.Equal(t, bad, read(t, filepath.Join(root, "projects", "p", "bad.md")))
}
