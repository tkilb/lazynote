package cli

import (
	"bytes"
	"errors"
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

// harness runs the command line against a temp notebook.
type harness struct {
	t       *testing.T
	root    string
	sources []string
	stdin   string
	tty     bool
	clip    string
	clipErr error
	opened  int
}

func newHarness(t *testing.T) *harness {
	return &harness{t: t, root: t.TempDir(), sources: []string{"adhoc", "meeting"}, tty: true}
}

// run executes args and returns what it wrote and its exit code.
func (h *harness) run(args ...string) (handled bool, code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	env := Env{
		Stdin:    strings.NewReader(h.stdin),
		StdinTTY: h.tty,
		Stdout:   &out,
		Stderr:   &errOut,
		Now:      now,
		Open: func() (store.Store, []string, error) {
			h.opened++
			return store.NewMarkdown(h.root), h.sources, nil
		},
		Clipboard: func() (string, error) { return h.clip, h.clipErr },
	}
	handled, code = Run(args, env)
	return handled, code, out.String(), errOut.String()
}

func (h *harness) note(rel string) store.Note {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.root, rel))
	require.NoError(h.t, err, rel)
	return store.ParseNote(filepath.Base(rel), data)
}

func TestOtherCommandsAreNotHandled(t *testing.T) {
	h := newHarness(t)
	for _, args := range [][]string{nil, {"version"}, {"--version"}, {"frobnicate"}} {
		handled, _, _, _ := h.run(args...)
		assert.False(t, handled, "%v", args)
	}
	assert.Zero(t, h.opened, "the notebook is not opened for commands that are not ours")
}

func TestAddTitleOnly(t *testing.T) {
	h := newHarness(t)
	handled, code, out, errOut := h.run("add", "Call mum")
	require.True(t, handled)
	require.Equal(t, 0, code, errOut)
	assert.Equal(t, "inbox/Call mum.md\n", out, "prints the new note's path")

	n := h.note("inbox/Call mum.md")
	assert.Nil(t, n.Frontmatter.Get("kind"), "no --kind, no kind property")
	assert.Equal(t, "adhoc", n.Frontmatter.Get("source").Value, "the first configured source is the default")
	assert.Equal(t, "2026-10-30", n.Frontmatter.Get("date").Value)
	assert.Empty(t, n.Frontmatter.Get("tags").Content)
	assert.Empty(t, n.Body)
}

func TestAddTitleAndBody(t *testing.T) {
	h := newHarness(t)
	_, code, _, errOut := h.run("add", "Water boils", "At 100C")
	require.Equal(t, 0, code, errOut)
	assert.Equal(t, "At 100C\n", h.note("inbox/Water boils.md").Body)
}

func TestFlagsMayComeAnywhere(t *testing.T) {
	h := newHarness(t)
	for i, args := range [][]string{
		{"add", "--kind", "idea", "--source", "web", "--tags", "a,b", "Title A", "body"},
		{"add", "Title B", "--kind", "idea", "--source", "web", "--tags", "a,b", "body"},
		{"add", "Title C", "body", "--kind=idea", "--source=web", "--tags=a,b"},
	} {
		_, code, _, errOut := h.run(args...)
		require.Equal(t, 0, code, "%d: %s", i, errOut)
		n := h.note("inbox/Title " + string(rune('A'+i)) + ".md")
		assert.Equal(t, store.KindIdea, n.Kind)
		assert.Equal(t, "web", n.Frontmatter.Get("source").Value)
		assert.Len(t, n.Frontmatter.Get("tags").Content, 2)
		assert.Equal(t, "body\n", n.Body)
	}
}

func TestSourceAcceptsAnyWord(t *testing.T) {
	h := newHarness(t)
	for i, src := range []string{"carrier-pigeon", "1:1", "Team Sync", "héllo"} {
		title := "T" + string(rune('A'+i))
		_, code, _, errOut := h.run("add", title, "--source", src)
		require.Equal(t, 0, code, errOut)
		assert.Equal(t, src, h.note("inbox/"+title+".md").Frontmatter.Get("source").Value)
	}
}

func TestDefaultSourceIsTheFirstConfiguredOne(t *testing.T) {
	h := newHarness(t)
	h.sources = []string{"standup", "adhoc"}
	_, code, _, _ := h.run("add", "T")
	require.Equal(t, 0, code)
	assert.Equal(t, "standup", h.note("inbox/T.md").Frontmatter.Get("source").Value)
}

