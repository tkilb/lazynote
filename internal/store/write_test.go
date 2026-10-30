package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkilb/lazynote/internal/undo"
)

func put(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// bump rewrites path with new content and a later mtime, like an outside edit.
func bump(t *testing.T, path, content string) {
	t.Helper()
	put(t, path, content)
	later := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(path, later, later))
}

// trashed lists the files stored under .lazynote/trash.
func trashed(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	_ = filepath.WalkDir(filepath.Join(root, ".lazynote", "trash"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	return files
}

func TestCreate(t *testing.T) {
	root, _ := newRoot(t)
	s := NewMarkdown(root)

	act, err := s.Create("inbox/deep/new.md", []byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", read(t, filepath.Join(root, "inbox", "deep", "new.md")))

	require.NoError(t, act.Undo())
	assert.False(t, exists(filepath.Join(root, "inbox")), "folders the create made are removed again")
	files := trashed(t, root)
	require.Len(t, files, 1)
	assert.Equal(t, "hello", read(t, files[0]), "undo moves the file to the trash, it does not delete it")

	require.NoError(t, act.Redo())
	assert.Equal(t, "hello", read(t, filepath.Join(root, "inbox", "deep", "new.md")))
	assert.Empty(t, trashed(t, root))
	assert.False(t, exists(filepath.Join(root, ".lazynote")), "empty trash folders are cleaned up")
}

func TestCreateNeverOverwrites(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "projects", "a.md"), "keep")

	_, err := NewMarkdown(root).Create("projects/a.md", []byte("clobber"))
	assert.ErrorIs(t, err, ErrExists)
	assert.Equal(t, "keep", read(t, filepath.Join(root, "projects", "a.md")))
}

func TestCreateUndoRefusesAfterOutsideEdit(t *testing.T) {
	root, _ := newRoot(t)
	act, err := NewMarkdown(root).Create("inbox/n.md", []byte("v1"))
	require.NoError(t, err)

	bump(t, filepath.Join(root, "inbox", "n.md"), "v2 edited elsewhere")

	assert.ErrorIs(t, act.Undo(), ErrChanged)
	assert.Equal(t, "v2 edited elsewhere", read(t, filepath.Join(root, "inbox", "n.md")))
	assert.Empty(t, trashed(t, root))
}

func TestMakeDir(t *testing.T) {
	root, _ := newRoot(t)
	s := NewMarkdown(root)

	act, err := s.MakeDir("resources/go/errors")
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(root, "resources", "go", "errors"))

	require.NoError(t, act.Undo())
	assert.False(t, exists(filepath.Join(root, "resources")), "created parents go too")

	require.NoError(t, act.Redo())
	assert.DirExists(t, filepath.Join(root, "resources", "go", "errors"))

	_, err = s.MakeDir("resources/go")
	assert.ErrorIs(t, err, ErrExists)
}

func TestMakeDirUndoRefusesWhenNotEmpty(t *testing.T) {
	root, _ := newRoot(t)
	act, err := NewMarkdown(root).MakeDir("areas/new")
	require.NoError(t, err)
	put(t, filepath.Join(root, "areas", "new", "x.md"), "x")

	assert.ErrorIs(t, act.Undo(), ErrChanged)
	assert.FileExists(t, filepath.Join(root, "areas", "new", "x.md"))
}

func TestMoveFile(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "n.md"), "body")
	s := NewMarkdown(root)

	act, err := s.Move("inbox/n.md", "archives/old/n.md")
	require.NoError(t, err)
	assert.False(t, exists(filepath.Join(root, "inbox", "n.md")))
	assert.Equal(t, "body", read(t, filepath.Join(root, "archives", "old", "n.md")))

	require.NoError(t, act.Undo())
	assert.Equal(t, "body", read(t, filepath.Join(root, "inbox", "n.md")))
	assert.False(t, exists(filepath.Join(root, "archives")), "folders created by the move are removed on undo")

	require.NoError(t, act.Redo())
	assert.Equal(t, "body", read(t, filepath.Join(root, "archives", "old", "n.md")))
}

func TestMoveDirectoryKeepsContents(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "projects", "p", "a.md"), "body-a")
	put(t, filepath.Join(root, "projects", "p", "sub", "pic.png"), "png")
	s := NewMarkdown(root)

	act, err := s.Move("projects/p", "archives/p")
	require.NoError(t, err)
	assert.Equal(t, "png", read(t, filepath.Join(root, "archives", "p", "sub", "pic.png")))

	require.NoError(t, act.Undo())
	assert.Equal(t, "body-a", read(t, filepath.Join(root, "projects", "p", "a.md")))
}

