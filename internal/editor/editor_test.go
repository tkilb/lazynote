package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		name     string
		env      []string
		wantArgs []string
	}{
		{"uses EDITOR", []string{"EDITOR=nvim"}, []string{"nvim", "/n/a.md"}},
		{"splits EDITOR into arguments", []string{"EDITOR=code --wait"}, []string{"code", "--wait", "/n/a.md"}},
		{"falls back to vi when unset", []string{"HOME=/h"}, []string{"vi", "/n/a.md"}},
		{"falls back to vi when blank", []string{"EDITOR=   "}, []string{"vi", "/n/a.md"}},
		{"last EDITOR wins like the environment", []string{"EDITOR=a", "EDITOR=b"}, []string{"b", "/n/a.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := Command("/n/a.md", tt.env)
			require.NoError(t, err)
			assert.Equal(t, tt.wantArgs, cmd.Args)
		})
	}
}

func TestCommandRejectsAnEmptyPath(t *testing.T) {
	_, err := Command("", []string{"EDITOR=vi"})
	assert.Error(t, err)
}

func TestCommandRunsTheEditorOnTheFile(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho edited >> \"$1\"\n"), 0o755))
	note := filepath.Join(dir, "a.md")
	require.NoError(t, os.WriteFile(note, []byte("hi\n"), 0o644))

	cmd, err := Command(note, []string{"EDITOR=" + script})
	require.NoError(t, err)
	require.NoError(t, cmd.Run())

	got, err := os.ReadFile(note)
	require.NoError(t, err)
	assert.Equal(t, "hi\nedited\n", string(got))
}