func TestBadInputIsRefusedAndNothingIsWritten(t *testing.T) {
	long := strings.Repeat("x", 41)
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no arguments", []string{"add"}, "title"},
		{"blank title", []string{"add", "  "}, "title"},
		{"unusable title", []string{"add", "///"}, "title"},
		{"too many words", []string{"add", "a", "b", "c"}, "too many"},
		{"unknown kind", []string{"add", "T", "--kind", "legend"}, "kind"},
		{"blank source", []string{"add", "T", "--source", "  "}, "source"},
		{"long source", []string{"add", "T", "--source", long}, "source"},
		{"control source", []string{"add", "T", "--source", "a\tb"}, "source"},
		{"bad tag", []string{"add", "T", "--tags", "bad+tag"}, "tag"},
		{"unknown flag", []string{"add", "T", "--nope"}, "nope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			_, code, out, errOut := h.run(tt.args...)
			assert.NotEqual(t, 0, code)
			assert.Empty(t, out)
			assert.Contains(t, strings.ToLower(errOut), tt.wantErr)
			assert.NoDirExists(t, filepath.Join(h.root, "inbox"))
		})
	}
}

func TestKindIsCaseInsensitiveAndUnsetIsAllowed(t *testing.T) {
	h := newHarness(t)
	_, code, _, _ := h.run("add", "A", "--kind", "FACT")
	require.Equal(t, 0, code)
	assert.Equal(t, store.KindFact, h.note("inbox/A.md").Kind)
	_, code, _, _ = h.run("add", "B", "--kind", "unset")
	require.Equal(t, 0, code)
	assert.Nil(t, h.note("inbox/B.md").Frontmatter.Get("kind"))
}

func TestAddedKindsAreAcceptedAndNamedInMessages(t *testing.T) {
	t.Cleanup(store.ResetKinds)
	require.NoError(t, store.SetKinds([]store.KindSpec{{Name: "recipe"}}))

	h := newHarness(t)
	_, code, _, errOut := h.run("add", "Pie", "--kind", "Recipe")
	require.Equal(t, 0, code, errOut)
	assert.Equal(t, store.Kind("recipe"), h.note("inbox/Pie.md").Kind)

	_, code, _, errOut = h.run("add", "Bad", "--kind", "nonsense")
	assert.Equal(t, 1, code)
	assert.Contains(t, errOut, "recipe", "the message lists the added kind")
	assert.NoFileExists(t, filepath.Join(h.root, "inbox", "Bad.md"))

	_, code, _, errOut = h.run("add", "-h")
	assert.Equal(t, 0, code)
	assert.Contains(t, errOut, "recipe", "the usage text lists it too")
}

func TestDoubleDashEndsTheFlags(t *testing.T) {
	h := newHarness(t)
	_, code, _, errOut := h.run("add", "--source", "web", "--", "--looks like a flag", "-5 degrees")
	require.Equal(t, 0, code, errOut)
	n := h.note("inbox/--looks like a flag.md")
	assert.Equal(t, "-5 degrees\n", n.Body)
	assert.Equal(t, "web", n.Frontmatter.Get("source").Value)
}

func TestPipedTextBecomesTheBodyOfAGivenTitle(t *testing.T) {
	h := newHarness(t)
	h.tty, h.stdin = false, "line one\nline two\n"
	_, code, _, errOut := h.run("add", "From a pipe")
	require.Equal(t, 0, code, errOut)
	assert.Equal(t, "line one\nline two\n", h.note("inbox/From a pipe.md").Body)
}

func TestPipedTextWithoutATitleUsesItsFirstLine(t *testing.T) {
	h := newHarness(t)
	h.tty, h.stdin = false, "Meeting notes\n\nfirst point\nsecond point\n"
	_, code, out, errOut := h.run("add", "--source", "clipboard")
	require.Equal(t, 0, code, errOut)
	assert.Equal(t, "inbox/Meeting notes.md\n", out)
	n := h.note("inbox/Meeting notes.md")
	assert.Equal(t, "first point\nsecond point\n", n.Body)
	assert.Equal(t, "clipboard", n.Frontmatter.Get("source").Value)
}

func TestSingleLinePipedTextIsAllTitle(t *testing.T) {
	h := newHarness(t)
	h.tty, h.stdin = false, "just one line\n"
	_, code, _, _ := h.run("add")
	require.Equal(t, 0, code)
	assert.Empty(t, h.note("inbox/just one line.md").Body)
}

func TestStdinIsNotReadFromATerminalOrWhenABodyIsGiven(t *testing.T) {
	h := newHarness(t)
	h.stdin = "never read"
	_, code, _, _ := h.run("add", "T1") // a terminal: stdin untouched
	require.Equal(t, 0, code)
	assert.Empty(t, h.note("inbox/T1.md").Body)

	h.tty = false
	_, code, _, _ = h.run("add", "T2", "given body")
	require.Equal(t, 0, code)
	assert.Equal(t, "given body\n", h.note("inbox/T2.md").Body, "an explicit body wins over a pipe")
}

func TestPipedNothingWithoutATitleIsAnError(t *testing.T) {
	h := newHarness(t)
	h.tty, h.stdin = false, "  \n"
	_, code, _, errOut := h.run("add")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, errOut, "title")
}