func TestMoveRejections(t *testing.T) {
	root, tmp := newRoot(t)
	put(t, filepath.Join(root, "inbox", "a.md"), "a")
	put(t, filepath.Join(root, "inbox", "b.md"), "b")
	put(t, filepath.Join(root, "projects", "p", "x.md"), "x")
	require.NoError(t, os.Symlink(filepath.Join(tmp, "outside"), filepath.Join(root, "out")))
	s := NewMarkdown(root)

	tests := []struct {
		name     string
		src, dst string
		want     error
	}{
		{"destination exists", "inbox/a.md", "inbox/b.md", ErrExists},
		{"onto itself", "inbox/a.md", "inbox/a.md", ErrExists},
		{"missing source", "inbox/gone.md", "inbox/c.md", ErrNotFound},
		{"folder into itself", "projects/p", "projects/p/inner", ErrIntoItself},
		{"folder onto itself", "projects/p", "projects/p", ErrExists},
		{"escaping destination", "inbox/a.md", "../outside/a.md", ErrParentSegment},
		{"symlink destination outside", "inbox/a.md", "out/a.md", ErrEscapesRoot},
		{"escaping source", "../outside/x.md", "inbox/c.md", ErrParentSegment},
		{"absolute source", "/etc/hosts", "inbox/c.md", ErrAbsolutePath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Move(tt.src, tt.dst)
			assert.ErrorIs(t, err, tt.want)
		})
	}
	assert.Equal(t, "a", read(t, filepath.Join(root, "inbox", "a.md")))
	assert.Equal(t, "b", read(t, filepath.Join(root, "inbox", "b.md")))
	assert.Equal(t, "x", read(t, filepath.Join(root, "projects", "p", "x.md")))
}

func TestMoveUndoRefusals(t *testing.T) {
	t.Run("original path was reused", func(t *testing.T) {
		root, _ := newRoot(t)
		put(t, filepath.Join(root, "inbox", "n.md"), "orig")
		act, err := NewMarkdown(root).Move("inbox/n.md", "areas/n.md")
		require.NoError(t, err)
		put(t, filepath.Join(root, "inbox", "n.md"), "someone else")

		assert.ErrorIs(t, act.Undo(), ErrExists)
		assert.Equal(t, "someone else", read(t, filepath.Join(root, "inbox", "n.md")))
		assert.Equal(t, "orig", read(t, filepath.Join(root, "areas", "n.md")))
	})
	t.Run("moved file was edited", func(t *testing.T) {
		root, _ := newRoot(t)
		put(t, filepath.Join(root, "inbox", "n.md"), "orig")
		act, err := NewMarkdown(root).Move("inbox/n.md", "areas/n.md")
		require.NoError(t, err)
		bump(t, filepath.Join(root, "areas", "n.md"), "edited")

		assert.ErrorIs(t, act.Undo(), ErrChanged)
		assert.Equal(t, "edited", read(t, filepath.Join(root, "areas", "n.md")))
	})
	t.Run("moved file was removed", func(t *testing.T) {
		root, _ := newRoot(t)
		put(t, filepath.Join(root, "inbox", "n.md"), "orig")
		act, err := NewMarkdown(root).Move("inbox/n.md", "areas/n.md")
		require.NoError(t, err)
		require.NoError(t, os.Remove(filepath.Join(root, "areas", "n.md")))

		assert.ErrorIs(t, act.Undo(), ErrChanged)
	})
	t.Run("redo refuses after an edit following undo", func(t *testing.T) {
		root, _ := newRoot(t)
		put(t, filepath.Join(root, "inbox", "n.md"), "orig")
		act, err := NewMarkdown(root).Move("inbox/n.md", "areas/n.md")
		require.NoError(t, err)
		require.NoError(t, act.Undo())
		bump(t, filepath.Join(root, "inbox", "n.md"), "edited after undo")

		assert.ErrorIs(t, act.Redo(), ErrChanged)
		assert.Equal(t, "edited after undo", read(t, filepath.Join(root, "inbox", "n.md")))
	})
}

func TestTrash(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "n.md"), "body")
	s := NewMarkdown(root)

	act, err := s.Trash("inbox/n.md")
	require.NoError(t, err)
	assert.False(t, exists(filepath.Join(root, "inbox", "n.md")))
	files := trashed(t, root)
	require.Len(t, files, 1)
	assert.Equal(t, "n.md", filepath.Base(files[0]), "the original name is kept for manual recovery")
	assert.Equal(t, "body", read(t, files[0]))

	require.NoError(t, act.Undo())
	assert.Equal(t, "body", read(t, filepath.Join(root, "inbox", "n.md")))
	assert.False(t, exists(filepath.Join(root, ".lazynote")))

	require.NoError(t, act.Redo())
	assert.False(t, exists(filepath.Join(root, "inbox", "n.md")))
	assert.Len(t, trashed(t, root), 1)
}

