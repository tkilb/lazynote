package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	home := t.TempDir()
	work := filepath.Join(home, "Notes", "work")
	require.NoError(t, os.MkdirAll(work, 0o755))
	file := filepath.Join(home, "afile")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	abs := t.TempDir()

	tests := []struct {
		name string
		yaml string
		want Config
	}{
		{
			name: "tilde expansion, order and default retention",
			yaml: "notebooks:\n  - ~/Notes/work\n  - " + abs + "\n",
			want: Config{
				Notebooks: []Notebook{
					{Path: work, Name: "work", Available: true},
					{Path: abs, Name: filepath.Base(abs), Available: true},
				},
				Retention: Retention{DailyDays: DefaultRetentionDays},
			},
		},
		{
			name: "bare tilde is the home directory",
			yaml: "notebooks: [\"~\"]\n",
			want: Config{
				Notebooks: []Notebook{{Path: home, Name: filepath.Base(home), Available: true}},
				Retention: Retention{DailyDays: DefaultRetentionDays},
			},
		},
		{
			name: "missing path is listed as unavailable",
			yaml: "notebooks:\n  - ~/Notes/gone\n",
			want: Config{
				Notebooks: []Notebook{{Path: filepath.Join(home, "Notes", "gone"), Name: "gone"}},
				Retention: Retention{DailyDays: DefaultRetentionDays},
			},
		},
		{
			name: "a file is unavailable",
			yaml: "notebooks:\n  - ~/afile\n",
			want: Config{
				Notebooks: []Notebook{{Path: file, Name: "afile"}},
				Retention: Retention{DailyDays: DefaultRetentionDays},
			},
		},
		{
			name: "trailing slash is cleaned",
			yaml: "notebooks:\n  - ~/Notes/work/\n",
			want: Config{
				Notebooks: []Notebook{{Path: work, Name: "work", Available: true}},
				Retention: Retention{DailyDays: DefaultRetentionDays},
			},
		},
		{
			name: "explicit retention",
			yaml: "notebooks:\n  - ~/Notes/work\nretention:\n  daily: 7d\n",
			want: Config{
				Notebooks: []Notebook{{Path: work, Name: "work", Available: true}},
				Retention: Retention{DailyDays: 7},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.yaml), home)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseNoNotebooks(t *testing.T) {
	tests := []struct{ name, yaml string }{
		{"empty file", ""},
		{"no notebooks key", "retention:\n  daily: 30d\n"},
		{"empty list", "notebooks: []\n"},
		{"null", "notebooks:\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml), t.TempDir())
			require.ErrorIs(t, err, ErrNoNotebooks)
			assert.Contains(t, err.Error(), ExampleConfig)
		})
	}
}

func TestParseInvalid(t *testing.T) {
	tests := []struct{ name, yaml, wantMsg string }{
		{"relative path", "notebooks:\n  - Notes/work\n", "absolute"},
		{"dot path", "notebooks:\n  - .\n", "absolute"},
		{"blank path", "notebooks:\n  - \"  \"\n", "absolute"},
		{"tilde user form", "notebooks:\n  - ~bob/Notes\n", "absolute"},
		{"malformed yaml", "notebooks: [\n", "parse config"},
		{"notebooks is a string", "notebooks: nope\n", "parse config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml), t.TempDir())
			require.Error(t, err)
			assert.NotErrorIs(t, err, ErrNoNotebooks)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestParseRetention(t *testing.T) {
	tests := []struct {
		daily   string
		want    int
		wantErr bool
	}{
		{"", DefaultRetentionDays, false},
		{"1d", 1, false},
		{"30d", 30, false},
		{"365d", 365, false},
		{"007d", 7, false},
		{"0d", 0, true},
		{"-5d", 0, true},
		{"+5d", 0, true},
		{"30", 0, true},
		{"d", 0, true},
		{"30h", 0, true},
		{"1.5d", 0, true},
		{"30D", 0, true},
		{" 30d", 0, true},
		{"30d ", 0, true},
		{"1w", 0, true},
		{"36500d", 36500, false},
		{"36501d", 0, true},
		{"9999999999d", 0, true},
		{"99999999999999999999d", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.daily, func(t *testing.T) {
			got, err := parseRetention(tt.daily)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "retention.daily")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	nb := filepath.Join(home, "Notes")
	require.NoError(t, os.MkdirAll(nb, 0o755))

	t.Run("reads a file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, os.WriteFile(path, []byte("notebooks:\n  - ~/Notes\n"), 0o600))

		got, err := Load(path)
		require.NoError(t, err)
		assert.Equal(t, []Notebook{{Path: nb, Name: "Notes", Available: true}}, got.Notebooks)
	})
	t.Run("missing file reports no notebooks", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
		require.ErrorIs(t, err, ErrNoNotebooks)
	})
	t.Run("unreadable path is a plain error", func(t *testing.T) {
		_, err := Load(t.TempDir()) // a directory, not a file
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrNoNotebooks)
	})
}

func TestDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := DefaultPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".config", "lazynote", "config.yaml"), got)
}
