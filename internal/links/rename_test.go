package links

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkilb/lazynote/internal/store"
)

func TestRenameReference(t *testing.T) {
	tests := []struct {
		name, in, want string
		changed        bool
	}{
		{"plain", "related: [\"[[Old]]\"]\n", "related: [\"[[New]]\"]\n", true},
		{"case-insensitive match keeps the new spelling", "related: [\"[[old]]\"]\n", "related: [\"[[New]]\"]\n", true},
		{"keeps alias and heading", "related: [\"[[Old#Goals|the goals]]\"]\n", "related: [\"[[New#Goals|the goals]]\"]\n", true},
		{"keeps a folder prefix and extension", "related: [\"[[projects/Old.md]]\"]\n", "related: [\"[[projects/New.md]]\"]\n", true},
		{"single string", "related: \"[[Old]]\"\n", "related: \"[[New]]\"\n", true},
		{"unquoted nested link", "related: [[Old]]\n", "related: [\"[[New]]\"]\n", true},
		{"only matching entries", "related: [\"[[A]]\", \"[[Old]]\", \"free\", 5]\n", "related: [\"[[A]]\", \"[[New]]\", \"free\", 5]\n", true},
		{"every spelling", "related: [\"[[Old]]\", \"[[old|again]]\"]\n", "related: [\"[[New]]\", \"[[New|again]]\"]\n", true},
		{"no match", "related: [\"[[A]]\"]\n", "related: [\"[[A]]\"]\n", false},
		{"no property", "kind: fact\n", "kind: fact\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := fmOf(t, tt.in)
			assert.Equal(t, tt.changed, RenameReference(fm, "Old", "New"))
			assert.Equal(t, tt.want, marshal(t, fm))
		})
	}
}

// notebook builds a small notebook and returns its root and an index.
func notebook(t *testing.T, files map[string]string) (string, store.Store, *Index) {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		put(t, root, rel, content)
	}
	s := store.NewMarkdown(root)
	x, err := Build(s)
	require.NoError(t, err)
	return root, s, x
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	require.NoError(t, err)
	return string(b)
}

func TestRenameNoteMovesRetitlesAndRewritesReferrers(t *testing.T) {
	root, s, x := notebook(t, map[string]string{
		"projects/Old.md": "---\nuid: u1\ntitle: Old\n---\nbody\n",
		"inbox/a.md":      note("[[Old#Goals|goals]]", "[[Other]]"),
		"areas/b.md":      note("[[old]]"),
		"areas/Other.md":  "x",
		"inbox/c.md":      note("[[Other]]"),
	})
	origA := readFile(t, root, "inbox/a.md")

	act, res, err := Rename(s, x, "projects/Old.md", false, "New")
	require.NoError(t, err)
	assert.Equal(t, RenameResult{Updated: 2}, res)

	assert.NoFileExists(t, filepath.Join(root, "projects", "Old.md"))
	n := store.ParseNote("New.md", []byte(readFile(t, root, "projects/New.md")))
	assert.Equal(t, "New", n.Frontmatter.Get("title").Value)
	assert.Equal(t, "u1", n.Frontmatter.Get("uid").Value, "a rename never changes the uid")
	assert.Equal(t, "body\n", n.Body)
	assert.Contains(t, readFile(t, root, "inbox/a.md"), "[[New#Goals|goals]]")
	assert.Contains(t, readFile(t, root, "inbox/a.md"), "[[Other]]")
	assert.Contains(t, readFile(t, root, "areas/b.md"), "[[New]]")
	assert.Equal(t, note("[[Other]]"), readFile(t, root, "inbox/c.md"), "unrelated notes are untouched")

	require.NoError(t, act.Undo())
	assert.Equal(t, "---\nuid: u1\ntitle: Old\n---\nbody\n", readFile(t, root, "projects/Old.md"))
	assert.Equal(t, origA, readFile(t, root, "inbox/a.md"), "one undo restores every note byte for byte")
	assert.NoFileExists(t, filepath.Join(root, "projects", "New.md"))

	require.NoError(t, act.Redo())
	assert.FileExists(t, filepath.Join(root, "projects", "New.md"))
	assert.Contains(t, readFile(t, root, "areas/b.md"), "[[New]]")
}

