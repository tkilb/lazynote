package gitsync

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in     string
		want   Status
		wantOK bool
	}{
		{"2\t3\n", Status{Tracked: true, Ahead: 2, Behind: 3}, true},
		{"0\t0\n", Status{Tracked: true}, true},
		{"", Status{}, false},
		{"1\n", Status{}, false},
		{"a\tb\n", Status{}, false},
		{"-1\t0\n", Status{}, false},
	}
	for _, tt := range tests {
		got, ok := parse(tt.in)
		assert.Equal(t, tt.wantOK, ok, tt.in)
		assert.Equal(t, tt.want, got, tt.in)
	}
}

func TestInSync(t *testing.T) {
	assert.True(t, Status{Tracked: true}.InSync())
	assert.False(t, Status{Tracked: true, Ahead: 1}.InSync())
	assert.False(t, Status{Tracked: true, Behind: 1}.InSync())
	assert.False(t, Status{}.InSync(), "without an upstream nothing is known to be in sync")
}

// git runs a git command in dir with a fixed identity and no user config.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func commit(t *testing.T, dir, name string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644))
	git(t, dir, "add", name)
	git(t, dir, "commit", "-m", name)
}

func TestReadAgainstRealRepositories(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	git(t, base, "init", "--bare", "-b", "main", remote)
	a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
	git(t, base, "clone", remote, a)
	commit(t, a, "one.md")
	git(t, a, "push", "-u", "origin", "HEAD:main")
	git(t, base, "clone", remote, b)

	got, ok := Read(context.Background(), a)
	assert.True(t, ok)
	assert.True(t, got.InSync())

	commit(t, a, "two.md")
	commit(t, a, "three.md")
	require.NoError(t, os.WriteFile(filepath.Join(a, "three.md"), []byte("edited"), 0o644))
	got, ok = Read(context.Background(), a)
	assert.True(t, ok)
	assert.Equal(t, Status{Tracked: true, Ahead: 2, Uncommitted: 1}, got, "uncommitted files count with an upstream too")

	commit(t, b, "other.md")
	git(t, b, "push", "origin", "HEAD:main")
	git(t, a, "fetch")
	got, ok = Read(context.Background(), a)
	assert.True(t, ok)
	assert.Equal(t, Status{Tracked: true, Ahead: 2, Behind: 1, Uncommitted: 1}, got)
}

func TestReadCountsUncommittedFilesWithoutAnUpstream(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	commit(t, repo, "a.md")
	got, ok := Read(context.Background(), repo)
	assert.True(t, ok)
	assert.Equal(t, Status{}, got, "clean, and nothing to compare with")

	require.NoError(t, os.WriteFile(filepath.Join(repo, "a.md"), []byte("changed"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "new", "deep"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "new", "deep", "b.md"), []byte("b"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "new", "c.md"), []byte("c"), 0o644))
	got, _ = Read(context.Background(), repo)
	assert.Equal(t, Status{Uncommitted: 3}, got, "a changed file and two new ones, listed one by one")

	sub := filepath.Join(repo, "new", "deep")
	got, _ = Read(context.Background(), sub)
	assert.Equal(t, Status{Uncommitted: 1}, got, "only the notebook folder counts when it is inside a repository")
}

func TestReadHidesWhatIsNotARepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	plain := t.TempDir()
	_, ok := Read(context.Background(), plain)
	assert.False(t, ok, "not a repository")

	_, ok = Read(context.Background(), filepath.Join(plain, "missing"))
	assert.False(t, ok, "missing folder")
}
