package intake

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkilb/lazynote/internal/store"
)

var now = time.Date(2026, 10, 30, 14, 30, 5, 0, time.Local)

func TestCleanTitle(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Call Mum about the thing", "Call Mum about the thing"},
		{"  spaced   out   words  ", "spaced out words"},
		{`a/b\c:d*e?f"g<h>i|j`, "abcdefghij"},
		{"C# [notes] ^caret #tag", "C notes caret tag"},
		{"..hidden", "hidden"},
		{"trailing dots...", "trailing dots"},
		{"Héllo Wörld", "Héllo Wörld"},
		{"", ""},
		{"///", ""},
		{strings.Repeat("x", 200), strings.Repeat("x", 80)},
		{"tab\tand\nnewline", "tab and newline"},
		{"nul\x00byte", "nulbyte"},
		{"keeps case AND punctuation, ok!", "keeps case AND punctuation, ok!"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, CleanTitle(tt.in))
		})
	}
}

func TestName(t *testing.T) {
	assert.Equal(t, "Call Mum.md", Name("Call Mum"))
	assert.Equal(t, "", Name("///"), "nothing usable, no name")
}

func newStore(t *testing.T) (string, store.Store) {
	t.Helper()
	root := t.TempDir()
	return root, store.NewMarkdown(root)
}

func TestCaptureWritesAnInboxNoteAndIsUndoable(t *testing.T) {
	root, s := newStore(t)

	res, err := Capture(s, now, store.KindFact, SourceQuick, "Water boils", "At 100C at sea level")
	require.NoError(t, err)
	assert.Equal(t, "inbox/Water boils.md", res.Rel, "named from the title, no timestamp")

	data, err := os.ReadFile(filepath.Join(root, res.Rel))
	require.NoError(t, err)
	n := store.ParseNote("x.md", data)
	assert.Equal(t, store.KindFact, n.Kind)
	assert.Equal(t, "quick", n.Frontmatter.Get("source").Value)
	assert.Equal(t, "2026-10-30", n.Frontmatter.Get("date").Value, "the date lives in the properties")
	assert.Equal(t, "Water boils", n.Frontmatter.Get("title").Value)
	assert.Equal(t, "At 100C at sea level\n", n.Body)

	require.NoError(t, res.Action.Undo())
	assert.NoFileExists(t, filepath.Join(root, res.Rel))
	require.NoError(t, res.Action.Redo())
	assert.FileExists(t, filepath.Join(root, res.Rel))
}

func TestCaptureWithoutABodyMakesAnEmptyBody(t *testing.T) {
	root, s := newStore(t)
	res, err := Capture(s, now, store.KindIdea, SourceQuick, "Just a title", "  ")
	require.NoError(t, err)
	data, _ := os.ReadFile(filepath.Join(root, res.Rel))
	assert.Empty(t, store.ParseNote("x.md", data).Body)
}

func TestCaptureNumbersACollidingName(t *testing.T) {
	root, s := newStore(t)
	a, err := Capture(s, now, store.KindIdea, SourceQuick, "Same", "one")
	require.NoError(t, err)
	b, err := Capture(s, now, store.KindIdea, SourceQuick, "Same", "two")
	require.NoError(t, err)
	c, err := Capture(s, now, store.KindIdea, SourceQuick, "Same", "three")
	require.NoError(t, err)

	assert.Equal(t, "inbox/Same.md", a.Rel)
	assert.Equal(t, "inbox/Same 2.md", b.Rel)
	assert.Equal(t, "inbox/Same 3.md", c.Rel)
	data, _ := os.ReadFile(filepath.Join(root, c.Rel))
	assert.Equal(t, "Same 3", store.ParseNote("x.md", data).Frontmatter.Get("title").Value, "the title follows the numbered name")
}

func TestCaptureRejectsAMissingTitle(t *testing.T) {
	_, s := newStore(t)
	for _, title := range []string{"", "   ", "\t", "///", "#"} {
		_, err := Capture(s, now, store.KindFact, SourceQuick, title, "has a body")
		assert.ErrorIs(t, err, ErrEmpty, "%q", title)
	}
}