func TestTrashDirectoryAndSameNameTwice(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "projects", "p", "a.md"), "first")
	s := NewMarkdown(root)

	act1, err := s.Trash("projects/p")
	require.NoError(t, err)
	put(t, filepath.Join(root, "projects", "p", "b.md"), "second")
	act2, err := s.Trash("projects/p")
	require.NoError(t, err)
	assert.Len(t, trashed(t, root), 2, "same names get separate slots")

	require.NoError(t, act2.Undo())
	assert.Equal(t, "second", read(t, filepath.Join(root, "projects", "p", "b.md")))
	assert.ErrorIs(t, act1.Undo(), ErrExists, "the folder name is taken again, so the older trash stays put")
	assert.Len(t, trashed(t, root), 1)
}

func TestTrashUndoRefusals(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "n.md"), "orig")
	s := NewMarkdown(root)
	act, err := s.Trash("inbox/n.md")
	require.NoError(t, err)

	put(t, filepath.Join(root, "inbox", "n.md"), "new occupant")
	assert.ErrorIs(t, act.Undo(), ErrExists)
	assert.Equal(t, "new occupant", read(t, filepath.Join(root, "inbox", "n.md")))
	assert.Len(t, trashed(t, root), 1, "the trashed copy stays put")

	_, err = s.Trash("inbox/missing.md")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCopyFile(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "n.md"), "---\nkind: fact\n---\nbody")
	s := NewMarkdown(root)

	act, _, err := s.Copy("inbox/n.md", "areas/n.md")
	require.NoError(t, err)
	got := ParseNote("n.md", []byte(read(t, filepath.Join(root, "areas", "n.md"))))
	assert.Equal(t, "fact", got.Frontmatter.Get("kind").Value)
	assert.Equal(t, "body", got.Body)

	require.NoError(t, act.Undo())
	assert.False(t, exists(filepath.Join(root, "areas")))
	assert.Equal(t, "---\nkind: fact\n---\nbody", read(t, filepath.Join(root, "inbox", "n.md")), "source untouched")
	assert.Len(t, trashed(t, root), 1)

	require.NoError(t, act.Redo())
	assert.FileExists(t, filepath.Join(root, "areas", "n.md"))
}

func TestCopyDirectoryTree(t *testing.T) {
	root, tmp := newRoot(t)
	put(t, filepath.Join(root, "projects", "p", "a.md"), "body-a")
	put(t, filepath.Join(root, "projects", "p", "sub", "pic.png"), "png")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "projects", "p", "empty"), 0o755))
	require.NoError(t, os.Symlink(filepath.Join(tmp, "outside"), filepath.Join(root, "projects", "p", "link")))
	s := NewMarkdown(root)

	_, _, err := s.Copy("projects/p", "archives/p")
	require.NoError(t, err)
	assert.Equal(t, "body-a", ParseNote("a.md", []byte(read(t, filepath.Join(root, "archives", "p", "a.md")))).Body)
	assert.Equal(t, "png", read(t, filepath.Join(root, "archives", "p", "sub", "pic.png")), "attachments are copied")
	assert.DirExists(t, filepath.Join(root, "archives", "p", "empty"))
	assert.False(t, exists(filepath.Join(root, "archives", "p", "link")), "nested symlinks are skipped")
	assert.Equal(t, "body-a", read(t, filepath.Join(root, "projects", "p", "a.md")))
}

func TestCopyRejections(t *testing.T) {
	root, tmp := newRoot(t)
	put(t, filepath.Join(root, "inbox", "a.md"), "a")
	put(t, filepath.Join(root, "inbox", "b.md"), "b")
	put(t, filepath.Join(root, "projects", "p", "x.md"), "x")
	require.NoError(t, os.Symlink(filepath.Join(root, "inbox", "a.md"), filepath.Join(root, "inbox", "link.md")))
	require.NoError(t, os.Symlink(filepath.Join(tmp, "outside"), filepath.Join(root, "out")))
	s := NewMarkdown(root)

	tests := []struct {
		name     string
		src, dst string
		want     error
	}{
		{"destination exists", "inbox/a.md", "inbox/b.md", ErrExists},
		{"missing source", "inbox/gone.md", "inbox/c.md", ErrNotFound},
		{"folder into itself", "projects/p", "projects/p/inner", ErrIntoItself},
		{"escaping destination", "inbox/a.md", "../outside/a.md", ErrParentSegment},
		{"symlink destination outside", "inbox/a.md", "out/a.md", ErrEscapesRoot},
		{"symlink as the top item", "inbox/link.md", "inbox/c.md", ErrUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := s.Copy(tt.src, tt.dst)
			assert.ErrorIs(t, err, tt.want)
		})
	}
	assert.False(t, exists(filepath.Join(root, "inbox", "c.md")), "a failed copy leaves nothing behind")
	assert.False(t, exists(filepath.Join(root, "projects", "p", "inner")))
}

