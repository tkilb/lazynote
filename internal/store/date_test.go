package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDateOnly(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"2026-10-30", "2026-10-30", true},
		{"2026-10-30T14:05:09Z", "2026-10-30", true},
		{"2026-10-30 14:05", "2026-10-30", true},
		{"  2026-10-30  ", "2026-10-30", true},
		{"2026-10-30x", "", false},
		{"2026-1-3", "", false},
		{"yesterday", "", false},
		{"", "", false},
		{"2026-13-45", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := DateOnly(tt.in)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