func TestRenameWithNoReferrersJustRenames(t *testing.T) {
	root, s, x := notebook(t, map[string]string{"inbox/Old.md": "plain text\n"})

	_, res, err := Rename(s, x, "inbox/Old.md", false, "New.md")
	require.NoError(t, err)
	assert.Equal(t, RenameResult{}, res)
	n := store.ParseNote("New.md", []byte(readFile(t, root, "inbox/New.md")))
	assert.Equal(t, "New", n.Frontmatter.Get("title").Value, "a note with no properties gets a block")
	assert.NotEmpty(t, n.Frontmatter.Get("uid").Value)
	assert.Equal(t, "plain text\n", n.Body)
}

func TestRenameRefusals(t *testing.T) {
	root, s, x := notebook(t, map[string]string{
		"inbox/Old.md":   "x",
		"inbox/Taken.md": "keep",
		"inbox/ref.md":   note("[[Old]]"),
	})
	tests := []struct{ name, typed string }{
		{"existing name", "Taken"},
		{"slash", "a/b"},
		{"empty", " "},
		{"dotdot", ".."},
		{"link-breaking pipe", "a|b"},
		{"link-breaking hash", "C# notes"},
		{"link-breaking bracket", "a]]b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Rename(s, x, "inbox/Old.md", false, tt.typed)
			assert.Error(t, err)
			assert.Equal(t, "x", readFile(t, root, "inbox/Old.md"), "nothing changed")
			assert.Equal(t, note("[[Old]]"), readFile(t, root, "inbox/ref.md"))
			assert.Equal(t, "keep", readFile(t, root, "inbox/Taken.md"))
		})
	}
	_, _, err := Rename(s, x, "inbox/missing.md", false, "New")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestRenameAllowsLinkBreakingCharactersWhenNothingReferencesTheNote(t *testing.T) {
	root, s, x := notebook(t, map[string]string{"inbox/Old.md": "x"})
	_, _, err := Rename(s, x, "inbox/Old.md", false, "C# notes")
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(root, "inbox", "C# notes.md"))
}

func TestRenameKeepsGoingWhenAFileCannotBeRewritten(t *testing.T) {
	root, s, x := notebook(t, map[string]string{
		"projects/Old.md": "---\nkind: [oops\n---\nbody\n",
		"inbox/good.md":   note("[[Old]]"),
		"inbox/bad.md":    note("[[Old]]"),
	})
	// bad.md turns invalid after the index was built, as if edited outside.
	require.NoError(t, os.WriteFile(filepath.Join(root, "inbox", "bad.md"), []byte("---\nbroken: [\n---\n"), 0o644))

	_, res, err := Rename(s, x, "projects/Old.md", false, "New")
	require.NoError(t, err)
	assert.Equal(t, RenameResult{Updated: 1, Skipped: 1, TitleKept: true}, res)
	assert.Equal(t, "---\nkind: [oops\n---\nbody\n", readFile(t, root, "projects/New.md"), "invalid properties are left exactly as they are")
	assert.Contains(t, readFile(t, root, "inbox/good.md"), "[[New]]")
}

func TestRenameFolderMovesItWithoutTouchingLinks(t *testing.T) {
	root, s, x := notebook(t, map[string]string{
		"projects/old/Inner.md": "---\nuid: u\ntitle: Inner\n---\n",
		"inbox/ref.md":          note("[[Inner]]"),
	})
	origRef := readFile(t, root, "inbox/ref.md")

	act, res, err := Rename(s, x, "projects/old", true, "new")
	require.NoError(t, err)
	assert.Equal(t, RenameResult{}, res)
	assert.FileExists(t, filepath.Join(root, "projects", "new", "Inner.md"))
	assert.Equal(t, origRef, readFile(t, root, "inbox/ref.md"), "links name files, so a folder rename leaves them alone")

	require.NoError(t, act.Undo())
	assert.DirExists(t, filepath.Join(root, "projects", "old"))
}

func TestRenameUndoRefusesAfterAnOutsideEditAndLeavesEverythingAlone(t *testing.T) {
	root, s, x := notebook(t, map[string]string{
		"projects/Old.md": "x",
		"inbox/a.md":      note("[[Old]]"),
	})
	act, _, err := Rename(s, x, "projects/Old.md", false, "New")
	require.NoError(t, err)
	renamed := readFile(t, root, "inbox/a.md")

	bumped := filepath.Join(root, "projects", "New.md")
	require.NoError(t, os.WriteFile(bumped, []byte("changed outside\n"), 0o644))
	later := timeLater()
	require.NoError(t, os.Chtimes(bumped, later, later))

	assert.Error(t, act.Undo())
	assert.Equal(t, renamed, readFile(t, root, "inbox/a.md"), "a failed undo puts the referrer back as it was")
	assert.Equal(t, "changed outside\n", readFile(t, root, "projects/New.md"))
}
