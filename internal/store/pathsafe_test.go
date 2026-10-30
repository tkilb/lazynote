package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRoot builds a temp tree: <tmp>/nb is the notebook root (with a real
// projects/ directory and a note), <tmp>/outside is a sibling directory,
// and <tmp>/nb-evil shares the root's name as a prefix.
func newRoot(t *testing.T) (root, tmp string) {
	t.Helper()
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	root = filepath.Join(tmp, "nb")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "projects"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "projects", "a.md"), nil, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "outside"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "nb-evil"), 0o755))
	return root, tmp
}

func TestResolveInRootRejectsLexical(t *testing.T) {
	root, _ := newRoot(t)
	tests := []struct {
		name string
		rel  string
		want error
	}{
		{"empty", "", ErrEmptyName},
		{"dot", ".", ErrEmptyName},
		{"dot slash", "./", ErrEmptyName},
		{"nul byte", "a\x00b.md", ErrNulByte},
		{"absolute", "/etc/passwd", ErrAbsolutePath},
		{"parent only", "..", ErrParentSegment},
		{"leading parent", "../outside/x.md", ErrParentSegment},
		{"embedded parent", "projects/../../outside", ErrParentSegment},
		{"parent that stays inside", "projects/../areas/x.md", ErrParentSegment},
		{"trailing parent", "projects/..", ErrParentSegment},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveInRoot(root, tt.rel)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestResolveInRootAccepts(t *testing.T) {
	root, _ := newRoot(t)
	tests := []struct {
		name string
		rel  string
		want string
	}{
		{"existing note", "projects/a.md", "projects/a.md"},
		{"existing dir", "projects", "projects"},
		{"missing file in existing dir", "projects/new.md", "projects/new.md"},
		{"missing nested dirs", "areas/deep/er/x.md", "areas/deep/er/x.md"},
		{"redundant separators", "projects//a.md", "projects/a.md"},
		{"dot segment", "./projects/./a.md", "projects/a.md"},
		{"name with dots", "projects/..a.md", "projects/..a.md"},
		{"file as parent of missing name", "projects/a.md/x", "projects/a.md/x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveInRoot(root, tt.rel)
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(root, tt.want), got)
		})
	}
}

func TestResolveInRootSymlinks(t *testing.T) {
	root, tmp := newRoot(t)
	link := func(target, name string) {
		t.Helper()
		require.NoError(t, os.Symlink(target, filepath.Join(root, name)))
	}
	link(filepath.Join(tmp, "outside"), "out-dir")
	link(filepath.Join(tmp, "nb-evil"), "evil-sibling")
	link(filepath.Join(tmp, "missing-target"), "dangling")
	link("../../etc", "relative-escape")
	link(filepath.Join(root, "projects"), "inside-link")
	link(root, "self")
	link("loop-b", "loop-a")
	link("loop-a", "loop-b")
	link("projects/a.md", "inside-file-link")
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "outside", "secret.md"), nil, 0o644))
	link(filepath.Join(tmp, "outside", "secret.md"), "file-escape.md")

	rejected := []struct{ name, rel string }{
		{"symlinked dir outside", "out-dir"},
		{"new file below outside link", "out-dir/new.md"},
		{"nested new dirs below outside link", "out-dir/a/b/c.md"},
		{"existing file below outside link", "out-dir/secret.md"},
		{"sibling sharing the root prefix", "evil-sibling/x.md"},
		{"dangling symlink", "dangling"},
		{"new file below dangling symlink", "dangling/x.md"},
		{"relative symlink escape", "relative-escape/x"},
		{"file symlink to outside", "file-escape.md"},
		{"symlink to the root itself", "self"},
	}
	for _, tt := range rejected {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			_, err := ResolveInRoot(root, tt.rel)
			assert.ErrorIs(t, err, ErrEscapesRoot)
		})
	}

	t.Run("allows symlink that stays inside", func(t *testing.T) {
		got, err := ResolveInRoot(root, "inside-link/new.md")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(root, "projects", "new.md"), got)
	})
	t.Run("returns the link itself for an inside symlink", func(t *testing.T) {
		got, err := ResolveInRoot(root, "inside-file-link")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(root, "inside-file-link"), got)
	})
	t.Run("symlink loop gives a sentinel without the path", func(t *testing.T) {
		_, err := ResolveInRoot(root, "loop-a")
		require.ErrorIs(t, err, ErrEscapesRoot)
	})
}

func TestResolveInRootSymlinkedRoot(t *testing.T) {
	root, tmp := newRoot(t)
	alias := filepath.Join(tmp, "alias")
	require.NoError(t, os.Symlink(root, alias))

	got, err := ResolveInRoot(alias, "projects/a.md")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "projects", "a.md"), got)
}

func TestResolveInRootMissingRoot(t *testing.T) {
	_, err := ResolveInRoot(filepath.Join(t.TempDir(), "nope"), "a.md")
	assert.ErrorIs(t, err, ErrUnresolvable)
}

func TestResolveInRootRelativeRoot(t *testing.T) {
	root, _ := newRoot(t)
	t.Chdir(filepath.Dir(root))

	got, err := ResolveInRoot("nb", "projects/a.md")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "projects", "a.md"), got)
}

func TestWithin(t *testing.T) {
	tests := []struct {
		name       string
		root, path string
		want       bool
	}{
		{"same", "/a/nb", "/a/nb", true},
		{"child", "/a/nb", "/a/nb/x", true},
		{"prefix sibling", "/a/nb", "/a/nb-evil", false},
		{"prefix sibling child", "/a/nb", "/a/nb-evil/x", false},
		{"parent", "/a/nb", "/a", false},
		{"filesystem root", "/", "/x", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, within(tt.root, tt.path))
		})
	}
}
