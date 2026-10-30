package links

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkilb/lazynote/internal/store"
)

func TestTarget(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"[[Project Alpha]]", "Project Alpha", true},
		{"  [[Project Alpha]]  ", "Project Alpha", true},
		{"[[Project Alpha|the alpha]]", "Project Alpha", true},
		{"[[Project Alpha#Goals]]", "Project Alpha", true},
		{"[[Project Alpha#Goals|goals]]", "Project Alpha", true},
		{"[[projects/Project Alpha]]", "Project Alpha", true},
		{"[[Project Alpha.md]]", "Project Alpha", true},
		{"[[Project Alpha.MD]]", "Project Alpha", true},
		{"[[ Spaced ]]", "Spaced", true},
		{"Project Alpha", "", false},
		{"[[Open", "", false},
		{"[[]]", "", false},
		{"[[|alias]]", "", false},
		{"[[#heading]]", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := Target(tt.in)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func put(t *testing.T, root, rel, content string) string {
	t.Helper()
	p := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func note(related ...string) string {
	s := "---\nuid: u\nrelated:\n"
	for _, r := range related {
		s += "  - \"" + r + "\"\n"
	}
	return s + "---\nbody\n"
}

func build(t *testing.T, root string) *Index {
	t.Helper()
	x, err := Build(store.NewMarkdown(root))
	require.NoError(t, err)
	return x
}

func TestOutgoingResolvesByNameIgnoringCaseAliasAndHeading(t *testing.T) {
	root := t.TempDir()
	put(t, root, "inbox/a.md", note("[[PROJECT alpha|x]]", "[[Meeting#Notes]]", "[[Nope]]"))
	put(t, root, "projects/Project Alpha.md", "---\nuid: p\n---\n")
	put(t, root, "areas/Meeting.md", "x")
	x := build(t, root)

	assert.Equal(t, []Link{
		{Name: "PROJECT alpha", Rel: "projects/Project Alpha.md"},
		{Name: "Meeting", Rel: "areas/Meeting.md"},
		{Name: "Nope", Rel: ""},
	}, x.Outgoing("inbox/a.md"))
}

func TestDuplicateNamesResolveToTheShortestPath(t *testing.T) {
	root := t.TempDir()
	put(t, root, "inbox/a.md", note("[[Dup]]"))
	put(t, root, "projects/deep/er/Dup.md", "x")
	put(t, root, "areas/Dup.md", "x")
	put(t, root, "archives/Dup.md", "x")
	x := build(t, root)

	assert.Equal(t, "areas/Dup.md", x.Outgoing("inbox/a.md")[0].Rel, "shortest, then alphabetical")
}

func TestIncomingListsNotesThatLinkHere(t *testing.T) {
	root := t.TempDir()
	put(t, root, "projects/Hub.md", "x")
	put(t, root, "inbox/b.md", note("[[hub]]", "[[Hub|again]]"))
	put(t, root, "inbox/a.md", note("[[Hub]]"))
	put(t, root, "projects/Hub2.md", note("[[Hub]]"))
	put(t, root, "inbox/self.md", note("[[self]]"))
	x := build(t, root)

	assert.Equal(t, []string{"inbox/a.md", "inbox/b.md", "projects/Hub2.md"}, x.Incoming("projects/Hub.md"),
		"sorted, one entry per note")
	assert.Empty(t, x.Incoming("inbox/self.md"), "a note linking itself is not its own backlink")
	assert.Empty(t, x.Incoming("inbox/a.md"))
}

func TestRelatedValueShapes(t *testing.T) {
	root := t.TempDir()
	put(t, root, "inbox/scalar.md", "---\nrelated: \"[[X]]\"\n---\n")
	put(t, root, "inbox/unquoted.md", "---\nrelated: [[X]]\n---\n")
	put(t, root, "inbox/mixed.md", "---\nrelated: [\"[[X]]\", 5, \"plain\", {a: b}]\n---\n")
	put(t, root, "inbox/none.md", "---\nkind: fact\n---\n")
	put(t, root, "inbox/bad.md", "---\nrelated: [oops\n---\n")
	put(t, root, "inbox/plain.md", "no frontmatter [[X]]")
	put(t, root, "areas/X.md", "x")
	x := build(t, root)

	want := []Link{{Name: "X", Rel: "areas/X.md"}}
	assert.Equal(t, want, x.Outgoing("inbox/scalar.md"))
	assert.Equal(t, want, x.Outgoing("inbox/unquoted.md"), "an unquoted [[X]] parses as a nested list")
	assert.Equal(t, want, x.Outgoing("inbox/mixed.md"), "only wikilink strings count")
	assert.Empty(t, x.Outgoing("inbox/none.md"))
	assert.Empty(t, x.Outgoing("inbox/bad.md"))
	assert.Empty(t, x.Outgoing("inbox/plain.md"), "inline body links are out of scope")
}

func TestHiddenFoldersAreNotIndexed(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".obsidian/Ghost.md", "x")
	put(t, root, ".lazynote/trash/ab/Ghost.md", "x")
	put(t, root, "projects/.hidden/Ghost.md", "x")
	put(t, root, "Ghost.md", "outside the PARA folders")
	put(t, root, "inbox/a.md", note("[[Ghost]]"))
	x := build(t, root)

	assert.Equal(t, []Link{{Name: "Ghost"}}, x.Outgoing("inbox/a.md"))
}

func TestRefreshPicksUpChangesAndRemovals(t *testing.T) {
	root := t.TempDir()
	s := store.NewMarkdown(root)
	a := put(t, root, "inbox/a.md", note("[[B]]"))
	put(t, root, "areas/B.md", "x")
	x, err := Build(s)
	require.NoError(t, err)
	require.Equal(t, "areas/B.md", x.Outgoing("inbox/a.md")[0].Rel)

	// a changes target, B disappears, C appears.
	require.NoError(t, os.WriteFile(a, []byte(note("[[C]]", "[[B]]")), 0o644))
	later := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(a, later, later))
	require.NoError(t, os.Remove(filepath.Join(root, "areas", "B.md")))
	put(t, root, "projects/C.md", "x")
	require.NoError(t, x.Refresh(s))

	assert.Equal(t, []Link{{Name: "C", Rel: "projects/C.md"}, {Name: "B"}}, x.Outgoing("inbox/a.md"))
	assert.Equal(t, []string{"inbox/a.md"}, x.Incoming("projects/C.md"))

	require.NoError(t, os.Remove(a))
	require.NoError(t, x.Refresh(s))
	assert.Empty(t, x.Outgoing("inbox/a.md"))
	assert.Empty(t, x.Incoming("projects/C.md"))
}

func TestAnUnreadableNoteDoesNotStopTheIndex(t *testing.T) {
	root := t.TempDir()
	put(t, root, "inbox/ok.md", note("[[T]]"))
	put(t, root, "areas/T.md", "x")
	big := put(t, root, "inbox/big.md", "")
	require.NoError(t, os.Truncate(big, 11<<20))
	x := build(t, root)

	assert.Equal(t, "areas/T.md", x.Outgoing("inbox/ok.md")[0].Rel)
	assert.Empty(t, x.Outgoing("inbox/big.md"))
}
