package loader

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, data, 0o644))
	return p
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name     string
		file     string
		data     string
		wantName string
		wantText string
	}{
		{"text file", "Standup 2026-11-03.txt", "Sam: hello\nAnna: hi\n", "Standup 2026-11-03", "Sam: hello\nAnna: hi\n"},
		{"markdown file", "Confluence Export.md", "# Title\n\nbody", "Confluence Export", "# Title\n\nbody"},
		{"no extension", "slack-thread", "a: b", "slack-thread", "a: b"},
		{"zoom captions", "meeting.vtt", "WEBVTT\n\n00:00 a", "meeting", "WEBVTT\n\n00:00 a"},
		{"only the last extension goes", "notes.v2.md", "x", "notes.v2", "x"},
		{"crlf is kept as it is", "win.txt", "a\r\nb\r\n", "win", "a\r\nb\r\n"},
		{"unicode", "ü.txt", "héllo — 日本語 🙂", "ü", "héllo — 日本語 🙂"},
		{"a byte order mark is dropped", "bom.txt", "\xef\xbb\xbfWEBVTT", "bom", "WEBVTT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := write(t, dir, tt.file, []byte(tt.data))
			got, err := ReadFile(p)
			require.NoError(t, err)
			assert.Equal(t, File, got.Kind)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.wantText, got.Text)
			assert.Empty(t, got.Origin)
		})
	}
}

func TestReadFileExpandsTheHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	write(t, home, "t.txt", []byte("hello"))
	got, err := ReadFile("~/t.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", got.Text)
}

func TestReadFileFollowsASymlinkTheUserTyped(t *testing.T) {
	dir := t.TempDir()
	target := write(t, dir, "real.txt", []byte("real"))
	link := filepath.Join(dir, "link.txt")
	require.NoError(t, os.Symlink(target, link))
	got, err := ReadFile(link)
	require.NoError(t, err)
	assert.Equal(t, "real", got.Text)
}

func TestReadFileFailures(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe")
	require.NoError(t, syscall.Mkfifo(fifo, 0o600))
	tests := []struct {
		name string
		path string
		want error
	}{
		{"missing", filepath.Join(dir, "nope.txt"), ErrNotFound},
		{"empty path", "", ErrNotFound},
		{"nul in the path", "a\x00b", ErrNotFound},
		{"a directory", dir, ErrNotFile},
		{"a named pipe", fifo, ErrNotFile},
		{"too large", write(t, dir, "big.txt", []byte(strings.Repeat("a", MaxBytes+1))), ErrTooLarge},
		{"binary", write(t, dir, "bin.dat", []byte("abc\x00def")), ErrNotText},
		{"not utf-8", write(t, dir, "latin.txt", []byte("caf\xe9")), ErrNotText},
		{"empty file", write(t, dir, "empty.txt", nil), ErrNothingToImport},
		{"only white space", write(t, dir, "blank.txt", []byte(" \n\t\r\n")), ErrNothingToImport},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadFile(tt.path)
			require.ErrorIs(t, err, tt.want)
			assert.NotContains(t, err.Error(), dir, "errors never show a path")
		})
	}
}

func TestReadFileExactlyTheLimitIsAccepted(t *testing.T) {
	p := write(t, t.TempDir(), "limit.txt", []byte(strings.Repeat("a", MaxBytes)))
	got, err := ReadFile(p)
	require.NoError(t, err)
	assert.Len(t, got.Text, MaxBytes)
}
