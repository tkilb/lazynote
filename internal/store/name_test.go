package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanRenamedName(t *testing.T) {
	tests := []struct {
		name, in string
		isDir    bool
		want     string
		wantErr  bool
	}{
		{"note gets .md", "Plan", false, "Plan.md", false},
		{"note keeps .md", "Plan.md", false, "Plan.md", false},
		{"note keeps .MD", "Plan.MD", false, "Plan.MD", false},
		{"note is trimmed", "  Plan  ", false, "Plan.md", false},
		{"folder name", "alpha", true, "alpha", false},
		{"folder trailing slash is fine", "alpha/", true, "alpha", false},
		{"note with trailing slash is rejected", "plan/", false, "", true},
		{"slash inside", "a/b", false, "", true},
		{"slash inside folder", "a/b", true, "", true},
		{"empty note", "  ", false, "", true},
		{"empty folder", "", true, "", true},
		{"dot", ".", true, "", true},
		{"dotdot", "..", false, "", true},
		{"nul byte", "a\x00b", false, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanRenamedName(tt.in, tt.isDir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
