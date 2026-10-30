package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMoveDestination(t *testing.T) {
	tests := []struct {
		name, rel, tab, want string
	}{
		{"to a tab root", "inbox/a.md", "projects", "projects/a.md"},
		{"from a subfolder lands at the root", "areas/deep/er/a.md", "resources", "resources/a.md"},
		{"to the inbox", "projects/a.md", "inbox", "inbox/a.md"},
		{"to the archives", "projects/a.md", "archives", "archives/a.md"},
		{"a daily goes to archives/daily", "inbox/daily/2026-10-30.md", "archives", "archives/daily/2026-10-30.md"},
		{"a daily elsewhere is plain", "inbox/daily/2026-10-30.md", "projects", "projects/2026-10-30.md"},
		{"a daily-looking name outside inbox/daily is plain", "projects/2026-10-30.md", "archives", "archives/2026-10-30.md"},
		{"a non-date note in inbox/daily is plain", "inbox/daily/notes.md", "archives", "archives/notes.md"},
		{"daily extension case", "inbox/daily/2026-10-30.MD", "archives", "archives/daily/2026-10-30.MD"},
		{"a bad date is not a daily", "inbox/daily/2026-13-45.md", "archives", "archives/2026-13-45.md"},
		{"unknown tab", "inbox/a.md", "elsewhere", ""},
		{"empty", "", "projects", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MoveDestination(tt.rel, tt.tab))
		})
	}
}

func TestDailyDate(t *testing.T) {
	tests := []struct {
		rel  string
		want string
		ok   bool
	}{
		{"inbox/daily/2026-10-30.md", "2026-10-30", true},
		{"archives/daily/2026-10-30.md", "2026-10-30", true},
		{"inbox/daily/2026-10-30.MD", "2026-10-30", true},
		{"inbox/daily/notes.md", "", false},
		{"inbox/daily/2026-13-45.md", "", false},
		{"inbox/2026-10-30.md", "", false},
		{"projects/daily/2026-10-30.md", "", false},
		{"inbox/daily/sub/2026-10-30.md", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			got, ok := DailyDate(tt.rel)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
