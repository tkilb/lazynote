package store

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath(t *testing.T) {
	root, _ := newRoot(t)
	s := NewMarkdown(root)

	got, err := s.Path("inbox/a.md")
	require.NoError(t, err)
	want, err := ResolveInRoot(root, "inbox/a.md")
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.True(t, filepath.IsAbs(got))

	for _, bad := range []string{"", "../x.md", "/etc/passwd", "a/../../x.md"} {
		_, err := s.Path(bad)
		assert.Error(t, err, "%q", bad)
	}
}