func TestOverwritePasteUndoesAsOneAction(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "n.md"), "incoming")
	put(t, filepath.Join(root, "areas", "n.md"), "existing")
	s := NewMarkdown(root)

	trash, err := s.Trash("areas/n.md")
	require.NoError(t, err)
	move, err := s.Move("inbox/n.md", "areas/n.md")
	require.NoError(t, err)
	paste := groupOf("paste n.md", trash, move)

	assert.Equal(t, "incoming", read(t, filepath.Join(root, "areas", "n.md")))

	require.NoError(t, paste.Undo())
	assert.Equal(t, "incoming", read(t, filepath.Join(root, "inbox", "n.md")))
	assert.Equal(t, "existing", read(t, filepath.Join(root, "areas", "n.md")), "the replaced note is back")

	require.NoError(t, paste.Redo())
	assert.Equal(t, "incoming", read(t, filepath.Join(root, "areas", "n.md")))
	assert.False(t, exists(filepath.Join(root, "inbox", "n.md")))
}

func TestGroupedUndoRollsBackWhenOneItemChanged(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "a.md"), "a")
	put(t, filepath.Join(root, "inbox", "b.md"), "b")
	s := NewMarkdown(root)

	moveA, err := s.Move("inbox/a.md", "areas/a.md")
	require.NoError(t, err)
	moveB, err := s.Move("inbox/b.md", "areas/b.md")
	require.NoError(t, err)
	paste := groupOf("paste 2 notes", moveA, moveB)

	// Undo runs b first, then a. Edit a outside so the undo of a is refused
	// after b has already been undone: b must end up back where it was.
	bump(t, filepath.Join(root, "areas", "a.md"), "edited")

	assert.ErrorIs(t, paste.Undo(), ErrChanged)
	assert.Equal(t, "b", read(t, filepath.Join(root, "areas", "b.md")), "rolled back to the pasted state")
	assert.False(t, exists(filepath.Join(root, "inbox", "b.md")))
	assert.Equal(t, "edited", read(t, filepath.Join(root, "areas", "a.md")))
}

func groupOf(desc string, actions ...undo.Action) undo.Action { return undo.Group(desc, actions...) }

func TestCopyOrMoveIntoItselfOnCaseInsensitiveFS(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "projects", "Dir", "a.md"), "a")
	if !exists(filepath.Join(root, "projects", "dir")) {
		t.Skip("filesystem is case-sensitive")
	}
	s := NewMarkdown(root)

	_, _, err := s.Copy("projects/Dir", "projects/dir/sub")
	assert.ErrorIs(t, err, ErrIntoItself)
	_, err = s.Move("projects/Dir", "projects/dir/sub")
	assert.ErrorIs(t, err, ErrIntoItself)
	assert.False(t, exists(filepath.Join(root, "projects", "Dir", "sub")), "nothing was created")
}

func TestRenameNoReplaceNeverClobbers(t *testing.T) {
	dir := t.TempDir()
	put(t, filepath.Join(dir, "from.md"), "from")
	put(t, filepath.Join(dir, "to.md"), "already here")
	want, err := takeSnapshot(filepath.Join(dir, "from.md"))
	require.NoError(t, err)

	err = renameNoReplace(filepath.Join(dir, "from.md"), filepath.Join(dir, "to.md"), want)
	assert.ErrorIs(t, err, ErrExists)
	assert.Equal(t, "already here", read(t, filepath.Join(dir, "to.md")))
	assert.Equal(t, "from", read(t, filepath.Join(dir, "from.md")))

	require.NoError(t, os.Remove(filepath.Join(dir, "to.md")))
	require.NoError(t, renameNoReplace(filepath.Join(dir, "from.md"), filepath.Join(dir, "to.md"), want))
	assert.Equal(t, "from", read(t, filepath.Join(dir, "to.md")))
	assert.False(t, exists(filepath.Join(dir, "from.md")))
}

