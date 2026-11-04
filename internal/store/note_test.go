package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNote(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		in        string
		wantTitle string
		wantBody  string
		wantFM    bool
		wantErr   bool
	}{
		{"title is the file name", "My Note.md", "---\ntags: [a]\n---\nbody", "My Note", "body", true, false},
		{"a title property never overrides it", "x.md", "---\ntitle: Real Title\n---\nbody", "x", "body", true, false},
		{"blank title property", "x.md", "---\ntitle: ''\n---\n", "x", "", true, false},
		{"non-scalar title property", "x.md", "---\ntitle: [a, b]\n---\n", "x", "", true, false},
		{"no frontmatter", "plain.md", "# Heading\ntext", "plain", "# Heading\ntext", false, false},
		{"invalid yaml hides block and reports", "bad.md", "---\na: [oops\n---\nbody", "bad", "body", false, true},
		{"non-mapping hides block and reports", "list.md", "---\n- a\n---\nbody", "list", "body", false, true},
		{"only the .md suffix is removed", "a.md.md", "text", "a.md", "text", false, false},
		{"extension in any case", "Upper.MD", "text", "Upper", "text", false, false},
		{"dots in the name are kept", "v1.2 notes.md", "text", "v1.2 notes", "text", false, false},
		{"unicode", "résumé 日本.md", "text", "résumé 日本", "text", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := ParseNote(tt.file, []byte(tt.in))
			assert.Equal(t, tt.file, n.Name)
			assert.Equal(t, tt.wantTitle, n.Title)
			assert.Equal(t, tt.wantBody, n.Body)
			assert.Equal(t, tt.wantFM, n.Frontmatter != nil)
			assert.Equal(t, tt.wantErr, n.FrontmatterErr != nil)
		})
	}
}

func TestIsNote(t *testing.T) {
	tests := map[string]bool{
		"a.md": true, "A.MD": true, "a.Md": true, ".md": true,
		"a.txt": false, "md": false, "a.md.bak": false, "": false, "m": false,
	}
	for name, want := range tests {
		assert.Equal(t, want, isNote(name), "%q", name)
	}
}

func TestReadNote(t *testing.T) {
	root, tmp := newRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "projects", "a.md"),
		[]byte("---\ntitle: Alpha\nuid: u1\n---\nHello\n"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(tmp, "outside"), filepath.Join(root, "out")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "big.md"), make([]byte, maxNoteBytes+1), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "max.md"), make([]byte, maxNoteBytes), 0o644))
	s := NewMarkdown(root)

	t.Run("reads and parses", func(t *testing.T) {
		n, err := s.ReadNote("projects/a.md")
		require.NoError(t, err)
		assert.Equal(t, "a.md", n.Name)
		assert.Equal(t, "a", n.Title, "the title is the file name, not the title property")
		assert.Equal(t, "Hello\n", n.Body)
		assert.Equal(t, "u1", n.Frontmatter.Get("uid").Value)
	})
	t.Run("missing note", func(t *testing.T) {
		_, err := s.ReadNote("projects/nope.md")
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("directory is not a note", func(t *testing.T) {
		_, err := s.ReadNote("projects")
		assert.ErrorIs(t, err, ErrUnreadable)
	})
	t.Run("size limit", func(t *testing.T) {
		_, err := s.ReadNote("big.md")
		assert.ErrorIs(t, err, ErrTooLarge)
		_, err = s.ReadNote("max.md")
		assert.NoError(t, err)
	})
	t.Run("path safety applies", func(t *testing.T) {
		for _, rel := range []string{"../outside/x.md", "/etc/passwd", "out/secret.md", ""} {
			_, err := s.ReadNote(rel)
			assert.Error(t, err, "%q", rel)
			assert.NotErrorIs(t, err, ErrNotFound, "%q", rel)
		}
	})
}

func TestParseNoteKeepsTheRawFrontmatter(t *testing.T) {
	tests := []struct {
		name, in string
		has      bool
		raw      string
	}{
		{"a block", "---\nuid: a\nkind: fact\n---\nbody\n", true, "uid: a\nkind: fact"},
		{"an empty block", "---\n---\nbody\n", true, ""},
		{"windows line endings", "---\r\nuid: a\r\n---\r\nbody\r\n", true, "uid: a"},
		{"an invalid block is kept as written", "---\nkey: [oops\n---\nbody\n", true, "key: [oops"},
		{"no block", "just text\n", false, ""},
		{"an unclosed block is not one", "---\nuid: a\nbody\n", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := ParseNote("a.md", []byte(tt.in))
			assert.Equal(t, tt.has, n.HasFrontmatter)
			assert.Equal(t, tt.raw, n.RawFrontmatter)
		})
	}
}