func TestCaptureKeepsTheBodyVerbatim(t *testing.T) {
	root, s := newStore(t)
	body := "---\nnot: frontmatter [[link]] # heading"
	res, err := Capture(s, now, store.KindRumor, SourceQuick, "T", body)
	require.NoError(t, err)
	data, _ := os.ReadFile(filepath.Join(root, res.Rel))
	n := store.ParseNote("x.md", data)
	require.NoError(t, n.FrontmatterErr)
	assert.Equal(t, body+"\n", n.Body)
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestImportTranscript(t *testing.T) {
	root, s := newStore(t)
	text := "Alice: hi\nBob: hello\n"
	src := writeTemp(t, "standup notes.txt", text)

	res, err := ImportTranscript(s, now, src)
	require.NoError(t, err)
	assert.Equal(t, "inbox/standup notes.md", res.Rel)
	data, _ := os.ReadFile(filepath.Join(root, res.Rel))
	n := store.ParseNote("x.md", data)
	assert.Equal(t, store.KindRumor, n.Kind, "raw and unconfirmed until reviewed")
	assert.Equal(t, "transcript", n.Frontmatter.Get("source").Value)
	assert.Equal(t, text, n.Body, "the original text is kept unmodified")

	srcAfter, _ := os.ReadFile(src)
	assert.Equal(t, text, string(srcAfter), "the source file is never touched")
}

func TestImportTranscriptKeepsFrontmatterLookingTextAsBody(t *testing.T) {
	root, s := newStore(t)
	text := "---\nkind: fact\n---\nreal text\n"
	res, err := ImportTranscript(s, now, writeTemp(t, "t.txt", text))
	require.NoError(t, err)
	data, _ := os.ReadFile(filepath.Join(root, res.Rel))
	n := store.ParseNote("x.md", data)
	assert.Equal(t, store.KindRumor, n.Kind)
	assert.Equal(t, text, n.Body)
}

func TestImportNoteKeepsItsNameAndGetsFreshIdentity(t *testing.T) {
	root, s := newStore(t)
	src := writeTemp(t, "Project Plan.md", "---\nuid: old\nkind: proposal\ntags: [x]\n---\nbody\n")

	res, err := ImportNote(s, now, src)
	require.NoError(t, err)
	assert.Equal(t, "inbox/Project Plan.md", res.Rel)
	assert.False(t, res.Unchanged)
	data, _ := os.ReadFile(filepath.Join(root, res.Rel))
	n := store.ParseNote("Project Plan.md", data)
	assert.Equal(t, store.KindProposal, n.Kind, "kind is left as it is")
	assert.Equal(t, "import", n.Frontmatter.Get("source").Value)
	assert.Equal(t, "2026-10-30", n.Frontmatter.Get("date").Value)
	assert.Equal(t, "Project Plan", n.Frontmatter.Get("title").Value)
	assert.NotEqual(t, "old", n.Frontmatter.Get("uid").Value)

	again, err := ImportNote(s, now, src)
	require.NoError(t, err)
	assert.Equal(t, "inbox/Project Plan 2.md", again.Rel, "a collision gets a number")
	require.NoError(t, again.Action.Undo())
	assert.NoFileExists(t, filepath.Join(root, again.Rel))
}

func TestImportNoteWithInvalidFrontmatterIsCopiedAsIs(t *testing.T) {
	root, s := newStore(t)
	content := "---\nkind: [oops\n---\nbody\n"
	res, err := ImportNote(s, now, writeTemp(t, "bad.md", content))
	require.NoError(t, err)
	assert.True(t, res.Unchanged)
	data, _ := os.ReadFile(filepath.Join(root, res.Rel))
	assert.Equal(t, content, string(data))
}

func TestImportNoteRequiresAMarkdownFile(t *testing.T) {
	_, s := newStore(t)
	_, err := ImportNote(s, now, writeTemp(t, "plain.txt", "x"))
	assert.ErrorIs(t, err, ErrNotNote)
}

func TestImportReadsOnlyRegularFilesOfReasonableSize(t *testing.T) {
	root, s := newStore(t)
	dir := t.TempDir()
	big := filepath.Join(dir, "big.md")
	require.NoError(t, os.WriteFile(big, nil, 0o644))
	require.NoError(t, os.Truncate(big, store.MaxNoteBytes+1))
	link := filepath.Join(dir, "link.md")
	require.NoError(t, os.Symlink(writeTemp(t, "real.md", "# hi\n"), link))

	tests := []struct {
		name, path string
		want       error
	}{
		{"missing", filepath.Join(dir, "nope.md"), ErrNotFound},
		{"directory", dir, ErrNotFile},
		{"too large", big, ErrTooLarge},
		{"empty path", "  ", ErrNotFound},
		{"nul byte", "a\x00b.md", ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ImportNote(s, now, tt.path)
			assert.ErrorIs(t, err, tt.want)
			_, err = ImportTranscript(s, now, tt.path)
			assert.ErrorIs(t, err, tt.want)
		})
	}
	// A symlink to a regular file is read, like any open would; the write
	// still lands inside the notebook.
	res, err := ImportNote(s, now, link)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(root, res.Rel))
	entries, _ := os.ReadDir(filepath.Join(root, "inbox"))
	assert.Len(t, entries, 1)
}

func TestImportExpandsAHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.WriteFile(filepath.Join(home, "n.md"), []byte("x\n"), 0o644))
	_, s := newStore(t)

	res, err := ImportNote(s, now, "~/n.md")
	require.NoError(t, err)
	assert.Equal(t, "inbox/n.md", res.Rel)
}

func TestErrorsNeverEchoThePath(t *testing.T) {
	_, s := newStore(t)
	_, err := ImportNote(s, now, "/definitely/secret/place/x.md")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret")
}