func TestClipboard(t *testing.T) {
	h := newHarness(t)
	h.clip = "Copied title\nCopied body\n"
	_, code, out, errOut := h.run("add", "--clipboard")
	require.Equal(t, 0, code, errOut)
	assert.Equal(t, "inbox/Copied title.md\n", out)
	n := h.note("inbox/Copied title.md")
	assert.Equal(t, "Copied body\n", n.Body)
	assert.Equal(t, "adhoc", n.Frontmatter.Get("source").Value)

	_, code, _, _ = h.run("add", "Given", "--clipboard")
	require.Equal(t, 0, code)
	assert.Equal(t, "Copied title\nCopied body\n", h.note("inbox/Given.md").Body, "with a title, the clipboard is the whole body")
}

func TestClipboardBeatsAPipe(t *testing.T) {
	h := newHarness(t)
	h.tty, h.stdin, h.clip = false, "piped", "from clipboard"
	_, code, _, _ := h.run("add", "--clipboard")
	require.Equal(t, 0, code)
	assert.FileExists(t, filepath.Join(h.root, "inbox", "from clipboard.md"))
}

func TestClipboardProblemsAreExplained(t *testing.T) {
	h := newHarness(t)
	h.clipErr = errors.New("exec: \"pbpaste\": executable file not found in $PATH")
	_, code, _, errOut := h.run("add", "--clipboard")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, errOut, "clipboard")
	assert.Contains(t, errOut, "pbpaste")

	h.clipErr, h.clip = nil, "  \n"
	_, code, _, errOut = h.run("add", "--clipboard")
	assert.NotEqual(t, 0, code)
	assert.Contains(t, errOut, "empty")
}

func TestAddNumbersACollidingName(t *testing.T) {
	h := newHarness(t)
	_, _, out1, _ := h.run("add", "Same")
	_, _, out2, _ := h.run("add", "Same")
	assert.Equal(t, "inbox/Same.md\n", out1)
	assert.Equal(t, "inbox/Same 2.md\n", out2)
}

func TestHelpExitsCleanly(t *testing.T) {
	h := newHarness(t)
	handled, code, _, errOut := h.run("add", "-h")
	assert.True(t, handled)
	assert.Equal(t, 0, code)
	assert.Contains(t, errOut, "--kind")
	assert.Zero(t, h.opened)
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestImportInfersTheTypeFromTheExtension(t *testing.T) {
	h := newHarness(t)
	md := writeFile(t, "Plan.md", "---\nkind: proposal\n---\nbody\n")
	txt := writeFile(t, "standup.txt", "Alice: hi\n")

	_, code, out, errOut := h.run("import", md)
	require.Equal(t, 0, code, errOut)
	assert.Equal(t, "inbox/Plan.md\n", out)
	n := h.note("inbox/Plan.md")
	assert.Equal(t, "import", n.Frontmatter.Get("source").Value)
	assert.Equal(t, store.KindProposal, n.Kind)

	_, code, out, errOut = h.run("import", txt)
	require.Equal(t, 0, code, errOut)
	assert.Equal(t, "inbox/standup.md\n", out)
	assert.Equal(t, "transcript", h.note("inbox/standup.md").Frontmatter.Get("source").Value)
}

func TestImportTypeFlag(t *testing.T) {
	h := newHarness(t)
	md := writeFile(t, "Meeting.md", "---\nkind: fact\n---\nraw text\n")
	_, code, _, errOut := h.run("import", "--type", "transcript", md)
	require.Equal(t, 0, code, errOut)
	n := h.note("inbox/Meeting.md")
	assert.Equal(t, store.KindRumor, n.Kind, "a transcript is rumor, whatever the file says")
	assert.Contains(t, n.Body, "raw text")

	other := writeFile(t, "other.txt", "x")
	_, code, _, errOut = h.run("import", "--type", "note", other)
	assert.NotEqual(t, 0, code)
	assert.Contains(t, errOut, "markdown")
}

func TestImportRefusals(t *testing.T) {
	h := newHarness(t)
	ok := writeFile(t, "a.md", "x")
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no file", []string{"import"}, "file"},
		{"two files", []string{"import", ok, ok}, "one file"},
		{"missing file", []string{"import", filepath.Join(t.TempDir(), "nope.md")}, "not found"},
		{"bad type", []string{"import", "--type", "pdf", ok}, "type"},
		{"directory", []string{"import", t.TempDir()}, "regular"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, code, out, errOut := h.run(tt.args...)
			assert.NotEqual(t, 0, code)
			assert.Empty(t, out)
			assert.Contains(t, strings.ToLower(errOut), tt.wantErr)
		})
	}
}

func TestErrorsNeverEchoThePathOfAMissingFile(t *testing.T) {
	h := newHarness(t)
	_, _, _, errOut := h.run("import", "/definitely/secret/place/x.md")
	assert.NotContains(t, errOut, "secret")
}
