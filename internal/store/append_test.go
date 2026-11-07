package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendAddsTextAfterOneBlankLine(t *testing.T) {
	tests := []struct {
		name, before, text, want string
	}{
		{"note with properties", "---\nuid: x\nkind: fact\n---\nbody\n", "## Added\n- a", "---\nuid: x\nkind: fact\n---\nbody\n\n## Added\n- a\n"},
		{"no final line break", "body", "added", "body\n\nadded\n"},
		{"surrounding blank lines of the text are dropped", "body\n", "\n\nadded\n\n\n", "body\n\nadded\n"},
		{"empty note", "", "added", "added\n"},
		{"properties are not reordered or normalized", "---\nzeta: 1\n# keep\nalpha: 2\n---\n", "x", "---\nzeta: 1\n# keep\nalpha: 2\n---\n\nx\n"},
		{"crlf note", "a\r\n", "x", "a\r\n\nx\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := newRoot(t)
			p := filepath.Join(root, "inbox", "n.md")
			put(t, p, tt.before)

			_, err := NewMarkdown(root).Append("inbox/n.md", tt.text)
			require.NoError(t, err)
			assert.Equal(t, tt.want, read(t, p))
		})
	}
}

func TestAppendUndoAndRedo(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	put(t, p, "---\nuid: x\n---\nbody\n")
	act, err := NewMarkdown(root).Append("inbox/n.md", "more")
	require.NoError(t, err)
	assert.Equal(t, "append to n.md", act.Description)

	require.NoError(t, act.Undo())
	assert.Equal(t, "---\nuid: x\n---\nbody\n", read(t, p))
	require.NoError(t, act.Redo())
	assert.Equal(t, "---\nuid: x\n---\nbody\n\nmore\n", read(t, p))
}

func TestAppendUndoRefusesAChangedNote(t *testing.T) {
	root, _ := newRoot(t)
	p := filepath.Join(root, "inbox", "n.md")
	put(t, p, "body\n")
	act, err := NewMarkdown(root).Append("inbox/n.md", "more")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(p, []byte("changed elsewhere, longer\n"), 0o644))
	require.ErrorIs(t, act.Undo(), ErrChanged)
	assert.Equal(t, "changed elsewhere, longer\n", read(t, p))
}

func TestAppendErrors(t *testing.T) {
	root, _ := newRoot(t)
	put(t, filepath.Join(root, "inbox", "n.md"), "body\n")
	put(t, filepath.Join(root, "inbox", "plain.txt"), "x")
	big := filepath.Join(root, "inbox", "big.md")
	put(t, big, "")
	require.NoError(t, os.Truncate(big, maxNoteBytes-1))
	s := NewMarkdown(root)

	tests := []struct {
		name, rel, text string
		err             error
	}{
		{"missing note", "inbox/none.md", "x", ErrNotFound},
		{"not a note", "inbox/plain.txt", "x", ErrUnsupported},
		{"blank text", "inbox/n.md", " \n\t\n", ErrNothingToAppend},
		{"result too large", "inbox/big.md", "more text", ErrTooLarge},
		{"outside the notebook", "../x.md", "x", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Append(tt.rel, tt.text)
			require.Error(t, err)
			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)
			}
		})
	}
	assert.Equal(t, "body\n", read(t, filepath.Join(root, "inbox", "n.md")))
}