func TestRenameNoReplaceKeepsModTime(t *testing.T) {
	dir := t.TempDir()
	put(t, filepath.Join(dir, "a.md"), "x")
	old := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "a.md"), old, old))
	want, err := takeSnapshot(filepath.Join(dir, "a.md"))
	require.NoError(t, err)

	require.NoError(t, renameNoReplace(filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md"), want))
	got, err := takeSnapshot(filepath.Join(dir, "b.md"))
	require.NoError(t, err)
	assert.Equal(t, want, got, "a move does not change the modification time")
}

func TestMakeDirUndoEdgeCases(t *testing.T) {
	t.Run("replaced by a file", func(t *testing.T) {
		root, _ := newRoot(t)
		act, err := NewMarkdown(root).MakeDir("areas/new")
		require.NoError(t, err)
		require.NoError(t, os.Remove(filepath.Join(root, "areas", "new")))
		put(t, filepath.Join(root, "areas", "new"), "a file now")

		assert.ErrorIs(t, act.Undo(), ErrChanged)
		assert.Equal(t, "a file now", read(t, filepath.Join(root, "areas", "new")))
	})
	t.Run("already removed", func(t *testing.T) {
		root, _ := newRoot(t)
		act, err := NewMarkdown(root).MakeDir("areas/new")
		require.NoError(t, err)
		require.NoError(t, os.Remove(filepath.Join(root, "areas", "new")))

		assert.NoError(t, act.Undo(), "nothing left to remove, so the undo is complete")
		assert.NoError(t, act.Redo())
		assert.DirExists(t, filepath.Join(root, "areas", "new"))
	})
	t.Run("failed redo keeps the bookkeeping", func(t *testing.T) {
		root, _ := newRoot(t)
		act, err := NewMarkdown(root).MakeDir("areas/new")
		require.NoError(t, err)
		require.NoError(t, act.Undo())
		put(t, filepath.Join(root, "areas", "new"), "taken")

		assert.ErrorIs(t, act.Redo(), ErrExists)
		assert.NotPanics(t, func() { _ = act.Undo() })
	})
}

func TestCopyReadOnlyDirectory(t *testing.T) {
	root, _ := newRoot(t)
	src := filepath.Join(root, "projects", "ro")
	put(t, filepath.Join(src, "a.md"), "a")
	require.NoError(t, os.Chmod(src, 0o555))
	t.Cleanup(func() {
		_ = os.Chmod(src, 0o755)
		_ = os.Chmod(filepath.Join(root, "archives", "ro"), 0o755)
	})

	_, _, err := NewMarkdown(root).Copy("projects/ro", "archives/ro")
	require.NoError(t, err)
	assert.Equal(t, "a", ParseNote("a.md", []byte(read(t, filepath.Join(root, "archives", "ro", "a.md")))).Body)
	info, err := os.Stat(filepath.Join(root, "archives", "ro"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o555), info.Mode().Perm(), "the source mode is applied after the copy")
}

func TestCopyFailureLeavesNothingBehind(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read any file")
	}
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "projects", "p", "a.md"), "body-a")
	put(t, filepath.Join(root, "projects", "p", "z-secret.md"), "z")
	secret := filepath.Join(root, "projects", "p", "z-secret.md")
	require.NoError(t, os.Chmod(secret, 0o000))
	t.Cleanup(func() { _ = os.Chmod(secret, 0o644) })

	_, _, err := NewMarkdown(root).Copy("projects/p", "archives/deep/p")
	assert.ErrorIs(t, err, ErrUnreadable)
	assert.False(t, exists(filepath.Join(root, "archives")), "partial copy and its new parents are removed")
	assert.Equal(t, "body-a", read(t, filepath.Join(root, "projects", "p", "a.md")))
}

func TestExists(t *testing.T) {
	root, tmp := newRoot(t)
	put(t, filepath.Join(root, "inbox", "n.md"), "x")
	require.NoError(t, os.Symlink(filepath.Join(tmp, "missing"), filepath.Join(root, "dangling")))
	require.NoError(t, os.Symlink(filepath.Join(tmp, "outside"), filepath.Join(root, "out")))
	s := NewMarkdown(root)

	for rel, want := range map[string]bool{
		"inbox/n.md":      true,
		"inbox":           true,
		"inbox/gone.md":   false,
		"inbox/n.md/deep": false,
		"nothing/at/all":  false,
	} {
		got, err := s.Exists(rel)
		require.NoError(t, err, rel)
		assert.Equal(t, want, got, rel)
	}
	for _, rel := range []string{"", "../outside", "/etc", "out/x", "dangling"} {
		_, err := s.Exists(rel)
		assert.Error(t, err, rel)
	}
}
